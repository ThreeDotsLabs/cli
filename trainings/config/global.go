package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ThreeDotsLabs/cli/internal"
)

type GlobalConfig struct {
	Token      string `toml:"token"`
	ServerAddr string `toml:"server_addr"`
	Region     string `toml:"region"`
	Insecure   bool   `toml:"insecure"`
}

func globalConfigPath() string {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	configDir := filepath.Join(userConfigDir, "three-dots-labs")
	configPath := filepath.Join(configDir, ".trainings-config")

	return configPath
}

func (c Config) ConfiguredGlobally() bool {
	return c.dirOrFileExists(c.osFs, globalConfigPath())
}

func (c Config) WriteGlobalConfig(cfg GlobalConfig) error {
	return c.writeConfigToml(c.osFs, globalConfigPath(), cfg)
}

func (c Config) GlobalConfig() GlobalConfig {
	configPath := globalConfigPath()

	logrus.WithField("path", configPath).Debug("Loading global config")

	if !c.dirOrFileExists(c.osFs, configPath) {
		panic(errors.Errorf(
			"trainings are not configured, please visit %s to get credentials and run %s",
			internal.WebsiteAddress, internal.SprintCommand("tdl training configure"),
		))
	}

	config := GlobalConfig{}
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		panic(errors.Wrapf(err, "unable to parse global config from %s", configPath))
	}

	if config.ServerAddr == "" {
		config.ServerAddr = internal.DefaultTrainingsServer
	}
	if config.Token == "" {
		panic(fmt.Sprintf("empty token in %s", configPath))
	}

	configStr := strings.ReplaceAll(fmt.Sprintf("%#v", config), config.Token, "[token]")
	logrus.WithField("training_config", configStr).Debug("Global config")

	return config
}
