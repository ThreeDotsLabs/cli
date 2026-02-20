package trainings

import (
	"fmt"
	"strings"
	"testing"
)

func TestRecoveryHint(t *testing.T) {
	hint := recoveryHint("go-event-driven")

	if !strings.Contains(hint, "progress is saved on the server") {
		t.Error("expected recovery hint to mention server-side progress")
	}
	if !strings.Contains(hint, "cd ..") {
		t.Error("expected recovery hint to escape broken directory")
	}
	if !strings.Contains(hint, "mkdir my-training") {
		t.Error("expected recovery hint to create new directory")
	}
	if !strings.Contains(hint, "tdl training init go-event-driven .") {
		t.Error("expected recovery hint to include training name and dot")
	}
	if !strings.Contains(hint, "re-download all your existing solutions") {
		t.Error("expected recovery hint to mention re-downloading solutions")
	}
}

func TestFormatGitWarning(t *testing.T) {
	err := fmt.Errorf("git merge: CONFLICT in file.go: exit status 1")
	warning := formatGitWarning("Could not merge", err)

	if !strings.Contains(warning, "Could not merge") {
		t.Error("expected warning to contain operation name")
	}
	if !strings.Contains(warning, "git merge: CONFLICT in file.go: exit status 1") {
		t.Error("expected warning to contain raw git error")
	}
	if !strings.Contains(warning, "⚠") {
		t.Error("expected warning to contain warning symbol")
	}
}

func TestFormatGitError(t *testing.T) {
	err := fmt.Errorf("git branch: fatal: branch already exists: exit status 128")
	msg := formatGitError("Could not save backup", err, "go-event-driven")

	if !strings.Contains(msg, "Could not save backup") {
		t.Error("expected error to contain operation name")
	}
	if !strings.Contains(msg, "git branch: fatal: branch already exists: exit status 128") {
		t.Error("expected error to contain raw git error")
	}
	if !strings.Contains(msg, "tdl training init go-event-driven .") {
		t.Error("expected error to contain recovery hint with training name")
	}
}
