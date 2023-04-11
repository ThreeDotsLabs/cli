package trainings

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/fatih/color"
	"github.com/pkg/errors"
)

func (h *Handlers) Info(ctx context.Context) error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return nil
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)

	trainingConfig := h.config.TrainingConfig(trainingRootFs)
	exerciseConfig := h.config.ExerciseConfig(trainingRootFs)

	fmt.Println("### Training")
	fmt.Println("Name:", color.CyanString(trainingConfig.TrainingName))
	fmt.Println("Root dir:", color.CyanString(trainingRoot))
	fmt.Println()

	fmt.Println("### Current exercise")
	fmt.Println("ID:", color.CyanString(exerciseConfig.ExerciseID))
	fmt.Println("Files:", color.CyanString(h.generateRunTerminalPath(trainingRootFs)))

	exerciseURL := internal.ExerciseURL(trainingConfig.TrainingName, exerciseConfig.ExerciseID)
	fmt.Println("Content:", color.CyanString(exerciseURL))

	return nil
}
