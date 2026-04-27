package internal

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectInstallMethodFromPath(t *testing.T) {
	tests := []struct {
		name         string
		resolvedPath string
		gopath       string
		gobin        string
		home         string
		goos         string
		expected     InstallMethod
	}{
		{
			name:         "Homebrew Apple Silicon",
			resolvedPath: "/opt/homebrew/Cellar/tdl/1.0.0/bin/tdl",
			home:         "/Users/bob",
			goos:         "darwin",
			expected:     InstallMethodHomebrew,
		},
		{
			name:         "Homebrew Intel macOS",
			resolvedPath: "/usr/local/Cellar/tdl/1.0.0/bin/tdl",
			home:         "/Users/bob",
			goos:         "darwin",
			expected:     InstallMethodHomebrew,
		},
		{
			name:         "Homebrew Linux",
			resolvedPath: "/home/linuxbrew/.linuxbrew/Cellar/tdl/1.0.0/bin/tdl",
			home:         "/home/bob",
			goos:         "linux",
			expected:     InstallMethodHomebrew,
		},
		{
			name:         "Go install default GOPATH",
			resolvedPath: "/Users/bob/go/bin/tdl",
			home:         "/Users/bob",
			goos:         "darwin",
			expected:     InstallMethodGoInstall,
		},
		{
			name:         "Go install custom GOPATH",
			resolvedPath: "/home/bob/go/bin/tdl",
			gopath:       "/home/bob/go",
			home:         "/home/bob",
			goos:         "linux",
			expected:     InstallMethodGoInstall,
		},
		{
			name:         "Go install custom GOBIN",
			resolvedPath: "/custom/gobin/tdl",
			gobin:        "/custom/gobin",
			home:         "/home/bob",
			goos:         "linux",
			expected:     InstallMethodGoInstall,
		},
		{
			name:         "Scoop apps Windows",
			resolvedPath: `C:\Users\bob\scoop\apps\tdl\current\tdl.exe`,
			home:         `C:\Users\bob`,
			goos:         "windows",
			expected:     InstallMethodScoop,
		},
		{
			name:         "Scoop shims Windows",
			resolvedPath: `C:\Users\bob\scoop\shims\tdl.exe`,
			home:         `C:\Users\bob`,
			goos:         "windows",
			expected:     InstallMethodScoop,
		},
		{
			name:         "Nix store",
			resolvedPath: "/nix/store/abc123-tdl-1.2.3/bin/tdl",
			home:         "/home/bob",
			goos:         "linux",
			expected:     InstallMethodNix,
		},
		{
			name:         "Direct binary /usr/local/bin",
			resolvedPath: "/usr/local/bin/tdl",
			home:         "/Users/bob",
			goos:         "darwin",
			expected:     InstallMethodDirectBinary,
		},
		{
			name:         "Direct binary Windows home",
			resolvedPath: `C:\Users\bob\ThreeDotsLabs\bin\tdl.exe`,
			home:         `C:\Users\bob`,
			goos:         "windows",
			expected:     InstallMethodDirectBinary,
		},
		{
			name:         "Direct binary random path",
			resolvedPath: "/some/random/path/tdl",
			home:         "/home/bob",
			goos:         "linux",
			expected:     InstallMethodDirectBinary,
		},
		{
			name:         "Scoop path on non-Windows is not Scoop",
			resolvedPath: "/home/bob/scoop/apps/tdl/tdl",
			home:         "/home/bob",
			goos:         "linux",
			expected:     InstallMethodDirectBinary,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectInstallMethodFromPath(tt.resolvedPath, tt.gopath, tt.gobin, tt.home, tt.goos)
			assert.Equal(t, tt.expected, result, "expected %s, got %s", tt.expected, result)
		})
	}
}

func TestInstallMethodString(t *testing.T) {
	assert.Equal(t, "Homebrew", InstallMethodHomebrew.String())
	assert.Equal(t, "go install", InstallMethodGoInstall.String())
	assert.Equal(t, "Scoop", InstallMethodScoop.String())
	assert.Equal(t, "Nix", InstallMethodNix.String())
	assert.Equal(t, "direct binary", InstallMethodDirectBinary.String())
	assert.Equal(t, "unknown", InstallMethodUnknown.String())
}

func TestCanWriteBinary(t *testing.T) {
	t.Run("writable directory", func(t *testing.T) {
		dir := t.TempDir()
		assert.True(t, canWriteBinary(filepath.Join(dir, "tdl")))
	})

	t.Run("read-only directory", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("dir mode bits don't gate file creation on Windows")
		}
		if os.Geteuid() == 0 {
			t.Skip("running as root bypasses permission checks")
		}
		dir := t.TempDir()
		require.NoError(t, os.Chmod(dir, 0o555))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
		assert.False(t, canWriteBinary(filepath.Join(dir, "tdl")))
	})

	t.Run("non-existent directory", func(t *testing.T) {
		assert.False(t, canWriteBinary(filepath.Join(t.TempDir(), "missing-dir", "tdl")))
	})
}

func TestFormatReleaseNotes(t *testing.T) {
	t.Run("empty body", func(t *testing.T) {
		assert.Equal(t, "", FormatReleaseNotes("", 15))
		assert.Equal(t, "", FormatReleaseNotes("   \n\n  ", 15))
	})

	t.Run("short body under limit", func(t *testing.T) {
		body := "- Fixed bug in exercise reset\n- Added module skipping"
		result := FormatReleaseNotes(body, 15)
		assert.Contains(t, result, "Fixed bug in exercise reset")
		assert.Contains(t, result, "Added module skipping")
		assert.NotContains(t, result, "see full release notes")
	})

	t.Run("long body truncated", func(t *testing.T) {
		var lines []string
		for i := 0; i < 20; i++ {
			lines = append(lines, "- Change number "+string(rune('A'+i)))
		}
		body := strings.Join(lines, "\n")
		result := FormatReleaseNotes(body, 5)
		assert.Contains(t, result, "Change number A")
		assert.Contains(t, result, "see full release notes")
		assert.NotContains(t, result, "Change number F")
	})

	t.Run("strips markdown headers", func(t *testing.T) {
		body := "## What's Changed\n- Bug fix"
		result := FormatReleaseNotes(body, 15)
		assert.Contains(t, result, "What's Changed")
		assert.NotContains(t, result, "##")
	})

	t.Run("strips bold markers", func(t *testing.T) {
		body := "**Important**: This is a breaking change"
		result := FormatReleaseNotes(body, 15)
		assert.Contains(t, result, "Important")
		assert.NotContains(t, result, "**")
	})

	t.Run("strips markdown links", func(t *testing.T) {
		body := "See [the docs](https://example.com) for details"
		result := FormatReleaseNotes(body, 15)
		assert.Contains(t, result, "the docs")
		assert.NotContains(t, result, "https://example.com")
		assert.NotContains(t, result, "[")
	})

	t.Run("trims leading and trailing blank lines", func(t *testing.T) {
		body := "\n\n- First line\n- Second line\n\n"
		result := FormatReleaseNotes(body, 15)
		assert.Contains(t, result, "First line")
		assert.Contains(t, result, "Second line")
	})
}
