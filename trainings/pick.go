package trainings

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/pkg/errors"
)

func (h *Handlers) SelectExercise(ctx context.Context) (string, error) {
	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return "", nil
	}
	trainingRootFs := newTrainingRootFs(trainingRoot)

	resp, err := h.newGrpcClient(ctx).GetExercises(ctx, &genproto.GetExercisesRequest{
		TrainingName: h.config.TrainingConfig(trainingRootFs).TrainingName,
		Token:        h.config.GlobalConfig().Token,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get solution files: %w", err)
	}

	for _, m := range resp.Modules {
		fmt.Println(m.Name)
		for _, e := range m.Exercises {
			fmt.Printf("  %s\n", e.Name)
		}
	}

	return "", nil
}

func (h *Handlers) Pick(ctx context.Context, exerciseID string) error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return nil
	}
	trainingRootFs := newTrainingRootFs(trainingRoot)

	resp, err := h.newGrpcClient(ctx).GetExercise(ctx, &genproto.GetExerciseRequest{
		TrainingName: h.config.TrainingConfig(trainingRootFs).TrainingName,
		Token:        h.config.GlobalConfig().Token,
		ExerciseId:   exerciseID,
	})
	if err != nil {
		return fmt.Errorf("failed to get exercise: %w", err)
	}

	_, err = h.setExercise(trainingRootFs, resp, trainingRoot)
	return err
}
