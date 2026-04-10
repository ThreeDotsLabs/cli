package trainings

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/trainings/config"
)

func (h *Handlers) runSettingsForm(cfg *config.TrainingConfig, globalCfg *config.GlobalConfig, gitAvailable bool, trainingRootFs *afero.BasePathFs) error {
	autoCommit := cfg.GitAutoCommit
	autoSync := cfg.GitAutoGolden
	syncMode := cfg.GitGoldenMode
	if syncMode == "" {
		syncMode = "compare"
	}
	mcpEnabled := globalCfg.MCPEnabled

	var fields []huh.Field

	if gitAvailable {
		fields = append(fields,
			huh.NewConfirm().
				Title("Auto-commit").
				Description("Automatically commit your progress when you pass each exercise (default: on)").
				Affirmative("on").
				Negative("off").
				WithButtonAlignment(lipgloss.Left).
				Value(&autoCommit),

			huh.NewConfirm().
				Title("Auto-sync").
				Description("After passing, automatically sync your code with the example solution (default: off)").
				Affirmative("on").
				Negative("off").
				WithButtonAlignment(lipgloss.Left).
				Value(&autoSync),

			huh.NewSelect[string]().
				Title("Sync mode").
				Description("How to apply the example solution when syncing (default: compare)").
				Options(
					huh.NewOption("compare", "compare"),
					huh.NewOption("merge", "merge"),
					huh.NewOption("override", "override"),
				).
				Inline(true).
				Value(&syncMode),
		)
	} else {
		fmt.Println("Git integration is not enabled for this training.")
		fmt.Println("To enable it, reinitialize with: " + color.CyanString("tdl training init"))
		fmt.Println()
	}

	fields = append(fields,
		huh.NewConfirm().
			Title("MCP server").
			Description("Let AI coding tools (Claude Code, Cursor, etc.) run exercises and check results (default: on)").
			Affirmative("on").
			Negative("off").
			WithButtonAlignment(lipgloss.Left).
			Value(&mcpEnabled),
	)

	form := huh.NewForm(
		huh.NewGroup(fields...),
	).
		WithTheme(huh.ThemeFunc(huh.ThemeBase)).
		WithKeyMap(settingsKeyMap())

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("Cancelled.")
			return nil
		}
		return errors.Wrap(err, "settings form error")
	}

	if gitAvailable {
		cfg.GitAutoCommit = autoCommit
		cfg.GitAutoGolden = autoSync
		cfg.GitGoldenMode = syncMode
	}
	globalCfg.MCPConfigured = true
	globalCfg.MCPEnabled = mcpEnabled

	if err := h.config.WriteTrainingConfig(*cfg, trainingRootFs); err != nil {
		return errors.Wrap(err, "can't update training config")
	}
	if err := h.config.WriteGlobalConfig(*globalCfg); err != nil {
		return errors.Wrap(err, "can't update global config")
	}

	fmt.Println()
	printCurrentSettings(*cfg, *globalCfg, gitAvailable)
	fmt.Println(color.GreenString("\nSettings saved."))
	return nil
}

// settingsKeyMap returns a keymap that adds up/down arrow navigation between fields.
func settingsKeyMap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()

	// Add up/down arrows for navigating between all fields.
	km.Confirm.Next = key.NewBinding(key.WithKeys("enter", "tab", "down"), key.WithHelp("↓/enter", "next"))
	km.Confirm.Prev = key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑", "back"))
	km.Select.Next = key.NewBinding(key.WithKeys("enter", "tab", "down"), key.WithHelp("↓/enter", "next"))
	km.Select.Prev = key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑", "back"))

	return km
}
