package course

import (
	"fmt"
	"os"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"
)

type globalConfig struct {
	Token      string
	ServerAddr string
}

const defaultServerAddress = "localhost:3000"

const globalCoursesConfigFile = ".courses-config"
const configDir = "three-dots-labs"

func ConfigureGlobally(token, serverAddr string, override bool) {
	configPath := globalConfigPath()

	if !override && fileExists(configPath) {
		fmt.Println("Courses are already configured. Please pass --override flag to configure again.")
		return
	}

	writeConfigToml(configPath, globalConfig{
		Token:      token,
		ServerAddr: serverAddr,
	})
}

func globalConfigPath() string {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	configDir := path.Join(userConfigDir, configDir)
	configPath := path.Join(configDir, globalCoursesConfigFile)

	return configPath
}

// todo - run once
func readGlobalConfig() globalConfig {
	configPath := globalConfigPath()

	if !fileExists(configPath) {
		// todo - better UX
		// todo - site url when to get token?
		panic("course not configured, plase run tdl course configure")
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

	logrus.WithField("course_config", config).Debug("Global config")

	return config
}
