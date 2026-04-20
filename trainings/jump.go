package trainings

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/manifoldco/promptui"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/ThreeDotsLabs/cli/trainings/git"
)

func (h *Handlers) SelectExercise(ctx context.Context) (string, error) {
	ctx = withSubAction(ctx, "jump")

	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return "", nil
	}
	trainingRootFs := newTrainingRootFs(trainingRoot)
	printGitNotices(h.config.TrainingConfig(trainingRootFs))

	currentExerciseID := h.config.ExerciseConfig(trainingRootFs).ExerciseID

	resp, err := h.newGrpcClient().GetExercises(ctx, &genproto.GetExercisesRequest{
		TrainingName: h.config.TrainingConfig(trainingRootFs).TrainingName,
		Token:        h.config.GlobalConfig().Token,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get solution files: %w", err)
	}

	if len(resp.Modules) == 0 {
		return "", nil
	}

	resp.Modules = append([]*genproto.GetExercisesResponse_Module{
		{
			Id:   "",
			Name: "(exit)",
		},
	}, resp.Modules...)

	for i := range resp.Modules {
		for j := range resp.Modules[i].Exercises {
			if resp.Modules[i].Exercises[j].Id == currentExerciseID {
				resp.Modules[i].Exercises[j].Name += " (current)"
			} else if resp.Modules[i].Exercises[j].IsSkipped {
				resp.Modules[i].Exercises[j].Name += " (skipped)"
			}
		}

		resp.Modules[i].Exercises = append([]*genproto.GetExercisesResponse_Exercise{
			{
				Id:   "",
				Name: "(back)",
			},
		}, resp.Modules[i].Exercises...)
	}

	moduleCursorPos := 0
	exerciseCursorPos := 0

	for i, module := range resp.Modules {
		for j, exercise := range module.Exercises {
			if exercise.Id == currentExerciseID {
				moduleCursorPos = i
				exerciseCursorPos = j
				break
			}
		}
	}

	for {
		moduleSelect := promptui.Select{
			Label:     "Choose module:",
			Items:     resp.Modules,
			Size:      len(resp.Modules),
			CursorPos: moduleCursorPos,
			Templates: &promptui.SelectTemplates{
				Label:    "{{ .Name }}",
				Active:   "{{ .Name | cyan }}",
				Inactive: "{{ .Name }}",
			},
			HideSelected: true,
		}

		index, _, err := moduleSelect.Run()
		if err != nil {
			return "", err
		}

		if index == 0 {
			return "", nil
		}

		if moduleCursorPos != index {
			moduleCursorPos = index
			exerciseCursorPos = 0
		}

		module := resp.Modules[index]

		exerciseSelect := promptui.Select{
			Label:     "Choose exercise:",
			Items:     module.Exercises,
			Size:      len(module.Exercises),
			CursorPos: exerciseCursorPos,
			Templates: &promptui.SelectTemplates{
				Label:    "{{ .Name }}",
				Active:   "{{ .Name | cyan }}",
				Inactive: "{{ .Name }}",
			},
			HideSelected: true,
		}

		index, _, err = exerciseSelect.Run()
		if err != nil {
			return "", err
		}

		if index == 0 {
			continue
		} else {
			fmt.Printf("Selected exercise: %v/%v\n", module.Name, module.Exercises[index].Name)
			return module.Exercises[index].Id, nil
		}
	}
}

func (h *Handlers) FindExercise(ctx context.Context, exerciseID string) (string, error) {
	ctx = withSubAction(ctx, "jump")

	exerciseID = strings.TrimSpace(exerciseID)

	_, err := uuid.Parse(exerciseID)
	if err == nil {
		return exerciseID, nil
	}

	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return "", nil
	}
	trainingRootFs := newTrainingRootFs(trainingRoot)

	currentExerciseID := h.config.ExerciseConfig(trainingRootFs).ExerciseID

	resp, err := h.newGrpcClient().GetExercises(ctx, &genproto.GetExercisesRequest{
		TrainingName: h.config.TrainingConfig(trainingRootFs).TrainingName,
		Token:        h.config.GlobalConfig().Token,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get solution files: %w", err)
	}

	if exerciseID == "latest" {
		lastModule := resp.Modules[len(resp.Modules)-1]
		lastExercise := lastModule.Exercises[len(lastModule.Exercises)-1]
		fmt.Printf("Selected exercise: %v/%v\n", lastModule.Name, lastExercise.Name)
		return lastExercise.Id, nil
	}

	targetModule := 0
	targetExercise := 0

	if strings.Contains(exerciseID, "/") {
		parts := strings.Split(exerciseID, "/")
		targetModule = numberFromName(parts[0])
		targetExercise = numberFromName(parts[1])
	} else {
		targetExercise = numberFromName(exerciseID)
	}

	if targetModule == 0 {
		for _, module := range resp.Modules {
			for _, exercise := range module.Exercises {
				if exercise.Id == currentExerciseID {
					targetModule = numberFromName(module.Name)
					break
				}
			}
		}
	}

	for _, module := range resp.Modules {
		if numberFromName(module.Name) == targetModule {
			for _, exercise := range module.Exercises {
				if numberFromName(exercise.Name) == targetExercise {
					fmt.Printf("Selected exercise: %v/%v\n", module.Name, exercise.Name)
					return exercise.Id, nil
				}
			}
		}
	}

	return "", fmt.Errorf("exercise not found")
}

func numberFromName(name string) int {
	parts := strings.Split(name, "-")
	return parseNumber(parts[0])
}

func parseNumber(number string) int {
	num, _ := strconv.Atoi(strings.TrimPrefix(number, "0"))
	return num
}

func (h *Handlers) Jump(ctx context.Context, exerciseID string) error {
	ctx = withSubAction(ctx, "jump")

	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return nil
	}
	trainingRootFs := newTrainingRootFs(trainingRoot)
	printGitNotices(h.config.TrainingConfig(trainingRootFs))

	// Save progress before jumping
	gitOps := h.newGitOps()
	if gitOps.Enabled() {
		exerciseCfg := h.config.ExerciseConfig(trainingRootFs)
		if !exerciseCfg.IsTextOnly && exerciseCfg.Directory != "" {
			saveProgress(gitOps, exerciseCfg.Directory, fmt.Sprintf("save progress on %s", exerciseCfg.ModuleExercisePath()))
		}
	}

	resp, err := h.newGrpcClient().GetExercise(ctx, &genproto.GetExerciseRequest{
		TrainingName: h.config.TrainingConfig(trainingRootFs).TrainingName,
		Token:        h.config.GlobalConfig().Token,
		ExerciseId:   exerciseID,
	})
	if err != nil {
		return fmt.Errorf("failed to get exercise: %w", err)
	}

	// Revisit prompt: if this exercise was visited before, ask what to do
	if gitOps.Enabled() && !resp.IsTextOnly && internal.IsStdinTerminal() {
		moduleExercisePath := moduleExercisePathFromResponse(resp)
		initBranch := git.InitBranchName(moduleExercisePath)

		if gitOps.BranchExists(initBranch) {
			// Check if a successful solution exists
			var successfulVerificationId string
			solResp, solErr := h.newGrpcClient().GetSolutions(ctx, &genproto.GetSolutionsRequest{
				ExerciseId: exerciseID,
				Token:      h.config.GlobalConfig().Token,
			})
			if solErr == nil {
				for _, sol := range solResp.Solutions {
					if sol.Successful {
						successfulVerificationId = sol.VerificationId
						break
					}
				}
			}

			backupBranch := git.BackupBranchName(moduleExercisePath)

			// Build prompt
			fmt.Println()
			fmt.Println(color.YellowString("  You've worked on this exercise before."))
			fmt.Println()

			actions := internal.Actions{
				{Shortcut: '\n', Action: "continue where you left off", ShortcutAliases: []rune{'\r'}},
				{Shortcut: 'r', Action: "reset to original exercise files"},
			}

			fmt.Printf("  %s  Continue where you left off (your files stay as they are)\n", color.New(color.Bold).Sprint("ENTER"))
			fmt.Printf("  %s   Reset to the original exercise files and solve it again from scratch\n", color.New(color.Bold).Sprint("   r"))

			if successfulVerificationId != "" {
				actions = append(actions, internal.Action{Shortcut: 's', Action: "restore your last successful solution"})
				fmt.Printf("  %s   Restore your last successful solution for this exercise\n", color.New(color.Bold).Sprint("   s"))
			}

			fmt.Println()
			if successfulVerificationId != "" {
				fmt.Println(color.YellowString("  Both r and s roll exercise files back to an earlier state."))
			} else {
				fmt.Println(color.YellowString("  r rolls exercise files back to an earlier state."))
			}
			fmt.Printf("  %s\n", color.YellowString("Code added by later exercises will be removed, but is saved in branch"))
			fmt.Printf("  %s%s\n", color.MagentaString(backupBranch), color.YellowString("."))
			fmt.Println()

			choice := internal.Prompt(actions, os.Stdin, os.Stdout)

			switch choice {
			case '\n':
				// Continue — no file ops, just update config pointer
			case 'r':
				if _, err := h.resetCleanFiles(ctx, gitOps, trainingRootFs, resp.ExerciseId, moduleExercisePath, resp.Dir); err != nil {
					if errors.Is(err, errBackupAborted) {
						return nil // user aborted — clean exit
					}
					return err
				}
				// resetCleanFiles already fetched fresh scaffold+golden and wrote the start state.
			case 's':
				if err := h.checkoutSolution(ctx, gitOps, successfulVerificationId, moduleExercisePath, trainingRootFs); err != nil {
					if errors.Is(err, errBackupAborted) {
						return nil // user aborted — clean exit
					}
					return err
				}
			}
		}
	}

	_, err = h.setExercise(ctx, trainingRootFs, resp, trainingRoot, false)
	return err
}

func (h *Handlers) checkoutSolution(
	ctx context.Context,
	gitOps *git.Ops,
	verificationId string,
	moduleExercisePath string,
	trainingRootFs *afero.BasePathFs,
) error {
	// 1. Fetch solution files from server
	solResp, err := h.newGrpcClient().GetSolutionFiles(ctx, &genproto.GetSolutionFilesRequest{
		ExecutionId: verificationId,
	})
	if err != nil {
		return fmt.Errorf("failed to get solution files: %w", err)
	}

	// 2. Save user's work to backup branch before overwriting files.
	backupBranch := git.BackupBranchName(moduleExercisePath)
	if err := saveToBackupBranch(gitOps, backupBranch); err != nil {
		return err
	}

	oldHead, _ := gitOps.RevParse("HEAD")

	// 3. Write solution files, stage, commit
	if err := files.NewFilesSilent().WriteExerciseFiles(solResp.FilesToCreate, trainingRootFs, solResp.Dir); err != nil {
		return fmt.Errorf("failed to write solution files: %w", err)
	}

	_ = gitOps.ResetStaging()
	if err := gitOps.AddAll(solResp.Dir); err != nil {
		fmt.Println(formatGitWarning("Could not stage solution files", err))
	}
	if gitOps.HasStagedChanges() {
		if err := gitOps.Commit(fmt.Sprintf("restore solution for %s", moduleExercisePath)); err != nil {
			fmt.Println(formatGitWarning("Could not commit restored solution", err))
		}
	}

	// 4. Show diff stat and backup info
	if oldHead != "" {
		if stat, err := gitOps.DiffStatPath(oldHead, "HEAD", solResp.Dir); err == nil && stat != "" {
			fmt.Println(stat)
		}
	}

	fmt.Println()
	fmt.Println(color.GreenString("  Last successful solution restored."))
	fmt.Printf("  Your code was saved to branch %s\n", color.MagentaString(backupBranch))
	fmt.Println("  Restore anytime with: " + color.CyanString("git checkout %s -- %s", backupBranch, solResp.Dir))
	fmt.Println()

	return nil
}
