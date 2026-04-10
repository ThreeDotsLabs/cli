package trainings

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/ThreeDotsLabs/cli/trainings/git"
)

func (h *Handlers) Reset(ctx context.Context) error {
	ctx = withSubAction(ctx, "reset")

	trainingRoot, err := h.config.FindTrainingRoot()
	if err != nil {
		return err
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)
	trainingConfig := h.config.TrainingConfig(trainingRootFs)
	printGitNotices(trainingConfig)

	exerciseCfg := h.config.ExerciseConfig(trainingRootFs)
	gitOps := h.newGitOps()

	if !gitOps.Enabled() || exerciseCfg.IsTextOnly || exerciseCfg.Directory == "" {
		// No git or text-only: fall back to re-downloading from server
		_, err = h.nextExercise(ctx, "", trainingRoot)
		return err
	}

	// Git-enabled reset: use existing init branch directly
	moduleExercisePath := exerciseCfg.ModuleExercisePath()
	initBranch := git.InitBranchName(moduleExercisePath)

	if !gitOps.BranchExists(initBranch) {
		// No init branch available — fall back to server re-download
		_, err = h.nextExercise(ctx, "", trainingRoot)
		return err
	}

	// Auto-commit uncommitted changes
	if gitOps.HasUncommittedChanges(exerciseCfg.Directory) {
		saveProgress(gitOps, exerciseCfg.Directory, fmt.Sprintf("save progress on %s", moduleExercisePath))
	}

	// Choose reset mode
	resetMode := 0 // default: clean files
	if internal.IsStdinTerminal() {
		fmt.Println()
		fmt.Println(color.YellowString("  Your exercise files will be restored to their original state."))
		fmt.Println()

		boxLines := []string{
			"💡 You also have full git history in this worktree: saved progress,",
			"   backup branches, and example solutions. Browse: " + color.CyanString("git log --oneline --all"),
		}
		if graph, err := gitOps.LogGraph(6); err == nil && graph != "" {
			boxLines = append(boxLines, "")
			for _, line := range strings.Split(graph, "\n") {
				if line != "" {
					boxLines = append(boxLines, line)
				}
			}
		}
		printColorBox(boxLines...)
		fmt.Println()
		fmt.Println(color.HiBlackString(strings.Repeat("─", internal.TerminalWidth())))
		fmt.Println()

		selectUI := promptui.Select{
			Label: "Choose reset mode",
			Items: []string{
				"Get clean exercise files (your code is saved to a backup branch)",
				"Restore deleted files only (your modifications are kept)",
				"(cancel)",
			},
			Templates: &promptui.SelectTemplates{
				Label:    "{{ . }}",
				Active:   "{{ . | cyan }}",
				Inactive: "{{ . }}",
			},
			HideSelected: true,
		}

		resetMode, _, err = selectUI.Run()
		if err != nil {
			return err
		}
	}

	switch resetMode {
	case 0:
		if err := h.resetCleanFiles(gitOps, initBranch, moduleExercisePath, exerciseCfg.Directory); err != nil {
			return err
		}
	case 1:
		if err := h.resetMissingOnly(gitOps, initBranch, moduleExercisePath, exerciseCfg.Directory, trainingRoot); err != nil {
			return err
		}
	case 2:
		fmt.Println("Cancelled")
		return nil
	}

	// exercise.md is gitignored, so checkout from init branch won't restore it.
	// Fetch from server and write directly.
	scaffoldResp, err := h.newGrpcClient().GetExercise(ctx, &genproto.GetExerciseRequest{
		TrainingName: trainingConfig.TrainingName,
		Token:        h.config.GlobalConfig().Token,
		ExerciseId:   exerciseCfg.ExerciseID,
	})
	if err != nil {
		logrus.WithError(err).Warn("Could not fetch exercise scaffold for exercise.md")
	} else {
		writeExerciseMd(scaffoldResp.FilesToCreate, trainingRootFs, exerciseCfg.Directory)
	}

	return nil
}

func (h *Handlers) resetCleanFiles(gitOps *git.Ops, initBranch, moduleExercisePath, exerciseDir string) error {
	// Save user's work to backup branch
	backupBranch := git.BackupBranchName(moduleExercisePath)
	if err := gitOps.CreateBranchFromHead(backupBranch); err != nil {
		fmt.Println(formatGitWarning("Could not save your code to a backup branch", err))
	} else {
		gitOps.PrintInfo(fmt.Sprintf("git branch %s", backupBranch))
	}

	oldHead, _ := gitOps.RevParse("HEAD")

	// Restore all exercise files from init branch
	if err := gitOps.CheckoutFiles(initBranch, exerciseDir); err != nil {
		return fmt.Errorf("could not restore exercise files: %w", err)
	}

	_ = gitOps.ResetStaging()
	if err := gitOps.AddAll(exerciseDir); err != nil {
		fmt.Println(formatGitWarning("Could not stage restored files", err))
	}
	if gitOps.HasStagedChanges() {
		if err := gitOps.Commit(fmt.Sprintf("reset exercise %s", moduleExercisePath)); err != nil {
			fmt.Println(formatGitWarning("Could not commit reset", err))
		}
	}

	if oldHead != "" {
		if stat, err := gitOps.DiffStatPath(oldHead, "HEAD", exerciseDir); err == nil && stat != "" {
			fmt.Println(stat)
		}
	}

	fmt.Println()
	fmt.Println(color.GreenString("  Exercise files restored to their original state."))
	fmt.Printf("  Your code was saved to branch %s\n", color.MagentaString(backupBranch))
	fmt.Println("  Restore anytime with: " + color.CyanString("git checkout %s -- %s", backupBranch, exerciseDir))
	fmt.Println()

	return nil
}

func (h *Handlers) resetMissingOnly(gitOps *git.Ops, initBranch, moduleExercisePath, exerciseDir, trainingRoot string) error {
	// List files on init branch
	initFiles, err := gitOps.ListFiles(initBranch, exerciseDir)
	if err != nil {
		return fmt.Errorf("could not list init branch files: %w", err)
	}

	// Find missing files
	var missingFiles []string
	for _, f := range initFiles {
		fullPath := filepath.Join(trainingRoot, f)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			missingFiles = append(missingFiles, f)
		}
	}

	if len(missingFiles) == 0 {
		fmt.Println("  All exercise files are present, nothing to restore.")
		return nil
	}

	oldHead, _ := gitOps.RevParse("HEAD")

	// Restore each missing file from init branch
	for _, f := range missingFiles {
		if err := gitOps.CheckoutPathFrom(initBranch, f); err != nil {
			fmt.Println(formatGitWarning(fmt.Sprintf("Could not restore %s", f), err))
			continue
		}
		fmt.Printf("  %s %s\n", color.GreenString("+"), f)
	}

	_ = gitOps.ResetStaging()
	if err := gitOps.AddAll(exerciseDir); err != nil {
		fmt.Println(formatGitWarning("Could not stage restored files", err))
	}
	if gitOps.HasStagedChanges() {
		if err := gitOps.Commit(fmt.Sprintf("restore missing files for %s", moduleExercisePath)); err != nil {
			fmt.Println(formatGitWarning("Could not commit restored files", err))
		}
	}

	if oldHead != "" {
		if stat, err := gitOps.DiffStatPath(oldHead, "HEAD", exerciseDir); err == nil && stat != "" {
			fmt.Println(stat)
		}
	}

	fmt.Println()
	return nil
}
