package trainings

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/internal"
	mcppkg "github.com/ThreeDotsLabs/cli/trainings/mcp"
)

// cliTools are AI coding tool binaries to look for in PATH.
var cliTools = []string{
	"claude",
	"cursor",
	"aider",
	"opencode",
	"copilot",
	"windsurf",
}

// macOSApps are AI coding tool application bundles to check on darwin.
var macOSApps = []string{
	"/Applications/Cursor.app",
	"/Applications/Windsurf.app",
}

// detectAICodingTools returns the names of AI coding tools found on this system.
func detectAICodingTools() []string {
	var found []string

	for _, tool := range cliTools {
		if path, err := exec.LookPath(tool); err == nil {
			logrus.WithFields(logrus.Fields{"tool": tool, "path": path}).Debug("AI coding tool found in PATH")
			found = append(found, tool)
		} else {
			logrus.WithField("tool", tool).Debug("AI coding tool not found in PATH")
		}
	}

	if runtime.GOOS == "darwin" {
		for _, app := range macOSApps {
			if _, err := os.Stat(app); err == nil {
				logrus.WithField("app", app).Debug("AI coding app found")
				// Extract app name (e.g. "Cursor.app")
				parts := strings.Split(app, "/")
				found = append(found, parts[len(parts)-1])
			} else {
				logrus.WithField("app", app).Debug("AI coding app not found")
			}
		}
	}

	// Deduplicate: if "cursor" CLI is found AND "Cursor.app" exists, keep just "cursor"
	seen := make(map[string]bool)
	var deduped []string
	for _, name := range found {
		key := strings.ToLower(strings.TrimSuffix(name, ".app"))
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, name)
		}
	}

	logrus.WithField("detected", deduped).Debug("AI coding tool detection complete")

	return deduped
}

// promptMCPSetup asks the user whether to enable the MCP server.
func promptMCPSetup() bool {
	fmt.Println()
	fmt.Println(color.New(color.Bold).Sprint("  MCP server (optional)"))
	fmt.Println()
	fmt.Println("  If you plan to use an AI coding tool (Claude Code, Cursor, etc.) alongside")
	fmt.Println("  this training, the MCP server lets it run exercises and check results.")
	fmt.Println("  It listens on 127.0.0.1 (localhost only) while " + color.CyanString("tdl tr run") + " is active.")
	fmt.Println()
	fmt.Println("  The training works perfectly fine without it.")
	fmt.Println("  You can change this later with: " + color.CyanString("tdl training settings"))
	fmt.Println()

	choice := internal.Prompt(
		internal.Actions{
			{Shortcut: '\n', Action: "enable MCP server", ShortcutAliases: []rune{'\r'}},
			{Shortcut: 'n', Action: "skip — not using AI tools"},
		},
		os.Stdin,
		os.Stdout,
	)
	fmt.Println()

	return choice == '\n'
}

// configureMCPIfNeeded checks the global config for MCP preference.
// If not yet configured, it detects AI coding tools and prompts the user.
//
// Key invariant: when no AI tools are detected, nothing is written to config.
// This ensures that if the user installs an AI tool later, MCPConfigured will
// still be false and the prompt will appear on the next run.
func (h *Handlers) configureMCPIfNeeded(trainingRootFs *afero.BasePathFs, agentInstructions []byte) {
	globalCfg := h.config.GlobalConfig()
	filePort := h.mcpPort // preserve for .mcp.json before we might zero h.mcpPort

	if globalCfg.MCPConfigured {
		if !globalCfg.MCPEnabled {
			h.mcpPort = 0
			h.loopState = nil
		}
		// Always ensure files exist when MCP has been configured
		// (user may enable later via settings, files should already be there)
		h.ensureMCPFiles(trainingRootFs, filePort, agentInstructions)
		return
	}

	// Not yet configured — detect AI tools
	tools := detectAICodingTools()

	if len(tools) == 0 {
		// No AI tools found. Do NOT write config — when the user installs
		// a tool later, MCPConfigured will still be false and this check
		// will run again, showing the prompt.
		logrus.Debug("No AI coding tools detected, skipping MCP setup")
		h.mcpPort = 0
		h.loopState = nil
		return
	}

	if !internal.IsStdinTerminal() {
		// Non-interactive — can't prompt, skip for now
		h.mcpPort = 0
		h.loopState = nil
		return
	}

	enabled := promptMCPSetup()

	globalCfg.MCPConfigured = true
	globalCfg.MCPEnabled = enabled
	if err := h.config.WriteGlobalConfig(globalCfg); err != nil {
		logrus.WithError(errors.WithStack(err)).Warn("Could not save MCP preference")
	}

	if !enabled {
		h.mcpPort = 0
		h.loopState = nil
	} else {
		if h.loopState == nil {
			h.loopState = mcppkg.NewLoopState()
			h.loopState.SetCLIVersion(h.cliMetadata.Version)
		}
	}

	// Always create files when AI tools are present
	h.ensureMCPFiles(trainingRootFs, filePort, agentInstructions)

	if enabled {
		fmt.Println(color.GreenString("  MCP enabled.") + " Created " + color.CyanString(".mcp.json") + " in your training directory.")
		fmt.Println("  Restart your AI coding tools to pick up the new MCP server config.")
		fmt.Println()
	}
}
