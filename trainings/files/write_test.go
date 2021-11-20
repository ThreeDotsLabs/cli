package files_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func TestFiles_WriteExerciseFiles(t *testing.T) {
	fs := &fsDecorator{
		Decorated:    afero.NewMemMapFs(),
		CreatedFiles: map[string]struct{}{},
	}
	stdin := bytes.NewBufferString("")
	stdout := bytes.NewBuffer(nil)

	f := files.NewFiles(fs, stdin, stdout)
	dir := "dir"

	err := f.WriteExerciseFiles(filesToCreate, dir)
	require.NoError(t, err)

	assert.Equal(t, []string{dir}, fs.CreatedDirs)
	assert.Empty(t, stdout.String())

	assertFilesCreated(t, fs, dir, filesToCreate)

	// check idempotency
	err = f.WriteExerciseFiles(filesToCreate, dir)
	require.NoError(t, err)
	assert.Equal(t, []string{dir}, fs.CreatedDirs)
	assert.Len(t, fs.CreatedFiles, len(filesToCreate))
	assert.Empty(t, stdout.String())
}

func TestFiles_WriteExerciseFiles_accept_override_existing(t *testing.T) {
	fs := &fsDecorator{
		Decorated:    afero.NewMemMapFs(),
		CreatedFiles: map[string]struct{}{},
	}
	stdin := bytes.NewBufferString(strings.Repeat("y\n", 3)) // this will accept diff
	stdout := bytes.NewBuffer(nil)

	f := files.NewFiles(fs, stdin, stdout)
	dir := "bar"

	err := f.WriteExerciseFiles([]*genproto.File{
		{
			Path: "main.go",
			Content: `package main

func main() {

}`,
		},
		{
			Path:    "go.mod",
			Content: "module foo\n\ngo 1.17\n",
		},
		{
			Path:    "baz/baz.go",
			Content: "package bar\n",
		},
	}, dir)
	require.NoError(t, err)

	// let's ignore current files
	fs.CreatedDirs = []string{}
	fs.CreatedFiles = map[string]struct{}{}

	filesToUpdate := []*genproto.File{
		{
			Path: "main.go",
			Content: `package main

func main() {
	fmt.Print("hello!")
}`,
		},
		{
			Path:    "go.mod",
			Content: "module foo\n\ngo 1.18\n",
		},
		{
			Path:    "baz/baz.go",
			Content: "package baz\n",
		},
	}
	err = f.WriteExerciseFiles(filesToUpdate, dir)
	require.NoError(t, err)

	assertFilesCreated(t, fs, dir, filesToUpdate)

	assert.Contains(
		t,
		stdout.String(),
		"already exists, diff:",
	)
}

func TestFiles_WriteExerciseFiles_reject_override_existing(t *testing.T) {
	fs := &fsDecorator{
		Decorated:    afero.NewMemMapFs(),
		CreatedFiles: map[string]struct{}{},
	}
	stdin := bytes.NewBufferString(strings.Repeat("n\n", 3)) // this will reject diff
	stdout := bytes.NewBuffer(nil)

	f := files.NewFiles(fs, stdin, stdout)
	dir := "bar"

	err := f.WriteExerciseFiles([]*genproto.File{
		{
			Path: "main.go",
			Content: `package main

func main() {

}`,
		},
		{
			Path:    "go.mod",
			Content: "module foo\n\ngo 1.17\n",
		},
		{
			Path:    "baz/baz.go",
			Content: "package bar\n",
		},
	}, dir)
	require.NoError(t, err)

	// let's ignore current files
	fs.CreatedDirs = []string{}
	fs.CreatedFiles = map[string]struct{}{}

	filesToUpdate := []*genproto.File{
		{
			Path: "main.go",
			Content: `package main

func main() {
	fmt.Print("hello!")
}`,
		},
		{
			Path:    "go.mod",
			Content: "module foo\n\ngo 1.18\n",
		},
		{
			Path:    "baz/baz.go",
			Content: "package baz\n",
		},
	}
	err = f.WriteExerciseFiles(filesToUpdate, dir)
	require.NoError(t, err)

	assert.Empty(t, fs.CreatedDirs)
	assert.Empty(t, fs.CreatedFiles)

	assert.Contains(
		t,
		stdout.String(),
		"already exists, diff:",
	)
}

// TestWriteExerciseFiles_path_traversal is checking if WriteExerciseFiles is valuable for path traversal.
// path.Join should protect us from that attack. But let's double-check.
func TestWriteExerciseFiles_path_traversal(t *testing.T) {
	testCases := []struct {
		Name     string
		FilePath string
	}{
		{
			Name:     "absolute_dir",
			FilePath: "/secret.txt",
		},
		{
			Name:     "parent_directory",
			FilePath: "../secret.txt",
		},
		{
			Name:     "parent_directory_no_slash",
			FilePath: "/../secret.txt",
		},
		{
			Name:     "parent_directory_windows",
			FilePath: "..\\secret.txt",
		},
		{
			Name:     "three_dots",
			FilePath: ".../foo/main.go",
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.Name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			originalFileContent := "original"
			err := afero.WriteFile(fs, "/secret.txt", []byte(originalFileContent), 0755)
			require.NoError(t, err)

			f := files.NewFiles(fs, bytes.NewBufferString(strings.Repeat("y\n", 2)), os.Stdout)

			err = f.WriteExerciseFiles(
				[]*genproto.File{
					{
						Path:    tc.FilePath,
						Content: "modified",
					},
				},
				"dir",
			)
			require.NoError(t, err)

			currentFileContent, err := afero.ReadFile(fs, "/secret.txt")
			require.NoError(t, err)

			assert.Equal(t, originalFileContent, string(currentFileContent))
		})
	}
}
