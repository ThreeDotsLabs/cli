package internal

import (
	"fmt"
	"strings"
)

// todo - test
func ConfirmPrompt(msg string) bool {
	defer fmt.Println()

	for {
		fmt.Printf("%s [y/n]: ", msg)

		var input string
		_, err := fmt.Scanln(&input)
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
		fmt.Printf("%s [Y/n]: ", msg)

		var input string
		_, _ = fmt.Scanln(&input)

		input = strings.ToLower(input)
		if input == "n" || input == "no" {
			return false
		}

		return true
	}
}
