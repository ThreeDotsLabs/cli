package trainings

import (
	"context"
	"fmt"
	"net/url"
	"path"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ThreeDotsLabs/cli/trainings/config"
)

func (h *Handlers) Info(ctx context.Context) error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return nil
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)

	trainingConfig := h.config.TrainingConfig(trainingRootFs)
	exerciseConfig := h.config.ExerciseConfig(trainingRootFs)

	exerciseURL, err := url.Parse(internal.WebsiteAddress)
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
