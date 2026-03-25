package trainings

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/ThreeDotsLabs/cli/trainings/git"
)

// isValidCompletedAt returns true when the timestamp is a real completion date
// (not a zero/epoch value from unset protobuf fields).
func isValidCompletedAt(t time.Time) bool {
	return !t.IsZero() && t.Year() >= 2000
}

// restore restores all solution files for the training in the given directory.
// When git is enabled, each exercise gets the full git structure (init branch,
// merge commit, solution commit with date, example solution branch) matching the normal flow.
func (h *Handlers) restore(ctx context.Context, trainingRoot string, gitOps *git.Ops) ([]string, error) {
	ctx = withSubAction(ctx, "restore")

	trainingRootFs := newTrainingRootFs(trainingRoot)
	trainingName := h.config.TrainingConfig(trainingRootFs).TrainingName

	resp, err := h.newGrpcClient().GetAllSolutionFiles(ctx, &genproto.GetAllSolutionFilesRequest{
		TrainingName:     trainingName,
		Token:            h.config.GlobalConfig().Token,
		SendAllExercises: true,
	}, grpc.MaxCallRecvMsgSize(50*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to get all solution files: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"solutions": len(resp.Solutions),
	}).Debug("Received solutions from server")

	// Create the initialize commit with a date before all exercise commits.
	// Files were staged but not committed by Init() so restore can set the date.
	if gitOps.Enabled() && gitOps.HasStagedChanges() {
		initMsg := fmt.Sprintf("initialize %s", trainingName)
		quietOps := git.NewQuietOps(trainingRoot)
		if earliest := earliestSolutionDate(resp.Solutions); !earliest.IsZero() {
			if err := quietOps.CommitWithDate(initMsg, earliest.Add(-10*time.Second)); err != nil {
				logrus.WithError(err).Warn("Could not create initialize commit")
			}
		} else {
			if err := quietOps.Commit(initMsg); err != nil {
				logrus.WithError(err).Warn("Could not create initialize commit")
			}
		}
	}

	var exerciseIDs []string
	var prevModuleExercisePath string
	total := len(resp.Solutions)

	bar := progressbar.NewOptions(total,
		progressbar.OptionSetDescription(
			color.New(color.Bold, color.FgYellow).Sprint("Restoring exercises"),
		),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(30),
		progressbar.OptionClearOnFinish(),
	)

	for _, solution := range resp.Solutions {
		moduleExercisePath := solution.Exercise.Module.Name + "/" + solution.Exercise.Name

		if !solution.IsTextOnly && gitOps.Enabled() {
			if err := h.restoreExerciseWithGit(
				ctx, trainingRootFs, gitOps, trainingRoot,
				solution, moduleExercisePath, prevModuleExercisePath,
			); err != nil {
				return nil, UserFacingError{
					Msg: fmt.Sprintf("Restore failed at exercise %s.", moduleExercisePath),
					SolutionHint: fmt.Sprintf(
						"Some exercises were restored, but the process stopped.\n\n%s",
						recoveryHint(trainingName),
					),
				}
			}
			prevModuleExercisePath = moduleExercisePath
		} else {
			// Text-only or no-git fallback: write files directly
			f := files.NewFilesSilent()
			if err := h.writeExerciseFiles(f, solution, trainingRootFs); err != nil {
				return nil, err
			}
			if err := addModuleToWorkspaceQuiet(trainingRoot, solution.Dir, true); err != nil {
				logrus.WithError(err).Warn("Failed to add module to workspace")
			}
		}

		exerciseIDs = append(exerciseIDs, solution.ExerciseId)
		bar.Add(1)
	}
	fmt.Fprintln(os.Stderr) // newline after progress

	return exerciseIDs, nil
}

// earliestSolutionDate returns the earliest valid CompletedAt or SubmittedAt
// across all solutions. Returns zero time if none are valid.
func earliestSolutionDate(solutions []*genproto.ExerciseSolution) time.Time {
	var earliest time.Time
	for _, s := range solutions {
		for _, t := range []time.Time{s.CompletedAt.AsTime(), s.SubmittedAt.AsTime()} {
			if isValidCompletedAt(t) && (earliest.IsZero() || t.Before(earliest)) {
				earliest = t
			}
		}
	}
	return earliest
}

// restoreExerciseWithGit produces the same git structure as the normal exercise flow:
// init branch → merge → solution commit (with date) → example solution branch.
func (h *Handlers) restoreExerciseWithGit(
	ctx context.Context,
	trainingRootFs *afero.BasePathFs,
	gitOps *git.Ops,
	trainingRoot string,
	solution *genproto.ExerciseSolution,
	moduleExercisePath string,
	prevModuleExercisePath string,
) error {
	// Compute date offsets so git log shows a natural chronology per exercise.
	completedAt := solution.CompletedAt.AsTime()
	submittedAt := solution.SubmittedAt.AsTime()
	completed := isValidCompletedAt(completedAt)

	var initDate, mergeDate, solutionDate, goldenDate time.Time
	if completed {
		solutionDate = completedAt
		initDate = completedAt.Add(-2 * time.Second)
		mergeDate = completedAt.Add(-1 * time.Second)
		goldenDate = completedAt.Add(1 * time.Second)
	} else if isValidCompletedAt(submittedAt) {
		solutionDate = submittedAt
		initDate = submittedAt.Add(-2 * time.Second)
		mergeDate = submittedAt.Add(-1 * time.Second)
	}

	// Use quiet ops so restore only shows the progress counter, not per-command output.
	quietOps := git.NewQuietOps(trainingRoot)

	// 1. Fetch scaffold files via GetExercise
	scaffoldResp, err := h.newGrpcClient().GetExercise(
		ctx,
		&genproto.GetExerciseRequest{
			TrainingName: h.config.TrainingConfig(trainingRootFs).TrainingName,
			Token:        h.config.GlobalConfig().Token,
			ExerciseId:   solution.ExerciseId,
		},
	)
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
			logrus.WithField("exercise", moduleExercisePath).Debug("Exercise scaffold no longer available, skipping git structure")
			scaffoldResp = nil
		} else {
			return fmt.Errorf("failed to get exercise scaffold: %w", err)
		}
	}

	// 2-3. Create init branch and merge — only when scaffold is available.
	if scaffoldResp != nil {
		// 2. Create init branch (shared function, quiet mode, with date)
		initBranch, err := createInitBranch(
			quietOps, solution.Dir, moduleExercisePath, prevModuleExercisePath,
			scaffoldResp.FilesToCreate, scaffoldResp.IsTextOnly, true, initDate,
		)
		if err != nil {
			return fmt.Errorf("failed to create init branch: %w", err)
		}

		// 3. Merge init branch into main.
		// Project exercises may conflict: init branch N+1 is based on init branch N (scaffold),
		// but main has exercise N's solution. Use MergeAutoResolve because step 4 overwrites
		// all files with the user's saved solution — the merge content is ephemeral.
		mergeMsg := fmt.Sprintf("start %s", moduleExercisePath)
		if !mergeDate.IsZero() {
			if err := quietOps.MergeAutoResolveWithDate(initBranch, mergeMsg, mergeDate); err != nil {
				return fmt.Errorf("failed to merge init branch: %w", err)
			}
		} else {
			if err := quietOps.MergeAutoResolve(initBranch, mergeMsg); err != nil {
				return fmt.Errorf("failed to merge init branch: %w", err)
			}
		}
	}

	// 4. Write user's solution files
	f := files.NewFilesSilent()
	if err := f.WriteExerciseFiles(solution.Files, trainingRootFs, solution.Dir); err != nil {
		return fmt.Errorf("failed to write solution files: %w", err)
	}

	// 5. Commit completed — with original date if available
	if err := quietOps.AddAll(solution.Dir); err != nil {
		return fmt.Errorf("failed to stage solution: %w", err)
	}

	// Also stage go.work if it exists (workspace may have been updated by merge)
	if hasGoWorkspace(trainingRoot) {
		_ = quietOps.AddFiles("go.work")
	}

	if quietOps.HasStagedChanges() {
		var commitMsg string
		if completed {
			commitMsg = fmt.Sprintf("completed %s", moduleExercisePath)
		} else {
			commitMsg = fmt.Sprintf("last submitted solution for %s", moduleExercisePath)
		}

		if !solutionDate.IsZero() {
			if err := quietOps.CommitWithDate(commitMsg, solutionDate); err != nil {
				return fmt.Errorf("failed to commit solution: %w", err)
			}
		} else {
			if err := quietOps.Commit(commitMsg); err != nil {
				return fmt.Errorf("failed to commit solution: %w", err)
			}
		}
	}

	// 6. Create example solution branch (silent) — only for completed exercises with available scaffold
	if scaffoldResp != nil && completed {
		exerciseCfg := config.ExerciseConfig{
			ExerciseID:   solution.ExerciseId,
			Directory:    solution.Dir,
			IsTextOnly:   solution.IsTextOnly,
			ModuleName:   solution.Exercise.Module.Name,
			ExerciseName: solution.Exercise.Name,
		}
		h.syncGoldenSolutionQuiet(ctx, trainingRootFs, quietOps, exerciseCfg, goldenDate)
	}

	// 7. Add module to workspace
	if err := addModuleToWorkspaceQuiet(trainingRoot, solution.Dir, true); err != nil {
		logrus.WithError(err).Warn("Failed to add module to workspace")
	}

	return nil
}
