package course

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/ThreeDotsLabs/cli/course/genproto"
	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// todo - separate file?
// todo - think if it's good enough to be final (no backward compabilityu!)
type ExerciseConfig struct {
	ExerciseID string `toml:"exercise_id"` // todo - use uuids here
	CourseID   string `toml:"course_id"`
}

const ExerciseConfigFile = ".tdl-exercise"

func Run() {
	success, lastExercise := returnExercise()
	if !success {
		return
	}

	if success && lastExercise {
		// todo - some CTA here
		fmt.Println("Congratulations, you finished the course " + color.YellowString("🏆"))
		return
	}

	fmt.Println()
	if !internal.ConfirmPromptDefaultYes("Do you want to go to the next exercise?") {
		return
	}

	// todo - is this assumption always valid about course dir?
	nextExercise()
}

func returnExercise() (bool, bool) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	config := ExerciseConfig{}

	exerciseConfig := path.Join(pwd, ExerciseConfigFile)
	if ok, err := fileExists(exerciseConfig); err != nil {
		panic(err)
	} else if !ok {
		fmt.Println("You are not in an exercise directory.")

		_, err := findCourseRoot()
		if errors.Is(err, courseRootNotFoundError) {
			fmt.Println("You are not in a course directory. If you already started the course, please go to the exercise directory.")
		} else {
			fmt.Println("Please go to the exercise directory.")
		}

		return false, false
	}

	if _, err := toml.DecodeFile(exerciseConfig, &config); err != nil {
		// todo - better handling
		panic(err)
	}

	logrus.WithFields(logrus.Fields{
		"course":   config.CourseID,
		"exercise": config.ExerciseID,
		"pwd":      pwd,
	}).Debug("Calculated course and exercise")

	files, err := getFiles()
	if err != nil {
		panic(err)
	}

	courseConfig := readCourseConfig()
	// todo - validate if exercise id == course exercise id? to ensure about consistency

	req := &genproto.VerifyExerciseRequest{
		CourseId: config.CourseID,
		Exercise: config.ExerciseID,
		Files:    files,
		Token:    courseConfig.Token,
	}
	logrus.WithField("req", req).Info("Request prepared")

	conn, err := grpc.Dial("localhost:3000", grpc.WithInsecure())
	if err != nil {
		// todo
		panic(err)
	}

	client := genproto.NewServerClient(conn)

	stream, err := client.VerifyExercise(context.Background(), req)
	if err != nil {
		// todo - remove all panics
		panic(err)
	}

	successful := false
	lastExercise := false

	for {
		response, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		if response.Finished {
			fmt.Println("--------")

			var msg string

			if response.Successful {
				msg = color.GreenString("SUCCESS")
				successful = true
				lastExercise = response.LastExercise
			} else {
				msg = color.RedString("FAIL")
			}

			fmt.Println(msg)
		}

		if len(response.Stdout) > 0 {
			fmt.Println(response.Stdout)
		}
		if len(response.Stderr) > 0 {
			_, _ = fmt.Fprintln(os.Stderr, response.Stderr)
			// todo log err
		}
		// todo - print result
		// todo - support stderr and commands
	}

	return successful, lastExercise
}

func getFiles() ([]*genproto.File, error) {
	pwd, err := os.Getwd()
	if err != nil {
		// todo - not panic
		panic(err)
	}

	// todo - make it comon
	// todo - add unit tests here
	var filesPaths []string
	err = filepath.Walk(
		pwd,
		func(filePath string, info os.FileInfo, err error) error {
			// todo - make it more secure (only go files?)
			if info.IsDir() {
				return nil
			}
			// todo - is it secure enough?
			if path.Ext(info.Name()) != ".go" && info.Name() != "go.mod" {
				return nil
			}

			filesPaths = append(filesPaths, filePath)
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("unable to read golden files %w", err)
	}

	var files []*genproto.File
	for _, filePath := range filesPaths {
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("unable to read golden file %s: %w", filePath, err)
		}

		relPath, err := filepath.Rel(pwd, filepath.Dir(filePath))
		if err != nil {
			return nil, err
		}

		files = append(files, &genproto.File{
			Name:    filepath.Base(filePath),
			Path:    relPath,
			Content: string(content),
		})
	}
	return files, nil
}
