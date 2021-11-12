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
	mainFs afero.Fs
	stdin  io.Reader
	stdout io.Writer
}

func NewDefaultFiles() Files {
	return NewFiles(afero.NewOsFs(), os.Stdin, os.Stdout)
}

func NewFiles(mainFs afero.Fs, stdin io.Reader, stdout io.Writer) Files {
	return Files{
		mainFs: mainFs,
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
	// We should never trust the remote server. We are using BasePath to protect from Path Traversal attack.
	// For more info please check: https://owasp.org/www-community/attacks/Path_Traversal
	exerciseBaseFs := afero.NewBasePathFs(f.mainFs, exerciseDir)

	if !f.dirOrFileExists("/", exerciseBaseFs) {
		if err := exerciseBaseFs.MkdirAll("/", 0755); err != nil {
			return errors.Wrapf(err, "can't create %s", exerciseDir)
		}
	}

	for _, file := range filesToCreate {
		shouldWrite, err := f.shouldWriteFile(file.Path, file, exerciseBaseFs)
		if err != nil {
			return err
		}
		if !shouldWrite {
			continue
		}

		f, err := exerciseBaseFs.Create(file.Path)
		if err != nil {
			return errors.Wrapf(err, "can't create %s", path.Join(exerciseDir, file.Path))
		}

		if _, err := f.WriteString(file.Content); err != nil {
			return errors.Wrapf(err, "can't write to %s", path.Join(exerciseDir, file.Path))
		}

		if err := f.Close(); err != nil {
			return errors.Wrapf(err, "can't close %s", path.Join(exerciseDir, file.Path))
		}
	}

	return nil
}

func (f Files) shouldWriteFile(filePath string, file *genproto.File, baseFs afero.Fs) (bool, error) {
	if !f.dirOrFileExists(filePath, baseFs) {
		return true, nil
	}

	actualContent, err := afero.ReadFile(baseFs, filePath)
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

func (f Files) dirOrFileExists(path string, baseFs afero.Fs) bool {
	_, err := baseFs.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}

	// it can be only some strange i/o error, let's not silently ignore it
	panic(err)
}
