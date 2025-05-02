package trainings

import (
	"context"
	"errors"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func (h *Handlers) Skip(ctx context.Context) error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return nil
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)
	exerciseConfig := h.config.ExerciseConfig(trainingRootFs)

	_, err = h.newGrpcClient(ctx).SkipExercise(context.Background(), &genproto.SkipExerciseRequest{
		TrainingName: h.config.TrainingConfig(trainingRootFs).TrainingName,
		ExerciseId:   exerciseConfig.ExerciseID,
		Token:        h.config.GlobalConfig().Token,
	})
	if err != nil {
		panic(err)
	}

	_, err = h.nextExercise(ctx, "", trainingRoot)
	if err != nil {
		panic(err)
	}

	return nil
}
