package trainings

import (
	"context"
)

func (h *Handlers) Reset(ctx context.Context) error {
	_, err := h.nextExercise(ctx, "")
	if err != nil {
		return err
	}
	return nil
}
