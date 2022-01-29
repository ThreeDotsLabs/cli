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
		Input          string
		ExpectedResult bool
	}{
		{
			Input:          "\n",
			ExpectedResult: true,
		},
		{
			Input:          "\ny\n",
			ExpectedResult: true,
		},
		{
			Input:          "\nn\n",
			ExpectedResult: true,
		},
		{
			Input:          "y\n",
			ExpectedResult: true,
		},
		{
			Input:          "Y\n",
			ExpectedResult: true,
		},
		{
			Input:          "n\n",
			ExpectedResult: false,
		},
		{
			Input:          "N\n",
			ExpectedResult: false,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.Input, func(t *testing.T) {
			stdin := bytes.NewBufferString(tc.Input)
			stdout := bytes.NewBuffer(nil)

			ok := FConfirmPromptDefaultYes("some msg", stdin, stdout)
			assert.Equal(t, tc.ExpectedResult, ok)
			assert.Equal(t, "some msg [y/n] (default: y): \n", stdout.String())
		})
	}
}
