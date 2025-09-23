package trainings

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

// restore restores all solution files for the training in the given directory.
func (h *Handlers) restore(ctx context.Context, trainingRoot string) error {
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

	for _, exercise := range resp.Solutions {
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
