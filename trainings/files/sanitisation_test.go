package files_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ThreeDotsLabs/cli/tdl/trainings/files"
	"github.com/ThreeDotsLabs/cli/tdl/trainings/genproto"
)

func TestValidateFilesToCreate(t *testing.T) {
	testCases := []struct {
		Name  string
		File  *genproto.File
		Valid bool
	}{
		{
			Name: "hidden_file",
			File: &genproto.File{
				Path: ".test",
			},
			Valid: true,
		},
		{
			Name: "empty_path",
			File: &genproto.File{
				Path: "",
			},
			Valid: false,
		},
		{
			Name: "single_dot_path",
			File: &genproto.File{
				Path: "./main.go",
			},
			Valid: true,
		},
		{
			Name: "parent_directory",
			File: &genproto.File{
				Path: "../main.go",
			},
			Valid: false,
		},
		{
			Name: "parent_directory_windows",
			File: &genproto.File{
				Path: "..\\main.go",
			},
			Valid: false,
		},
		{
			Name: "parent_directory_foo",
			File: &genproto.File{
				Path: "../foo/main.go",
			},
			Valid: false,
		},
		{
			Name: "dots_inside",
			File: &genproto.File{
				Path: "/var/www/images/../../../etc/passwd",
			},
			Valid: false,
		},
		{
			Name: "two_dots",
			File: &genproto.File{
				Path: "../main.go",
			},
			Valid: false,
		},
		{
			Name: "two_levels_up",
			File: &genproto.File{
				Path: "../../main.go",
			},
			Valid: false,
		},
		{
			Name: "three_dots",
			File: &genproto.File{
				Path: ".../main.go",
			},
			Valid: false,
		},
		{
			Name: "four_dots",
			File: &genproto.File{
				Path: "..../main.go",
			},
			Valid: false,
		},
		{
			Name: "root",
			File: &genproto.File{
				Path: "/main.go",
			},
			Valid: false,
		},
		{
			Name: "root_dir",
			File: &genproto.File{
				Path: "/foo/bar/main.go",
			},
			Valid: false,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.Name, func(t *testing.T) {
			err := files.ValidateFilesToCreate([]*genproto.File{
				tc.File,
			})

			if !tc.Valid {
				assert.Error(t, err)
				assert.IsType(t, err, files.InvalidFilePathError{})
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateExerciseDir(t *testing.T) {
	testCases := []struct {
		Dir   string
		Valid bool
	}{
		{
			Dir:   "exercise",
			Valid: true,
		},
		{
			Dir:   "exercise-1",
			Valid: true,
		},
		{
			Dir:   "exercise_1",
			Valid: true,
		},
		{
			Dir:   "/etc/password",
			Valid: false,
		},
		{
			Dir:   "/",
			Valid: false,
		},
		{
			Dir:   "/exercise",
			Valid: false,
		},
		{
			Dir:   "exercise\\1",
			Valid: false,
		},
		{
			Dir:   "exercise!!",
			Valid: false,
		},
		{
			Dir:   "",
			Valid: false,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.Dir, func(t *testing.T) {
			result := files.ValidateExerciseDir(tc.Dir)
			assert.Equal(t, tc.Valid, result)
		})
	}
}
