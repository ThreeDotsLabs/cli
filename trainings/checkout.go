package trainings

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/manifoldco/promptui"
	"github.com/mergestat/timediff"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *Handlers) Checkout(ctx context.Context) error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return nil
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)

	resp, err := h.newGrpcClient().GetSolutions(ctx, &genproto.GetSolutionsRequest{
		ExerciseId: h.config.ExerciseConfig(trainingRootFs).ExerciseID,
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

	selectUI := promptui.Select{
		Label: "Select a solution to checkout",
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

	if err := h.writeExerciseFiles(files.NewFilesWithConfig(true, true), getSolutionFilesToExerciseSolution(getResp), trainingRootFs); err != nil {
		return err
	}

	err = addModuleToWorkspace(trainingRoot, getResp.Dir)
	if err != nil {
		logrus.WithError(err).Warn("Failed to add module to workspace")
	}

	return nil
}
