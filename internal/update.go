package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/fatih/color"
)

type releaseResponse struct {
	TagName string `json:"tag_name"`
}

const repoURL = "https://github.com/ThreeDotsLabs/cli"
const releasesURL = "https://api.github.com/repos/ThreeDotsLabs/cli/releases/latest"

func CheckForUpdate(currentVersion string) {
	if currentVersion == "" || currentVersion == "dev" {
		return
	}

	updateInfo, _ := getUpdateInfo()

	if updateInfo.UpdateAvailable && updateInfo.CurrentVersion == currentVersion {
		printVersionNotice(updateInfo.CurrentVersion, updateInfo.AvailableVersion)
		return
	}

	if time.Since(updateInfo.LastChecked) < time.Hour {
		return
	}

	latestVersion := getLatestVersion()

	if latestVersion != "" && latestVersion != currentVersion {
		updateInfo.CurrentVersion = currentVersion
		updateInfo.AvailableVersion = latestVersion
		updateInfo.UpdateAvailable = true

		printVersionNotice(currentVersion, latestVersion)
	} else {
		updateInfo.CurrentVersion = currentVersion
		updateInfo.AvailableVersion = ""
		updateInfo.UpdateAvailable = false
	}

	updateInfo.LastChecked = time.Now()

	_ = storeUpdateInfo(updateInfo)
}

func printVersionNotice(currentVersion string, availableVersion string) {
	c := color.New(color.FgHiYellow)
	_, _ = c.Printf("A new version of the CLI is available: %s (current: %s)\n", availableVersion, currentVersion)
	_, _ = c.Printf("Some features may be missing or not work correctly. Please update soon!\n")
	_, _ = c.Printf("See instructions at: %v\n", repoURL)
	fmt.Println()
}

func getLatestVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesURL, nil)
	if err != nil {
		return ""
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var release releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return ""
	}

	return strings.TrimLeft(release.TagName, "v")
}

func updateInfoPath() string {
	return path.Join(GlobalConfigDir(), "update")
}

type UpdateInfo struct {
	CurrentVersion   string    `json:"current_version"`
	AvailableVersion string    `json:"available_version"`
	UpdateAvailable  bool      `json:"update_available"`
	LastChecked      time.Time `json:"last_checked"`
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}

	return false
}
