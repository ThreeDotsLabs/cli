package internal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/term"
)

// On Windows, golang.org/x/term uses fd directly as a Win32 HANDLE.
// os.Stdin.Fd() returns the real Win32 handle (not 0); hardcoding 0
// would pass NULL to GetConsoleMode and always return "not a terminal".
// On Unix, os.Stdin.Fd() returns 0 as usual — no behaviour change.
var stdinFileDescriptor = int(os.Stdin.Fd())
var stdoutFileDescriptor = int(os.Stdout.Fd())

func stdinTerminalReason() (bool, string) {
	if v, _ := strconv.ParseBool(os.Getenv("TDL_FORCE_INTERACTIVE")); v {
		return true, "true (TDL_FORCE_INTERACTIVE)"
	}
	if term.IsTerminal(stdinFileDescriptor) {
		return true, "true (native console)"
	}
	// mintty (Git Bash) and MSYS2 use pipes instead of native console handles;
	// TERM env var is their reliable signal.
	if t := os.Getenv("TERM"); t != "" {
		return true, fmt.Sprintf("true (TERM=%s)", t)
	}
	// Windows Terminal sets WT_SESSION in every child process.
	if wt := os.Getenv("WT_SESSION"); wt != "" {
		return true, "true (WT_SESSION)"
	}
	// VS Code integrated terminal and other modern emulators set TERM_PROGRAM.
	if tp := os.Getenv("TERM_PROGRAM"); tp != "" {
		return true, fmt.Sprintf("true (TERM_PROGRAM=%s)", tp)
	}
	return false, "false"
}

// IsStdinTerminal returns true if stdin is connected to a terminal (not a pipe or file).
func IsStdinTerminal() bool         { v, _ := stdinTerminalReason(); return v }
func IsStdinTerminalReason() string { _, s := stdinTerminalReason(); return s }

// NewRawTerminalReader returns raw terminal reader which allows reading stdin without hitting enter.
func NewRawTerminalReader(stdin io.Reader) (*bufio.Reader, func(), error) {
	if stdin != os.Stdin {
		logrus.Info("Mock mode, returning mock reader")
		return bufio.NewReader(stdin), func() {}, nil
	}

	state, err := term.MakeRaw(stdinFileDescriptor)
	if err != nil {
		return nil, func() {}, errors.Wrap(err, "can't set stdin to raw")
	}

	return bufio.NewReader(stdin), func() {
		if err := term.Restore(stdinFileDescriptor, state); err != nil {
			logrus.WithError(err).Warn("Failed to restore terminal")
		}
	}, err
}

func DoNotTrack() bool {
	v, _ := strconv.ParseBool(os.Getenv("DO_NOT_TRACK"))
	return v
}

// TerminalWidth returns the current terminal width, falling back to 60 if not a TTY.
func TerminalWidth() int {
	w, _, err := term.GetSize(stdoutFileDescriptor)
	if err != nil || w <= 0 {
		return 60
	}
	return w
}
