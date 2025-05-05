package trainings

import (
	"context"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

func (h *Handlers) nextExercise(ctx context.Context, currentExerciseID string, trainingRoot string) (finished bool, err error) {
	h.solutionHintDisplayed = false
	clear(h.notifications)

	// We should never trust the remote server.
	// Writing files based on external name is a vector for Path Traversal attack.
	// For more info please check: https://owasp.org/www-community/attacks/Path_Traversal
	//
	// To avoid that we are using afero.BasePathFs with base on training root for all operations in trainings dir.
	trainingRootFs := afero.NewBasePathFs(afero.NewOsFs(), trainingRoot).(*afero.BasePathFs)

	resp, err := h.getNextExercise(ctx, currentExerciseID, trainingRootFs)
	if err != nil {
		return false, err
	}

	return h.setExercise(trainingRootFs, resp, trainingRoot)
}

func (h *Handlers) setExercise(fs *afero.BasePathFs, exercise *genproto.NextExerciseResponse, trainingRoot string) (finished bool, err error) {
	if exercise.TrainingStatus == genproto.NextExerciseResponse_FINISHED {
		printFinished()
		return true, nil
	}
	if exercise.TrainingStatus == genproto.NextExerciseResponse_PAYMENT_REQUIRED {
		printPaymentRequired()
		return false, nil
	}

	h.printCurrentExercise(
		exercise.GetExercise().GetModule().GetName(),
		exercise.GetExercise().GetName(),
	)

	if err := h.writeExerciseFiles(exercise, fs); err != nil {
		return false, err
	}

	if exercise.IsTextOnly {
		printTextOnlyExerciseInfo(
			h.config.TrainingConfig(fs).TrainingName,
			exercise.ExerciseId,
		)
	} else {
		err = addModuleToWorkspace(trainingRoot, exercise.Dir)
		if err != nil {
			logrus.WithError(err).Warn("Failed to add module to workspace")
		}
	}

	return false, nil
}

func (h *Handlers) getNextExercise(
	ctx context.Context,
	currentExerciseID string,
	trainingRootFs *afero.BasePathFs,
) (resp *genproto.NextExerciseResponse, err error) {
	resp, err = h.newGrpcClient(ctx).NextExercise(
		ctxWithRequestHeader(ctx, h.cliMetadata),
		&genproto.NextExerciseRequest{
			TrainingName:      h.config.TrainingConfig(trainingRootFs).TrainingName,
			CurrentExerciseId: currentExerciseID,
			Token:             h.config.GlobalConfig().Token,
		},
	)

	logrus.WithFields(logrus.Fields{
		"resp": resp,
		"err":  err,
	}).Debug("Received exercise from server")

	return resp, err
}

func (h *Handlers) writeExerciseFiles(resp *genproto.NextExerciseResponse, trainingRootFs *afero.BasePathFs) error {
	if resp.Dir == "" {
		return errors.New("exercise dir is empty")
	}
	if resp.ExerciseId == "" {
		return errors.New("exercise id is empty")
	}

	if err := files.NewFiles().WriteExerciseFiles(resp.FilesToCreate, trainingRootFs, resp.Dir); err != nil {
		return err
	}

	return h.config.WriteExerciseConfig(
		trainingRootFs,
		config.ExerciseConfig{
			ExerciseID:  resp.ExerciseId,
			Directory:   resp.Dir,
			IsTextOnly:  resp.IsTextOnly,
			IsSkippable: resp.IsSkippable,
		},
	)
}
