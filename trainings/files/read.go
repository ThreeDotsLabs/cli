package files

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func (f Files) ReadSolutionFiles(trainingRootFs afero.Fs, dir string) ([]*genproto.File, error) {
	var filesPaths []string
	err := afero.Walk(
		trainingRootFs,
		dir,
		func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				logrus.WithError(err).Warn("Error while reading solution files")
				return nil
			}

			if info.IsDir() {
				return nil
			}
			if filepath.Ext(info.Name()) != ".go" && info.Name() != "go.mod" {
				return nil
			}

			filesPaths = append(filesPaths, filePath)
			return nil
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read solution files")
	}

	var files []*genproto.File
	for _, filePath := range filesPaths {
		content, err := afero.ReadFile(trainingRootFs, filePath)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to read solution file %s", filePath)
		}

		relPath, err := filepath.Rel(dir, filePath)
		if err != nil {
			return nil, err
		}

		// Normalize filepath to slashes
		slashPath := filepath.ToSlash(relPath)

		files = append(files, &genproto.File{
			Path:    slashPath,
			Content: string(content),
		})
	}

	return files, nil
}
