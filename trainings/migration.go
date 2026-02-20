package trainings

import (
	"fmt"
	"strings"

	"github.com/fatih/color"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
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
	fmt.Println("  The new version tracks your progress with git — branches,")
	fmt.Println("  commits, and diff with official solutions.")
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
