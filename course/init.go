package course

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/ThreeDotsLabs/cli/tdl/course/genproto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type globalConfig struct {
	Token      string
	ServerAddr string
}

const defaultServerAddress = "localhost:3000"

const globalCoursesConfigFile = ".courses-config"
const configDir = "three-dots-labs"

func ConfigureGlobally(token, serverAddr string, override bool) error {
	logrus.WithFields(logrus.Fields{"serverAddr": serverAddr, "override": override}).Debug("Configuring")

	configPath := globalConfigPath()

	if !override && fileExists(configPath) {
		fmt.Println("Courses are already configured. Please pass --override flag to configure again.")
		return nil
	}

	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		// todo
		panic(err)
	}

	client := genproto.NewServerClient(conn)

	if _, err = client.Init(context.Background(), &genproto.InitRequest{Token: token}); err != nil {
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
