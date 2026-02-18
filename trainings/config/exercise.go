package config

import (
	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

const exerciseConfigFile = ".tdl-exercise"

type ExerciseConfig struct {
	ExerciseID   string `toml:"exercise_id"`
	Directory    string `toml:"directory"`
	IsTextOnly   bool   `toml:"is_text_only"`
	IsOptional   bool   `toml:"is_optional"`
	ModuleName   string `toml:"module_name"`
	ExerciseName string `toml:"exercise_name"`
}

// ModuleExercisePath returns "module/exercise" for use in branch names and commit messages.
// Falls back to Directory for old configs that don't have module/exercise names.
func (c ExerciseConfig) ModuleExercisePath() string {
	if c.ModuleName != "" && c.ExerciseName != "" {
		return c.ModuleName + "/" + c.ExerciseName
	}
	return c.Directory
}

func (c Config) WriteExerciseConfig(trainingRootFs afero.Fs, cfg ExerciseConfig) error {
	return c.writeConfigToml(trainingRootFs, exerciseConfigFile, cfg)
}

func (c Config) ExerciseConfig(trainingRootFs afero.Fs) ExerciseConfig {
	b, err := afero.ReadFile(trainingRootFs, exerciseConfigFile)
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
