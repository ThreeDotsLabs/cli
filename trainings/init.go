package trainings

import (
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

func (h *Handlers) Init(ctx context.Context, trainingName string, dir string) error {
	logrus.WithFields(logrus.Fields{
		"training_name": trainingName,
	}).Debug("Starting training")

	wd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}
	trainingRootDir := path.Join(wd, dir)

	// trainingRootDir may be different when doing init in already existing workspace
	trainingRootDir, err = h.startTraining(ctx, trainingName, trainingRootDir)
	if errors.Is(err, ErrInterrupted) {
		fmt.Println("Interrupted")
		return nil
	} else if errors.Is(err, ErrTrainingNotFound) {
		fmt.Printf("Training '%v' not found.\nPlease check the valid training name on the website!\n", trainingName)
	} else if err != nil {
		return err
	}

	// todo - handle situation when training was started but something failed here and someone is starting excersise again (because he have no local files)
	_, err = h.nextExercise(ctx, "", trainingRootDir)
	if err != nil {
		return err
	}

	if !isInTrainingRoot(trainingRootDir) {
		relDir, err := filepath.Rel(wd, trainingRootDir)
		if err != nil {
			return errors.Wrap(err, "can't get relative path")
		}

		fmt.Println("\nNow run " + color.CyanString("cd "+relDir+"/") + " to enter the training workspace")
	}

	return nil
}

func isInTrainingRoot(trainingRoot string) bool {
	pwd, err := os.Getwd()
	if err != nil {
		logrus.WithError(err).Warn("Can't get current working directory")
		return false
	}

	absPwd, err := filepath.Abs(pwd)
	if err != nil {
		logrus.WithError(err).Warn("Can't get absolute path of current working directory")
		return false
	}

	absTrainingRoot, err := filepath.Abs(trainingRoot)
	if err != nil {
		logrus.WithError(err).Warn("Can't get absolute path of training root")
		return false
	}

	return absPwd == absTrainingRoot
}

var (
	ErrInterrupted      = errors.New("interrupted")
	ErrTrainingNotFound = errors.New("training not found")
)

func (h *Handlers) startTraining(
	ctx context.Context,
	trainingName string,
	trainingRootDir string,
) (string, error) {
	alreadyExistingTrainingRoot, err := h.config.FindTrainingRoot()
	if err == nil {
		fmt.Println(color.BlueString("Training was already initialised. Training root:" + alreadyExistingTrainingRoot))
		trainingRootDir = alreadyExistingTrainingRoot
	} else if !errors.Is(err, config.TrainingRootNotFoundError) {
		return "", errors.Wrap(err, "can't check if training root exists")
	} else {
		if err := h.showTrainingStartPrompt(trainingRootDir); err != nil {
			return "", err
		}

		// we will create training root in current working directory
		logrus.Debug("No training root yet")
	}

	trainingRootFs := newTrainingRootFs(trainingRootDir)

	if alreadyExistingTrainingRoot != "" {
		cfg := h.config.TrainingConfig(trainingRootFs)
		if cfg.TrainingName != trainingName {
			return "", fmt.Errorf(
				"training %s was already started in this directory, please go to other directory and run `tdl training init`",
				cfg.TrainingName,
			)
		}
	} else {
		err := os.MkdirAll(trainingRootDir, 0755)
		if err != nil {
			return "", errors.Wrap(err, "can't create training root dir")
		}

		err = createGoWorkspace(trainingRootDir, trainingName)
		if err != nil {
			logrus.WithError(err).Warn("Could not create go workspace")
		}
	}

	_, err = h.newGrpcClient(ctx).StartTraining(
		ctxWithRequestHeader(ctx, h.cliMetadata),
		&genproto.StartTrainingRequest{
			TrainingName: trainingName,
			Token:        h.config.GlobalConfig().Token,
		},
	)
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return "", ErrTrainingNotFound
		}
		return "", errors.Wrap(err, "start training gRPC call failed")
	}

	if err := h.config.WriteTrainingConfig(config.TrainingConfig{TrainingName: trainingName}, trainingRootFs); err != nil {
		return "", errors.Wrap(err, "can't write training config")
	}

	if err := writeGitignore(trainingRootFs); err != nil {
		return "", err
	}

	return trainingRootDir, nil
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

func createGoWorkspace(trainingRoot, trainingName string) error {
	cmd := exec.Command("go", "work", "init")
	cmd.Dir = trainingRoot

	printlnCommand(trainingRoot, "go work init")

	out, err := cmd.CombinedOutput()
	if strings.Contains(string(out), "already exists") {
		logrus.Debug("go.work already exists")
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "can't run go work init: %s", string(out))
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

	cmd := exec.Command("go", "work", "use", modulePath)
	cmd.Dir = trainingRoot

	printlnCommand(trainingRoot, fmt.Sprintf("go work use %v", modulePath))

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "can't run go work use")
	}

	return nil
}

func (h *Handlers) showTrainingStartPrompt(trainingDir string) error {
	fmt.Printf(
		"This command will clone training source code to %s directory.\n",
		trainingDir,
	)

	if !internal.ConfirmPromptDefaultYes("continue") {
		return ErrInterrupted
	}

	return nil
}
