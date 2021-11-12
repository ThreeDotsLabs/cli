package files

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/tdl/internal"
	"github.com/ThreeDotsLabs/cli/tdl/trainings/genproto"
)

type Files struct {
	fs     afero.Fs
	stdin  io.Reader
	stdout io.Writer
}

func NewDefaultFiles() Files {
	return NewFiles(afero.NewOsFs(), os.Stdin, os.Stdout)
}

func NewFiles(fs afero.Fs, stdin io.Reader, stdout io.Writer) Files {
	return Files{
		fs:     fs,
		stdin:  stdin,
		stdout: stdout,
	}
}

type InvalidFilePathError struct {
	pathValue string
}

func (i InvalidFilePathError) Error() string {
	return fmt.Sprintf("invalid file.Path '%s'", i.pathValue)
}

func (f Files) WriteExerciseFiles(filesToCreate []*genproto.File, exerciseDir string) error {
	if !f.dirOrFileExists(exerciseDir) {
		if err := f.fs.MkdirAll(exerciseDir, 0755); err != nil {
			return errors.Wrapf(err, "can't create %s", exerciseDir)
		}
	}

	for _, file := range filesToCreate {
		// We should never trust the remote server.
		// Writing files based on external name is a vector for Path Traversal attack.
		// For more info please check: https://owasp.org/www-community/attacks/Path_Traversal
		//
		// Fortunately, path.Join is protecting us from that by calling path.Clean().
		fullFilePath := path.Join(exerciseDir, file.Path)

		shouldWrite, err := f.shouldWriteFile(fullFilePath, file)
		if err != nil {
			return err
		}
		if !shouldWrite {
			continue
		}

		f, err := f.fs.Create(fullFilePath)
		if err != nil {
			return errors.Wrapf(err, "can't create %s", fullFilePath)
		}

		if _, err := f.WriteString(file.Content); err != nil {
			return errors.Wrapf(err, "can't write to %s", fullFilePath)
		}

		if err := f.Close(); err != nil {
			return errors.Wrapf(err, "can't close %s", fullFilePath)
		}
	}

	return nil
}

func (f Files) shouldWriteFile(filePath string, file *genproto.File) (bool, error) {
	if !f.dirOrFileExists(filePath) {
		return true, nil
	}

	actualContent, err := afero.ReadFile(f.fs, filePath)
	if err != nil {
		return false, errors.Wrapf(err, "can't read %s", filePath)
	}

	if string(actualContent) == file.Content {
		logrus.Debugf("File %s already exists, skipping\n", filePath)
		return false, nil
	}

	_, _ = fmt.Fprintf(f.stdout, "\nFile %s already exists, diff:\n", filePath)

	edits := myers.ComputeEdits(span.URIFromPath("local "+path.Base(file.Path)), string(actualContent), file.Content)
	diff := fmt.Sprint(gotextdiff.ToUnified("local "+path.Base(file.Path), "remote "+path.Base(file.Path), string(actualContent), edits))
	_, _ = fmt.Fprintln(f.stdout, diff)

	if !internal.FConfirmPrompt("Should it be overridden?", f.stdin, f.stdout) {
		_, _ = fmt.Fprintln(f.stdout, "Skipping file")
		return false, nil
	} else {
		return true, nil
	}
}

func (f Files) dirOrFileExists(path string) bool {
	_, err := f.fs.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}

	// it can be only some strange i/o error, let's not silently ignore it
	panic(err)
}
