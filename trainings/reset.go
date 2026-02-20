package trainings

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

func (h *Handlers) Reset(ctx context.Context) error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if err != nil {
		return err
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)
	printGitNotices(h.config.TrainingConfig(trainingRootFs))

	// Warn about uncommitted changes before reset
	gitOps := h.newGitOps()
	if gitOps.Enabled() {
		exerciseCfg := h.config.ExerciseConfig(trainingRootFs)

		if !exerciseCfg.IsTextOnly && exerciseCfg.Directory != "" {
			if gitOps.HasUncommittedChanges(exerciseCfg.Directory) {
				fmt.Println(color.YellowString("  You have uncommitted changes in %s.", exerciseCfg.Directory))
				fmt.Println(color.YellowString("  Reset will overwrite these files via git merge."))

				// Save progress before reset
				if err := gitOps.AddAll(exerciseCfg.Directory); err == nil && gitOps.HasStagedChanges() {
					if err := gitOps.Commit(fmt.Sprintf("save progress on %s", exerciseCfg.ModuleExercisePath())); err != nil {
						logrus.WithError(err).Warn("Could not commit progress before reset")
					}
				}
			}
		}
	}

	_, err = h.nextExercise(ctx, "", trainingRoot)
	if err != nil {
		return err
	}
	return nil
}
