package trainings

import (
	"fmt"

	"github.com/fatih/color"
)

type UserFacingError struct {
	Msg          string
	SolutionHint string
}

func (u UserFacingError) Error() string {
	return u.Msg + " " + u.SolutionHint
}

// recoveryHint generates the re-init recovery instructions for a training.
// Directs users to create a new directory (not re-use the broken one).
func recoveryHint(trainingName string) string {
	return fmt.Sprintf(`Your progress is saved on the server.
To start fresh, create a new directory and re-initialize:

  cd ..
  mkdir my-training && cd my-training
  %s

This will re-download all your existing solutions.`,
		color.CyanString("tdl training init %s .", trainingName),
	)
}

// formatGitWarning formats a non-blocking git failure for user display.
// Always shows the actual git error — never hides it.
func formatGitWarning(operation string, err error) string {
	return color.YellowString("  ⚠ %s: %s", operation, err.Error())
}

// formatGitError formats a blocking git failure with recovery instructions.
// Always shows the actual git error + recovery hint.
func formatGitError(operation string, err error, trainingName string) string {
	return formatGitWarning(operation, err) + "\n\n" + recoveryHint(trainingName)
}
