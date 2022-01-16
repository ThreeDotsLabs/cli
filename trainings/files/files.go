package files

import (
	"io"
	"os"
)

type Files struct {
	stdin  io.Reader
	stdout io.Writer
}

func NewFiles() Files {
	return NewFilesWithStdOuts(os.Stdin, os.Stdout)
}

func NewFilesWithStdOuts(stdin io.Reader, stdout io.Writer) Files {
	return Files{
		stdin:  stdin,
		stdout: stdout,
	}
}
