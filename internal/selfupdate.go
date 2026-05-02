package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

const (
	repoOwner = "ThreeDotsLabs"
	repoName  = "cli"
)

// InstallMethod represents how the tdl binary was installed.
type InstallMethod int

const (
	InstallMethodUnknown InstallMethod = iota
	InstallMethodHomebrew
	InstallMethodGoInstall
	InstallMethodScoop
	InstallMethodNix
	InstallMethodDirectBinary
)

func (m InstallMethod) String() string {
	switch m {
	case InstallMethodHomebrew:
		return "Homebrew"
	case InstallMethodGoInstall:
		return "go install"
	case InstallMethodScoop:
		return "Scoop"
	case InstallMethodNix:
		return "Nix"
	case InstallMethodDirectBinary:
		return "direct binary"
	default:
		return "unknown"
	}
}

// UpdateOptions configures the update behavior.
type UpdateOptions struct {
	SkipConfirm   bool
	TargetVersion string // e.g., "v1.2.3", "master", or "" for latest
	ForceUpdate   bool   // skip "already on latest" check
}

// DetectInstallMethod determines the installation method by examining the binary path.
func DetectInstallMethod() InstallMethod {
	exePath, err := os.Executable()
	if err != nil {
		logrus.WithError(err).Debug("Cannot determine executable path")
		return InstallMethodUnknown
	}

	resolved, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		logrus.WithError(err).Debug("Cannot resolve symlinks for executable path")
		resolved = exePath
	}

	home, _ := os.UserHomeDir()

	method := detectInstallMethodFromPath(resolved, os.Getenv("GOPATH"), os.Getenv("GOBIN"), home, runtime.GOOS)
	logrus.WithFields(logrus.Fields{
		"binary_path": resolved,
		"method":      method.String(),
	}).Debug("Detected install method")
	return method
}

// detectInstallMethodFromPath is the testable core of DetectInstallMethod.
func detectInstallMethodFromPath(resolvedPath, gopath, gobin, home, goos string) InstallMethod {
	// Normalize all backslashes to forward slashes for consistent matching across platforms.
	// filepath.ToSlash only converts the OS-native separator, so on macOS it won't
	// convert Windows backslashes in test paths. We do a manual replace for robustness.
	normalizedPath := strings.ReplaceAll(resolvedPath, "\\", "/")
	lowerPath := strings.ToLower(normalizedPath)

	// Homebrew: check for /Cellar/ or /homebrew/ in the resolved path
	if strings.Contains(lowerPath, "/cellar/") || strings.Contains(lowerPath, "/homebrew/") {
		return InstallMethodHomebrew
	}
	// Linux Homebrew
	if strings.Contains(lowerPath, "/linuxbrew/") {
		return InstallMethodHomebrew
	}

	// Nix: check for /nix/store/ in the path
	if strings.Contains(lowerPath, "/nix/store/") {
		return InstallMethodNix
	}

	// Scoop (Windows): check for scoop/apps/ or scoop/shims/ in the path
	if goos == "windows" {
		if strings.Contains(lowerPath, "scoop/apps/") || strings.Contains(lowerPath, "scoop/shims/") {
			return InstallMethodScoop
		}
	}

	// Go install: check if binary is in GOBIN, GOPATH/bin, or $HOME/go/bin
	if gobin != "" {
		gobinNorm := strings.ReplaceAll(gobin, "\\", "/")
		if strings.HasPrefix(normalizedPath, gobinNorm+"/") {
			return InstallMethodGoInstall
		}
	}
	if gopath != "" {
		gopathBin := strings.ReplaceAll(gopath, "\\", "/") + "/bin"
		if strings.HasPrefix(normalizedPath, gopathBin+"/") {
			return InstallMethodGoInstall
		}
	}
	if home != "" {
		defaultGoBin := strings.ReplaceAll(home, "\\", "/") + "/go/bin"
		if strings.HasPrefix(normalizedPath, defaultGoBin+"/") {
			return InstallMethodGoInstall
		}
	}

	return InstallMethodDirectBinary
}

// canWriteBinary reports whether the current process can replace the binary
// at path. The selfupdate library replaces the binary via tempfile + rename
// in the same directory (see creativeprojects/go-selfupdate update/apply.go),
// so write access to the parent directory is the necessary and sufficient
// condition.
//
// We deliberately do NOT probe the binary file itself: on Linux the kernel
// returns ETXTBSY for any write-open of a currently-running executable, which
// would produce a false negative for every `tdl update` invocation.
func canWriteBinary(path string) bool {
	dir := filepath.Dir(path)
	if err := dirWritable(dir); err != nil {
		logrus.WithFields(logrus.Fields{"dir": dir, "error": err}).Debug("Update directory not writable")
		return false
	}
	logrus.WithField("dir", dir).Debug("Update directory writable")
	return true
}

func newUpdater() (*selfupdate.Updater, error) {
	return selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
}

func repoSlug() selfupdate.RepositorySlug {
	return selfupdate.NewRepositorySlug(repoOwner, repoName)
}

func parseBrewFormulaVersion(data []byte) (string, error) {
	var result struct {
		Formulae []struct {
			Versions struct {
				Stable string `json:"stable"`
			} `json:"versions"`
		} `json:"formulae"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	if len(result.Formulae) == 0 || result.Formulae[0].Versions.Stable == "" {
		return "", fmt.Errorf("no version found in brew info output")
	}
	return result.Formulae[0].Versions.Stable, nil
}

// brewFormulaVersion returns the version from the local Homebrew tap formula.
//
// Homebrew caches third-party tap formulae as local git clones and only
// refreshes them when auto-update fires. Auto-update is triggered by
// commands like `brew install` or `brew upgrade`, but has a 24h debounce
// (HOMEBREW_AUTO_UPDATE_SECS, default 86400s) — if any brew command ran
// within that window, the refresh is skipped. There is no background
// process; taps stay stale until the next brew command after the debounce
// expires, or until the user runs `brew update` manually.
//
// Unlike `go install @vX.Y.Z`, there is no way to tell brew to install a
// specific version of a tap formula. The `@version` syntax (e.g.
// `python@3.11`) only works when a separate versioned formula file exists
// in the tap, which we don't publish.
//
// This function lets us detect when the local formula is behind the latest
// GitHub release, so we can suppress misleading update prompts rather than
// pointing the user at an upgrade that would silently no-op.
func brewFormulaVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "brew", "info", "--json=v2", "threedotslabs/tap/tdl")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("brew info failed: %w", err)
	}
	return parseBrewFormulaVersion(out)
}

// isVersionAvailableForInstallMethod checks whether the given version
// is available through the user's install method. Currently only
// Homebrew is checked — see brewFormulaVersion for why.
// Returns true for all other methods or on any error (graceful fallback).
func isVersionAvailableForInstallMethod(version string) bool {
	method := DetectInstallMethod()
	if method != InstallMethodHomebrew {
		return true
	}
	formulaVer, err := brewFormulaVersion(context.Background())
	if err != nil {
		logrus.WithError(err).Debug("Could not check brew formula version")
		return true
	}
	target := strings.TrimPrefix(version, "v")
	return !isNewerVersion(target, formulaVer)
}

// RunUpdate checks for updates and applies them based on the installation method.
func RunUpdate(ctx context.Context, currentVersion string, opts UpdateOptions) error {
	if os.Getenv("TDL_NO_UPDATE_CHECK") != "" {
		logrus.Debug("Update disabled via TDL_NO_UPDATE_CHECK")
		fmt.Println("Update checks are disabled (TDL_NO_UPDATE_CHECK is set).")
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"current_version": currentVersion,
		"target_version":  opts.TargetVersion,
		"skip_confirm":    opts.SkipConfirm,
	}).Debug("Starting update")

	if currentVersion == "" {
		fmt.Println("You are running a development build. Update is only available for released versions.")
		fmt.Printf("Run %s to install from source.\n", SprintCommand("go install github.com/ThreeDotsLabs/cli/tdl@latest"))
		return nil
	}

	updater, err := newUpdater()
	if err != nil {
		return fmt.Errorf("failed to initialize updater: %w", err)
	}

	method := DetectInstallMethod()

	// Detect the target release
	var release *selfupdate.Release
	var found bool

	if opts.TargetVersion != "" {
		target := opts.TargetVersion
		if !strings.HasPrefix(target, "v") {
			target = "v" + target
		}
		release, found, err = updater.DetectVersion(ctx, repoSlug(), target)
		if err != nil {
			return fmt.Errorf("failed to check for version %s: %w", opts.TargetVersion, err)
		}
		if !found {
			logrus.WithField("target", opts.TargetVersion).Debug("Version not found as release tag, trying as branch")
			return handleBranchInstall(ctx, opts.TargetVersion, method, opts.SkipConfirm)
		}
	} else {
		release, found, err = updater.DetectLatest(ctx, repoSlug())
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}
		if !found {
			logrus.Debug("No releases found on GitHub")
			fmt.Println("No releases found.")
			return nil
		}
		logrus.WithField("latest", release.Version()).Debug("Latest release detected")
		if !opts.ForceUpdate && release.LessOrEqual(currentVersion) {
			fmt.Printf("You are already running the latest version (%s).\n", currentVersion)
			return nil
		}
	}

	targetVersion := release.Version()

	// Show update info with release notes BEFORE confirmation (skip if caller already showed it)
	if !opts.SkipConfirm {
		fmt.Printf("\nUpdate available: %s → %s\n", currentVersion, targetVersion)

		if notes := release.ReleaseNotes; notes != "" {
			formatted := FormatReleaseNotes(notes, 15)
			if formatted != "" {
				fmt.Println()
				fmt.Println("Release notes:")
				fmt.Println(formatted)
			}
		}
		fmt.Println()
	}

	logrus.WithField("method", method.String()).Debug("Updating via install method")

	// Branch on install method
	switch method {
	case InstallMethodHomebrew:
		// Homebrew's local tap may still point to an older formula version
		// (24h auto-update debounce). Bail out early instead of running a
		// doomed `brew upgrade` that would print "already installed".
		formulaVer, err := brewFormulaVersion(ctx)
		if err == nil {
			target := strings.TrimPrefix(targetVersion, "v")
			if isNewerVersion(target, formulaVer) {
				fmt.Println(color.YellowString(
					"Version %s is not yet available via Homebrew (formula has %s).",
					targetVersion, formulaVer,
				))
				fmt.Println("Homebrew updates its formula cache periodically. To refresh manually, run:")
				fmt.Printf("  %s\n\n", SprintCommand("brew update && brew upgrade tdl"))
				return nil
			}
		}
		return updateViaCommand(ctx, "brew", opts, currentVersion, targetVersion,
			[]string{"upgrade", "tdl"})

	case InstallMethodGoInstall:
		// Pin to the explicit version we just resolved via the GitHub API
		// rather than letting `go install` resolve "@latest" itself. The Go
		// module proxy caches "@latest" and can serve a stale version right
		// after a release, which would silently re-install the current
		// version and cause the update prompt to loop.
		ref := "v" + strings.TrimPrefix(targetVersion, "v")
		return updateViaCommand(ctx, "go", opts, currentVersion, targetVersion,
			[]string{"install", "github.com/ThreeDotsLabs/cli/tdl@" + ref})

	case InstallMethodNix:
		return updateViaCommand(ctx, "nix", opts, currentVersion, targetVersion,
			[]string{"profile", "upgrade", "--flake", "github:ThreeDotsLabs/cli"})

	case InstallMethodScoop:
		return updateViaCommand(ctx, "scoop", opts, currentVersion, targetVersion,
			[]string{"update", "tdl"})

	case InstallMethodDirectBinary, InstallMethodUnknown:
		return updateDirectBinary(ctx, updater, currentVersion, targetVersion, release, opts)
	}

	return nil
}

func updateViaCommand(ctx context.Context, tool string, opts UpdateOptions, currentVer, targetVer string, args []string) error {
	fullCmd := tool + " " + strings.Join(args, " ")

	toolPath, err := exec.LookPath(tool)
	if err != nil {
		logrus.WithFields(logrus.Fields{"tool": tool, "error": err}).Debug("Tool not found in PATH")
		fmt.Printf("Could not find %s in PATH. Please run manually:\n", tool)
		fmt.Printf("  %s\n", color.CyanString(fullCmd))
		return nil
	}
	logrus.WithFields(logrus.Fields{"tool": tool, "path": toolPath}).Debug("Found tool in PATH")
	fmt.Println(color.CyanString("••• ") + fullCmd)

	if !opts.SkipConfirm && IsStdinTerminal() {
		if !ConfirmPromptDefaultYes("update") {
			return nil
		}
	}

	cmd := exec.CommandContext(ctx, tool, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", fullCmd, err)
	}

	fmt.Println(color.GreenString("\nSuccessfully updated to %s.", targetVer))
	return nil
}

func updateDirectBinary(ctx context.Context, updater *selfupdate.Updater, currentVer, targetVer string, release *selfupdate.Release, opts UpdateOptions) error {
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine binary path: %w", err)
	}
	binaryPath, err = filepath.EvalSymlinks(binaryPath)
	if err != nil {
		return fmt.Errorf("cannot resolve binary path: %w", err)
	}
	logrus.WithField("binary_path", binaryPath).Debug("Resolved binary path for direct update")

	// Check writability BEFORE any confirmation
	if !canWriteBinary(binaryPath) {
		fmt.Printf("The binary at %s requires elevated permissions to update.\n\n", binaryPath)

		cmdName := os.Args[0]
		if runtime.GOOS == "windows" {
			fmt.Println("Please re-open your terminal as Administrator and run:")
			fmt.Println("  " + color.CyanString("%s update", cmdName))
		} else {
			fmt.Println("Please run:")
			fmt.Println("  " + color.CyanString("sudo %s update", cmdName))
		}

		fmt.Printf("\nOr download from: %s/releases/latest\n", repoURL)

		return nil
	}

	fmt.Printf("Updating tdl: %s → %s (%s)\n", currentVer, targetVer, binaryPath)

	if !opts.SkipConfirm && IsStdinTerminal() {
		if !ConfirmPromptDefaultYes("update") {
			return nil
		}
	}

	if err := updater.UpdateTo(ctx, release, binaryPath); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Println(color.GreenString("\nSuccessfully updated to %s.", targetVer))
	return nil
}

// handleBranchInstall handles the case where --version is a branch name (not a release tag).
func handleBranchInstall(ctx context.Context, branch string, method InstallMethod, skipConfirm bool) error {
	logrus.WithFields(logrus.Fields{"branch": branch, "method": method.String()}).Debug("Handling branch install")

	switch method {
	case InstallMethodGoInstall:
		return updateViaCommand(ctx, "go", UpdateOptions{SkipConfirm: skipConfirm}, "", branch,
			[]string{"install", "github.com/ThreeDotsLabs/cli/tdl@" + branch})

	case InstallMethodNix:
		return updateViaCommand(ctx, "nix", UpdateOptions{SkipConfirm: skipConfirm}, "", branch,
			[]string{"profile", "install", "github:ThreeDotsLabs/cli/" + branch})

	default:
		fmt.Printf("'%s' is not a release tag. Only tagged releases are available for %s installs.\n", branch, method)
		fmt.Printf("To install from a branch, use:\n")
		fmt.Printf("  %s\n", SprintCommand("go install github.com/ThreeDotsLabs/cli/tdl@"+branch))
		return nil
	}
}

// FormatReleaseNotes prepares release notes for terminal display.
func FormatReleaseNotes(body string, maxLines int) string {
	if strings.TrimSpace(body) == "" {
		return ""
	}

	// Light markdown stripping
	lines := strings.Split(body, "\n")
	var cleaned []string
	for _, line := range lines {
		// Strip markdown headers
		line = stripMarkdownHeader(line)
		// Strip bold markers
		line = strings.ReplaceAll(line, "**", "")
		// Strip links: [text](url) → text
		line = stripMarkdownLinks(line)
		cleaned = append(cleaned, line)
	}

	// Trim leading/trailing blank lines
	cleaned = trimBlankLines(cleaned)

	if len(cleaned) == 0 {
		return ""
	}

	truncated := false
	if len(cleaned) > maxLines {
		cleaned = cleaned[:maxLines]
		truncated = true
	}

	var result strings.Builder
	for _, line := range cleaned {
		result.WriteString(color.HiBlackString("  " + line))
		result.WriteString("\n")
	}
	if truncated {
		result.WriteString(color.HiBlackString(fmt.Sprintf("  ... see full release notes at %s/releases", repoURL)))
		result.WriteString("\n")
	}

	return strings.TrimRight(result.String(), "\n")
}

func stripMarkdownHeader(line string) string {
	trimmed := strings.TrimLeft(line, " ")
	if strings.HasPrefix(trimmed, "# ") {
		return strings.TrimPrefix(trimmed, "# ")
	}
	if strings.HasPrefix(trimmed, "## ") {
		return strings.TrimPrefix(trimmed, "## ")
	}
	if strings.HasPrefix(trimmed, "### ") {
		return strings.TrimPrefix(trimmed, "### ")
	}
	return line
}

func stripMarkdownLinks(line string) string {
	result := line
	for {
		start := strings.Index(result, "[")
		if start == -1 {
			break
		}
		mid := strings.Index(result[start:], "](")
		if mid == -1 {
			break
		}
		mid += start
		end := strings.Index(result[mid:], ")")
		if end == -1 {
			break
		}
		end += mid
		text := result[start+1 : mid]
		result = result[:start] + text + result[end+1:]
	}
	return result
}

func trimBlankLines(lines []string) []string {
	// Trim leading blank lines
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	// Trim trailing blank lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
