package trainings

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// formatServerError translates gRPC errors into user-friendly messages.
// Unknown codes pass through unchanged for formatUnexpectedError in main.go.
func formatServerError(err error) error {
	logrus.WithError(err).Debug("server error")

	switch status.Code(errors.Cause(err)) {
	case codes.Unavailable:
		return UserFacingError{
			Msg:          "Verification server is not reachable.",
			SolutionHint: "Check your internet connection and try again. If the problem persists, the server may be temporarily down.",
		}
	case codes.DeadlineExceeded:
		return UserFacingError{
			Msg:          "Verification timed out.",
			SolutionHint: "Check your internet connection and try again.",
		}
	case codes.Unauthenticated:
		return UserFacingError{
			Msg:          "Authentication failed.",
			SolutionHint: "Run " + color.CyanString("tdl training configure <token>") + " to set up your token.",
		}
	case codes.ResourceExhausted:
		return UserFacingError{
			Msg:          "Server is overloaded, please try again later.",
			SolutionHint: "Wait a moment and try again.",
		}
	default:
		return err
	}
}

// formatConnectionError wraps a failed Ping into a user-friendly connectivity error.
func formatConnectionError(err error) error {
	return UserFacingError{
		Msg: "Could not connect to the server.",
		SolutionHint: fmt.Sprintf(
			"Please check:\n"+
				"  1. Your internet connection\n"+
				"  2. Firewall or VPN settings that may block outgoing connections\n"+
				"  3. The server may be temporarily unavailable — try again in a few minutes\n\n"+
				"%s",
			color.HiBlackString("Raw error: %s", err),
		),
	}
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
