package trainings

import (
	"fmt"
	"os"
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

// printGitNowAvailableNotice shows a banner (at most once per 24 h) when git is available
// but the workspace has git disabled. Covers both workspaces where git was missing at init
// and older workspaces that were silently disabled before the terminal-detection fix.
func printGitNowAvailableNotice(cfg config.TrainingConfig) {
	if !cfg.GitConfigured || cfg.GitEnabled {
		return
	}

	if _, err := git.CheckVersion(); err != nil {
		// git still not usable
		return
	}

	if !internal.ShouldShowGitInstallNotice() {
		return
	}
	_ = internal.RecordGitInstallNoticeShown()

	sep := color.HiBlackString(strings.Repeat("─", internal.TerminalWidth()))
	title := color.New(color.Bold, color.FgHiGreen).Sprint("  *** Git is now available! ***")
	initCmd := color.CyanString("tdl training init %s .", cfg.TrainingName)

	fmt.Println(sep)
	fmt.Println(title)
	fmt.Println()
	fmt.Println("  Git was not enabled for this workspace.")
	fmt.Println("  You can enable git integration by reinitializing:")
	fmt.Println()
	fmt.Printf("    cd ..\n")
	fmt.Printf("    mkdir my-training && cd my-training\n")
	fmt.Printf("    %s\n", initCmd)
	fmt.Println()
	fmt.Println("  Your progress will be restored automatically.")
	fmt.Println(sep)
	fmt.Println()

	if internal.IsStdinTerminal() {
		choice := internal.Prompt(
			internal.Actions{
				{Shortcut: '\n', Action: "continue for now", ShortcutAliases: []rune{'\r'}},
				{Shortcut: 'q', Action: "quit to reinitialize"},
			},
			os.Stdin,
			os.Stdout,
		)
		fmt.Println()
		if choice == 'q' {
			os.Exit(0)
		}
	}
}

// printGitNotices shows all relevant git migration/availability notices.
func printGitNotices(cfg config.TrainingConfig) {
	printGitMigrationNotice(cfg)
	printGitNowAvailableNotice(cfg)
}

// printInitNeedsFreshDir is shown during `tdl training init` when the user runs init
// in a directory that was already set up without git. Unlike printGitNowAvailableNotice
// (rate-limited, shown during run/next/etc.), this fires unconditionally because the
// user explicitly ran init and needs clear, actionable guidance.
func printInitNeedsFreshDir(cfg config.TrainingConfig) {
	sep := color.HiBlackString(strings.Repeat("─", internal.TerminalWidth()))
	initCmd := color.CyanString("tdl training init %s .", cfg.TrainingName)

	_, gitErr := git.CheckVersion()
	gitAvailable := gitErr == nil

	fmt.Println(sep)
	if gitAvailable {
		title := color.New(color.Bold, color.FgHiGreen).Sprint("  *** Git is now available! ***")
		fmt.Println(title)
		fmt.Println()
		fmt.Println("  Git is now installed, but this workspace was set up without git tracking.")
		fmt.Println("  To enable git integration, initialize in a fresh empty directory.")
	} else {
		title := color.New(color.Bold, color.FgHiYellow).Sprint("  *** Fresh directory required ***")
		fmt.Println(title)
		fmt.Println()
		fmt.Println("  This workspace was set up without git tracking.")
		fmt.Println("  Once git is installed, initialize in a fresh directory to enable it.")
	}
	fmt.Println()
	fmt.Println("  Your progress is saved on the server and will be restored")
	fmt.Println("  automatically: solutions, completion history, everything.")
	fmt.Println()
	fmt.Println("  Create a new directory and reinitialize:")
	fmt.Println()
	fmt.Printf("    cd ..\n")
	fmt.Printf("    mkdir my-training && cd my-training\n")
	fmt.Printf("    %s\n", initCmd)
	fmt.Println(sep)
	fmt.Println()
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
