package config

import (
	"os"

	"github.com/spf13/afero"
)

type Config struct {
	osFs afero.Fs
}

func NewConfig() Config {
	return Config{
		osFs: afero.NewOsFs(),
	}
}

func (c Config) dirOrFileExists(fs afero.Fs, path string) bool {
	_, err := fs.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}

	// it can be only some strange i/o error, let's not silently ignore it
	panic(err)
}
