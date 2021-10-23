package internal

import (
	"fmt"
	"io"
	"os"
	"strings"
)

var stdin io.Reader = os.Stdin
var stdout io.Writer = os.Stdout

func ConfirmPrompt(msg string) bool {
	defer fmt.Fprintln(stdout)

	for {
		fmt.Fprintf(stdout, "%s [y/n]: ", msg)

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
	defer fmt.Println()

	for {
		fmt.Fprintf(stdout, "%s [Y/n]: ", msg)

		var input string
		_, _ = fmt.Fscanln(stdin, &input)

		input = strings.ToLower(input)
		if input == "n" || input == "no" {
			return false
		}

		return true
	}
}
