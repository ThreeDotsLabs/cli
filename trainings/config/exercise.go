package config

import (
	"path"

	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"
)

type ExerciseConfig struct {
	ExerciseID   string `toml:"exercise_id"`
	TrainingName string `toml:"training_name"`
}

const ExerciseConfigFile = ".tdl-exercise"

func (c Config) WriteExerciseConfig(dir string, cfg ExerciseConfig) error {
	return c.writeConfigToml(path.Join(dir, ExerciseConfigFile), cfg)
}

func (c Config) ExerciseConfig(dir string) ExerciseConfig {
	exerciseConfig := ExerciseConfig{}
	if _, err := toml.DecodeFile(c.ExerciseConfigPath(dir), &exerciseConfig); err != nil {
		panic(err)
	}

	logrus.WithFields(logrus.Fields{
		"training": exerciseConfig.TrainingName,
		"exercise": exerciseConfig.ExerciseID,
	}).Debug("Calculated training and exercise")

	return exerciseConfig
}

func (c Config) ExerciseConfigPath(dir string) string {
	return path.Join(dir, ExerciseConfigFile)
}

func (c Config) ExerciseConfigExists(dir string) bool {
	return c.dirOrFileExists(c.ExerciseConfigPath(dir))
}
