package internal

import (
	"bufio"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

const stdinFileDescriptor = 0

// NewRawTerminalReader returns raw terminal reader which allows reading stdin without hitting enter.
func NewRawTerminalReader(stdin io.Reader) (*bufio.Reader, func(), error) {
	if stdin != os.Stdin {
		logrus.Info("Mock mode, returning mock reader")
		return bufio.NewReader(stdin), func() {}, nil
	}

	state, err := terminal.MakeRaw(stdinFileDescriptor)
	if err != nil {
		return nil, nil, errors.Wrap(err, "can't set stdin to raw")
	}

	return bufio.NewReader(stdin), func() {
		if err := terminal.Restore(stdinFileDescriptor, state); err != nil {
			logrus.WithError(err).Warn("Failed to restore terminal")
		}
	}, err
}
