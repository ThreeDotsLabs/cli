package trainings

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/fatih/color"
	"github.com/spf13/afero"

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
	_, err := h.nextExercise(ctx, "")
	if err != nil {
		return err
	}

	fmt.Println("To see exercise content, please go back to " + color.CyanString(internal.WebsiteAddress))
	return nil
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
	} else {
		_ = createGoWorkspace(trainingRoot)
	}

	_, err = h.newGrpcClient(ctx).StartTraining(
		ctxWithRequestHeader(ctx, h.cliMetadata),
		&genproto.StartTrainingRequest{
			TrainingName: trainingName,
			Token:        h.config.GlobalConfig().Token,
		},
	)
	if err != nil {
		return errors.Wrap(err, "start training gRPC call failed")
	}

	if err := h.config.WriteTrainingConfig(config.TrainingConfig{TrainingName: trainingName}, trainingRootFs); err != nil {
		return errors.Wrap(err, "can't write training config")
	}

	if err := writeGitignore(trainingRootFs); err != nil {
		return err
	}

	return nil
}

var gitignore = strings.Join(
	[]string{
		"# Exercise content is subject to Three Dots Labs' copyright.",
		"**/" + files.ExerciseFile,
		"",
	},
	"\n",
)

func writeGitignore(trainingRootFs *afero.BasePathFs) error {
	if !files.DirOrFileExists(trainingRootFs, ".gitignore") {
		f, err := trainingRootFs.Create(".gitignore")
		if err != nil {
			return errors.Wrap(err, "can't create .gitignore")
		}

		if _, err := f.Write([]byte(gitignore)); err != nil {
			return errors.Wrap(err, "can't write .gitignore")
		}
	}

	return nil
}

func createGoWorkspace(trainingRoot string) error {
	cmd := exec.Command("go", "work", "init")
	cmd.Dir = trainingRoot

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "can't run go work init")
	}

	return nil
}

func hasGoWorkspace(trainingRoot string) bool {
	_, err := os.Stat(path.Join(trainingRoot, "go.work"))
	return err == nil
}

func addModuleToWorkspace(trainingRoot string, modulePath string) error {
	if !hasGoWorkspace(trainingRoot) {
		return nil
	}

	cmd := exec.Command("go", "work", "use", ".")
	cmd.Dir = path.Join(trainingRoot, modulePath)

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "can't run go work use")
	}

	return nil
}

func (h *Handlers) showTrainingStartPrompt() error {
	pwd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "can't get wd")
	}

	fmt.Printf(
		"This command will clone training source code to %s directory.\n",
		pwd,
	)

	if !internal.ConfirmPromptDefaultYes("continue") {
		return ErrInterrupted
	}

	return nil
}
