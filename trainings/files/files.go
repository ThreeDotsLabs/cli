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
	forceOverwrite    bool
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

func NewFilesForceOverwrite() Files {
	f := NewFiles()
	f.forceOverwrite = true
	return f
}

func NewFilesWithStdOuts(stdin io.Reader, stdout io.Writer) Files {
	return Files{
		stdin:  stdin,
		stdout: stdout,
	}
}

// NewFilesSilent creates a Files that writes silently (no output).
// Use for internal operations like writing to worktrees.
func NewFilesSilent() Files {
	return Files{
		forceOverwrite: true,
		stdout:         io.Discard,
		stdin:          os.Stdin,
	}
}

// NewFilesSilentDeleteUnused creates a Files that writes silently and deletes
// files not present in the server response. Use for override operations where
// the exercise directory should exactly match the example solution.
func NewFilesSilentDeleteUnused() Files {
	return Files{
		forceOverwrite:    true,
		deleteUnusedFiles: true,
		stdout:            io.Discard,
		stdin:             os.Stdin,
	}
}
