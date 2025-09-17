package trainings

import (
	"context"
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
)

const (
	actionSkipAll     = "Skip all remaining optional modules"
	actionSkipCurrent = "Skip the current module"
	actionCancel      = "(cancel)"
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
		return err
	}

	if !resp.CanSkip {
		fmt.Println(color.New(color.Bold, color.FgYellow).Sprint("You cannot skip this module."))
		return nil
	}

	fmt.Println()
	fmt.Println(`Some modules are optional and you can skip them.

` + color.New(color.Bold, color.FgYellow).Sprint("Before you skip, please keep in mind: ") + `
	- We recommend skipping only modules that you are already familiar with.
	- The example solutions in the following modules may contain code from the skipped module.
	- You can always come back to the skipped module later using "tdl training jump".
`)

	actions := []string{actionSkipCurrent, actionCancel}

	if resp.CanSkipAllOptional {
		actions = append([]string{actionSkipAll}, actions...)

		fmt.Println(color.New(color.Bold, color.FgYellow).Sprint("\nYou can also skip all the remaining optional modules in this training."))
		fmt.Printf("It will let you get the certificate now and you can always come back to the skipped modules later.\n\n")
	}

	moduleSelect := promptui.Select{
		Label: "Choose what to do",
		Items: actions,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "{{ . | cyan }}",
			Inactive: "{{ . }}",
		},
		HideSelected: true,
	}

	_, choice, err := moduleSelect.Run()
	if err != nil {
		fmt.Println("Skipping cancelled")
		return err
	}

	var skipAll bool
	if choice == actionSkipAll {
		fmt.Println("Skipping all remaining optional modules.")
		skipAll = true
	} else if choice == actionSkipCurrent {
		fmt.Println("Skipping current module.")
		skipAll = false
	} else {
		fmt.Println("Skipping cancelled")
		return nil
	}

	_, err = h.newGrpcClient().SkipExercise(context.Background(), &genproto.SkipExerciseRequest{
		TrainingName:    h.config.TrainingConfig(trainingRootFs).TrainingName,
		ExerciseId:      exerciseConfig.ExerciseID,
		Token:           h.config.GlobalConfig().Token,
		SkipAllOptional: skipAll,
	})
	if err != nil {
		return err
	}

	_, err = h.nextExercise(ctx, "", trainingRoot)
	if err != nil {
		return err
	}

	return nil
}
