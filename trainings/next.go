package trainings

import (
	"context"

	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func (h *Handlers) nextExercise(ctx context.Context, currentExerciseID string) (finished bool, err error) {
	h.solutionHintDisplayed = false

	trainingRoot, err := h.config.FindTrainingRoot()
	if err != nil {
		return false, err
	}

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

	if resp.TrainingStatus == genproto.NextExerciseResponse_FINISHED {
		printFinished()
		return true, nil
	}
	if resp.TrainingStatus == genproto.NextExerciseResponse_PAYMENT_REQUIRED {
		printPaymentRequired()
		return false, nil
	}

	if err := h.writeExerciseFiles(resp, trainingRootFs); err != nil {
		return false, err
	}

	if resp.IsTextOnly {
		printTextOnlyExerciseInfo(
			h.config.TrainingConfig(trainingRootFs).TrainingName,
			resp.ExerciseId,
		)
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
			ExerciseID: resp.ExerciseId,
			Directory:  resp.Dir,
			IsTextOnly: resp.IsTextOnly,
		},
	)
}
