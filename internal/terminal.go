package internal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

const stdinFileDescriptor = 0
const stdoutFileDescriptor = 1

func stdinTerminalReason() (bool, string) {
	if v, _ := strconv.ParseBool(os.Getenv("TDL_FORCE_INTERACTIVE")); v {
		return true, "true (TDL_FORCE_INTERACTIVE)"
	}
	if terminal.IsTerminal(stdinFileDescriptor) {
		return true, "true (native console)"
	}
	// On Windows, mintty (Git Bash) and MSYS2 terminals use pipes instead of
	// native console handles, so IsTerminal returns false even though they fully
	// support ANSI/VT sequences. The TERM env var is a reliable signal for these.
	if term := os.Getenv("TERM"); term != "" {
		return true, fmt.Sprintf("true (TERM=%s)", term)
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

	state, err := terminal.MakeRaw(stdinFileDescriptor)
	if err != nil {
		return nil, func() {}, errors.Wrap(err, "can't set stdin to raw")
	}

	return bufio.NewReader(stdin), func() {
		if err := terminal.Restore(stdinFileDescriptor, state); err != nil {
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
	w, _, err := terminal.GetSize(stdoutFileDescriptor)
	if err != nil || w <= 0 {
		return 60
	}
	return w
}
