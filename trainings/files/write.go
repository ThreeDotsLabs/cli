package files

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

const (
	ExerciseFile = "exercise.md"
)

type InvalidFilePathError struct {
	pathValue string
}

func (i InvalidFilePathError) Error() string {
	return fmt.Sprintf("invalid file.Path '%s'", i.pathValue)
}

type savedFile struct {
	Name  string
	Lines int
}

func (f Files) WriteExerciseFiles(filesToCreate []*genproto.File, trainingRootFs afero.Fs, exerciseDir string) error {
	if !f.dirOrFileExists(trainingRootFs, exerciseDir) {
		if err := trainingRootFs.MkdirAll(exerciseDir, 0755); err != nil {
			return errors.Wrapf(err, "can't create %s", exerciseDir)
		}
	}

	logrus.WithFields(logrus.Fields{
		"exercise_dir": exerciseDir,
		"files_num":    len(filesToCreate),
	}).Debugf("Writing exercise files to %s", exerciseDir)

	var savedFiles []savedFile

	filesToDelete := map[string]struct{}{}

	if f.deleteUnusedFiles {
		err := afero.Walk(trainingRootFs, exerciseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && filepath.Base(path) != ExerciseFile && filepath.Base(path) != "go.sum" {
				filesToDelete[path] = struct{}{}
			}

			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "can't walk through %s", exerciseDir)
		}

		for _, fileFromServer := range filesToCreate {
			fullFilePath := filepath.Join(exerciseDir, fileFromServer.Path)
			delete(filesToDelete, fullFilePath)
		}
	}

	if f.showFullDiff {
		proceed, err := f.shouldWriteAllFiles(trainingRootFs, exerciseDir, filesToCreate, filesToDelete)
		if err != nil {
			return err
		}

		if !proceed {
			return nil
		}
	}

	for _, fileFromServer := range filesToCreate {
		// We should never trust the remote server.
		// Writing files based on external name is a vector for Path Traversal attack.
		// For more info please check: https://owasp.org/www-community/attacks/Path_Traversal
		//
		// To avoid that we are using afero.BasePathFs with base on training root.
		fullFilePath := filepath.Join(exerciseDir, fileFromServer.Path)
		fullFileDir := filepath.Dir(fullFilePath)

		if !f.showFullDiff {
			shouldWrite, err := f.shouldWriteFile(trainingRootFs, fullFilePath, fileFromServer)
			if err != nil {
				return err
			}
			if !shouldWrite {
				continue
			}
		}

		if err := trainingRootFs.MkdirAll(fullFileDir, 0755); err != nil {
			return errors.Wrapf(err, "can't create %s dir", fileFromServer.Path)
		}

		file, err := trainingRootFs.Create(fullFilePath)
		if err != nil {
			return errors.Wrapf(err, "can't create %s", fullFilePath)
		}

		if _, err := file.WriteString(fileFromServer.Content); err != nil {
			return errors.Wrapf(err, "can't write to %s", fullFilePath)
		}

		if err := file.Close(); err != nil {
			return errors.Wrapf(err, "can't close %s", fullFilePath)
		}

		linesAdded := len(strings.Split(fileFromServer.Content, "\n"))

		savedFiles = append(savedFiles, savedFile{
			Name:  fullFilePath,
			Lines: linesAdded,
		})
	}

	for _, file := range savedFiles {
		savedFileRelativePath, err := calculateSavedFileRelativePath(trainingRootFs, file.Name)
		if err != nil {
			logrus.WithError(err).Warn("Can't calculate savedFileRelativePath")
			savedFileRelativePath = file.Name
		}

		fmt.Fprintf(f.stdout, color.GreenString("+")+" %s (%d lines)\n", savedFileRelativePath, file.Lines)
	}

	if f.deleteUnusedFiles {
		var deletedFiles []string
		for path := range filesToDelete {
			if !f.showFullDiff {
				shouldDelete := internal.FConfirmPrompt(fmt.Sprintf("File %s is not used anymore, should it be deleted?", path), f.stdin, f.stdout)
				if !shouldDelete {
					continue
				}
			}

			if err := trainingRootFs.Remove(path); err != nil {
				return errors.Wrapf(err, "can't delete %s", path)
			}

			deletedFiles = append(deletedFiles, path)
			deletedFileRelativePath, err := calculateSavedFileRelativePath(trainingRootFs, path)
			if err != nil {
				logrus.WithError(err).Warn("Can't calculate deletedFileRelativePath")
				deletedFileRelativePath = path
			}

			fmt.Fprintf(f.stdout, color.RedString("-")+" %s\n", deletedFileRelativePath)
		}
	}

	if len(savedFiles) == 1 {
		fmt.Fprintf(f.stdout, "Exercise ready, 1 file saved.\n\n")
	} else if len(savedFiles) > 0 {
		fmt.Fprintf(f.stdout, "Exercise ready, %d files saved.\n\n", len(savedFiles))
	} else {
		fmt.Fprintf(f.stdout, "Exercise ready.\n\n")
	}

	return nil
}

func calculateSavedFileRelativePath(trainingRootFs afero.Fs, fileName string) (string, error) {
	trainingRootFsRelPather, ok := trainingRootFs.(RealPather)
	if !ok {
		return fileName, nil
	}

	realPath, err := trainingRootFsRelPather.RealPath(fileName)
	if err != nil {
		return "", errors.Wrapf(err, "can't get real path of %s", fileName)
	}

	wd, err := syscall.Getwd()
	if err != nil {
		return "", err
	}

	terminalPath, err := filepath.Rel(wd, realPath)
	if err != nil {
		return "", err
	}

	return terminalPath, nil
}

func (f Files) shouldWriteAllFiles(fs afero.Fs, exerciseDir string, filesToCreate []*genproto.File, pathsToDelete map[string]struct{}) (bool, error) {
	externalFiles := map[string]*genproto.File{}

	allPaths := map[string]struct{}{}
	for _, file := range filesToCreate {
		fullFilePath := filepath.Join(exerciseDir, file.Path)
		allPaths[fullFilePath] = struct{}{}
		externalFiles[fullFilePath] = file
	}

	for path := range pathsToDelete {
		allPaths[path] = struct{}{}
	}

	allPathsSorted := make([]string, 0, len(allPaths))
	for path := range allPaths {
		allPathsSorted = append(allPathsSorted, path)
	}

	sort.Strings(allPathsSorted)

	additions := false
	changes := false

	for _, filePath := range allPathsSorted {
		exists, err := afero.Exists(fs, filePath)
		if err != nil {
			return false, errors.Wrapf(err, "can't check if %s exists", filePath)
		}

		var localContent []byte
		if exists {
			localContent, err = afero.ReadFile(fs, filePath)
			if err != nil {
				return false, errors.Wrapf(err, "can't read %s", filePath)
			}
		}

		var externalContent string
		externalFile, ok := externalFiles[filePath]
		if ok {
			externalContent = externalFile.Content
		}

		if string(localContent) != externalContent {
			if exists {
				changes = true
			} else {
				additions = true
			}

			relPath, err := filepath.Rel(exerciseDir, filePath)
			if err != nil {
				return false, errors.Wrapf(err, "can't get relative path for %s", filePath)
			}

			edits := myers.ComputeEdits(span.URIFromPath("local "+filePath), string(localContent), externalContent)
			diff := fmt.Sprint(gotextdiff.ToUnified("local "+relPath, "remote "+relPath, string(localContent), edits))
			_, _ = fmt.Fprintln(f.stdout, colorDiff(diff))
		}
	}

	if !changes && len(pathsToDelete) == 0 {
		if additions {
			return true, nil
		} else {
			fmt.Println("All files are up to date.")
			return false, nil
		}
	}

	if len(pathsToDelete) > 0 {
		fmt.Printf("Warning! Files to be deleted:\n")
		for path := range pathsToDelete {
			relPath, err := filepath.Rel(exerciseDir, path)
			if err != nil {
				return false, errors.Wrapf(err, "can't get relative path for %s", path)
			}

			fmt.Printf(color.RedString("- %s\n"), relPath)
		}
		fmt.Println()
	}

	proceed := internal.FConfirmPrompt("Should all files be overridden?", f.stdin, f.stdout)
	if !proceed {
		fmt.Println("Skipping all files.")
		return false, nil
	}

	return true, nil
}

func (f Files) shouldWriteFile(fs afero.Fs, filePath string, file *genproto.File) (bool, error) {
	if !f.dirOrFileExists(fs, filePath) {
		return true, nil
	}

	actualContent, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return false, errors.Wrapf(err, "can't read %s", filePath)
	}

	if string(actualContent) == file.Content {
		logrus.Debugf("File %s already exists, skipping\n", filePath)
		return false, nil
	}

	if strings.Contains(filePath, ExerciseFile) {
		// always override exercise.md
		return true, nil
	}

	_, _ = fmt.Fprintf(f.stdout, "\nFile %s already exists, diff:\n", filePath)

	edits := myers.ComputeEdits(span.URIFromPath("local "+filepath.Base(file.Path)), string(actualContent), file.Content)
	diff := fmt.Sprint(gotextdiff.ToUnified("local "+filepath.Base(file.Path), "remote "+filepath.Base(file.Path), string(actualContent), edits))
	_, _ = fmt.Fprintln(f.stdout, colorDiff(diff))

	if !internal.FConfirmPrompt("Should it be overridden?", f.stdin, f.stdout) {
		_, _ = fmt.Fprintln(f.stdout, "Skipping file")
		return false, nil
	} else {
		return true, nil
	}
}

func (f Files) dirOrFileExists(fs afero.Fs, path string) bool {
	return DirOrFileExists(fs, path)
}

func DirOrFileExists(fs afero.Fs, path string) bool {
	_, err := fs.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}

	// it can be only some strange i/o error, let's not silently ignore it
	panic(err)
}

func colorDiff(diffText string) string {
	red := color.New(color.FgRed)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	lines := strings.Split(diffText, "\n")
	var coloredLines []string

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "-"):
			coloredLines = append(coloredLines, red.Sprint(line))
		case strings.HasPrefix(line, "+"):
			coloredLines = append(coloredLines, green.Sprint(line))
		case strings.HasPrefix(line, "@@"):
			coloredLines = append(coloredLines, yellow.Sprint(line))
		default:
			coloredLines = append(coloredLines, line)
		}
	}

	return strings.Join(coloredLines, "\n")
}
