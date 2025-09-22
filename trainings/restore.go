package trainings

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *Handlers) restore(ctx context.Context) error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return nil
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)

	resp, err := h.newGrpcClient().GetAllSolutionFiles(ctx, &genproto.GetAllSolutionFilesRequest{
		TrainingName: h.config.TrainingConfig(trainingRootFs).TrainingName,
		Token:        h.config.GlobalConfig().Token,
	})
	if err != nil {
		return fmt.Errorf("failed to get all solution files: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"resp": resp,
		"err":  err,
	}).Debug("Received solutions from server")

	for _, exercise := range resp.Exercises {
		fmt.Println(color.New(color.Bold, color.FgYellow).Sprint("\nRestoring exercise:"), exercise.Exercise.Module.Name, "/", exercise.Exercise.Name)

		if err := h.writeExerciseFiles(files.NewFilesWithConfig(false, true), exercise, trainingRootFs); err != nil {
			return err
		}

		err = addModuleToWorkspace(trainingRoot, exercise.Dir)
		if err != nil {
			logrus.WithError(err).Warn("Failed to add module to workspace")
		}
	}

	return nil
}
