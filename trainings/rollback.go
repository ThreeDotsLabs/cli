package trainings

import (
	"context"
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/mergestat/timediff"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func (h *Handlers) Rollback(ctx context.Context) error {
	ctx = withSubAction(ctx, "rollback")

	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return nil
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)
	printGitNotices(h.config.TrainingConfig(trainingRootFs))

	exerciseCfg := h.config.ExerciseConfig(trainingRootFs)

	resp, err := h.newGrpcClient().GetSolutions(ctx, &genproto.GetSolutionsRequest{
		ExerciseId: exerciseCfg.ExerciseID,
		Token:      h.config.GlobalConfig().Token,
	})
	if err != nil {
		return fmt.Errorf("failed to get solutions: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"resp": resp,
		"err":  err,
	}).Debug("Received solutions from server")

	items := []string{"(cancel)"}
	for _, solution := range resp.Solutions {
		text := ""
		if solution.Successful {
			text += "✅"
		} else {
			text += "❌"
		}
		text += " "
		text += solution.VerificationId
		text += " "
		text += timediff.TimeDiff(solution.ExecutedAt.AsTime())

		items = append(items, text)
	}

	gitOps := h.newGitOps()

	if gitOps.Enabled() {
		fmt.Println()
		printDimBox(
			"💡 All your past successful solutions are also saved in git history.",
			fmt.Sprintf("   Browse with: git log -- %s  (or in your IDE)", exerciseCfg.ModuleExercisePath()),
		)
		fmt.Println()
	}

	selectUI := promptui.Select{
		Label: "Select a solution to rollback to",
		Items: items,
		Size:  10,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "{{ . | cyan }}",
			Inactive: "{{ . }}",
		},
		HideSelected: true,
	}

	index, _, err := selectUI.Run()
	if err != nil {
		return err
	}

	if index == 0 {
		fmt.Println("Cancelled")
		return nil
	}

	getResp, err := h.newGrpcClient().GetSolutionFiles(ctx, &genproto.GetSolutionFilesRequest{
		ExecutionId: resp.Solutions[index-1].VerificationId,
	})
	if err != nil {
		return fmt.Errorf("failed to get solution files: %w", err)
	}

	if gitOps.Enabled() && getResp.Dir != "" {
		// Save uncommitted changes before overwriting
		if gitOps.HasUncommittedChanges(getResp.Dir) {
			saveProgress(gitOps, getResp.Dir, fmt.Sprintf("save progress before rollback for %s", getResp.Dir))
		}

		// Write files directly (skip interactive prompts — user explicitly chose this solution)
		if err := files.NewFilesSilent().WriteExerciseFiles(getResp.FilesToCreate, trainingRootFs, getResp.Dir); err != nil {
			return err
		}

		// Show what changed
		if stat, err := gitOps.DiffStatWorkingTree(getResp.Dir); err == nil && stat != "" {
			fmt.Println(stat)
		}

		// Leave files as unstaged changes for user to review
		fmt.Println("  Solution restored. Review with " + fmt.Sprintf("`git diff %s`", getResp.Dir))

	} else {
		// No git — use existing interactive file writer
		if err := h.writeExerciseFiles(files.NewFilesWithConfig(true, true), getSolutionFilesToExerciseSolution(getResp), trainingRootFs); err != nil {
			return err
		}
	}

	err = addModuleToWorkspace(trainingRoot, getResp.Dir)
	if err != nil {
		logrus.WithError(err).Warn("Failed to add module to workspace")
	}

	return nil
}
