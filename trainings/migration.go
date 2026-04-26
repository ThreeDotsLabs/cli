package trainings

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/fatih/color"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/git"
)

// printGitMigrationNotice prints a prominent banner for workspaces created
// before git integration was added. Old configs lack git_configured (zero value = false).
// New workspaces and --no-git workspaces both have git_configured = true.
func printGitMigrationNotice(cfg config.TrainingConfig) {
	if cfg.GitConfigured {
		return
	}

	sep := color.HiBlackString(strings.Repeat("─", internal.TerminalWidth()))
	title := color.New(color.Bold, color.FgHiYellow).Sprint("  *** CLI UPGRADE: Git Integration Available ***")
	initCmd := color.CyanString("tdl training init %s .", cfg.TrainingName)

	fmt.Println(sep)
	fmt.Println(title)
	fmt.Println()
	fmt.Println("  This workspace was created with an older version of the CLI.")
	fmt.Println("  The new version tracks your progress with git: branches,")
	fmt.Println("  commits, and diff with example solutions.")
	fmt.Println()
	fmt.Println("  All your progress is saved on the server and will be")
	fmt.Println("  restored automatically when you reinitialize — solutions,")
	fmt.Println("  completion history, everything migrated.")
	fmt.Println()
	fmt.Println("  To upgrade, create a new directory and reinitialize:")
	fmt.Println()
	fmt.Printf("    cd ..\n")
	fmt.Printf("    mkdir my-training && cd my-training\n")
	fmt.Printf("    %s\n", initCmd)
	fmt.Println(sep)
	fmt.Println()
}

// printGitNowAvailableNotice shows a banner when git has become available
// since the workspace was created without it (git was missing/too old).
// Does not trigger for users who chose --no-git (GitUnavailable = false).
func printGitNowAvailableNotice(cfg config.TrainingConfig) {
	if !cfg.GitConfigured || cfg.GitEnabled || !cfg.GitUnavailable {
		return
	}

	if _, err := git.CheckVersion(); err != nil {
		// git still not usable
		return
	}

	sep := color.HiBlackString(strings.Repeat("─", internal.TerminalWidth()))
	title := color.New(color.Bold, color.FgHiGreen).Sprint("  *** Git is now available! ***")
	initCmd := color.CyanString("tdl training init %s .", cfg.TrainingName)

	fmt.Println(sep)
	fmt.Println(title)
	fmt.Println()
	fmt.Println("  Git was not available when this workspace was created.")
	fmt.Println("  You can enable git integration by reinitializing:")
	fmt.Println()
	fmt.Printf("    cd ..\n")
	fmt.Printf("    mkdir my-training && cd my-training\n")
	fmt.Printf("    %s\n", initCmd)
	fmt.Println()
	fmt.Println("  Your progress will be restored automatically.")
	fmt.Println(sep)
	fmt.Println()
}

// printGitNotices shows all relevant git migration/availability notices.
func printGitNotices(cfg config.TrainingConfig) {
	printGitMigrationNotice(cfg)
	printGitNowAvailableNotice(cfg)
}

// showGitInstallNoticeIfDue shows the "git not installed" notice at most once per 24 h.
// Fires whenever git is disabled and still not installed — covers both workspaces where
// git was missing at init (GitUnavailable=true) and older workspaces that were silently
// disabled before the terminal-detection fix. Returns false if the user chose to quit.
func showGitInstallNoticeIfDue(cfg config.TrainingConfig) bool {
	if !cfg.GitConfigured || cfg.GitEnabled {
		return true
	}
	if _, err := git.CheckVersion(); err == nil {
		return true // git is available — printGitNowAvailableNotice handles the reinitialize prompt
	}
	if !internal.ShouldShowGitInstallNotice() {
		return true // shown recently, skip
	}
	_ = internal.RecordGitInstallNoticeShown()
	printGitUnavailableNotice("Git is not installed.", git.InstallHint(runtime.GOOS))
	if !internal.IsStdinTerminal() {
		return true
	}
	return promptContinueWithoutGit()
}
