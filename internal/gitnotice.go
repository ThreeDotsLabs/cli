package internal

import (
	"encoding/json"
	"os"
	"path"
	"time"
)

const gitInstallNoticeCooldown = 24 * time.Hour

type gitInstallNoticeInfo struct {
	LastShown time.Time `json:"last_shown"`
}

func gitInstallNoticePath() string {
	return path.Join(GlobalConfigDir(), "git-install-notice")
}

func ShouldShowGitInstallNotice() bool {
	info, err := readGitInstallNoticeInfo()
	if err != nil {
		return true
	}
	return time.Since(info.LastShown) >= gitInstallNoticeCooldown
}

func RecordGitInstallNoticeShown() error {
	info := gitInstallNoticeInfo{LastShown: time.Now()}
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return os.WriteFile(gitInstallNoticePath(), data, 0644)
}

func readGitInstallNoticeInfo() (gitInstallNoticeInfo, error) {
	data, err := os.ReadFile(gitInstallNoticePath())
	if err != nil {
		return gitInstallNoticeInfo{}, err
	}
	var info gitInstallNoticeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return gitInstallNoticeInfo{}, err
	}
	return info, nil
}
