package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Version represents a semantic version with major, minor, and patch components.
type Version struct {
	Major, Minor, Patch int
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// AtLeast returns true if v is greater than or equal to min.
func (v Version) AtLeast(min Version) bool {
	if v.Major != min.Major {
		return v.Major > min.Major
	}
	if v.Minor != min.Minor {
		return v.Minor > min.Minor
	}
	return v.Patch >= min.Patch
}

// MinVersion is the minimum git version required by the CLI.
// Driven by `git merge-tree --write-tree` (Git 2.38+).
var MinVersion = Version{2, 38, 0}

// GitNotInstalledError indicates that the git binary was not found in PATH.
type GitNotInstalledError struct{}

func (e *GitNotInstalledError) Error() string {
	return "git is not installed"
}

// GitTooOldError indicates the installed git version is below the minimum required.
type GitTooOldError struct {
	Detected Version
	Required Version
}

func (e *GitTooOldError) Error() string {
	return fmt.Sprintf("git version %s is below minimum required %s", e.Detected, e.Required)
}

// versionRegexp matches the version numbers in `git version` output.
// Handles formats like "2.39.0", "2.39.3 (Apple Git-146)", "2.9".
var versionRegexp = regexp.MustCompile(`(\d+)\.(\d+)(?:\.(\d+))?`)

// parseGitVersion extracts a Version from `git version` output.
func parseGitVersion(output string) (Version, error) {
	m := versionRegexp.FindStringSubmatch(output)
	if m == nil {
		return Version{}, fmt.Errorf("cannot parse git version from %q", output)
	}

	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	var patch int
	if m[3] != "" {
		patch, _ = strconv.Atoi(m[3])
	}

	return Version{major, minor, patch}, nil
}

// InstallHint returns OS-specific instructions for installing or upgrading git.
func InstallHint(goos string) string {
	switch goos {
	case "darwin":
		return strings.Join([]string{
			"Install or upgrade git:",
			"  brew install git",
			"",
			"Or update Xcode Command Line Tools:",
			"  xcode-select --install",
		}, "\n")
	case "linux":
		return strings.Join([]string{
			"Install or upgrade git using your package manager:",
			"  Ubuntu/Debian:  sudo apt-get install git",
			"  Fedora/RHEL:    sudo dnf install git",
			"  Arch:           sudo pacman -S git",
		}, "\n")
	case "windows":
		return strings.Join([]string{
			"Install or upgrade git:",
			"  winget install Git.Git",
			"",
			"Or download from https://git-scm.com/downloads",
		}, "\n")
	default:
		return "Install or upgrade git from https://git-scm.com/downloads"
	}
}

var (
	checkOnce   sync.Once
	checkResult error
)

// CheckVersion verifies that git is installed and meets the minimum version requirement.
// Results are cached — the check runs at most once per process.
func CheckVersion() error {
	checkOnce.Do(func() {
		_, err := exec.LookPath("git")
		if err != nil {
			checkResult = &GitNotInstalledError{}
			return
		}

		out, err := exec.Command("git", "version").CombinedOutput()
		if err != nil {
			checkResult = fmt.Errorf("could not run git version: %w", err)
			return
		}

		v, err := parseGitVersion(string(out))
		if err != nil {
			checkResult = err
			return
		}

		if !v.AtLeast(MinVersion) {
			checkResult = &GitTooOldError{Detected: v, Required: MinVersion}
			return
		}
	})

	return checkResult
}

// ResetCheckVersion resets the cached version check (for testing only).
func ResetCheckVersion() {
	checkOnce = sync.Once{}
	checkResult = nil
}
