package trainings

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
)

// SettingsOptions holds optional flag overrides for the settings command.
// Nil pointers mean "not set" (use interactive form); non-nil means apply directly.
type SettingsOptions struct {
	AutoCommit *bool
	AutoSync   *bool
	MCP        *bool
	SyncMode   *string
}

func (o SettingsOptions) anyFlagSet() bool {
	return o.AutoCommit != nil || o.AutoSync != nil || o.MCP != nil || o.SyncMode != nil
}

// Settings lets users view and change training settings (git + MCP).
func (h *Handlers) Settings(opts SettingsOptions) error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if err != nil {
		h.printNotInATrainingDirectory()
		return nil
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)
	cfg := h.config.TrainingConfig(trainingRootFs)
	printGitNotices(cfg)

	globalCfg := h.config.GlobalConfig()
	gitAvailable := cfg.GitConfigured && cfg.GitEnabled

	if opts.anyFlagSet() {
		return h.applySettingsFlags(opts, &cfg, &globalCfg, gitAvailable, trainingRootFs)
	}

	if !internal.IsStdinTerminal() {
		printCurrentSettings(cfg, globalCfg, gitAvailable)
		fmt.Println()
		fmt.Println("To change settings, use flags:")
		fmt.Println("  " + color.CyanString(internal.BinaryName()+" training settings --auto-commit=on --mcp=off"))
		return nil
	}

	return h.runSettingsForm(&cfg, &globalCfg, gitAvailable, trainingRootFs)
}

func (h *Handlers) applySettingsFlags(opts SettingsOptions, cfg *config.TrainingConfig, globalCfg *config.GlobalConfig, gitAvailable bool, trainingRootFs *afero.BasePathFs) error {
	if opts.SyncMode != nil {
		switch *opts.SyncMode {
		case "compare", "merge", "override":
		default:
			fmt.Printf("Invalid sync mode %q. Must be one of: compare, merge, override\n", *opts.SyncMode)
			return nil
		}
	}

	gitFlagSet := opts.AutoCommit != nil || opts.AutoSync != nil || opts.SyncMode != nil
	if gitFlagSet && !gitAvailable {
		fmt.Println("Git integration is not enabled for this training.")
		fmt.Println("To enable it, reinitialize with: " + color.CyanString(internal.BinaryName()+" training init"))
		return nil
	}

	if gitAvailable {
		if opts.AutoCommit != nil {
			cfg.GitAutoCommit = *opts.AutoCommit
		}
		if opts.AutoSync != nil {
			cfg.GitAutoGolden = *opts.AutoSync
		}
		if opts.SyncMode != nil {
			cfg.GitGoldenMode = *opts.SyncMode
		}
	}

	if opts.MCP != nil {
		globalCfg.MCPConfigured = true
		globalCfg.MCPEnabled = *opts.MCP
	}

	if err := h.config.WriteTrainingConfig(*cfg, trainingRootFs); err != nil {
		return errors.Wrap(err, "can't update training config")
	}
	if err := h.config.WriteGlobalConfig(*globalCfg); err != nil {
		return errors.Wrap(err, "can't update global config")
	}

	printCurrentSettings(*cfg, *globalCfg, gitAvailable)
	fmt.Println(color.GreenString("\nSettings saved."))
	return nil
}

func printCurrentSettings(cfg config.TrainingConfig, globalCfg config.GlobalConfig, gitAvailable bool) {
	fmt.Printf("Settings for %s:\n", color.CyanString(cfg.TrainingName))
	if gitAvailable {
		syncMode := cfg.GitGoldenMode
		if syncMode == "" {
			syncMode = "compare"
		}
		fmt.Printf("  Auto-commit:  %s\n", formatBool(cfg.GitAutoCommit))
		fmt.Printf("  Auto-sync:    %s\n", formatBool(cfg.GitAutoGolden))
		fmt.Printf("  Sync mode:    %s\n", syncMode)
	} else {
		fmt.Println("  Git:          " + color.YellowString("not enabled"))
	}
	fmt.Printf("  MCP server:   %s\n", formatBool(globalCfg.MCPEnabled))
}

func formatBool(v bool) string {
	if v {
		return color.GreenString("on")
	}
	return color.YellowString("off")
}
