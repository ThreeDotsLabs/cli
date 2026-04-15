package trainings

import (
	"encoding/json"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureMCPJson_CreatesFileWhenMissing(t *testing.T) {
	fs := afero.NewMemMapFs()

	ensureMCPJson(fs, 39131)

	data, err := afero.ReadFile(fs, ".mcp.json")
	require.NoError(t, err)

	var root map[string]any
	require.NoError(t, json.Unmarshal(data, &root))

	servers := root["mcpServers"].(map[string]any)
	entry := servers["tdl-training"].(map[string]any)
	assert.Equal(t, "http://127.0.0.1:39131/mcp", entry["url"])
}

func TestEnsureMCPJson_PreservesOtherServers(t *testing.T) {
	fs := afero.NewMemMapFs()

	existing := `{
  "mcpServers": {
    "my-custom-server": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
`
	require.NoError(t, afero.WriteFile(fs, ".mcp.json", []byte(existing), 0644))

	ensureMCPJson(fs, 39131)

	data, err := afero.ReadFile(fs, ".mcp.json")
	require.NoError(t, err)

	var root map[string]any
	require.NoError(t, json.Unmarshal(data, &root))

	servers := root["mcpServers"].(map[string]any)

	// Our entry added
	tdl := servers["tdl-training"].(map[string]any)
	assert.Equal(t, "http://127.0.0.1:39131/mcp", tdl["url"])

	// User's entry preserved
	custom := servers["my-custom-server"].(map[string]any)
	assert.Equal(t, "http://localhost:8080/mcp", custom["url"])
}

func TestEnsureMCPJson_UpdatesPort(t *testing.T) {
	fs := afero.NewMemMapFs()

	ensureMCPJson(fs, 39131)
	ensureMCPJson(fs, 12345)

	data, err := afero.ReadFile(fs, ".mcp.json")
	require.NoError(t, err)

	var root map[string]any
	require.NoError(t, json.Unmarshal(data, &root))

	servers := root["mcpServers"].(map[string]any)
	entry := servers["tdl-training"].(map[string]any)
	assert.Equal(t, "http://127.0.0.1:12345/mcp", entry["url"])
}

// testAgentInstructions is a stand-in for server-provided agent instructions in tests.
var testAgentInstructions = []byte("# Training Companion\n\nServer-provided agent instructions.\n")

func TestEnsureManagedFile_CreatesFileWhenMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	hashes := map[string]string{}

	changed := ensureManagedFile(fs, "CLAUDE.md", testAgentInstructions, hashes)

	assert.True(t, changed)
	assert.NotEmpty(t, hashes["CLAUDE.md"])

	content, err := afero.ReadFile(fs, "CLAUDE.md")
	require.NoError(t, err)
	assert.Equal(t, string(testAgentInstructions), string(content))
}

func TestEnsureManagedFile_SkipsWhenTemplateUnchanged(t *testing.T) {
	fs := afero.NewMemMapFs()
	hashes := map[string]string{}

	// First write
	ensureManagedFile(fs, "CLAUDE.md", testAgentInstructions, hashes)

	// Modify file on disk (user edit)
	require.NoError(t, afero.WriteFile(fs, "CLAUDE.md", []byte("user content"), 0644))

	// Run again — template unchanged, should skip
	changed := ensureManagedFile(fs, "CLAUDE.md", testAgentInstructions, hashes)

	assert.False(t, changed)

	// User's content preserved
	content, err := afero.ReadFile(fs, "CLAUDE.md")
	require.NoError(t, err)
	assert.Equal(t, "user content", string(content))
}

func TestEnsureManagedFile_OverwritesUnmodifiedFile(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Simulate: old template was written
	oldContent := []byte("old template v1\n")
	oldHash := hashContent(oldContent)
	require.NoError(t, afero.WriteFile(fs, "CLAUDE.md", oldContent, 0644))
	hashes := map[string]string{"CLAUDE.md": oldHash}

	// Now new template (generateClaudeMd returns different content)
	changed := ensureManagedFile(fs, "CLAUDE.md", testAgentInstructions, hashes)

	assert.True(t, changed)

	content, err := afero.ReadFile(fs, "CLAUDE.md")
	require.NoError(t, err)
	assert.Equal(t, string(testAgentInstructions), string(content))
	assert.Equal(t, hashContent(testAgentInstructions), hashes["CLAUDE.md"])
}

func TestEnsureManagedFile_AdoptsExistingFileFirstTime(t *testing.T) {
	fs := afero.NewMemMapFs()

	// File exists but no stored hash (first-time tracking)
	require.NoError(t, afero.WriteFile(fs, "CLAUDE.md", []byte("user created this"), 0644))
	hashes := map[string]string{}

	changed := ensureManagedFile(fs, "CLAUDE.md", testAgentInstructions, hashes)

	assert.True(t, changed) // hash was updated
	assert.Equal(t, hashContent(testAgentInstructions), hashes["CLAUDE.md"])

	// File NOT overwritten — user's content preserved
	content, err := afero.ReadFile(fs, "CLAUDE.md")
	require.NoError(t, err)
	assert.Equal(t, "user created this", string(content))
}

func TestEnsureManagedFile_DeclineStopsNagging(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Simulate: template v1 was written, user edited, template v2 arrived
	require.NoError(t, afero.WriteFile(fs, "CLAUDE.md", []byte("user edited content"), 0644))
	hashes := map[string]string{"CLAUDE.md": "old-template-hash"}

	// This would normally prompt — but in tests stdin is not a terminal,
	// so it falls through to non-interactive path (skips, sets hash)
	changed := ensureManagedFile(fs, "CLAUDE.md", testAgentInstructions, hashes)

	assert.True(t, changed)
	// hash updated to new template hash — won't ask again for this version
	assert.Equal(t, hashContent(testAgentInstructions), hashes["CLAUDE.md"])

	// User's content preserved (non-interactive = skip)
	content, err := afero.ReadFile(fs, "CLAUDE.md")
	require.NoError(t, err)
	assert.Equal(t, "user edited content", string(content))

	// Run again — hash matches new template, skips entirely
	changed = ensureManagedFile(fs, "CLAUDE.md", testAgentInstructions, hashes)
	assert.False(t, changed)
}

func TestEnsureManagedFile_IndependentHashes(t *testing.T) {
	fs := afero.NewMemMapFs()
	hashes := map[string]string{}
	content := testAgentInstructions

	// Create both files
	ensureManagedFile(fs, "CLAUDE.md", content, hashes)
	ensureManagedFile(fs, "AGENTS.md", content, hashes)

	assert.Equal(t, hashes["CLAUDE.md"], hashes["AGENTS.md"])

	// Edit only CLAUDE.md on disk
	require.NoError(t, afero.WriteFile(fs, "CLAUDE.md", []byte("user edit"), 0644))

	// Both use same template content, so neither should trigger (template unchanged)
	assert.False(t, ensureManagedFile(fs, "CLAUDE.md", content, hashes))
	assert.False(t, ensureManagedFile(fs, "AGENTS.md", content, hashes))

	// AGENTS.md untouched
	agentsContent, err := afero.ReadFile(fs, "AGENTS.md")
	require.NoError(t, err)
	assert.Equal(t, string(content), string(agentsContent))
}

func TestEnsureManagedFile_WorksForGitignore(t *testing.T) {
	fs := afero.NewMemMapFs()
	hashes := map[string]string{}

	changed := ensureManagedFile(fs, ".gitignore", []byte(gitignore), hashes)

	assert.True(t, changed)
	assert.NotEmpty(t, hashes[".gitignore"])

	content, err := afero.ReadFile(fs, ".gitignore")
	require.NoError(t, err)
	assert.Equal(t, gitignore, string(content))
	assert.Contains(t, string(content), "AGENTS.md")
}

func TestEnsureManagedFile_UpdatesOldFileWithoutHash(t *testing.T) {
	fs := afero.NewMemMapFs()
	hashes := map[string]string{}

	// Simulate: old CLI wrote .gitignore without hash tracking
	oldGitignore := "# old content\nCLAUDE.md\n"
	require.NoError(t, afero.WriteFile(fs, ".gitignore", []byte(oldGitignore), 0644))

	// New template has AGENTS.md — no stored hash, file differs from template
	// Non-interactive: stores hash, preserves file (user will be prompted in interactive mode)
	changed := ensureManagedFile(fs, ".gitignore", []byte(gitignore), hashes)
	assert.True(t, changed)
	assert.Equal(t, hashContent([]byte(gitignore)), hashes[".gitignore"])

	// On next run with same template, stored hash matches — nothing to do
	changed = ensureManagedFile(fs, ".gitignore", []byte(gitignore), hashes)
	assert.False(t, changed)
}

func TestHashContent(t *testing.T) {
	h1 := hashContent([]byte("hello"))
	h2 := hashContent([]byte("hello"))
	h3 := hashContent([]byte("world"))

	assert.Equal(t, h1, h2)
	assert.NotEqual(t, h1, h3)
	assert.Len(t, h1, 64) // sha256 hex
}
