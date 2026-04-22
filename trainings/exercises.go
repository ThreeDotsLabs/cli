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
