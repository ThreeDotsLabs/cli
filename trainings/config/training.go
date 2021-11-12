package config

import (
	"path"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ThreeDotsLabs/cli/tdl/internal"
)

const trainingConfigFile = ".tdl-training"

type TrainingConfig struct {
	TrainingName string `toml:"training_name"`
}

func (c Config) WriteTrainingConfig(config TrainingConfig, dir string) error {
	logrus.WithField("training_root", dir).Debug("Creating training root")

	return c.writeConfigToml(path.Join(dir, trainingConfigFile), config)
}

func (c Config) TrainingConfig(dir string) TrainingConfig {
	trainingRoot, err := c.FindTrainingRoot(dir)
	if err != nil {
		panic(err)
	}

	trainingConfigPath := path.Join(trainingRoot, trainingConfigFile)

	config := TrainingConfig{}
	if _, err := toml.DecodeFile(trainingConfigPath, &config); err != nil {
		panic(errors.Wrapf(err, "can't decode %s", trainingConfigPath))
	}

	logrus.WithField("training_config", config).Debug("Training config")

	return config
}

// todo - check if it's printing properly
var TrainingRootNotFoundError = errors.Errorf("training root not found, did you run %s?", internal.SprintCommand("tdl trainings init"))

func (c Config) FindTrainingRoot(dir string) (string, error) {
	for {
		if c.dirOrFileExists(path.Join(dir, trainingConfigFile)) {
			return dir, nil
		}

		dir = path.Dir(dir)
		if dir == "/" {
			break
		}
	}

	return "", TrainingRootNotFoundError
}
