package trainings

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func (h *Handlers) Init(ctx context.Context, trainingName string) error {
	wd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}

	logrus.WithFields(logrus.Fields{
		"training_name": trainingName,
		"dir":           wd,
	}).Debug("Starting training")

	if err := h.startTraining(ctx, trainingName, wd); errors.Is(err, ErrInterrupted) {
		fmt.Println("Interrupted")
		return nil
	} else if err != nil {
		return err
	}

	// todo - handle situation when training was started but something failed here and someone is starting excersise again (because he have no local files)
	return h.nextExercise(ctx, "", wd)
}

var ErrInterrupted = errors.New("interrupted")

func (h *Handlers) startTraining(ctx context.Context, trainingName string, dir string) error {
	if err := h.showTrainingStartPrompt(); err != nil {
		return err
	}

	alreadyExistingTrainingRoot, err := h.config.FindTrainingRoot(dir)
	if err != nil {
		if errors.Is(err, config.TrainingRootNotFoundError) {
			logrus.Debug("No training root yet")
		} else {
			return errors.Wrap(err, "can't check if training root exists")
		}
	} else {
		fmt.Println(color.BlueString("Training was already started. Training root:" + alreadyExistingTrainingRoot))

		cfg := h.config.TrainingConfig(dir)
		if cfg.TrainingName != trainingName {
			return fmt.Errorf("training %s was already started in this directory", cfg.TrainingName)
		}

		// training root already exists, let's use it
		dir = alreadyExistingTrainingRoot
	}

	_, err = h.newGrpcClient(ctx).StartTraining(context.Background(), &genproto.StartTrainingRequest{
		TrainingName: trainingName,
		Token:        h.config.GlobalConfig().Token,
	})
	if err != nil {
		return errors.Wrap(err, "start training gRPC call failed")
	}

	return h.config.WriteTrainingConfig(config.TrainingConfig{TrainingName: trainingName}, dir)
}

func (h *Handlers) checkIfTrainingWasAlreadyStarted(trainingName string, dir string) (bool, error) {
	if trainingRoot, err := h.config.FindTrainingRoot(dir); err == nil {
		fmt.Println("Training was already started. Training root:", trainingRoot)

		cfg := h.config.TrainingConfig(dir)
		if cfg.TrainingName != trainingName {
			return true, fmt.Errorf("training %s was already started in this directory", cfg.TrainingName)
		}

		return true, nil
	} else if !errors.Is(err, config.TrainingRootNotFoundError) {
		return false, errors.Wrap(err, "can't check if training root exists")
	}

	return false, nil
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
