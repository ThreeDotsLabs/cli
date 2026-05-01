package internal

import (
	"os"
	"path/filepath"
)

func BinaryName() string {
	return filepath.Base(os.Args[0])
}
