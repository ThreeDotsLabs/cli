package internal

import (
	"bufio"
	"io"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

const stdinFileDescriptor = 0
const stdoutFileDescriptor = 1

// IsStdinTerminal returns true if stdin is connected to a terminal (not a pipe or file).
func IsStdinTerminal() bool {
	if v, _ := strconv.ParseBool(os.Getenv("TDL_FORCE_INTERACTIVE")); v {
		return true
	}
	return terminal.IsTerminal(stdinFileDescriptor)
}

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
