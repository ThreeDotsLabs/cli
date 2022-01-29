package internal

import (
	"fmt"
	"io"
	"os"
	"strings"
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

func ConfirmPromptDefaultYes(msg string) bool {
	return FConfirmPromptDefaultYes(msg, os.Stdin, os.Stdout)
}

func FConfirmPromptDefaultYes(msg string, stdin io.Reader, stdout io.Writer) bool {
	defer func() {
		_, _ = fmt.Fprintln(stdout)
	}()

	for {
		_, _ = fmt.Fprintf(stdout, "%s [y/n] (default: y): ", msg)

		var input string
		_, _ = fmt.Fscanln(stdin, &input)

		input = strings.ToLower(input)
		if input == "n" || input == "no" {
			return false
		} else {
			return true
		}
	}
}
