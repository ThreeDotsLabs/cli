package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func (f Files) ReadSolutionFiles(trainingRootFs afero.Fs, dir string) ([]*genproto.File, error) {
	var filesPaths []string
	var vendorDirSkipped bool

	err := afero.Walk(
		trainingRootFs,
		dir,
		func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				logrus.WithError(err).Warn("Error while reading solution files")
				return nil
			}

			if info.IsDir() {
				if isGoVendorDir(trainingRootFs, filePath) {
					vendorDirSkipped = true
					return filepath.SkipDir
				}
				return nil
			}
			if !IsSolutionFile(filePath) {
				return nil
			}

			filesPaths = append(filesPaths, filePath)
			return nil
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read solution files")
	}

	if vendorDirSkipped {
		warnAboutSkippedVendorDirs(f.stdout)
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

// vendorMarkerFiles are the files whose presence makes a sibling "vendor" directory a real
// Go vendor directory: go.mod for a module root, go.work for a workspace root (`go work vendor`).
var vendorMarkerFiles = []string{"go.mod", "go.work"}

// isGoVendorDir reports whether dirPath is a Go vendor directory, i.e. a directory named
// "vendor" sitting next to a go.mod or go.work. Go only honours vendoring at a module or
// workspace root, so this deliberately doesn't match something like
// internal/marketplace/vendor, which is ordinary user code that happens to share the name.
func isGoVendorDir(fs afero.Fs, dirPath string) bool {
	if filepath.Base(dirPath) != "vendor" {
		return false
	}

	for _, marker := range vendorMarkerFiles {
		exists, err := afero.Exists(fs, filepath.Join(filepath.Dir(dirPath), marker))
		if err != nil {
			logrus.WithError(err).WithField("dir", dirPath).WithField("marker", marker).
				Warn("Can't check if vendor dir is at a module or workspace root")
			continue
		}

		if exists {
			return true
		}
	}

	return false
}

func warnAboutSkippedVendorDirs(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, color.YellowString("Vendor directory detected, not sending it to the server."))
}

var solutionFiles = []string{
	"go.mod",
	"go.work",
	"Dockerfile",
	".gitattributes",
}

var solutionExtensions = []string{
	".go",
	".yaml",
	".yml",
	".conf",
	".html",
	".sql",
}

func IsSolutionFile(filePath string) bool {
	name := filepath.Base(filePath)

	for _, solutionFile := range solutionFiles {
		if name == solutionFile {
			return true
		}
	}

	for _, ext := range solutionExtensions {
		if filepath.Ext(filePath) == ext {
			return true
		}
	}

	return false
}
