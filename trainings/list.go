package trainings

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
)

func List() error {
	trainings, err := NewGrpcClient(readGlobalConfig().ServerAddr).GetTrainings(context.Background(), &empty.Empty{})
	if err != nil {
		panic(err)
	}

	for _, training := range trainings.Trainings {
		fmt.Println(training.Id)
	}

	return nil
}
