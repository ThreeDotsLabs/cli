package files

import (
	"io"
	"os"
)

type Files struct {
	stdin  io.Reader
	stdout io.Writer

	// stdinCh, when set, takes precedence over stdin for confirm prompts.
	// Callers running inside interactiveRun should pass h.stdinCh here so
	// confirm prompts don't race with the MCP stdin reader goroutine for
	// bytes on os.Stdin. See trainings/run.go interactiveRun for context.
	stdinCh <-chan rune

	deleteUnusedFiles bool
	showFullDiff      bool
	forceOverwrite    bool
}

// WithStdinCh returns a copy of f that reads confirm-prompt answers from ch
// instead of f.stdin. Use this when the caller is inside interactiveRun and
// MCP is active (h.stdinCh != nil).
func (f Files) WithStdinCh(ch <-chan rune) Files {
	f.stdinCh = ch
	return f
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
