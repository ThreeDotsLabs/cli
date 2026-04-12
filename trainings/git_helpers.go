package trainings

import (
	"errors"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/git"
)

// errBackupAborted signals that the user (or non-interactive environment) refused
// to proceed with a destructive operation after backup-branch creation failed.
// Callers that detect this via errors.Is should treat it as a clean abort, not a crash.
var errBackupAborted = errors.New("aborted: backup branch creation failed")

// gitDefaultConfig sets the default git integration fields on a training config.
// Centralizes the 5 fields so init and clone can't drift.
func gitDefaultConfig(cfg *config.TrainingConfig) {
	cfg.GitConfigured = true
	cfg.GitEnabled = true
	cfg.GitAutoCommit = true
	cfg.GitAutoGolden = false
	cfg.GitGoldenMode = "compare"
}

// stageInitialFiles stages the base set of files for an initial commit:
// .tdl-training, .gitignore, and optionally go.work if a Go workspace exists.
// Extra files (e.g. a cloned exercise directory) can be appended.
func stageInitialFiles(gitOps *git.Ops, trainingRootDir string, extraFiles ...string) {
	files := []string{".tdl-training", ".gitignore"}
	if hasGoWorkspace(trainingRootDir) {
		files = append(files, "go.work")
	}
	files = append(files, extraFiles...)
	if err := gitOps.AddFiles(files...); err != nil {
		logrus.WithError(err).Warn("Could not stage initial files")
	}
}

// saveToBackupBranch creates a named backup branch at HEAD before a destructive
// operation. On success it prints the standard "git branch X" info line and returns
// nil. On failure it explains the risk and asks the user whether to continue anyway:
//   - In interactive mode: prompts "Continue anyway? [y/n]". Yes → nil, no → errBackupAborted.
//   - In non-interactive mode: always aborts by returning errBackupAborted (can't prompt).
//
// Returning errBackupAborted means the caller MUST NOT proceed with the destructive
// operation. Returning nil means either the branch was created, or the user explicitly
// confirmed they want to continue without one.
func saveToBackupBranch(gitOps *git.Ops, backupBranch string) error {
	if err := gitOps.CreateBranchFromHead(backupBranch); err == nil {
		gitOps.PrintInfo(fmt.Sprintf("git branch %s", backupBranch))
		return nil
	} else {
		fmt.Println(formatGitWarning("Could not save your code to a backup branch", err))
		fmt.Println()
		fmt.Println(color.YellowString("  Without a backup branch, finding your previous code"))
		fmt.Println(color.YellowString("  will require ") + color.CyanString("git reflog") + color.YellowString("."))
		fmt.Println()

		if !internal.IsStdinTerminal() {
			fmt.Println(color.YellowString("  Aborting to protect your code (not running interactively)."))
			return errBackupAborted
		}

		if !internal.FConfirmPrompt(color.YellowString("Continue anyway?"), os.Stdin, os.Stdout) {
			fmt.Println(color.YellowString("  Aborted."))
			return errBackupAborted
		}
		fmt.Println(color.YellowString("  Continuing without backup."))
		return nil
	}
}

// saveProgress performs the reset→stage→commit pattern used to save user work
// before switching exercises. It is a no-op if there are no changes to commit.
func saveProgress(gitOps *git.Ops, dir string, commitMsg string) {
	_ = gitOps.ResetStaging()
	if err := gitOps.AddAll(dir); err == nil && gitOps.HasStagedChanges() {
		if err := gitOps.Commit(commitMsg); err != nil {
			fmt.Println(formatGitWarning("Could not auto-commit your progress", err))
		}
	}
}
