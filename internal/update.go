package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"net/http"
	"os"
	"path"
	"time"
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

	lastUpdate, _ := LastUpdateCheckTime()

	if time.Since(lastUpdate) < 24*time.Hour {
		return
	}

	defer func() {
		_ = StoreUpdateCheckTime(time.Now().UTC())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesURL, nil)
	if err != nil {
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var release releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return
	}

	if release.TagName != currentVersion {
		c := color.New(color.FgHiYellow)
		_, _ = c.Printf("A new version is available: %s (current: %s)\n", release.TagName, currentVersion)
		_, _ = c.Printf("Visit %v to update\n", repoURL)
		fmt.Println()
	}
}

func lastUpdateCheckPath() string {
	return path.Join(GlobalConfigDir(), "last-update-check")
}

func LastUpdateCheckTime() (time.Time, error) {
	if !fileExists(lastUpdateCheckPath()) {
		return time.Time{}, nil
	}

	content, err := os.ReadFile(lastUpdateCheckPath())
	if err != nil {
		return time.Time{}, err
	}

	t, err := time.Parse(time.RFC3339, string(content))
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

func StoreUpdateCheckTime(t time.Time) error {
	return os.WriteFile(lastUpdateCheckPath(), []byte(t.Format(time.RFC3339)), 0644)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}

	return false
}
