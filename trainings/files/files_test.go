package files_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func testDataDir(t *testing.T, testName string) string {
	wd, err := os.Getwd()
	require.NoError(t, err)

	return filepath.Join(wd, "testdata", testName)
}

type fsDecorator struct {
	Decorated afero.Fs

	CreatedFiles map[string]struct{}
	CreatedDirs  []string
}

func (f *fsDecorator) Create(name string) (afero.File, error) {
	file, err := f.Decorated.Create(name)

	if err == nil {
		f.CreatedFiles[name] = struct{}{}
	}

	return file, err
}

func (f *fsDecorator) Mkdir(name string, perm os.FileMode) error {
	f.CreatedDirs = append(f.CreatedDirs, name)

	return f.Decorated.Mkdir(name, perm)
}

func (f *fsDecorator) MkdirAll(path string, perm os.FileMode) error {
	f.CreatedDirs = append(f.CreatedDirs, path)

	return f.Decorated.MkdirAll(path, perm)
}

func (f fsDecorator) Open(name string) (afero.File, error) {
	return f.Decorated.Open(name)
}

func (f fsDecorator) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return f.Decorated.OpenFile(name, flag, perm)
}

func (f fsDecorator) Remove(name string) error {
	return f.Decorated.Remove(name)
}

func (f fsDecorator) RemoveAll(path string) error {
	return f.Decorated.RemoveAll(path)
}

func (f fsDecorator) Rename(oldname, newname string) error {
	return f.Decorated.Rename(oldname, newname)
}

func (f fsDecorator) Stat(name string) (os.FileInfo, error) {
	return f.Decorated.Stat(name)
}

func (f fsDecorator) Name() string {
	return f.Decorated.Name()
}

func (f fsDecorator) Chmod(name string, mode os.FileMode) error {
	return f.Decorated.Chmod(name, mode)
}

func (f fsDecorator) Chown(name string, uid, gid int) error {
	return f.Decorated.Chown(name, uid, gid)
}

func (f fsDecorator) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return f.Decorated.Chtimes(name, atime, mtime)
}

func assertFilesCreated(t *testing.T, fs *fsDecorator, dir string, filesToCreate []*genproto.File) {
	assert.Len(t, fs.CreatedFiles, len(filesToCreate))

	for _, file := range filesToCreate {
		expectedPath := filepath.Join(dir, file.Path)

		_, ok := fs.CreatedFiles[expectedPath]
		if !assert.True(t, ok, "file %s doesn't exist", expectedPath) {
			continue
		}

		fileContent, err := afero.ReadFile(fs, expectedPath)
		require.NoError(t, err)
		assert.Equal(t, file.Content, string(fileContent))
	}
}
