package trainings

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

func (h *Handlers) Clone(ctx context.Context, executionID string, directory string) error {
	resp, err := h.newGrpcClient().GetSolutionFiles(ctx, &genproto.GetSolutionFilesRequest{
		ExecutionId: executionID,
	})
	if err != nil {
		return fmt.Errorf("failed to get solution files: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"resp": resp,
		"err":  err,
	}).Debug("Received exercise from server")

	absoluteDirToClone, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}

	absoluteDirToClone = path.Join(absoluteDirToClone, directory)

	if _, err := h.startTraining(ctx, resp.TrainingName, absoluteDirToClone); err != nil {
		return err
	}

	trainingRootFs := afero.NewBasePathFs(afero.NewOsFs(), absoluteDirToClone).(*afero.BasePathFs)

	if err := h.config.WriteTrainingConfig(config.TrainingConfig{TrainingName: resp.TrainingName}, trainingRootFs); err != nil {
		return errors.Wrap(err, "can't write training config")
	}

	files := &genproto.NextExerciseResponse{
		TrainingStatus: genproto.NextExerciseResponse_IN_PROGRESS,
		Dir:            resp.Dir,
		ExerciseId:     resp.ExerciseId,
		FilesToCreate:  resp.FilesToCreate,
		IsTextOnly:     false,
	}

	if err := h.writeExerciseFiles(files, trainingRootFs); err != nil {
		return err
	}

	err = addModuleToWorkspace(absoluteDirToClone, resp.Dir)
	if err != nil {
		logrus.WithError(err).Warn("Failed to add module to workspace")
	}

	return nil
}
