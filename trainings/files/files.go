package files

import (
	"io"
	"os"
)

type Files struct {
	stdin  io.Reader
	stdout io.Writer

	deleteUnusedFiles bool
	showFullDiff      bool
}

func NewFiles() Files {
	return NewFilesWithStdOuts(os.Stdin, os.Stdout)
}

func NewFilesWithConfig(deleteUnusedFiles bool, showFullDiff bool) Files {
	f := NewFiles()
	f.deleteUnusedFiles = deleteUnusedFiles
	f.showFullDiff = showFullDiff
	return f
}

func NewFilesWithStdOuts(stdin io.Reader, stdout io.Writer) Files {
	return Files{
		stdin:  stdin,
		stdout: stdout,
	}
}
