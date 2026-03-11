package trainings

import (
	"context"
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/ThreeDotsLabs/cli/trainings/git"
)

func (h *Handlers) Clone(ctx context.Context, executionID string, directory string) error {
	ctx = withSubAction(ctx, "clone")

	resp, err := h.newGrpcClient().GetSolutionFiles(ctx, &genproto.GetSolutionFilesRequest{
		ExecutionId: executionID,
	})
	if err != nil {
		return fmt.Errorf("failed to get solution files: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"resp": resp,
		"err":  err,
	}).Debug("Received exercise from server")

	absoluteDirToClone, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}

	absoluteDirToClone = path.Join(absoluteDirToClone, directory)

	if _, _, _, err := h.startTraining(ctx, resp.TrainingName, absoluteDirToClone); err != nil {
		return err
	}

	trainingRootFs := afero.NewBasePathFs(afero.NewOsFs(), absoluteDirToClone).(*afero.BasePathFs)

	if err := h.config.WriteTrainingConfig(config.TrainingConfig{TrainingName: resp.TrainingName}, trainingRootFs); err != nil {
		return errors.Wrap(err, "can't write training config")
	}

	if err := h.writeExerciseFiles(files.NewFiles(), getSolutionFilesToExerciseSolution(resp), trainingRootFs); err != nil {
		return err
	}

	err = addModuleToWorkspace(absoluteDirToClone, resp.Dir)
	if err != nil {
		logrus.WithError(err).Warn("Failed to add module to workspace")
	}

	// Initialize git repo (mirrors init.go git setup)
	cfg := h.config.TrainingConfig(trainingRootFs)
	if !cfg.GitConfigured {
		gitOps := git.NewOps(absoluteDirToClone, false)
		_, versionErr := git.CheckVersion()
		if versionErr != nil {
			var notInstalled *git.GitNotInstalledError
			var tooOld *git.GitTooOldError
			if errors.As(versionErr, &notInstalled) || errors.As(versionErr, &tooOld) {
				logrus.WithError(versionErr).Debug("Git unavailable for clone")
				_ = git.InstallHint(runtime.GOOS) // consume for logging
				cfg.GitConfigured = true
				cfg.GitEnabled = false
				cfg.GitUnavailable = true
				if err := h.config.WriteTrainingConfig(cfg, trainingRootFs); err != nil {
					return errors.Wrap(err, "can't update training config")
				}
				return nil
			}
			logrus.WithError(versionErr).Warn("Could not verify git version")
		}

		if _, err := gitOps.Init(); err != nil {
			logrus.WithError(err).Warn("Could not initialize git repository for clone")
			return nil
		}

		gitDefaultConfig(&cfg)
		if err := h.config.WriteTrainingConfig(cfg, trainingRootFs); err != nil {
			return errors.Wrap(err, "can't update training config with git preferences")
		}

		var extraFiles []string
		if resp.Dir != "" {
			extraFiles = append(extraFiles, resp.Dir)
		}
		stageInitialFiles(gitOps, absoluteDirToClone, extraFiles...)
		if gitOps.HasStagedChanges() {
			if err := gitOps.Commit(fmt.Sprintf("clone solution for %s", resp.TrainingName)); err != nil {
				fmt.Println(formatGitWarning("Could not create initial git commit", err))
			}
		}
	}

	return nil
}

func getSolutionFilesToExerciseSolution(resp *genproto.GetSolutionFilesResponse) *genproto.ExerciseSolution {
	return &genproto.ExerciseSolution{
		ExerciseId: resp.ExerciseId,
		Dir:        resp.Dir,
		Files:      resp.FilesToCreate,
		IsTextOnly: false,
		IsOptional: false,
	}
}
