package trainings

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/manifoldco/promptui"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func (h *Handlers) SelectExercise(ctx context.Context) (string, error) {
	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return "", nil
	}
	trainingRootFs := newTrainingRootFs(trainingRoot)

	currentExerciseID := h.config.ExerciseConfig(trainingRootFs).ExerciseID

	resp, err := h.newGrpcClient(ctx).GetExercises(ctx, &genproto.GetExercisesRequest{
		TrainingName: h.config.TrainingConfig(trainingRootFs).TrainingName,
		Token:        h.config.GlobalConfig().Token,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get solution files: %w", err)
	}

	if len(resp.Modules) == 0 {
		return "", nil
	}

	resp.Modules = append([]*genproto.GetExercisesResponse_Module{
		{
			Id:   "",
			Name: "(exit)",
		},
	}, resp.Modules...)

	for i := range resp.Modules {
		resp.Modules[i].Exercises = append([]*genproto.GetExercisesResponse_Exercise{
			{
				Id:   "",
				Name: "(back)",
			},
		}, resp.Modules[i].Exercises...)
	}

	moduleCursorPos := 0
	exerciseCursorPos := 0

	for i, module := range resp.Modules {
		for j, exercise := range module.Exercises {
			if exercise.Id == currentExerciseID {
				moduleCursorPos = i
				exerciseCursorPos = j
				break
			}
		}
	}

	for {
		moduleSelect := promptui.Select{
			Label:     "Choose module:",
			Items:     resp.Modules,
			Size:      len(resp.Modules),
			CursorPos: moduleCursorPos,
			Templates: &promptui.SelectTemplates{
				Label:    "{{ .Name }}",
				Active:   "{{ .Name | cyan }}",
				Inactive: "{{ .Name }}",
			},
			HideSelected: true,
		}

		index, _, err := moduleSelect.Run()
		if err != nil {
			return "", err
		}

		if index == 0 {
			return "", nil
		}

		if moduleCursorPos != index {
			moduleCursorPos = index
			exerciseCursorPos = 0
		}

		module := resp.Modules[index]

		exerciseSelect := promptui.Select{
			Label:     "Choose exercise:",
			Items:     module.Exercises,
			Size:      len(module.Exercises),
			CursorPos: exerciseCursorPos,
			Templates: &promptui.SelectTemplates{
				Label:    "{{ .Name }}",
				Active:   "{{ .Name | cyan }}",
				Inactive: "{{ .Name }}",
			},
			HideSelected: true,
		}

		index, _, err = exerciseSelect.Run()
		if err != nil {
			return "", err
		}

		if index == 0 {
			continue
		} else {
			fmt.Printf("Selected exercise: %v/%v\n", module.Name, module.Exercises[index].Name)
			return module.Exercises[index].Id, nil
		}
	}
}

func (h *Handlers) FindExercise(ctx context.Context, exerciseID string) (string, error) {
	exerciseID = strings.TrimSpace(exerciseID)

	_, err := uuid.Parse(exerciseID)
	if err == nil {
		return exerciseID, nil
	}

	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return "", nil
	}
	trainingRootFs := newTrainingRootFs(trainingRoot)

	currentExerciseID := h.config.ExerciseConfig(trainingRootFs).ExerciseID

	resp, err := h.newGrpcClient(ctx).GetExercises(ctx, &genproto.GetExercisesRequest{
		TrainingName: h.config.TrainingConfig(trainingRootFs).TrainingName,
		Token:        h.config.GlobalConfig().Token,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get solution files: %w", err)
	}

	if exerciseID == "latest" {
		lastModule := resp.Modules[len(resp.Modules)-1]
		lastExercise := lastModule.Exercises[len(lastModule.Exercises)-1]
		fmt.Printf("Selected exercise: %v/%v\n", lastModule.Name, lastExercise.Name)
		return lastExercise.Id, nil
	}

	targetModule := 0
	targetExercise := 0

	if strings.Contains(exerciseID, "/") {
		parts := strings.Split(exerciseID, "/")
		targetModule = numberFromName(parts[0])
		targetExercise = numberFromName(parts[1])
	} else {
		targetExercise = numberFromName(exerciseID)
	}

	if targetModule == 0 {
		for _, module := range resp.Modules {
			for _, exercise := range module.Exercises {
				if exercise.Id == currentExerciseID {
					targetModule = numberFromName(module.Name)
					break
				}
			}
		}
	}

	for _, module := range resp.Modules {
		if numberFromName(module.Name) == targetModule {
			for _, exercise := range module.Exercises {
				if numberFromName(exercise.Name) == targetExercise {
					fmt.Printf("Selected exercise: %v/%v\n", module.Name, exercise.Name)
					return exercise.Id, nil
				}
			}
		}
	}

	return "", fmt.Errorf("exercise not found")
}

func numberFromName(name string) int {
	parts := strings.Split(name, "-")
	return parseNumber(parts[0])
}

func parseNumber(number string) int {
	num, _ := strconv.Atoi(strings.TrimPrefix(number, "0"))
	return num
}

func (h *Handlers) Jump(ctx context.Context, exerciseID string) error {
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
