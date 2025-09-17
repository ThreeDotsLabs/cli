package trainings

import (
	"context"
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/fatih/color"
)

func (h *Handlers) Skip(ctx context.Context) error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return nil
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)
	exerciseConfig := h.config.ExerciseConfig(trainingRootFs)

	resp, err := h.newGrpcClient().CanSkipExercise(context.Background(), &genproto.CanSkipExerciseRequest{
		TrainingName: h.config.TrainingConfig(trainingRootFs).TrainingName,
		ExerciseId:   exerciseConfig.ExerciseID,
		Token:        h.config.GlobalConfig().Token,
	})
	if err != nil {
		panic(err)
	}

	if !resp.CanSkip {
		fmt.Println(color.New(color.FgYellow).Sprint("You cannot skip this module."))
		return nil
	}

	fmt.Println()
	fmt.Println(`Some modules are optional and you can skip them.

` + color.New(color.Bold, color.FgYellow).Sprint("Before you skip, please keep in mind: ") + `
	- We recommend skipping only modules that you are already familiar with.
	- The example solutions in the following modules may contain code from the skipped module.
	- You can always come back to the skipped module later using "tdl training jump".
`)

	if !internal.ConfirmPromptDefaultYes("skip the current module") {
		fmt.Println("Skipping cancelled")
		return nil
	}

	var skipAll bool
	if resp.CanSkipAllOptional {
		fmt.Println("You can also skip all the remaining optional modules in this training.")
		fmt.Printf("It will let you get the certificate now and you can always come back to the skipped modules later using \"tdl training jump\".\n\n")

		skipAll = internal.ConfirmPromptDefaultYes("skip all the remaining optional modules in this training")
	}

	_, err = h.newGrpcClient().SkipExercise(context.Background(), &genproto.SkipExerciseRequest{
		TrainingName:    h.config.TrainingConfig(trainingRootFs).TrainingName,
		ExerciseId:      exerciseConfig.ExerciseID,
		Token:           h.config.GlobalConfig().Token,
		SkipAllOptional: skipAll,
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
