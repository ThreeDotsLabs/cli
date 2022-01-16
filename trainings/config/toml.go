package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func (c Config) writeConfigToml(fs afero.Fs, destPath string, v interface{}) error {
	dir := filepath.Dir(destPath)

	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return errors.Wrapf(err, "can't create config dir %s", dir)
	}

	f, err := fs.OpenFile(destPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return errors.Wrapf(err, "can't open config file %s", destPath)
	}

	if err := toml.NewEncoder(f).Encode(v); err != nil {
		return errors.Wrap(err, "can't encode config")
	}
	if err := f.Close(); err != nil {
		return errors.Wrap(err, "can't close config file")
	}

	return nil
}
