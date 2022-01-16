package trainings

import (
	"context"
	"fmt"
	"net/url"
	"path"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/web"
)

func (h *Handlers) Info(ctx context.Context) error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		fmt.Println("You are not in a training directory. If you already started the training, please go to the exercise directory.")
		fmt.Printf("Please run %s if you didn't start training yet.\n", internal.SprintCommand("tdl training init"))
		return nil
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)

	trainingConfig := h.config.TrainingConfig(trainingRootFs)
	exerciseConfig := h.config.ExerciseConfig(trainingRootFs)

	exerciseURL, err := url.Parse(web.Website)
	if err != nil {
		logrus.WithError(err).Warn("Can't parse website URL")
	}
	exerciseURL.Path = path.Join("trainings/" + trainingConfig.TrainingName + "/exercise/" + exerciseConfig.ExerciseID)

	fmt.Println("### Training")
	fmt.Println("Name:", color.CyanString(trainingConfig.TrainingName))
	fmt.Println("Root dir:", color.CyanString(trainingRoot))
	fmt.Println()

	fmt.Println("### Current exercise")
	fmt.Println("ID:", color.CyanString(exerciseConfig.ExerciseID))
	fmt.Println("Files:", color.CyanString(h.generateRunTerminalPath(trainingRootFs)))
	fmt.Println("Content:", color.CyanString(exerciseURL.String()))

	return nil
}
