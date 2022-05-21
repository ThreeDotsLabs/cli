package internal

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfirmPrompt(t *testing.T) {
	testCases := []struct {
		Input           string
		ExpectedMessage string
		ExpectedResult  bool
	}{
		{
			Input:           "y\n",
			ExpectedMessage: "some msg [y/n]: \n",
			ExpectedResult:  true,
		},
		{
			Input:           "Y\n",
			ExpectedMessage: "some msg [y/n]: \n",
			ExpectedResult:  true,
		},
		{
			Input:           "\ny\n",
			ExpectedMessage: "some msg [y/n]: some msg [y/n]: \n",
			ExpectedResult:  true,
		},
		{
			Input:           "n\n",
			ExpectedMessage: "some msg [y/n]: \n",
			ExpectedResult:  false,
		},
		{
			Input:           "N\n",
			ExpectedMessage: "some msg [y/n]: \n",
			ExpectedResult:  false,
		},
		{
			Input:           "\nn\n",
			ExpectedMessage: "some msg [y/n]: some msg [y/n]: \n",
			ExpectedResult:  false,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.Input, func(t *testing.T) {
			stdin := bytes.NewBufferString(tc.Input)
			stdout := bytes.NewBuffer(nil)

			ok := FConfirmPrompt("some msg", stdin, stdout)
			assert.Equal(t, tc.ExpectedResult, ok)
			assert.Equal(t, tc.ExpectedMessage, stdout.String())
		})
	}
}

func TestConfirmPromptDefaultYes(t *testing.T) {
	testCases := []struct {
		Input           string
		Actions         Actions
		ExpectedMessage string
		ExpectedResult  rune
	}{
		{
			Input: "\n",
			Actions: Actions{
				{
					Shortcut:        '\n',
					ShortcutAliases: nil,
					Action:          "some msg",
				},
				{
					Shortcut:        'q',
					ShortcutAliases: nil,
					Action:          "quit",
				},
			},
			ExpectedMessage: "Press ENTER to some msg or q to quit \n",
			ExpectedResult:  '\n',
		},
		{
			Input: "q",
			Actions: Actions{
				{
					Shortcut:        '\n',
					ShortcutAliases: nil,
					Action:          "some msg",
				},
				{
					Shortcut:        'q',
					ShortcutAliases: nil,
					Action:          "quit",
				},
			},
			ExpectedMessage: "Press ENTER to some msg or q to quit \n",
			ExpectedResult:  'q',
		},
		{
			Input: "na\n",
			Actions: Actions{
				{
					Shortcut:        '\n',
					ShortcutAliases: nil,
					Action:          "some msg",
				},
				{
					Shortcut:        'q',
					ShortcutAliases: nil,
					Action:          "quit",
				},
			},
			ExpectedMessage: "Press ENTER to some msg or q to quit \n",
			ExpectedResult:  '\n',
		},
		{
			Input: "\r",
			Actions: Actions{
				{
					Shortcut:        '\n',
					ShortcutAliases: []rune{'\r'},
					Action:          "some msg",
				},
				{
					Shortcut:        'q',
					ShortcutAliases: nil,
					Action:          "quit",
				},
			},
			ExpectedMessage: "Press ENTER to some msg or q to quit \n",
			ExpectedResult:  '\n',
		},
		{
			Input: "\n",
			Actions: Actions{
				{
					Shortcut: '\n',
					Action:   "some msg",
				},
				{
					Shortcut: 'r',
					Action:   "retry",
				},
				{
					Shortcut: 'q',
					Action:   "quit",
				},
			},
			ExpectedMessage: "Press ENTER to some msg, r to retry or q to quit \n",
			ExpectedResult:  '\n',
		},
		{
			Input: "\n",
			Actions: Actions{
				{
					Shortcut: '\n',
					Action:   "some msg",
				},
			},
			ExpectedMessage: "Press ENTER to some msg \n",
			ExpectedResult:  '\n',
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.Input, func(t *testing.T) {
			stdin := bytes.NewBufferString(tc.Input)
			stdout := bytes.NewBuffer(nil)

			ok := Prompt(tc.Actions, stdin, stdout)
			assert.Equal(t, tc.ExpectedResult, ok)
			assert.Equal(t, tc.ExpectedMessage, stdout.String())
		})
	}
}
