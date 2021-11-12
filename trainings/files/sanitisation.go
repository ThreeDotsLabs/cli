package files

import (
	"strings"

	"github.com/ThreeDotsLabs/cli/tdl/trainings/genproto"
)

func ValidateFilesToCreate(filesToCreate []*genproto.File) error {
	for _, file := range filesToCreate {
		if !validateFilePath(file) {
			return InvalidFilePathError{file.Path}
		}
	}
	return nil
}

func validateFilePath(file *genproto.File) bool {
	if file.Path == "" {
		return false
	}
	if strings.Contains(file.Path, "..") {
		return false
	}
	if strings.HasPrefix(file.Path, "/") {
		return false
	}

	return true
}

const exerciseDirAllowedChars = "abcdefghijklmnopqrstuvwxyz0123456789-_"

func ValidateExerciseDir(dir string) bool {
	if dir == "" {
		return false
	}

	for _, char := range dir {
		if !strings.Contains(exerciseDirAllowedChars, strings.ToLower(string(char))) {
			return false
		}
	}

	return true
}
