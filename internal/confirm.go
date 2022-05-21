package internal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

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

type Action struct {
	Shortcut        rune
	ShortcutAliases []rune

	Action string
}

func (a Action) KeyString() string {
	if a.Shortcut == '\n' {
		return "ENTER"
	}

	return string(a.Shortcut)
}

type Actions []Action

func (a Actions) ReadKeyFromInput(char rune) (rune, bool) {
	for _, action := range a {
		if action.Shortcut == char {
			return action.Shortcut, true
		}

		for _, alias := range action.ShortcutAliases {
			if alias == char {
				return action.Shortcut, true
			}
		}
	}

	return rune(0), false
}

func ConfirmPromptDefaultYes(action string) bool {
	promptValue := Prompt(
		Actions{
			{Shortcut: '\n', Action: action, ShortcutAliases: []rune{'\r'}},
			{Shortcut: 'q', Action: "quit"},
		},
		os.Stdin,
		os.Stdout,
	)
	return promptValue == '\n'
}

const endOfTextChar = "\x03"

func Prompt(actions Actions, stdin io.Reader, stdout io.Writer) rune {
	defer func() {
		_, _ = fmt.Fprintln(stdout)
	}()

	in, clean, err := NewRawTerminalReader(stdin)
	defer clean()

	enterRequired := false

	if err != nil {
		logrus.WithError(err).Info("Can't read char from terminal, fallback to standard stdin reader")
		enterRequired = true
		in = bufio.NewReader(stdin)
	}

	var actionsStr []string
	for _, action := range actions {
		keyString := action.KeyString()
		if enterRequired && action.Shortcut == '\n' {
			keyString += " and ENTER"
		}

		actionsStr = append(actionsStr, fmt.Sprintf(
			"%s to %s",
			color.New(color.Bold).Sprint(keyString),
			action.Action,
		))
	}

	_, _ = fmt.Fprintf(stdout, "Press "+formatActionsMessage(actionsStr)+" ")

	for {
		char, _, err := in.ReadRune()
		if err != nil {
			logrus.WithError(err).Fatal("Can't read char from terminal")
		}
		input := strings.ToLower(string(char))

		logrus.WithField("input", input).Debug("Received input")

		if input == endOfTextChar {
			clean()
			os.Exit(0)
		}

		if key, ok := actions.ReadKeyFromInput(char); ok {
			return key
		}
	}
}

func formatActionsMessage(actionsStr []string) string {
	switch len(actionsStr) {
	case 0:
		return ""
	case 1:
		return actionsStr[0]
	default:
		return strings.Join(actionsStr[:len(actionsStr)-1], ", ") + " or " + actionsStr[len(actionsStr)-1]
	}
}
