package trainings

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/ThreeDotsLabs/cli/tdl/internal"
	"github.com/ThreeDotsLabs/cli/tdl/trainings/genproto"
	"github.com/fatih/color"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const trainingConfigFile = ".tdl-training"

type trainingConfig struct {
	TrainingName string
}

var trainingRootNotFoundError = errors.New("training root not found")

func Init(trainingName string) error {
	logrus.WithField("training_name", trainingName).Debug("Starting training")

	err := startTraining(trainingName)
	if err != nil {
		// todo - handle it nicer
		panic(err)
	}

	// todo - handle situation when training was started but something failed here and someone is starting excersise again (because he have no local files)
	nextExercise("")

	return nil
}

func nextExercise(currentExerciseID string) {
	trainingRoot, err := findTrainingRoot()
	if err != nil {
		panic(err)
	}

	trainingConfig := readTrainingConfig()

	logrus.WithFields(logrus.Fields{
		"training_name": trainingConfig.TrainingName,
		"training_root": trainingRoot,
	}).Debug("Starting exercise")

	// todo - dedup?
	conn, err := grpc.Dial(readGlobalConfig().ServerAddr, grpc.WithInsecure())
	if err != nil {
		// todo
		panic(err)
	}
	client := genproto.NewServerClient(conn)

	resp, err := client.NextExercise(context.Background(), &genproto.NextExerciseRequest{
		TrainingName:      trainingConfig.TrainingName,
		CurrentExerciseId: currentExerciseID,
		Token:             readGlobalConfig().Token,
	})
	if status.Code(err) == codes.NotFound {
		trainingFinished()
		return
	}
	if err != nil {
		panic(err)
	}

	logrus.WithFields(logrus.Fields{
		"resp": resp,
	}).Debug("Received exercise from server")

	// todo - validate if resp.GetDir() or anything is empty!

	expectedDir := path.Join(trainingRoot, resp.GetDir())

	if !fileExists(expectedDir) {
		if err := os.MkdirAll(expectedDir, 0755); err != nil {
			panic(err)
		}
	}
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	relExpectedDir, err := filepath.Rel(pwd, expectedDir)
	if err != nil {
		panic(err)
	}

	requireCd := pwd != expectedDir

	for _, file := range resp.GetFilesToCreate() {
		filePath := path.Join(expectedDir, file.Path, file.Name)

		if !shouldWriteFile(filePath, file) {
			continue
		}

		// todo - this needs to be escaped very good!!!!!!!! sec!!!
		f, err := os.Create(filePath)
		if err != nil {
			panic(err)
		}

		if _, err := f.WriteString(file.Content); err != nil {
			panic(err)
		}
	}

	writeConfigToml(path.Join(expectedDir, ExerciseConfigFile), ExerciseConfig{
		ExerciseID:   resp.GetExerciseId(),
		TrainingName: trainingConfig.TrainingName,
	})

	if requireCd {
		fmt.Printf("Exercise files were created in '%s' directory.\n", relExpectedDir)
		fmt.Println("Please execute", color.CyanString("cd "+relExpectedDir), "to get there.")
	}

	fmt.Printf("\nPlase go to http://localhost:3002/about see exercise content.\n")
	fmt.Printf("To run solution, please execute " + color.CyanString("tdl training run"))
	if requireCd {
		fmt.Print(" in ", relExpectedDir)
	}
	fmt.Println()
}

func trainingFinished() {
	// todo - some CTA here
	fmt.Println("Congratulations, you finished the training " + color.YellowString("🏆"))
	return
}

func writeConfigToml(destPath string, v interface{}) {
	// todo - verify security
	err := os.MkdirAll(path.Dir(destPath), 0700)
	if err != nil {
		panic(err)
	}

	// todo - verify security
	f, err := os.OpenFile(destPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(err)
	}

	if err := toml.NewEncoder(f).Encode(v); err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
}

func startTraining(trainingName string) error {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	if !internal.ConfirmPromptDefaultYes(fmt.Sprintf("This command will clone training source code to %s directory. Do you want to continue?", pwd)) {
		// todo - better handling
		return errors.New("interrupted")
	}

	// todo - dedup?
	conn, err := grpc.Dial(readGlobalConfig().ServerAddr, grpc.WithInsecure())
	if err != nil {
		// todo
		panic(err)
	}
	client := genproto.NewServerClient(conn)

	_, err = client.StartTraining(context.Background(), &genproto.StartTrainingRequest{
		TrainingName: trainingName, // todo - it should be some kind of uuid
		Token:        readGlobalConfig().Token,
	})
	if err != nil {
		panic(err)
	}

	trainingRoot, err := findTrainingRoot()
	if errors.Is(err, trainingRootNotFoundError) {
		logrus.WithField("training_root", trainingRoot).Debug("Creating training root")

		trainingRoot = pwd
		f, err := os.Create(path.Join(trainingRoot, trainingConfigFile))
		if err != nil {
			panic(err)
		}

		if err := toml.NewEncoder(f).Encode(trainingConfig{
			TrainingName: trainingName,
		}); err != nil {
			panic(err)
		}

		// todo - put some content
		if err := f.Close(); err != nil {
			panic(err)
		}
	} else {
		logrus.WithField("training_root", trainingRoot).Debug("Found training root")
		fmt.Println("Training was already started. Training root:", pwd)

		cfg := readTrainingConfig()

		if cfg.TrainingName != trainingName {
			return fmt.Errorf("training %s was already started in this directory", cfg.TrainingName)
		}

		return nil
	}

	return nil
}

func readTrainingConfig() trainingConfig {
	// todo - it would be nice to not read it every time
	trainingRoot, err := findTrainingRoot()
	if err != nil {
		panic(err)
	}

	config := trainingConfig{}
	if _, err := toml.DecodeFile(path.Join(trainingRoot, trainingConfigFile), &config); err != nil {
		// todo - better handling
		panic(err)
	}

	logrus.WithField("training_config", config).Debug("Training config")

	return config
}

func shouldWriteFile(filePath string, file *genproto.File) bool {
	// todo - next!
	if !fileExists(filePath) {
		return true
	}

	// todo - test it

	actualContent, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	if string(actualContent) == file.Content {
		logrus.Debug("File %s already exists, skipping\n", filePath)
		return false
	}

	fmt.Printf("\nFile %s already exists, diff:\n", filePath)

	edits := myers.ComputeEdits(span.URIFromPath("local "+file.Name), string(actualContent), file.Content)
	diff := fmt.Sprint(gotextdiff.ToUnified("local "+file.Name, "remote "+file.Name, string(actualContent), edits))
	fmt.Println(diff)

	if !internal.ConfirmPrompt("Should it be overridden?") {
		fmt.Println("Skipping file")
		return false
	} else {
		return true
	}
}

func findTrainingRoot() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := pwd

	for {
		if fileExists(path.Join(dir, trainingConfigFile)) {
			return dir, nil
		}

		dir = path.Dir(dir)
		if dir == "/" {
			break
		}
	}

	return "", trainingRootNotFoundError
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}

	panic(err)
}
