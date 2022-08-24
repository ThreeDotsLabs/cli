package trainings

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func (h *Handlers) Run(ctx context.Context, detached bool) error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return nil
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)

	if detached {
		return h.detachedRun(ctx, trainingRootFs)
	} else {
		return h.interactiveRun(ctx, trainingRootFs)
	}
}

func (h *Handlers) detachedRun(ctx context.Context, trainingRootFs *afero.BasePathFs) error {
	successful, err := h.run(ctx, trainingRootFs)
	if err != nil {
		return err
	}
	if !successful {
		os.Exit(1)
	}

	promptResult := internal.Prompt(
		internal.Actions{
			{Shortcut: '\n', Action: "go to the next exercise", ShortcutAliases: []rune{'\r'}},
			{Shortcut: 'q', Action: "quit"},
		},
		os.Stdin,
		os.Stdout,
	)
	if promptResult == 'q' {
		os.Exit(0)
	}

	finished, err := h.nextExercise(ctx, h.config.ExerciseConfig(trainingRootFs).ExerciseID)
	if err != nil {
		return err
	}
	if finished {
		return nil
	}

	return nil
}

func (h *Handlers) interactiveRun(ctx context.Context, trainingRootFs *afero.BasePathFs) error {
	for {
		successful, err := h.run(ctx, trainingRootFs)
		if err != nil {
			return err
		}

		if !successful {
			if !internal.ConfirmPromptDefaultYes("run solution again") {
				return nil
			} else {
				continue
			}
		}

		promptResult := internal.Prompt(
			internal.Actions{
				{Shortcut: '\n', Action: "go to the next exercise", ShortcutAliases: []rune{'\r'}},
				{Shortcut: 'r', Action: "re-run solution", ShortcutAliases: []rune{'\r'}},
				{Shortcut: 'q', Action: "quit"},
			},
			os.Stdin,
			os.Stdout,
		)
		if promptResult == 'q' {
			os.Exit(0)
		}
		if promptResult == 'r' {
			continue
		}

		finished, err := h.nextExercise(ctx, h.config.ExerciseConfig(trainingRootFs).ExerciseID)
		if err != nil {
			return err
		}
		if finished {
			return nil
		}

		if !internal.ConfirmPromptDefaultYes("run your solution") {
			return nil
		}
	}
}

func (h *Handlers) run(ctx context.Context, trainingRootFs *afero.BasePathFs) (bool, error) {
	// todo - validate if exercise id == training exercise id? to ensure about consistency
	success, err := h.runExercise(ctx, trainingRootFs)

	if isExerciseNoLongerAvailable(err) {
		fmt.Println(color.YellowString("We did update of the exercise code. Your local workspace is out of sync."))

		if !internal.ConfirmPromptDefaultYes("update your local workspace") {
			os.Exit(0)
		}

		_, err = h.nextExercise(ctx, "")
		return true, err
	}

	fmt.Println()

	return success, err
}

func isExerciseNoLongerAvailable(err error) bool {
	return status.Code(errors.Cause(err)) == codes.NotFound
}

func (h *Handlers) runExercise(ctx context.Context, trainingRootFs *afero.BasePathFs) (bool, error) {
	exerciseConfig := h.config.ExerciseConfig(trainingRootFs)

	solutionFiles, err := files.NewFiles().ReadSolutionFiles(trainingRootFs, exerciseConfig.Directory)
	if err != nil {
		return false, err
	}

	if len(solutionFiles) == 0 {
		solutionFilesRealPath, err := trainingRootFs.RealPath(exerciseConfig.Directory)
		if err != nil {
			logrus.WithField("exercise_dir", exerciseConfig.Directory).Warn("Can't get realpath of solution")
		}

		hintCommand := "tdl training init " + h.config.TrainingConfig(trainingRootFs).TrainingName
		return false, UserFacingError{
			Msg:          fmt.Sprintf("No solution files found in %s.", solutionFilesRealPath),
			SolutionHint: "Please run " + color.CyanString(hintCommand) + " to init exercise files.",
		}
	}

	terminalPath := h.generateRunTerminalPath(trainingRootFs)

	req := &genproto.VerifyExerciseRequest{
		ExerciseId: exerciseConfig.ExerciseID,
		Files:      solutionFiles,
		Token:      h.config.GlobalConfig().Token,
	}

	reqStr := strings.ReplaceAll(fmt.Sprintf("%s", req.String()), h.config.GlobalConfig().Token, "[token]")
	logrus.WithField("req", reqStr).Info("Request prepared")

	runCtx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	stream, err := h.newGrpcClient(ctxWithRequestHeader(ctx, h.cliMetadata)).VerifyExercise(runCtx, req)
	if err != nil {
		return false, err
	}

	successful := false
	finished := false
	verificationID := ""

	for {
		response, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, errors.Wrap(err, "error response from server")
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

		if verificationID == "" && response.VerificationId != "" {
			verificationID = response.VerificationId
			logrus.
				WithField("verification_id", verificationID).
				WithField("metadata", response.Metadata).
				Debug("Verification started")
		}

		if len(response.Command) > 0 {
			fmt.Print(color.CyanString(fmt.Sprintf("••• %s ➜ ", terminalPath)) + response.Command)
		}
		if len(response.Stdout) > 0 {
			fmt.Print(response.Stdout)
		}
		if len(response.Stderr) > 0 {
			_, _ = fmt.Fprint(os.Stderr, response.Stderr)
		}
		// todo - support stderr and commands

		if !h.solutionHintDisplayed && response.Finished && !response.Successful && response.SolutionAvailable {
			fmt.Println(color.HiYellowString("\nFeeling stuck? Don't give up! If you want to check the solution, you can now do it on the website."))
			h.solutionHintDisplayed = true
		}
	}

	if !finished {
		return false, errors.New("execution didn't finish")
	} else {
		return successful, nil
	}
}

func (h *Handlers) generateRunTerminalPath(trainingRootFs *afero.BasePathFs) string {
	exerciseConfig := h.config.ExerciseConfig(trainingRootFs)

	wd, err := syscall.Getwd()
	if err != nil {
		logrus.WithError(err).Warn("Can't get wd")
		return "???"
	}

	exerciseDir, err := trainingRootFs.RealPath(exerciseConfig.Directory)
	if err != nil {
		logrus.WithError(err).Warn("Can't get exercise real path")
		return "???"
	}

	terminalPath, err := filepath.Rel(wd, exerciseDir)
	if err != nil {
		logrus.WithError(err).Warn("Can't get relative exercise path")
		return wd
	}

	if terminalPath == exerciseConfig.Directory {
		terminalPath = "./" + terminalPath
	}

	return terminalPath
}
