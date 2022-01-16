package trainings

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ThreeDotsLabs/cli/trainings/files"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/ThreeDotsLabs/cli/trainings/web"
)

func (h *Handlers) nextExercise(ctx context.Context, currentExerciseID string, dir string) error {
	trainingRoot, err := h.config.FindTrainingRoot(dir)
	if err != nil {
		return err
	}

	// We should never trust the remote server.
	// Writing files based on external name is a vector for Path Traversal attack.
	// For more info please check: https://owasp.org/www-community/attacks/Path_Traversal
	//
	// To avoid that we are using afero.BasePathFs with base on training root for all operations in trainings dir.
	trainingRootFs := afero.NewBasePathFs(afero.NewOsFs(), trainingRoot)

	finished, resp, err := h.getNextExercise(ctx, currentExerciseID, trainingRootFs)
	if err != nil {
		return err
	}
	if finished {
		trainingFinished()
		return nil
	}

	if err := h.writeExerciseFiles(resp, trainingRootFs); err != nil {
		return err
	}

	return h.showExerciseTips(trainingRoot, resp.Dir)
}

func (h *Handlers) getNextExercise(ctx context.Context, currentExerciseID string, trainingRootFs afero.Fs) (finished bool, resp *genproto.NextExerciseResponse, err error) {
	resp, err = h.newGrpcClient(ctx).NextExercise(ctx, &genproto.NextExerciseRequest{
		TrainingName:      h.config.TrainingConfig(trainingRootFs).TrainingName,
		CurrentExerciseId: currentExerciseID,
		Token:             h.config.GlobalConfig().Token,
	})
	if status.Code(err) == codes.NotFound {
		return true, nil, nil
	} else if err != nil {
		return false, nil, errors.Wrap(err, "Can't get next exercise")
	}

	logrus.WithFields(logrus.Fields{"resp": resp}).Debug("Received exercise from server")

	return false, resp, nil
}

func (h *Handlers) writeExerciseFiles(resp *genproto.NextExerciseResponse, trainingRootFs afero.Fs) error {
	if err := files.NewFiles().WriteExerciseFiles(resp.FilesToCreate, trainingRootFs, resp.Dir); err != nil {
		return err
	}

	return h.config.WriteExerciseConfig(
		trainingRootFs,
		config.ExerciseConfig{
			ExerciseID: resp.ExerciseId,
			Directory:  resp.Dir,
		},
	)
}

func (h *Handlers) showExerciseTips(trainingRoot string, exerciseDir string) error {
	exerciseAbsDir := filepath.Join(trainingRoot, exerciseDir)

	pwd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "can't get working directory")
	}
	cdRequired := pwd != exerciseAbsDir

	relExpectedDir, err := filepath.Rel(pwd, exerciseAbsDir)
	if err != nil {
		return errors.Wrapf(err, "can't generate rel path for %s and %s", pwd, exerciseAbsDir)
	}

	if cdRequired {
		fmt.Printf("Exercise files were created in '%s' directory.\n", relExpectedDir)
		fmt.Println("Please execute", internal.SprintCommand("cd "+relExpectedDir), "to get there.")
	}

	fmt.Printf("\nPlase go to %s see exercise content.\n", web.Website)
	fmt.Printf("To run solution, please execute " + internal.SprintCommand("tdl training run"))
	if cdRequired {
		fmt.Print(" in ", relExpectedDir)
	}
	fmt.Println()

	return nil
}

func trainingFinished() {
	fmt.Println("Congratulations, you finished the training " + color.YellowString("🏆"))
}
