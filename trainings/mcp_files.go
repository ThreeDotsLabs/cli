package trainings

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/files"
)

const mcpJsonFile = ".mcp.json"
const claudeMdFile = "CLAUDE.md"
const agentsMdFile = "AGENTS.md"

// ensureMCPFiles creates or updates .mcp.json, CLAUDE.md, AGENTS.md, and .gitignore in the training root.
// port is the MCP server port to write into .mcp.json (passed explicitly because h.mcpPort
// may already be zeroed when MCP is disabled, but we still want .mcp.json to have the right port).
// agentInstructions is the server-provided content written to both CLAUDE.md and AGENTS.md.
func (h *Handlers) ensureMCPFiles(trainingRootFs *afero.BasePathFs, port int, agentInstructions []byte) {
	cfg := h.config.TrainingConfig(trainingRootFs)

	ensureMCPJson(trainingRootFs, port)

	changed := ensureManagedFile(trainingRootFs, claudeMdFile, agentInstructions, cfg.FileHashes)
	changed = ensureManagedFile(trainingRootFs, agentsMdFile, agentInstructions, cfg.FileHashes) || changed
	changed = ensureManagedFile(trainingRootFs, ".gitignore", []byte(gitignore), cfg.FileHashes) || changed

	if changed {
		if err := h.config.WriteTrainingConfig(cfg, trainingRootFs); err != nil {
			logrus.WithError(errors.WithStack(err)).Warn("Could not save file hashes")
		}
	}
}

// ensureMCPJson upserts the tdl-training entry in .mcp.json, preserving other entries.
func ensureMCPJson(fs afero.Fs, port int) {
	var root map[string]any

	data, err := afero.ReadFile(fs, mcpJsonFile)
	if err == nil {
		if jsonErr := json.Unmarshal(data, &root); jsonErr != nil {
			logrus.WithError(jsonErr).Warn("Could not parse .mcp.json, recreating")
			root = nil
		}
	}

	if root == nil {
		root = make(map[string]any)
	}

	servers, _ := root["mcpServers"].(map[string]any)
	if servers == nil {
		servers = make(map[string]any)
	}

	servers["tdl-training"] = map[string]any{
		"type": "http",
		"url":  fmt.Sprintf("http://127.0.0.1:%d/mcp", port),
	}
	root["mcpServers"] = servers

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		logrus.WithError(err).Warn("Could not marshal .mcp.json")
		return
	}
	out = append(out, '\n')

	if err := afero.WriteFile(fs, mcpJsonFile, out, 0644); err != nil {
		logrus.WithError(err).Warn("Could not write .mcp.json")
	}
}

// ensureManagedFile creates or updates a managed file using hash-based conflict detection.
// Returns true if hashes was modified (caller should save config).
//
// The hash in hashes[filename] tracks the template version we last wrote or offered — NOT the file on disk.
// This ensures: user declines v2 → we stop asking about v2 → v3 ships → we ask again.
func ensureManagedFile(fs afero.Fs, filename string, newContent []byte, hashes map[string]string) bool {
	newHash := hashContent(newContent)
	storedHash := hashes[filename]

	// Template unchanged since last write/offer — nothing to do
	if newHash == storedHash {
		return false
	}

	diskContent, err := afero.ReadFile(fs, filename)

	// File missing
	if err != nil {
		if err := afero.WriteFile(fs, filename, newContent, 0644); err != nil {
			logrus.WithError(err).Warnf("Could not write %s", filename)
			return false
		}
		hashes[filename] = newHash
		return true
	}

	diskHash := hashContent(diskContent)

	// No stored hash but file exists — first-time tracking
	// If disk already matches template, just adopt. Otherwise fall through to
	// conflict handling so the user sees the diff and can choose.
	if storedHash == "" && diskHash == newHash {
		hashes[filename] = newHash
		return true
	}

	// File on disk matches what we last wrote — user didn't edit, safe to overwrite
	if diskHash == storedHash {
		if err := afero.WriteFile(fs, filename, newContent, 0644); err != nil {
			logrus.WithError(err).Warnf("Could not write %s", filename)
			return false
		}
		hashes[filename] = newHash
		return true
	}

	// Conflict: user edited AND template changed
	if !internal.IsStdinTerminal() {
		logrus.Infof("%s has updates but can't prompt in non-interactive mode, skipping", filename)
		hashes[filename] = newHash
		return true
	}

	edits := myers.ComputeEdits(span.URIFromPath(filename), string(diskContent), string(newContent))
	diff := fmt.Sprint(gotextdiff.ToUnified("current "+filename, "updated "+filename, string(diskContent), edits))

	fmt.Println()
	fmt.Println(color.New(color.Bold).Sprintf("%s has been updated by the training:", filename))
	fmt.Println(files.ColorDiff(diff))

	choice := internal.Prompt(
		internal.Actions{
			{Shortcut: '\n', Action: "replace with new version", ShortcutAliases: []rune{'\r'}},
			{Shortcut: 'n', Action: "keep your version"},
		},
		os.Stdin,
		os.Stdout,
	)
	fmt.Println()

	if choice == '\n' {
		if err := afero.WriteFile(fs, filename, newContent, 0644); err != nil {
			logrus.WithError(err).Warnf("Could not write %s", filename)
			return false
		}
	}

	// Update hash regardless of choice — marks this template version as processed
	hashes[filename] = newHash
	return true
}

func hashContent(content []byte) string {
	h := sha256.Sum256(content)
	return fmt.Sprintf("%x", h)
}
