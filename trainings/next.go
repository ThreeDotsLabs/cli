package trainings

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ThreeDotsLabs/cli/tdl/internal"
	"github.com/ThreeDotsLabs/cli/tdl/trainings/config"
	"github.com/ThreeDotsLabs/cli/tdl/trainings/genproto"
	"github.com/ThreeDotsLabs/cli/tdl/trainings/web"
)

func (h *Handlers) nextExercise(ctx context.Context, currentExerciseID string, dir string) error {
	finished, resp, err := h.getNextExercise(ctx, currentExerciseID, dir)
	if err != nil {
		return err
	}
	if finished {
		trainingFinished()
		return nil
	}

	trainingRoot, err := h.config.FindTrainingRoot(dir)
	if err != nil {
		return err
	}

	exerciseDir := h.calculateExerciseDir(resp, trainingRoot)

	if err := h.writeExerciseFiles(resp.ExerciseId, resp.FilesToCreate, exerciseDir); err != nil {
		return err
	}

	return h.showExerciseTips(exerciseDir)
}

func (h *Handlers) getNextExercise(ctx context.Context, currentExerciseID string, dir string) (finished bool, resp *genproto.NextExerciseResponse, err error) {
	resp, err = h.newGrpcClient(ctx).NextExercise(context.Background(), &genproto.NextExerciseRequest{
		TrainingName:      h.config.TrainingConfig(dir).TrainingName,
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

func (h *Handlers) calculateExerciseDir(resp *genproto.NextExerciseResponse, trainingRoot string) string {
	// We should never trust the remote server.
	// Writing files based on external name is a vector for Path Traversal attack.
	// For more info please check: https://owasp.org/www-community/attacks/Path_Traversal
	//
	// Fortunately, path.Join is protecting us from that by calling path.Clean().
	return path.Join(trainingRoot, resp.Dir)
}

func (h *Handlers) writeExerciseFiles(exerciseID string, filesToCreate []*genproto.File, exerciseDir string) error {
	if err := h.files.WriteExerciseFiles(filesToCreate, exerciseDir); err != nil {
		return err
	}

	return h.config.WriteExerciseConfig(exerciseDir, config.ExerciseConfig{
		ExerciseID:   exerciseID,
		TrainingName: h.config.TrainingConfig(exerciseDir).TrainingName,
	})
}

func (h *Handlers) showExerciseTips(exerciseDir string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "can't get working directory")
	}
	cdRequired := pwd != exerciseDir

	relExpectedDir, err := filepath.Rel(pwd, exerciseDir)
	if err != nil {
		return errors.Wrapf(err, "can't generate rel path for %s and %s", pwd, exerciseDir)
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
