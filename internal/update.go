package internal

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"net/http"
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

	resp, err := http.Get(releasesURL)
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
