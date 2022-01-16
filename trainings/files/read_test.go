package files_test

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func TestFiles_ReadSolutionFiles(t *testing.T) {
	fs := afero.NewBasePathFs(afero.NewOsFs(), testDataDir(t, "TestFiles_ReadSolutionFiles"))
	wd := "/foo"

	f := files.NewFilesWithStdOuts(os.Stdin, os.Stdout)

	protoFiles, err := f.ReadSolutionFiles(fs, wd)
	require.NoError(t, err)

	assert.Equal(t, []*genproto.File{
		{
			Path:    "baz/baz.go",
			Content: "package baz\n",
		},
		{
			Path:    "go.mod",
			Content: "module foo\n\ngo 1.17\n",
		},
		{
			Path:    "main.go",
			Content: "package main\n\nfunc main() {\n\n}\n",
		},
	}, protoFiles)
}

var filesToCreate = []*genproto.File{
	{
		Path:    "baz/baz.go",
		Content: "package bar\n",
	},
	{
		Path:    "go.mod",
		Content: "module foo\n\ngo 1.17\n",
	},
	{
		Path:    "main.go",
		Content: "package main\n\nfunc main() {\n\n}\n",
	},
}
