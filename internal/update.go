package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	semver "github.com/Masterminds/semver/v3"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

type releaseResponse struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
}

const repoURL = "https://github.com/ThreeDotsLabs/cli"
const releasesURL = "https://api.github.com/repos/ThreeDotsLabs/cli/releases/latest"
const updateCheckInterval = 30 * time.Minute
const dismissalDuration = 30 * time.Minute

type latestRelease struct {
	Version      string
	ReleaseNotes string
}

func CheckForUpdate(currentVersion string, commandName string, forcePrompt bool) {
	if os.Getenv("TDL_NO_UPDATE_CHECK") != "" {
		logrus.Debug("Update check disabled via TDL_NO_UPDATE_CHECK")
		return
	}

	if currentVersion == "" {
		return
	}

	isUpdateCommand := commandName == "update" || commandName == "u"

	updateInfo, _ := getUpdateInfo()

	// Fast path: cached update available — no API call needed
	if updateInfo.UpdateAvailable && isNewerVersion(updateInfo.AvailableVersion, currentVersion) {
		showUpdatePromptOrNotice(updateInfo, currentVersion, isUpdateCommand, forcePrompt)
		return
	}

	// Fast path: check interval not elapsed — return immediately
	if !forcePrompt && time.Since(updateInfo.LastChecked) < updateCheckInterval {
		return
	}

	release := getLatestRelease()
	if release == nil {
		return
	}

	isNewer := release.Version != "" && isNewerVersion(release.Version, currentVersion)
	isDifferent := release.Version != "" && release.Version != currentVersion

	if isNewer || (forcePrompt && isDifferent) {
		// Don't advertise updates that aren't installable yet — e.g. the
		// Homebrew tap formula may still point to the old version (24h
		// auto-update debounce). The notice will appear once the tap refreshes.
		if !forcePrompt && !isVersionAvailableForInstallMethod(release.Version) {
			logrus.Debug("Version not yet available via install method, skipping notice")
			updateInfo.LastChecked = time.Now()
			_ = storeUpdateInfo(updateInfo)
			return
		}

		updateInfo.CurrentVersion = currentVersion
		updateInfo.AvailableVersion = release.Version
		updateInfo.UpdateAvailable = true
		updateInfo.ReleaseNotes = release.ReleaseNotes

		updateInfo.LastChecked = time.Now()
		_ = storeUpdateInfo(updateInfo)

		showUpdatePromptOrNotice(updateInfo, currentVersion, isUpdateCommand, forcePrompt)
	} else {
		updateInfo.CurrentVersion = currentVersion
		updateInfo.AvailableVersion = ""
		updateInfo.UpdateAvailable = false
		updateInfo.ReleaseNotes = ""
		// Clear stale dismissal since there's no pending update
		updateInfo.DismissedVersion = ""
		updateInfo.DismissedAt = time.Time{}

		updateInfo.LastChecked = time.Now()
		_ = storeUpdateInfo(updateInfo)
	}
}

// CheckUpdateAvailable performs a silent update check suitable for background
// goroutines that poll on a long timer. It never prints and never prompts.
// Returns whether a newer release is known for currentVersion, along with the
// new version string and its release notes (if any).
//
// Honors TDL_NO_UPDATE_CHECK, skips when currentVersion is empty or "dev",
// fast-paths the cached state file, and only hits the GitHub API once the
// updateCheckInterval has elapsed. Respects the on-disk dismissal window so
// a user who already declined the same version at startup isn't re-notified
// mid-session.
//
// forcePrompt mirrors the semantics of CheckForUpdate's forcePrompt: bypass
// the interval gate, accept any release that differs from currentVersion
// (not just newer), and bypass the dismissal window. Intended for the hidden
// --force-update-prompt testing flag.
func CheckUpdateAvailable(currentVersion string, forcePrompt bool) (available bool, newVersion string, releaseNotes string) {
	if os.Getenv("TDL_NO_UPDATE_CHECK") != "" {
		return false, "", ""
	}
	if currentVersion == "" {
		return false, "", ""
	}
	if !forcePrompt && currentVersion == "dev" {
		return false, "", ""
	}

	updateInfo, _ := getUpdateInfo()

	if forcePrompt || time.Since(updateInfo.LastChecked) >= updateCheckInterval {
		if release := getLatestRelease(); release != nil {
			isNewer := release.Version != "" && isNewerVersion(release.Version, currentVersion)
			isDifferent := release.Version != "" && release.Version != currentVersion
			if isNewer || (forcePrompt && isDifferent) {
				// Same install-method gate as in CheckForUpdate — see comment there.
				if forcePrompt || isVersionAvailableForInstallMethod(release.Version) {
					updateInfo.CurrentVersion = currentVersion
					updateInfo.AvailableVersion = release.Version
					updateInfo.UpdateAvailable = true
					updateInfo.ReleaseNotes = release.ReleaseNotes
				} else {
					logrus.Debug("Version not yet available via install method, skipping")
				}
			} else {
				updateInfo.CurrentVersion = currentVersion
				updateInfo.AvailableVersion = ""
				updateInfo.UpdateAvailable = false
				updateInfo.ReleaseNotes = ""
				updateInfo.DismissedVersion = ""
				updateInfo.DismissedAt = time.Time{}
			}
			updateInfo.LastChecked = time.Now()
			_ = storeUpdateInfo(updateInfo)
		}
	}

	if !updateInfo.UpdateAvailable {
		return false, "", ""
	}

	if !forcePrompt {
		if !isNewerVersion(updateInfo.AvailableVersion, currentVersion) {
			return false, "", ""
		}
		// Respect the existing dismissal window — don't nag a user who already
		// declined the same version at startup.
		if !shouldShowBlockingPrompt(updateInfo) {
			return false, "", ""
		}
	}

	return true, updateInfo.AvailableVersion, updateInfo.ReleaseNotes
}

func showUpdatePromptOrNotice(updateInfo UpdateInfo, currentVersion string, isUpdateCommand bool, forcePrompt bool) {
	// If user is running "tdl update", skip — they're already updating
	if isUpdateCommand {
		return
	}

	// Non-interactive terminal (CI, piped stdin) — passive notice only
	if !IsStdinTerminal() {
		printVersionNotice(currentVersion, updateInfo.AvailableVersion)
		return
	}

	if forcePrompt || shouldShowBlockingPrompt(updateInfo) {
		showBlockingUpdatePrompt(updateInfo, currentVersion)
	} else {
		printVersionNotice(currentVersion, updateInfo.AvailableVersion)
	}
}

func shouldShowBlockingPrompt(info UpdateInfo) bool {
	// Never dismissed — show prompt
	if info.DismissedVersion == "" {
		return true
	}

	// Dismissed a different version — new release, re-prompt
	if info.DismissedVersion != info.AvailableVersion {
		return true
	}

	// Dismissed same version — only re-prompt after dismissal duration
	return time.Since(info.DismissedAt) > dismissalDuration
}

func showBlockingUpdatePrompt(updateInfo UpdateInfo, currentVersion string) {
	c := color.New(color.FgHiYellow)
	_, _ = c.Printf("A new version of the CLI is available: %s \u2192 %s\n", currentVersion, updateInfo.AvailableVersion)
	_, _ = c.Printf("Some features may be missing or not work correctly.\n")

	if updateInfo.ReleaseNotes != "" {
		formatted := FormatReleaseNotes(updateInfo.ReleaseNotes, 15)
		if formatted != "" {
			fmt.Println()
			fmt.Println("Release notes:")
			fmt.Println(formatted)
		}
	}
	fmt.Println()

	method := DetectInstallMethod()

	// Check if binary requires elevated permissions (direct binary install)
	if method == InstallMethodDirectBinary || method == InstallMethodUnknown {
		binaryPath, err := os.Executable()
		if err == nil {
			binaryPath, _ = filepath.EvalSymlinks(binaryPath)
		}
		if err != nil || !canWriteBinary(binaryPath) {
			cmdName := os.Args[0]
			var updateCmd string
			if runtime.GOOS == "windows" {
				updateCmd = fmt.Sprintf("%s update", cmdName)
				fmt.Println("The binary requires elevated permissions to update.")
				fmt.Println("To update, re-open your terminal as Administrator and run:")
			} else {
				updateCmd = fmt.Sprintf("sudo %s update", cmdName)
				fmt.Printf("The binary at %s requires elevated permissions to update.\n", binaryPath)
				fmt.Println("To update, run:")
			}
			fmt.Println("  " + SprintCommand(updateCmd))
			fmt.Printf("\nOr download from: %s/releases/latest\n", repoURL)
			fmt.Println()

			result := Prompt(
				Actions{
					{Shortcut: '\n', Action: "exit", ShortcutAliases: []rune{'\r'}},
					{Shortcut: 's', Action: "skip and continue"},
				},
				os.Stdin,
				os.Stdout,
			)

			// Store dismissal regardless of choice
			updateInfo.DismissedVersion = updateInfo.AvailableVersion
			updateInfo.DismissedAt = time.Now()
			_ = storeUpdateInfo(updateInfo)

			if result == '\n' {
				os.Exit(0)
			}
			fmt.Println()
			return
		}
	}

	hint := updateCommandHint(method, updateInfo.AvailableVersion)
	action := "update now"
	if hint != "" {
		action = fmt.Sprintf("run %s", SprintCommand(hint))
	}

	result := Prompt(
		Actions{
			{Shortcut: '\n', Action: action, ShortcutAliases: []rune{'\r'}},
			{Shortcut: 's', Action: "skip"},
		},
		os.Stdin,
		os.Stdout,
	)

	if result == 's' {
		// User declined — record dismissal
		updateInfo.DismissedVersion = updateInfo.AvailableVersion
		updateInfo.DismissedAt = time.Now()
		_ = storeUpdateInfo(updateInfo)
		fmt.Println()
		return
	}

	// User pressed ENTER — run update with SkipConfirm (no double confirmation)
	fmt.Println()
	ctx := context.Background()
	err := RunUpdate(ctx, currentVersion, UpdateOptions{SkipConfirm: true, ForceUpdate: true})
	if err != nil {
		fmt.Println(color.RedString("Update failed: %v", err))
		fmt.Println(color.HiBlackString("Continuing with current version..."))
		fmt.Println()
		return
	}

	// Update succeeded — binary is replaced, must exit
	fmt.Println()
	fmt.Println("Please re-run your command.")
	os.Exit(0)
}

func updateCommandHint(method InstallMethod, availableVersion string) string {
	switch method {
	case InstallMethodHomebrew:
		return "brew upgrade tdl"
	case InstallMethodGoInstall:
		// Pin to the explicit version when we know it so copy-pasting the
		// hint does not hit a stale "@latest" resolution in the Go proxy.
		ref := "latest"
		if availableVersion != "" {
			ref = "v" + strings.TrimPrefix(availableVersion, "v")
		}
		return "go install github.com/ThreeDotsLabs/cli/tdl@" + ref
	case InstallMethodNix:
		return "nix profile upgrade --flake github:ThreeDotsLabs/cli"
	case InstallMethodScoop:
		return "scoop update tdl"
	default:
		return ""
	}
}

func printVersionNotice(currentVersion string, availableVersion string) {
	c := color.New(color.FgHiYellow)
	_, _ = c.Printf("A new version of the CLI is available: %s (current: %s)\n", availableVersion, currentVersion)
	_, _ = c.Printf("Some features may be missing or not work correctly. Please update soon!\n")
	_, _ = c.Printf("Run %s to update, or see: %v/releases\n", SprintCommand(os.Args[0]+" update"), repoURL)
	fmt.Println()
}

func getLatestRelease() *latestRelease {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesURL, nil)
	if err != nil {
		return nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var release releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil
	}

	version := strings.TrimLeft(release.TagName, "v")
	if version == "" {
		return nil
	}

	return &latestRelease{
		Version:      version,
		ReleaseNotes: release.Body,
	}
}

func updateInfoPath() string {
	return path.Join(GlobalConfigDir(), "update")
}

type UpdateInfo struct {
	CurrentVersion   string    `json:"current_version"`
	AvailableVersion string    `json:"available_version"`
	UpdateAvailable  bool      `json:"update_available"`
	LastChecked      time.Time `json:"last_checked"`
	ReleaseNotes     string    `json:"release_notes,omitempty"`
	DismissedVersion string    `json:"dismissed_version,omitempty"`
	DismissedAt      time.Time `json:"dismissed_at,omitempty"`
}

func getUpdateInfo() (UpdateInfo, error) {
	if !fileExists(updateInfoPath()) {
		return UpdateInfo{}, nil
	}

	content, err := os.ReadFile(updateInfoPath())
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("failed to read update file: %w", err)
	}

	var info UpdateInfo
	if err := json.Unmarshal(content, &info); err != nil {
		return UpdateInfo{}, fmt.Errorf("failed to unmarshal update info: %w", err)
	}

	return info, nil
}

func storeUpdateInfo(info UpdateInfo) error {
	content, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal update info: %w", err)
	}

	if err := os.WriteFile(updateInfoPath(), content, 0644); err != nil {
		return fmt.Errorf("failed to write update info file: %w", err)
	}

	return nil
}

func isNewerVersion(latest, current string) bool {
	latestV, err := semver.NewVersion(latest)
	if err != nil {
		return latest != current
	}
	currentV, err := semver.NewVersion(current)
	if err != nil {
		return latest != current
	}
	return latestV.GreaterThan(currentV)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}

	return false
}
