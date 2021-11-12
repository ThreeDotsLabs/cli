package trainings

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ThreeDotsLabs/cli/tdl/internal"
	"github.com/ThreeDotsLabs/cli/tdl/trainings/config"
	"github.com/ThreeDotsLabs/cli/tdl/trainings/genproto"
)

func (h *Handlers) Run(ctx context.Context) (bool, error) {
	wd, err := os.Getwd()
	if err != nil {
		return false, errors.WithStack(err)
	}

	if _, err := h.config.FindTrainingRoot(wd); errors.Is(err, config.TrainingRootNotFoundError) {
		fmt.Println("You are not in a training directory. If you already started the training, please go to the exercise directory.")
		fmt.Printf("Please run %s if you didn't start training yet.\n", internal.SprintCommand("tdl training init"))
		return false, nil
	}

	if !h.config.ExerciseConfigExists(wd) {
		fmt.Println("You are not in an exercise directory.")
		fmt.Println("Please go to the exercise directory.")
		return false, nil
	}

	// todo - validate if exercise id == training exercise id? to ensure about consistency
	success, err := h.runExercise(ctx, wd)
	if !success || err != nil {
		return success, err
	}

	fmt.Println()
	if !internal.ConfirmPromptDefaultYes("Do you want to go to the next exercise?") {
		return success, nil
	}

	// todo - is this assumption always valid about training dir?
	return success, h.nextExercise(ctx, h.config.ExerciseConfig(wd).ExerciseID, wd)
}

func (h *Handlers) runExercise(ctx context.Context, dir string) (bool, error) {
	files, err := h.files.ReadSolutionFiles(dir)
	if err != nil {
		return false, err
	}

	req := &genproto.VerifyExerciseRequest{
		ExerciseId: h.config.ExerciseConfig(dir).ExerciseID,
		Files:      files,
		Token:      h.config.GlobalConfig().Token,
	}
	logrus.WithField("req", req).Info("Request prepared")

	runCtx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	stream, err := h.newGrpcClient(ctx).VerifyExercise(runCtx, req)
	if err != nil {
		return false, err
	}

	successful := false
	finished := false

	for {
		response, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		if response.Finished {
			fmt.Println("--------")

			var msg string

			if response.Successful {
				msg = color.GreenString("SUCCESS")
				successful = true
				finished = true
			} else {
				msg = color.RedString("FAIL")
				finished = true
			}

			fmt.Println(msg)
		}

		if len(response.Stdout) > 0 {
			fmt.Println(response.Stdout)
		}
		if len(response.Stderr) > 0 {
			_, _ = fmt.Fprintln(os.Stderr, response.Stderr)
		}
		// todo - support stderr and commands
	}

	if !finished {
		return false, errors.New("execution didn't finish")
	} else {
		return successful, nil
	}
}
