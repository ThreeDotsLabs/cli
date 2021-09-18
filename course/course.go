package course

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/ThreeDotsLabs/cli/course/genproto"
	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/fatih/color"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func List() {
	// todo - dedup?
	conn, err := grpc.Dial("localhost:3000", grpc.WithInsecure())
	if err != nil {
		// todo
		panic(err)
	}
	client := genproto.NewServerClient(conn)

	courses, err := client.GetCourses(context.Background(), &empty.Empty{})
	if err != nil {
		panic(err)
	}

	for _, course := range courses.Courses {
		fmt.Println(course.Id)
	}
}

const courseConfigFile = ".tdl-course"

type courseConfig struct {
	CourseID string
	Token    string
}

var courseRootNotFoundError = errors.New("course root not found")

func Start(courseID string) {
	logrus.WithField("course_id", courseID).Debug("Starting course")

	err := startCourse(courseID)
	if err != nil {
		// todo - handle it nicer
		panic(err)
	}

	// todo - handle situation when course was started but something failed here and someone is starting excersise again (because he have no local files)
	nextExercise()
}

func nextExercise() {
	courseRoot, err := findCourseRoot()
	if err != nil {
		panic(err)
	}

	courseConfig := readCourseConfig()

	logrus.WithFields(logrus.Fields{
		"course_id":   courseConfig.CourseID,
		"course_root": courseRoot,
	}).Debug("Starting exercise")

	// todo - dedup?
	conn, err := grpc.Dial("localhost:3000", grpc.WithInsecure())
	if err != nil {
		// todo
		panic(err)
	}
	client := genproto.NewServerClient(conn)

	resp, err := client.NextExercise(context.Background(), &genproto.NextExerciseRequest{
		CourseId: courseConfig.CourseID,
		Token:    courseConfig.Token,
	})
	if err != nil {
		panic(err)
	}

	logrus.WithFields(logrus.Fields{
		"resp": resp,
	}).Debug("Received exercise from server")

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

	exerciseConfig := ExerciseConfig{
		ExerciseID: resp.GetExerciseId(),
		CourseID:   courseConfig.CourseID,
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

	if requireCd {
		fmt.Printf("Exercise files were created in '%s' directory.\n", relExpectedDir)
		fmt.Println("Please execute", color.CyanString("cd "+relExpectedDir), "to get there.")
	}

	fmt.Printf("\nTo run solution, please execute " + color.CyanString("tdl course run"))
	if requireCd {
		fmt.Print(" in ", relExpectedDir)
	}
	fmt.Println()
}

func startCourse(courseID string) error {
	// todo - move somewhere
	token := os.Getenv("TDL_TOKEN")
	if token == "" {
		panic("missing token")
	}

	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	courseRoot, err := findCourseRoot()
	if errors.Is(err, courseRootNotFoundError) {
		logrus.WithField("course_root", courseRoot).Debug("Creating course root")

		courseRoot = pwd
		f, err := os.Create(path.Join(courseRoot, courseConfigFile))
		if err != nil {
			panic(err)
		}

		if err := toml.NewEncoder(f).Encode(courseConfig{
			CourseID: courseID,
			Token:    token,
		}); err != nil {
			panic(err)
		}

		// todo - put some content
		if err := f.Close(); err != nil {
			panic(err)
		}
	} else {
		logrus.WithField("course_root", courseRoot).Debug("Found course root")
		fmt.Println("Course was already started. Course root:", pwd)

		cfg := readCourseConfig()

		if cfg.CourseID != courseID {
			return fmt.Errorf("course %s was already started in this directory", cfg.CourseID)
		}

		return nil
	}

	if !internal.ConfirmPromptDefaultYes(fmt.Sprintf("This command will clone course source code to %s directory. Do you want to continue?", pwd)) {

	}

	// todo - dedup?
	conn, err := grpc.Dial("localhost:3000", grpc.WithInsecure())
	if err != nil {
		// todo
		panic(err)
	}
	client := genproto.NewServerClient(conn)

	_, err = client.StartCourse(context.Background(), &genproto.StartCourseRequest{
		CourseId: courseID, // todo - it should be some kind of uuid
		Token:    token,
	})
	if err != nil {
		panic(err)
	}

	// todo - create course config

	return nil
}

func readCourseConfig() courseConfig {
	// todo - it would be nice to not read it every time
	courseRoot, err := findCourseRoot()
	if err != nil {
		panic(err)
	}

	config := courseConfig{}

	if _, err := toml.DecodeFile(path.Join(courseRoot, courseConfigFile), &config); err != nil {
		// todo - better handling
		panic(err)
	}

	logrus.WithField("course_config", config).Debug("Course config")

	return config
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
