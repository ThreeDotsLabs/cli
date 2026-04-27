package trainings

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/git"
)

func printInfoSection(name string) {
	fmt.Println()
	fmt.Println(color.New(color.Bold).Sprint(name))
	fmt.Println(color.HiBlackString(strings.Repeat("─", len(name))))
}

func (h *Handlers) Info(ctx context.Context) error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return nil
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)

	trainingConfig := h.config.TrainingConfig(trainingRootFs)
	printGitNotices(trainingConfig)

	exerciseConfig := h.config.ExerciseConfig(trainingRootFs)

	printInfoSection("Training")
	fmt.Println("Name:    ", color.CyanString(trainingConfig.TrainingName))
	fmt.Println("Root dir:", color.CyanString(trainingRoot))

	printInfoSection("Current exercise")
	fmt.Println("ID:     ", color.CyanString(exerciseConfig.ExerciseID))
	fmt.Println("Files:  ", color.CyanString(h.generateRunTerminalPath(trainingRootFs)))
	exerciseURL := internal.ExerciseURL(trainingConfig.TrainingName, exerciseConfig.ExerciseID)
	fmt.Println("Content:", color.CyanString(exerciseURL))

	if trainingConfig.GitConfigured {
		printInfoSection("Git")
		if !trainingConfig.GitEnabled {
			fmt.Println("Status:", color.YellowString("disabled"))
		} else {
			gitOps := h.newGitOps()
			fmt.Println("Status:", color.GreenString("enabled"))

			if branch, err := gitOps.CurrentBranch(); err == nil && branch != "" {
				fmt.Println("Branch:", color.CyanString(branch))
			}

			if !exerciseConfig.IsTextOnly && exerciseConfig.Directory != "" {
				if gitOps.HasUncommittedChanges(exerciseConfig.Directory) {
					fmt.Println("Changes:", color.YellowString("uncommitted changes in %s", exerciseConfig.Directory))
				} else {
					fmt.Println("Changes:", color.HiBlackString("none"))
				}
			}

			if log, err := gitOps.Log(1); err == nil && log != "" {
				fmt.Println("Last commit:", color.HiBlackString(log))
			}
		}
	}

	printInfoSection("Environment")
	fmt.Println("CLI version:", color.CyanString(internal.BinaryVersion()))
	fmt.Println("OS:         ", color.CyanString(runtime.GOOS+"/"+runtime.GOARCH))

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = os.Getenv("COMSPEC")
	}
	if shell == "" {
		shell = "unknown"
	}
	fmt.Println("Shell:      ", color.CyanString(shell))
	fmt.Println("Terminal:   ", color.CyanString(internal.IsStdinTerminalReason()))

	if gitPath, err := exec.LookPath("git"); err == nil {
		fmt.Println("Git path:   ", color.CyanString(gitPath))
		if v, err := git.CheckVersion(); err == nil {
			fmt.Println("Git version:", color.CyanString(v.String()))
		}
	} else {
		fmt.Println("Git path:   ", color.YellowString("not found in PATH"))
	}

	fmt.Println()
	return nil
}
