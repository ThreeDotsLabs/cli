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

	if runtime.GOOS == "windows" {
		// Windows PATH detection misses many real installs (stale IDE
		// environments, GUI installers that skip PATH, WSL split-brain,
		// PowerShell function shims). Treat Windows as "tool present" so
		// the MCP prompt always runs; users without a tool can decline it.
		logrus.Debug("Windows: assuming AI coding tool is present")
		found = append(found, "windows-assumed")
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
	fmt.Println("  It listens on 127.0.0.1 (localhost only) while " + color.CyanString(internal.BinaryName()+" tr run") + " is active.")
	fmt.Println()
	fmt.Println("  The training works perfectly fine without it.")
	fmt.Println("  It will integrate your coding agent with this CLI.")
	fmt.Println(color.New(color.Bold).Sprint("  Writing by hand is still the default, and may help the ideas stick."))
	fmt.Println("  You can change this later with: " + color.CyanString(internal.BinaryName()+" training settings"))
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

// configureMCPIfNeeded reconciles MCP runtime state (h.mcpPort, h.loopState)
// and on-disk files (.mcp.json, CLAUDE.md, AGENTS.md) with the user's saved
// preference.
//
// Once MCP is configured — whether the user enabled or declined it — the files
// are always written on every run. When enabled, the server runs as well; when
// declined, the files remain as a discovery hint so users can learn the feature
// exists and flip it on later via `tdl training settings`. See ensureMCPFiles
// for the rationale behind the unconditional file writes.
//
// AI tool detection is used ONLY to decide whether to prompt on the first run.
// We skip the prompt entirely when no tool is installed, leaving config
// unconfigured so it re-arms next time a tool is present. Once the user has
// answered the prompt, detection is no longer consulted: their saved preference
// drives everything, so uninstalling/reinstalling the tool doesn't force them
// to re-opt-in.
func (h *Handlers) configureMCPIfNeeded(trainingRootFs *afero.BasePathFs, agentInstructions []byte) {
	globalCfg := h.config.GlobalConfig()

	if globalCfg.MCPConfigured {
		// Write files first (while h.mcpPort is still valid), then zero
		// runtime state if disabled. This is both the normal-enabled path
		// and the discovery-dump path, unified.
		h.ensureMCPFiles(trainingRootFs, agentInstructions)
		if !globalCfg.MCPEnabled {
			h.mcpPort = 0
			h.loopState = nil
		}
		return
	}

	// First run — detect AI tools to decide whether to prompt.
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
		// Non-interactive — can't prompt, skip for now. Leave config
		// unconfigured so the prompt re-arms on the next interactive run.
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

	h.ensureMCPFiles(trainingRootFs, agentInstructions)

	if !enabled {
		h.mcpPort = 0
		h.loopState = nil
		return
	}

	if h.loopState == nil {
		h.loopState = mcppkg.NewLoopState()
		h.loopState.SetCLIVersion(h.cliMetadata.Version)
	}

	fmt.Println(color.GreenString("  MCP enabled.") + " Created " + color.CyanString(".mcp.json") + " in your training directory.")
	fmt.Println("  Restart your AI coding tools to pick up the new MCP server config.")
	fmt.Println()
	fmt.Println("  We did our best to avoid hallucinations, but AI tools can still make")
	fmt.Println("  things up. If the agent contradicts the training, trust the training.")
	fmt.Println()
}
