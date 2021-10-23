package trainings

import (
	"context"
	"os"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/ThreeDotsLabs/cli/tdl/trainings/genproto"
	"github.com/sirupsen/logrus"
)

type globalConfig struct {
	Token      string
	ServerAddr string
}

const defaultServerAddress = "localhost:3000"

const globalTrainingsConfigFile = ".trainings-config"
const configDir = "three-dots-labs"

func ConfigureGlobally(token, serverAddr string, override bool) error {
	configPath := globalConfigPath()

	logrus.WithFields(logrus.Fields{
		"serverAddr": serverAddr,
		"override":   override,
		"configPath": configPath,
	}).Debug("Configuring")

	if !override && fileExists(configPath) {
		panic("Trainings are already configured. Please pass --override flag to configure again.")
	}

	if _, err := NewGrpcClient(serverAddr).Init(context.Background(), &genproto.InitRequest{Token: token}); err != nil {
		// todo - remove all panics
		panic(err)
	}

	writeConfigToml(configPath, globalConfig{
		Token:      token,
		ServerAddr: serverAddr,
	})

	return nil
}

func globalConfigPath() string {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	configDir := path.Join(userConfigDir, configDir)
	configPath := path.Join(configDir, globalTrainingsConfigFile)

	return configPath
}

// todo - run once
func readGlobalConfig() globalConfig {
	configPath := globalConfigPath()

	if !fileExists(configPath) {
		// todo - better UX
		// todo - site url when to get token?
		panic("training not configured, please run tdl training configure")
	}

	config := globalConfig{}
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		// todo - better handling
		panic(err)
	}

	if config.ServerAddr == "" {
		config.ServerAddr = defaultServerAddress
	}
	if config.Token == "" {
		panic("empty token")
	}

	logrus.WithField("training_config", config).Debug("Global config")

	return config
}
