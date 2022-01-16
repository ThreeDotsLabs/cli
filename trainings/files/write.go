package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
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

	var savedFiles []savedFile

	for _, fileFromServer := range filesToCreate {
		// We should never trust the remote server.
		// Writing files based on external name is a vector for Path Traversal attack.
		// For more info please check: https://owasp.org/www-community/attacks/Path_Traversal
		//
		// To avoid that we are using afero.BasePathFs with base on training root.
		fullFilePath := filepath.Join(exerciseDir, fileFromServer.Path)

		shouldWrite, err := f.shouldWriteFile(trainingRootFs, fullFilePath, fileFromServer)
		if err != nil {
			return err
		}
		if !shouldWrite {
			continue
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
		fmt.Fprintf(f.stdout, "+ %s (%d lines)\n", file.Name, file.Lines)
	}

	if len(savedFiles) > 0 {
		fmt.Fprintf(f.stdout, "%d files saved\n\n", len(savedFiles))
	}

	return nil
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

	_, _ = fmt.Fprintf(f.stdout, "\nFile %s already exists, diff:\n", filePath)

	edits := myers.ComputeEdits(span.URIFromPath("local "+filepath.Base(file.Path)), string(actualContent), file.Content)
	diff := fmt.Sprint(gotextdiff.ToUnified("local "+filepath.Base(file.Path), "remote "+filepath.Base(file.Path), string(actualContent), edits))
	_, _ = fmt.Fprintln(f.stdout, diff)

	if !internal.FConfirmPrompt("Should it be overridden?", f.stdin, f.stdout) {
		_, _ = fmt.Fprintln(f.stdout, "Skipping file")
		return false, nil
	} else {
		return true, nil
	}
}

func (f Files) dirOrFileExists(fs afero.Fs, path string) bool {
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
