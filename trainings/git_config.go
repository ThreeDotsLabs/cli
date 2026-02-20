package trainings

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/pkg/errors"
)

// ConfigureGit lets users change git integration settings for the current training.
func (h *Handlers) ConfigureGit() error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if err != nil {
		h.printNotInATrainingDirectory()
		return nil
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)
	cfg := h.config.TrainingConfig(trainingRootFs)
	printGitMigrationNotice(cfg)

	if !cfg.GitConfigured || !cfg.GitEnabled {
		fmt.Println("Git integration is not enabled for this training.")
		fmt.Println("To enable it, reinitialize with: " + color.CyanString("tdl training init"))
		return nil
	}

	fmt.Printf("Current settings for %s:\n", color.CyanString(cfg.TrainingName))
	fmt.Printf("  Auto-commit:  %s\n", formatBool(cfg.GitAutoCommit))
	fmt.Printf("  Auto-golden:  %s\n\n", formatBool(cfg.GitAutoGolden))

	autoCommit, autoGolden := promptGitPreferences()

	cfg.GitAutoCommit = autoCommit
	cfg.GitAutoGolden = autoGolden

	if err := h.config.WriteTrainingConfig(cfg, trainingRootFs); err != nil {
		return errors.Wrap(err, "can't update training config")
	}

	fmt.Println(color.GreenString("Settings updated."))
	return nil
}

func formatBool(v bool) string {
	if v {
		return color.GreenString("on")
	}
	return color.YellowString("off")
}
