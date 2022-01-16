package config

import (
	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

const ExerciseConfigFile = ".tdl-exercise"

type ExerciseConfig struct {
	ExerciseID string `toml:"exercise_id"`
	Directory  string `toml:"directory"`
}

func (c Config) WriteExerciseConfig(trainingRootFs afero.Fs, cfg ExerciseConfig) error {
	return c.writeConfigToml(trainingRootFs, ExerciseConfigFile, cfg)
}

func (c Config) ExerciseConfig(trainingRootFs afero.Fs) ExerciseConfig {
	b, err := afero.ReadFile(trainingRootFs, ExerciseConfigFile)
	if err != nil {
		panic(errors.Wrap(err, "can't read exercise config"))
	}

	exerciseConfig := ExerciseConfig{}
	if _, err := toml.Decode(string(b), &exerciseConfig); err != nil {
		panic(errors.Wrapf(err, "can't decode exercise config: %s", string(b)))
	}

	logrus.WithFields(logrus.Fields{
		"exercise": exerciseConfig.ExerciseID,
		"dir":      exerciseConfig.Directory,
	}).Debug("Calculated training and exercise")

	return exerciseConfig
}
