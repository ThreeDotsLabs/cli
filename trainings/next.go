package trainings

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

func (h *Handlers) nextExercise(ctx context.Context, currentExerciseID string, trainingRoot string) (finished bool, err error) {
	return h.nextExerciseWithSkipped(ctx, currentExerciseID, trainingRoot, nil)
}

func (h *Handlers) nextExerciseWithSkipped(ctx context.Context, currentExerciseID string, trainingRoot string, skipExerciseIDs []string) (finished bool, err error) {
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

	writeFiles := true
	for _, skipExerciseID := range skipExerciseIDs {
		if resp.ExerciseId == skipExerciseID {
			// Exercise already has a local solution, don't overwrite files
			writeFiles = false
			break
		}
	}

	return h.setExercise(trainingRootFs, resp, trainingRoot, writeFiles)
}

func (h *Handlers) setExercise(fs *afero.BasePathFs, exercise *genproto.NextExerciseResponse, trainingRoot string, writeFiles bool) (finished bool, err error) {
	if exercise.TrainingStatus == genproto.NextExerciseResponse_FINISHED {
		printFinished()
		return true, nil
	}
	if exercise.TrainingStatus == genproto.NextExerciseResponse_COHORT_BATCH_DONE {
		var date *time.Time
		if exercise.GetNextBatchDate() != nil {
			t := exercise.GetNextBatchDate().AsTime()
			date = &t
		}

		printCohortBatchDone(date)
		return true, nil
	}
	if exercise.TrainingStatus == genproto.NextExerciseResponse_PAYMENT_REQUIRED {
		printPaymentRequired()
		return false, nil
	}

	if exercise.GetExercise() != nil {
		h.printCurrentExercise(
			exercise.GetExercise().GetModule().GetName(),
			exercise.GetExercise().GetName(),
		)
	}

	if writeFiles {
		// In the relaxed mode, we clone the complete example solutions, so it makes sense to delete unused files
		isEasy := exercise.TrainingDifficulty == genproto.TrainingDifficulty_EASY
		f := files.NewFilesWithConfig(isEasy, isEasy)
		if err := h.writeExerciseFiles(f, nextExerciseResponseToExerciseSolution(exercise), fs); err != nil {
			return false, err
		}
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
	resp, err = h.newGrpcClient().NextExercise(
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

func (h *Handlers) writeExerciseFiles(files files.Files, resp *genproto.ExerciseSolution, trainingRootFs *afero.BasePathFs) error {
	if resp.Dir == "" {
		return errors.New("exercise dir is empty")
	}
	if resp.ExerciseId == "" {
		return errors.New("exercise id is empty")
	}

	if err := files.WriteExerciseFiles(resp.Files, trainingRootFs, resp.Dir); err != nil {
		return err
	}

	return h.config.WriteExerciseConfig(
		trainingRootFs,
		config.ExerciseConfig{
			ExerciseID: resp.ExerciseId,
			Directory:  resp.Dir,
			IsTextOnly: resp.IsTextOnly,
			IsOptional: resp.IsOptional,
		},
	)
}

func nextExerciseResponseToExerciseSolution(resp *genproto.NextExerciseResponse) *genproto.ExerciseSolution {
	return &genproto.ExerciseSolution{
		ExerciseId: resp.ExerciseId,
		Dir:        resp.Dir,
		Files:      resp.FilesToCreate,
		IsTextOnly: resp.IsTextOnly,
		IsOptional: resp.IsOptional,
	}
}
