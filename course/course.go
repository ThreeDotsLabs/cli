package course

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/ThreeDotsLabs/cli/tdl/course/genproto"
	"github.com/ThreeDotsLabs/cli/tdl/internal"
	"github.com/fatih/color"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func List() error {
	// todo - dedup?
	conn, err := grpc.Dial(readGlobalConfig().ServerAddr, grpc.WithInsecure())
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

	return nil
}

const courseConfigFile = ".tdl-course"

type courseConfig struct {
	CourseName string
}

var courseRootNotFoundError = errors.New("course root not found")

func Init(courseName string) error {
	logrus.WithField("course_name", courseName).Debug("Starting course")

	err := startCourse(courseName)
	if err != nil {
		// todo - handle it nicer
		panic(err)
	}

	// todo - handle situation when course was started but something failed here and someone is starting excersise again (because he have no local files)
	nextExercise("")

	return nil
}

func nextExercise(currentExerciseID string) {
	courseRoot, err := findCourseRoot()
	if err != nil {
		panic(err)
	}

	courseConfig := readCourseConfig()

	logrus.WithFields(logrus.Fields{
		"course_name": courseConfig.CourseName,
		"course_root": courseRoot,
	}).Debug("Starting exercise")

	// todo - dedup?
	conn, err := grpc.Dial(readGlobalConfig().ServerAddr, grpc.WithInsecure())
	if err != nil {
		// todo
		panic(err)
	}
	client := genproto.NewServerClient(conn)

	resp, err := client.NextExercise(context.Background(), &genproto.NextExerciseRequest{
		CourseName:        courseConfig.CourseName,
		CurrentExerciseId: currentExerciseID,
		Token:             readGlobalConfig().Token,
	})
	if status.Code(err) == codes.NotFound {
		courseFinished()
		return
	}
	if err != nil {
		panic(err)
	}

	logrus.WithFields(logrus.Fields{
		"resp": resp,
	}).Debug("Received exercise from server")

	// todo - validate if resp.GetDir() or anything is empty!

	expectedDir := path.Join(courseRoot, resp.GetDir())

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
		ExerciseID: resp.GetExerciseId(),
		CourseName: courseConfig.CourseName,
	})

	if requireCd {
		fmt.Printf("Exercise files were created in '%s' directory.\n", relExpectedDir)
		fmt.Println("Please execute", color.CyanString("cd "+relExpectedDir), "to get there.")
	}

	fmt.Printf("\nPlase go to http://localhost:3002/about see exercise content.\n")
	fmt.Printf("To run solution, please execute " + color.CyanString("tdl course run"))
	if requireCd {
		fmt.Print(" in ", relExpectedDir)
	}
	fmt.Println()
}

func courseFinished() {
	// todo - some CTA here
	fmt.Println("Congratulations, you finished the course " + color.YellowString("🏆"))
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

func startCourse(courseName string) error {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	if !internal.ConfirmPromptDefaultYes(fmt.Sprintf("This command will clone course source code to %s directory. Do you want to continue?", pwd)) {
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

	_, err = client.StartCourse(context.Background(), &genproto.StartCourseRequest{
		CourseName: courseName, // todo - it should be some kind of uuid
		Token:      readGlobalConfig().Token,
	})
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
			CourseName: courseName,
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

		if cfg.CourseName != courseName {
			return fmt.Errorf("course %s was already started in this directory", cfg.CourseName)
		}

		return nil
	}

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

func findCourseRoot() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := pwd

	for {
		if fileExists(path.Join(dir, courseConfigFile)) {
			return dir, nil
		}

		dir = path.Dir(dir)
		if dir == "/" {
			break
		}
	}

	return "", courseRootNotFoundError
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
