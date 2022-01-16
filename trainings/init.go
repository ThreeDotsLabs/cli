package trainings

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func (h *Handlers) Init(ctx context.Context, trainingName string) error {
	logrus.WithFields(logrus.Fields{
		"training_name": trainingName,
	}).Debug("Starting training")

	if err := h.startTraining(ctx, trainingName); errors.Is(err, ErrInterrupted) {
		fmt.Println("Interrupted")
		return nil
	} else if err != nil {
		return err
	}

	// todo - handle situation when training was started but something failed here and someone is starting excersise again (because he have no local files)
	return h.nextExercise(ctx, "")
}

var ErrInterrupted = errors.New("interrupted")

func (h *Handlers) startTraining(ctx context.Context, trainingName string) error {
	var trainingRoot string

	alreadyExistingTrainingRoot, err := h.config.FindTrainingRoot()
	if err == nil {
		fmt.Println(color.BlueString("Training was already initialised. Training root:" + alreadyExistingTrainingRoot))
		trainingRoot = alreadyExistingTrainingRoot
	} else if !errors.Is(err, config.TrainingRootNotFoundError) {
		return errors.Wrap(err, "can't check if training root exists")
	} else {
		if err := h.showTrainingStartPrompt(); err != nil {
			return err
		}

		wd, err := os.Getwd()
		if err != nil {
			return errors.WithStack(err)
		}

		// we will create training root in current working directory
		trainingRoot = wd
		logrus.Debug("No training root yet")
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)

	if alreadyExistingTrainingRoot != "" {
		cfg := h.config.TrainingConfig(trainingRootFs)
		if cfg.TrainingName != trainingName {
			return fmt.Errorf(
				"training %s was already started in this directory, please go to other directory and run `tdl training init`",
				cfg.TrainingName,
			)
		}
	}

	_, err = h.newGrpcClient(ctx).StartTraining(ctx, &genproto.StartTrainingRequest{
		TrainingName: trainingName,
		Token:        h.config.GlobalConfig().Token,
	})
	if err != nil {
		return errors.Wrap(err, "start training gRPC call failed")
	}

	return h.config.WriteTrainingConfig(config.TrainingConfig{TrainingName: trainingName}, trainingRootFs)
}

func (h *Handlers) showTrainingStartPrompt() error {
	pwd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "can't get wd")
	}

	msg := fmt.Sprintf(
		"This command will clone training source code to %s directory. Do you want to continue?",
		pwd,
	)

	if !internal.ConfirmPromptDefaultYes(msg) {
		return ErrInterrupted
	}

	return nil
}
