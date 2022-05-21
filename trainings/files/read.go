package files

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func (f Files) ReadSolutionFiles(trainingRootFs afero.Fs, dir string) ([]*genproto.File, error) {
	var filesPaths []string
	err := afero.Walk(
		trainingRootFs,
		dir,
		func(filePath string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if filepath.Ext(info.Name()) != ".go" && info.Name() != "go.mod" {
				return nil
			}

			// Normalize filepath to slashes
			slashPath := filepath.ToSlash(filePath)

			filesPaths = append(filesPaths, slashPath)
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

		files = append(files, &genproto.File{
			Path:    relPath,
			Content: string(content),
		})
	}

	return files, nil
}
