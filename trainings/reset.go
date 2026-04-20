package trainings

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/files"
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
		if _, err := h.resetCleanFiles(ctx, gitOps, trainingRootFs, exerciseCfg.ExerciseID, moduleExercisePath, exerciseCfg.Directory); err != nil {
			if errors.Is(err, errBackupAborted) {
				return nil // user chose to abort — already explained above
			}
			return err
		}
	case 1:
		if err := h.resetMissingOnly(ctx, gitOps, trainingRootFs, exerciseCfg.ExerciseID, moduleExercisePath, exerciseCfg.Directory, trainingRoot); err != nil {
			return err
		}
	case 2:
		fmt.Println("Cancelled")
		return nil
	}

	// Note: resetCleanFiles/resetMissingOnly already fetch fresh scaffold+golden from
	// the server and write the start state — no additional overlay step is needed.

	return nil
}

// resetCleanFiles replaces exerciseDir with its starting state
// (golden(prev) + scaffold(current)) — 1:1, deletes extras. Saves user's work
// to a timestamped backup branch first.
//
// INVARIANT: on success exerciseDir is 1:1 with the start state — no stale user
// files. Enforced by replaceExerciseFilesAndCommit → replaceExerciseFiles.
// Do not replace this with a CheckoutFiles(initBranch, ...) call: init branches
// in project-style trainings accumulate empty placeholder files from earlier
// scaffolds (see exercise_replace.go for the full story).
func (h *Handlers) resetCleanFiles(
	ctx context.Context,
	gitOps *git.Ops,
	fs *afero.BasePathFs,
	exerciseID, moduleExercisePath, exerciseDir string,
) (string, error) {
	startState, err := h.fetchStartStateFiles(ctx, fs, exerciseID)
	if err != nil {
		return "", err
	}

	backupBranch := git.BackupBranchName(moduleExercisePath)
	oldHead, _ := gitOps.RevParse("HEAD")

	if _, err := replaceExerciseFilesAndCommit(
		gitOps, fs, startState, exerciseDir, backupBranch,
		fmt.Sprintf("reset exercise %s", moduleExercisePath),
	); err != nil {
		return "", err
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

	return backupBranch, nil
}

// fetchStartStateFiles returns the starting state of the given exercise:
// golden(prev) with scaffold(current) overlaid on top. Makes up to 3
// gRPC calls (GetExercises for prev resolution, GetGoldenSolution for prev golden,
// GetExercise for current scaffold). Pass "" for exerciseID only in edge cases
// where the caller knows the current exercise has no meaningful prev.
func (h *Handlers) fetchStartStateFiles(
	ctx context.Context,
	fs *afero.BasePathFs,
	exerciseID string,
) ([]*genproto.File, error) {
	trainingName := h.config.TrainingConfig(fs).TrainingName
	token := h.config.GlobalConfig().Token

	prevExerciseID, _, err := h.resolvePreviousExercise(ctx, trainingName, token, exerciseID)
	if err != nil {
		return nil, fmt.Errorf("could not resolve previous exercise: %w", err)
	}

	scaffoldResp, err := h.newGrpcClient().GetExercise(ctx, &genproto.GetExerciseRequest{
		TrainingName: trainingName,
		Token:        token,
		ExerciseId:   exerciseID,
	})
	if err != nil {
		return nil, fmt.Errorf("could not fetch exercise scaffold: %w", err)
	}

	var goldenFiles []*genproto.File
	if prevExerciseID != "" {
		gf, err := h.fetchGoldenFiles(ctx, trainingName, prevExerciseID, token)
		if err != nil {
			return nil, err
		}
		goldenFiles = gf
	}

	return mergeStartStateFiles(goldenFiles, scaffoldResp.FilesToCreate), nil
}

// resetMissingOnly restores files from the exercise's start state
// (golden(prev) + scaffold(current)) that are missing on disk. Unlike
// resetCleanFiles, this keeps the user's modifications to existing files —
// it only fills gaps.
func (h *Handlers) resetMissingOnly(
	ctx context.Context,
	gitOps *git.Ops,
	fs *afero.BasePathFs,
	exerciseID, moduleExercisePath, exerciseDir, trainingRoot string,
) error {
	startState, err := h.fetchStartStateFiles(ctx, fs, exerciseID)
	if err != nil {
		return err
	}

	// Find files that exist in the start state but are missing on disk.
	var missingFiles []*genproto.File
	for _, f := range startState {
		fullPath := filepath.Join(trainingRoot, exerciseDir, f.Path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			missingFiles = append(missingFiles, f)
		}
	}

	if len(missingFiles) == 0 {
		fmt.Println("  All exercise files are present, nothing to restore.")
		return nil
	}

	oldHead, _ := gitOps.RevParse("HEAD")

	// Additive write — do NOT use NewFilesSilentDeleteUnused here: resetMissingOnly
	// must preserve the user's modifications. This is a gap-fill, not a full reset.
	fw := files.NewFilesSilent()
	if err := fw.WriteExerciseFiles(missingFiles, fs, exerciseDir); err != nil {
		return fmt.Errorf("could not write missing files: %w", err)
	}

	for _, f := range missingFiles {
		fmt.Printf("  %s %s\n", color.GreenString("+"), filepath.Join(exerciseDir, f.Path))
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
