package config

import (
	"fmt"
	"os"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ThreeDotsLabs/cli/tdl/internal"
	"github.com/ThreeDotsLabs/cli/tdl/trainings/web"
)

type GlobalConfig struct {
	Token      string `toml:"token"`
	ServerAddr string `toml:"server_addr"`
}

func globalConfigPath() string {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	configDir := path.Join(userConfigDir, "three-dots-labs")
	configPath := path.Join(configDir, ".trainings-config")

	return configPath
}

const defaultServer = "localhost:3000"

func (c Config) ConfiguredGlobally() bool {
	return c.dirOrFileExists(globalConfigPath())
}

func (c Config) WriteGlobalConfig(cfg GlobalConfig) error {
	return c.writeConfigToml(globalConfigPath(), cfg)
}

func (c Config) GlobalConfig() GlobalConfig {
	configPath := globalConfigPath()

	logrus.WithField("path", configPath).Debug("Loading global config")

	if !c.dirOrFileExists(configPath) {
		panic(errors.Errorf(
			"trainings are not configured, please visit %s to get credentials and run %s",
			web.Website, internal.SprintCommand("tdl training configure"),
		))
	}

	config := GlobalConfig{}
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		panic(errors.Wrapf(err, "unable to parse global config from %s", configPath))
	}

	if config.ServerAddr == "" {
		config.ServerAddr = defaultServer
	}
	if config.Token == "" {
		panic(fmt.Sprintf("empty token in %s", configPath))
	}

	logrus.WithField("training_config", config).Debug("Global config")

	return config
}
