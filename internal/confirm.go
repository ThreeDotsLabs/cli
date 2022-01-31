package internal

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

func ConfirmPrompt(msg string) bool {
	return FConfirmPrompt(msg, os.Stdin, os.Stdout)
}

func FConfirmPrompt(msg string, stdin io.Reader, stdout io.Writer) bool {
	defer func() {
		_, _ = fmt.Fprintln(stdout)
	}()

	for {
		_, _ = fmt.Fprintf(stdout, "%s [y/n]: ", msg)

		var input string
		_, err := fmt.Fscanln(stdin, &input)
		if err != nil {
			continue
		}
		if len(input) == 0 {
			continue
		}

		input = strings.ToLower(input)

		if input == "y" || input == "yes" {
			return true
		}
		if input == "n" || input == "no" {
			return false
		}
	}
}

func ConfirmPromptDefaultYes(action string) bool {
	return FConfirmPromptDefaultYes(action, os.Stdin, os.Stdout)
}

const endOfTextChar = "\x03"

func FConfirmPromptDefaultYes(action string, stdin io.Reader, stdout io.Writer) bool {
	defer func() {
		_, _ = fmt.Fprintln(stdout)
	}()

	_, _ = fmt.Fprintf(stdout, "\nPress ENTER to %s or q to quit ", action)

	in, clean, err := NewRawTerminalReader(stdin)
	if err != nil {
		logrus.WithError(err).Fatal("Can't read char from terminal")
	}
	defer clean()

	for {
		char, _, err := in.ReadRune()
		if err != nil {
			logrus.WithError(err).Fatal("Can't read char from terminal")
		}
		input := strings.ToLower(string(char))

		logrus.WithField("input", input).Debug("Received input")

		if input == "n" || input == "no" || input == "q" || input == endOfTextChar {
			return false
		} else if input == "\r" || input == "\n" {
			return true
		} else {
			continue
		}
	}
}
