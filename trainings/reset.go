package trainings

import (
	"context"
)

func (h *Handlers) Reset(ctx context.Context) error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if err != nil {
		return err
	}

	_, err = h.nextExercise(ctx, "", trainingRoot)
	if err != nil {
		return err
	}
	return nil
}
