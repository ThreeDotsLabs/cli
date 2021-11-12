package files

import (
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/tdl/trainings/genproto"
)

func (f Files) ReadSolutionFiles(dir string) ([]*genproto.File, error) {
	var filesPaths []string
	err := afero.Walk(
		f.fs,
		dir,
		func(filePath string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if path.Ext(info.Name()) != ".go" && info.Name() != "go.mod" {
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
		content, err := afero.ReadFile(f.fs, filePath)
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
