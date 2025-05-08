package trainings

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
)

func (h *Handlers) List(ctx context.Context) error {
	trainings, err := h.newGrpcClient().GetTrainings(context.Background(), &empty.Empty{})
	if err != nil {
		panic(err)
	}

	for _, training := range trainings.Trainings {
		fmt.Println(training.Id)
	}

	return nil
}
