package internal

import (
	"fmt"
	"strings"
)

// todo - test
func ConfirmPrompt(msg string) bool {
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
