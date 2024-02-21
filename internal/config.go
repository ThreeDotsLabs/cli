package internal

import (
	"os"
	"path/filepath"
)

func GlobalConfigDir() string {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	return filepath.Join(userConfigDir, "three-dots-labs")
}
