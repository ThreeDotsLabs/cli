package trainings

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

// fetchGoldenFiles wraps GetGoldenSolution with the standard error path.
// Used by overrideWithGolden ('s'), g handlers, and reset — every caller that
// needs the example-solution file list for a specific exercise.
func (h *Handlers) fetchGoldenFiles(
	ctx context.Context,
	trainingName, exerciseID, token string,
) ([]*genproto.File, error) {
	resp, err := h.newGrpcClient().GetGoldenSolution(ctx, &genproto.GetGoldenSolutionRequest{
		TrainingName: trainingName,
		ExerciseId:   exerciseID,
		Token:        token,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch example solution for exercise %s: %w", exerciseID, err)
	}
	return resp.Files, nil
}

// resolvePreviousExercise returns the ID and module/exercise path of the exercise
// immediately preceding currentExerciseID in the training's order. Returns
// ("", "", nil) when currentExerciseID is the first exercise.
//
// One GetExercises call per invocation — callers may cache the result per-command
// if they need it more than once, but the common case (single reset / single 'g')
// only needs it once.
func (h *Handlers) resolvePreviousExercise(
	ctx context.Context,
	trainingName, token, currentExerciseID string,
) (prevExerciseID, prevModuleExercisePath string, err error) {
	resp, err := h.newGrpcClient().GetExercises(ctx, &genproto.GetExercisesRequest{
		TrainingName: trainingName,
		Token:        token,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to list exercises: %w", err)
	}

	type flatExercise struct {
		id                 string
		moduleExercisePath string
	}
	var flat []flatExercise
	for _, module := range resp.Modules {
		for _, exercise := range module.Exercises {
			flat = append(flat, flatExercise{
				id:                 exercise.Id,
				moduleExercisePath: module.Name + "/" + exercise.Name,
			})
		}
	}

	for i, e := range flat {
		if e.id == currentExerciseID {
			if i == 0 {
				return "", "", nil // first exercise — no predecessor
			}
			return flat[i-1].id, flat[i-1].moduleExercisePath, nil
		}
	}

	return "", "", fmt.Errorf("exercise %s not found in training %s", currentExerciseID, trainingName)
}
