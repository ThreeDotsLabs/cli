package config

import (
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/internal"
)

const trainingConfigFile = ".tdl-training"

type TrainingConfig struct {
	TrainingName string `toml:"training_name"`
}

func (c Config) WriteTrainingConfig(config TrainingConfig, trainingRootFs afero.Fs) error {
	logrus.Debug("Creating training root")

	return c.writeConfigToml(trainingRootFs, trainingConfigFile, config)
}

func (c Config) TrainingConfig(trainingRootFs afero.Fs) TrainingConfig {
	b, err := afero.ReadFile(trainingRootFs, trainingConfigFile)
	if err != nil {
		panic(errors.Wrap(err, "can't read training config"))
	}

	config := TrainingConfig{}
	if _, err := toml.Decode(string(b), &config); err != nil {
		panic(errors.Wrapf(err, "can't decode training config: %s", string(b)))
	}

	logrus.WithField("training_config", config).Debug("Training config")

	return config
}

// todo - check if it's printing properly
var TrainingRootNotFoundError = errors.Errorf("training root not found, did you run %s?", internal.SprintCommand("tdl trainings init"))

func (c Config) FindTrainingRoot(dir string) (string, error) {
	for {
		if c.dirOrFileExists(c.osFs, filepath.Join(dir, trainingConfigFile)) {
			return dir, nil
		}

		dir = filepath.Dir(dir)
		if dir == "/" {
			break
		}
	}

	return "", TrainingRootNotFoundError
}
