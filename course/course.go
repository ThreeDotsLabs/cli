package course

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/ThreeDotsLabs/cli/course/genproto"
	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"google.golang.org/grpc"
)

const courseConfigFile = ".tdl-course"

var courseRootNotFoundError = errors.New("course root not found")

func Start() {
	courseID := "example"

	courseRoot := startCourse(courseID)

	// todo - handle situation when course was started but something failed here and someone is starting excersise again (because he have no local files)
	nextExercise(courseRoot, courseID)
}

func nextExercise(courseRoot string, courseID string) {
	// todo - dedup?
	conn, err := grpc.Dial("localhost:3000", grpc.WithInsecure())
	if err != nil {
		// todo
		panic(err)
	}
	client := genproto.NewServerClient(conn)

	resp, err := client.NextExercise(context.Background(), &empty.Empty{})
	if err != nil {
		panic(err)
	}

	// todo - validate if resp.GetDir() or anything is empty!

	expectedDir := path.Join(courseRoot, resp.GetDir())

	expectedDirExists, err := fileExists(expectedDir)
	if err != nil {
		panic(err)
	}
	if !expectedDirExists {
		if err := os.MkdirAll(expectedDir, 0755); err != nil {
			panic(err)
		}
	}
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	if pwd != expectedDir {
		fmt.Printf("Exercise was created in '%s' directory.\nPlase run `cd %s` and `tdl course run` to execute your solution.\n", resp.GetDir(), resp.GetDir())
	}

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

	exerciseConfig := ExerciseConfig{
		ExerciseID: resp.GetExerciseId(),
		CourseID:   courseID,
	}

	f, err := os.Create(path.Join(expectedDir, ExerciseConfigFile))
	if err != nil {
		panic(err)
	}

	if err := toml.NewEncoder(f).Encode(exerciseConfig); err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}

	fmt.Printf("Starting %s exercise. Please run `tdl course run` to check your solution.\n", exerciseConfig.ExerciseID)
}

func startCourse(courseID string) string {
	// todo - dedup?
	conn, err := grpc.Dial("localhost:3000", grpc.WithInsecure())
	if err != nil {
		// todo
		panic(err)
	}
	client := genproto.NewServerClient(conn)

	_, err = client.StartCourse(context.Background(), &genproto.StartCourseRequest{
		CourseId: courseID, // todo - it should be some kind of uuid
	})
	if err != nil {
		panic(err)
	}

	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// todo - create course config

	courseRoot, err := findCourseRoot()
	if errors.Is(err, courseRootNotFoundError) {
		courseRoot = pwd
		f, err := os.Create(path.Join(courseRoot, courseConfigFile))
		if err != nil {
			panic(err)
		}
		// todo - put some content
		if err := f.Close(); err != nil {
			panic(err)
		}
	}
	return courseRoot
}

func shouldWriteFile(filePath string, file *genproto.File) bool {
	// todo - next!
	ok, err := fileExists(filePath)
	if err != nil {
		panic(err)
	}
	if !ok {
		return true
	}

	// todo - test it

	actualContent, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	if string(actualContent) == file.Content {
		fmt.Printf("File %s already exists, skipping\n", filePath)
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

func findCourseRoot() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := pwd

	for {
		ok, err := fileExists(path.Join(dir, courseConfigFile))
		if err != nil {
			return "", err
		}
		if ok {
			return dir, nil
		}

		dir = path.Dir(dir)
		if dir == "/" {
			break
		}
	}

	return "", courseRootNotFoundError
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}
