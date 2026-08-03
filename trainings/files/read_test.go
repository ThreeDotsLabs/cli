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
		// baz has no go.mod, so baz/vendor is ordinary user code and must still be sent.
		{
			Path:    "baz/baz.go",
			Content: "package baz\n",
		},
		{
			Path:    "baz/vendor/vendor.go",
			Content: "package vendor\n",
		},
		{
			Path:    "go.mod",
			Content: "module foo\n\ngo 1.17\n",
		},
		{
			Path:    "main.go",
			Content: "package main\n\nfunc main() {\n\n}\n",
		},
		// /foo/vendor sits next to /foo/go.mod, so it's a real Go vendor dir and is skipped.
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
