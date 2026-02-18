package trainings

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/ThreeDotsLabs/cli/trainings/git"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var errMergeAborted = fmt.Errorf("merge aborted by user")

func (h *Handlers) nextExercise(ctx context.Context, currentExerciseID string, trainingRoot string) (finished bool, err error) {
	return h.nextExerciseWithSkipped(ctx, currentExerciseID, trainingRoot, nil)
}

func (h *Handlers) nextExerciseWithSkipped(ctx context.Context, currentExerciseID string, trainingRoot string, skipExerciseIDs []string) (finished bool, err error) {
	h.solutionHintDisplayed = false
	clear(h.notifications)

	// We should never trust the remote server.
	// Writing files based on external name is a vector for Path Traversal attack.
	// For more info please check: https://owasp.org/www-community/attacks/Path_Traversal
	//
	// To avoid that we are using afero.BasePathFs with base on training root for all operations in trainings dir.
	trainingRootFs := afero.NewBasePathFs(afero.NewOsFs(), trainingRoot).(*afero.BasePathFs)

	resp, err := h.getNextExercise(ctx, currentExerciseID, trainingRootFs)
	if err != nil {
		return false, err
	}

	writeFiles := true
	for _, skipExerciseID := range skipExerciseIDs {
		if resp.ExerciseId == skipExerciseID {
			// Exercise already has a local solution, don't overwrite files
			writeFiles = false
			break
		}
	}

	return h.setExercise(trainingRootFs, resp, trainingRoot, writeFiles)
}

func (h *Handlers) setExercise(fs *afero.BasePathFs, exercise *genproto.NextExerciseResponse, trainingRoot string, writeFiles bool) (finished bool, err error) {
	if exercise.TrainingStatus == genproto.NextExerciseResponse_FINISHED {
		printFinished()
		return true, nil
	}
	if exercise.TrainingStatus == genproto.NextExerciseResponse_COHORT_BATCH_DONE {
		var date *time.Time
		if exercise.GetNextBatchDate() != nil {
			t := exercise.GetNextBatchDate().AsTime()
			date = &t
		}

		printCohortBatchDone(date)
		return true, nil
	}
	if exercise.TrainingStatus == genproto.NextExerciseResponse_PAYMENT_REQUIRED {
		printPaymentRequired()
		return false, nil
	}

	if exercise.GetExercise() != nil {
		h.printCurrentExercise(
			exercise.GetExercise().GetModule().GetName(),
			exercise.GetExercise().GetName(),
		)
	}

	if writeFiles {
		gitOps := h.newGitOps()
		if gitOps.Enabled() && !exercise.IsTextOnly {
			// Read current (soon-to-be-previous) exercise config for init chain
			var prevModuleExercisePath string
			if files.DirOrFileExists(fs, ".tdl-exercise") {
				prevCfg := h.config.ExerciseConfig(fs)
				prevModuleExercisePath = prevCfg.ModuleExercisePath()
			}

			if err := h.setExerciseWithGit(gitOps, fs, exercise, trainingRoot, prevModuleExercisePath); err != nil {
				return false, fmt.Errorf("git flow failed: %w", err)
			}
			// Git handled file writing; just write the exercise config
			if err := h.config.WriteExerciseConfig(
				fs,
				config.ExerciseConfig{
					ExerciseID:   exercise.ExerciseId,
					Directory:    exercise.Dir,
					IsTextOnly:   exercise.IsTextOnly,
					IsOptional:   exercise.IsOptional,
					ModuleName:   exercise.GetExercise().GetModule().GetName(),
					ExerciseName: exercise.GetExercise().GetName(),
				},
			); err != nil {
				return false, err
			}
			goto postWrite
		}

		// Existing behavior (no git or text-only)
		isEasy := exercise.TrainingDifficulty == genproto.TrainingDifficulty_EASY
		f := files.NewFilesWithConfig(isEasy, isEasy)
		if err := h.writeExerciseFiles(f, nextExerciseResponseToExerciseSolution(exercise), fs); err != nil {
			return false, err
		}
	} else {
		// Files already exist (e.g. restored), but still update exercise state
		if err := h.config.WriteExerciseConfig(
			fs,
			config.ExerciseConfig{
				ExerciseID:   exercise.ExerciseId,
				Directory:    exercise.Dir,
				IsTextOnly:   exercise.IsTextOnly,
				IsOptional:   exercise.IsOptional,
				ModuleName:   exercise.GetExercise().GetModule().GetName(),
				ExerciseName: exercise.GetExercise().GetName(),
			},
		); err != nil {
			return false, err
		}
	}

postWrite:
	if exercise.IsTextOnly {
		printTextOnlyExerciseInfo(
			h.config.TrainingConfig(fs).TrainingName,
			exercise.ExerciseId,
		)
	} else {
		err = addModuleToWorkspace(trainingRoot, exercise.Dir)
		if err != nil {
			logrus.WithError(err).Warn("Failed to add module to workspace")
		}
	}

	return false, nil
}

func (h *Handlers) getNextExercise(
	ctx context.Context,
	currentExerciseID string,
	trainingRootFs *afero.BasePathFs,
) (resp *genproto.NextExerciseResponse, err error) {
	cfg := h.config.TrainingConfig(trainingRootFs)
	resp, err = h.newGrpcClient().NextExercise(
		ctxWithRequestHeader(ctx, h.cliMetadata),
		&genproto.NextExerciseRequest{
			TrainingName:      cfg.TrainingName,
			CurrentExerciseId: currentExerciseID,
			Token:             h.config.GlobalConfig().Token,
			SendAllFiles:      cfg.GitConfigured && cfg.GitEnabled,
		},
	)

	logrus.WithFields(logrus.Fields{
		"resp": resp,
		"err":  err,
	}).Debug("Received exercise from server")

	return resp, err
}

func (h *Handlers) writeExerciseFiles(files files.Files, resp *genproto.ExerciseSolution, trainingRootFs *afero.BasePathFs) error {
	if resp.Dir == "" {
		return errors.New("exercise dir is empty")
	}
	if resp.ExerciseId == "" {
		return errors.New("exercise id is empty")
	}

	if err := files.WriteExerciseFiles(resp.Files, trainingRootFs, resp.Dir); err != nil {
		return err
	}

	return h.config.WriteExerciseConfig(
		trainingRootFs,
		config.ExerciseConfig{
			ExerciseID:   resp.ExerciseId,
			Directory:    resp.Dir,
			IsTextOnly:   resp.IsTextOnly,
			IsOptional:   resp.IsOptional,
			ModuleName:   resp.GetModule().GetName(),
			ExerciseName: resp.GetExercise().GetName(),
		},
	)
}

func (h *Handlers) setExerciseWithGit(
	gitOps *git.Ops,
	fs *afero.BasePathFs,
	exercise *genproto.NextExerciseResponse,
	trainingRoot string,
	prevModuleExercisePath string,
) error {
	exerciseDir := exercise.Dir
	moduleExercisePath := exercise.Dir // fallback
	if exercise.GetExercise() != nil {
		moduleName := exercise.GetExercise().GetModule().GetName()
		exerciseName := exercise.GetExercise().GetName()
		if moduleName != "" && exerciseName != "" {
			moduleExercisePath = moduleName + "/" + exerciseName
		}
	}

	// 1-5. Create init branch (shared with restore flow)
	initBranch, err := createInitBranch(
		gitOps, exerciseDir, moduleExercisePath, prevModuleExercisePath,
		exercise.FilesToCreate, exercise.IsTextOnly, false, time.Time{},
	)
	if err != nil {
		return err
	}

	// 6. Conflict preview (read-only) — only in interactive mode
	var conflictPrompt rune
	var previewedConflictFiles []string
	var previewBackupBranch string
	if internal.IsStdinTerminal() {
		conflictFiles, previewErr := gitOps.MergeTreeCheck(initBranch)
		if previewErr != nil {
			logrus.WithError(previewErr).Debug("merge-tree not available, skipping conflict preview")
		} else if len(conflictFiles) > 0 {
			previewedConflictFiles = conflictFiles
			previewBackupBranch = git.BackupBranchName(moduleExercisePath)
			fmt.Println(color.YellowString("\n  The next exercise updates files you've modified:"))
			for _, cf := range conflictFiles {
				fmt.Printf("    %s\n", color.CyanString(cf))
			}
			fmt.Println()
			fmt.Println("  You can merge the changes yourself, or we'll replace all exercise files")
			fmt.Println("  with our versions (only this exercise is affected).")
			fmt.Printf("  Your code will be saved to %s — you can restore it anytime.\n", color.MagentaString(previewBackupBranch))
			fmt.Println()
			conflictPrompt = internal.Prompt(
				internal.Actions{
					{Shortcut: '\n', Action: "merge (resolve in editor)", ShortcutAliases: []rune{'\r'}},
					{Shortcut: 'g', Action: "replace exercise files with ours"},
					{Shortcut: 'q', Action: "abort — stay on current exercise"},
				},
				os.Stdin, os.Stdout,
			)
			if conflictPrompt == 'q' {
				fmt.Println(color.YellowString("  Merge aborted, staying on current exercise."))
				return errMergeAborted
			}
		}
	}

	// 7. Actual merge
	mergeMsg := fmt.Sprintf("start %s", moduleExercisePath)
	var oldHead string
	for {
		oldHead, _ = gitOps.RevParse("HEAD")
		mergeErr := gitOps.Merge(initBranch, mergeMsg)
		if mergeErr == nil {
			break // clean merge
		}

		if strings.Contains(mergeErr.Error(), "would be overwritten") {
			// Dirty working tree — git refused, nothing touched
			fmt.Println(color.YellowString("\n  You have uncommitted changes in files the exercise needs to update."))
			fmt.Println(color.YellowString("  In another terminal, commit or stash your changes:"))
			fmt.Println(color.CyanString("    git add -A && git commit -m \"my changes\""))
			if !internal.ConfirmPromptDefaultYes("retry") {
				return fmt.Errorf("merge blocked by uncommitted changes")
			}
			continue // retry after user commits
		}

		// Merge conflict — working tree has conflict markers
		if conflictPrompt == 'g' && len(previewedConflictFiles) > 0 {
			// User chose "replace all exercise files" — save their work, then overwrite entire exercise dir
			_ = gitOps.CreateBranchFromHead(previewBackupBranch)
			gitOps.PrintInfo(fmt.Sprintf("git branch %s", previewBackupBranch))
			_ = gitOps.CheckoutFiles(initBranch, exerciseDir)
			_ = gitOps.AddAll(exerciseDir)
			_ = gitOps.Commit(mergeMsg)
			fmt.Printf("  Your code saved to branch %s\n", color.MagentaString(previewBackupBranch))
			fmt.Println("  Restore anytime with: " + color.CyanString("git checkout %s -- %s", previewBackupBranch, exerciseDir))
			fmt.Println(color.GreenString("  All exercise files replaced with our versions."))
			fmt.Println()
		} else if internal.IsStdinTerminal() {
			// Interactive conflict resolution loop
			if err := resolveConflictsInteractive(gitOps, initBranch, mergeMsg, moduleExercisePath, exerciseDir); err != nil {
				return err
			}
		} else {
			// Non-interactive fallback
			fmt.Println(color.YellowString("\n  Merge conflict detected."))
			fmt.Println(color.YellowString("  After resolving conflicts:"))
			fmt.Println(color.CyanString("    git add -A && git commit"))
		}
		break
	}

	if stat, err := gitOps.DiffStatPath(oldHead, "HEAD", exerciseDir); err == nil && stat != "" {
		fmt.Println(stat)
	}
	fmt.Printf("\n%s\n\n", color.GreenString("Exercise ready."))

	return nil
}

// createInitBranch creates the init branch for an exercise via worktree.
// Returns the init branch name.
// When quiet is true, existing branches are silently deleted and simulated fetch output is skipped.
// When commitDate is non-zero, the init commit uses that date.
func createInitBranch(
	gitOps *git.Ops,
	exerciseDir string,
	moduleExercisePath string,
	prevModuleExercisePath string,
	scaffoldFiles []*genproto.File,
	isTextOnly bool,
	quiet bool,
	commitDate time.Time,
) (initBranch string, err error) {
	initBranch = git.InitBranchName(moduleExercisePath)

	// Clean up existing init branch if present
	if gitOps.BranchExists(initBranch) {
		if quiet {
			if err := gitOps.DeleteBranch(initBranch); err != nil {
				return "", fmt.Errorf("can't delete existing branch %s: %w", initBranch, err)
			}
		} else {
			backupName := git.BackupBranchName(moduleExercisePath)
			if err := gitOps.CreateBranchFrom(backupName, initBranch); err != nil {
				logrus.WithError(err).Debug("Could not back up existing init branch")
			}
			fmt.Printf("  Branch %s already existed, backed up to %s\n", initBranch, backupName)
			if err := gitOps.DeleteBranch(initBranch); err != nil {
				return "", fmt.Errorf("can't delete existing branch %s: %w", initBranch, err)
			}
		}
	}

	// Create worktree — based on previous init branch (chain) or HEAD (first exercise)
	tmpDir, err := os.MkdirTemp("", "tdl-init-")
	if err != nil {
		return "", fmt.Errorf("can't create temp dir for worktree: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	prevInitBranch := ""
	if prevModuleExercisePath != "" {
		prevInitBranch = git.InitBranchName(prevModuleExercisePath)
	}

	if prevInitBranch != "" && gitOps.BranchExists(prevInitBranch) {
		if err := gitOps.WorktreeAddFrom(tmpDir, initBranch, prevInitBranch); err != nil {
			return "", fmt.Errorf("can't create worktree from %s: %w", prevInitBranch, err)
		}
	} else {
		if err := gitOps.WorktreeAdd(tmpDir, initBranch); err != nil {
			return "", fmt.Errorf("can't create worktree: %w", err)
		}
	}
	defer gitOps.WorktreeRemove(tmpDir)

	// Write exercise files into the worktree silently
	worktreeFs := afero.NewBasePathFs(afero.NewOsFs(), tmpDir).(*afero.BasePathFs)
	f := files.NewFilesSilent()
	if err := f.WriteExerciseFiles(scaffoldFiles, worktreeFs, exerciseDir); err != nil {
		return "", fmt.Errorf("can't write exercise files to worktree: %w", err)
	}

	// Add module to workspace in worktree
	if !isTextOnly {
		if err := addModuleToWorkspaceQuiet(tmpDir, exerciseDir, true); err != nil {
			logrus.WithError(err).Warn("Failed to add module to workspace in worktree")
		}
	}

	// Commit on init branch
	worktreeOps := git.NewQuietOps(tmpDir)
	if err := worktreeOps.AddAll(exerciseDir); err != nil {
		return "", fmt.Errorf("can't stage exercise files: %w", err)
	}
	if hasGoWorkspace(tmpDir) {
		_ = worktreeOps.AddFiles("go.work")
	}
	if worktreeOps.HasStagedChanges() {
		commitMsg := fmt.Sprintf("init files for %s", moduleExercisePath)
		if !commitDate.IsZero() {
			if err := worktreeOps.CommitWithDate(commitMsg, commitDate); err != nil {
				return "", fmt.Errorf("can't commit exercise files: %w", err)
			}
		} else {
			if err := worktreeOps.Commit(commitMsg); err != nil {
				return "", fmt.Errorf("can't commit exercise files: %w", err)
			}
		}
	}

	// Show simulated fetch (skipped in quiet mode)
	if !quiet {
		gitOps.PrintInfo(fmt.Sprintf("git fetch cli %s", initBranch))
	}

	return initBranch, nil
}

func nextExerciseResponseToExerciseSolution(resp *genproto.NextExerciseResponse) *genproto.ExerciseSolution {
	sol := &genproto.ExerciseSolution{
		ExerciseId: resp.ExerciseId,
		Dir:        resp.Dir,
		Files:      resp.FilesToCreate,
		IsTextOnly: resp.IsTextOnly,
		IsOptional: resp.IsOptional,
	}
	if resp.GetExercise() != nil {
		sol.Module = &genproto.ExerciseSolution_Module{
			Name: resp.GetExercise().GetModule().GetName(),
		}
		sol.Exercise = &genproto.ExerciseSolution_Exercise{
			Name: resp.GetExercise().GetName(),
		}
	}
	return sol
}

// resolveConflictsInteractive runs a loop where the user can fix conflicts in their editor,
// replace all exercise files with init branch versions (saving to a backup branch), or quit.
func resolveConflictsInteractive(gitOps *git.Ops, initBranch, mergeMsg, moduleExercisePath, exerciseDir string) error {
	conflictFiles, _ := gitOps.UnmergedFiles()
	fmt.Println(color.YellowString("\n  Merge conflict detected."))
	fmt.Println(color.YellowString("  Files with conflicts:"))
	for _, cf := range conflictFiles {
		fmt.Printf("    %s\n", color.CyanString(cf))
	}

	backupBranch := git.BackupBranchName(moduleExercisePath)

	fmt.Println()
	fmt.Println("  Resolve the conflicts in your editor, or press 'g' to replace all exercise")
	fmt.Println("  files with our versions (only this exercise is affected).")
	fmt.Printf("  Your code will be saved to %s — you can restore it anytime.\n", color.MagentaString(backupBranch))

	for {
		choice := internal.Prompt(
			internal.Actions{
				{Shortcut: '\n', Action: "confirm (conflicts resolved)", ShortcutAliases: []rune{'\r'}},
				{Shortcut: 'g', Action: "replace exercise files with ours"},
				{Shortcut: 'q', Action: "abort — cancel merge"},
			},
			os.Stdin, os.Stdout,
		)

		switch choice {
		case '\n':
			// User says they resolved — verify
			remaining, _ := gitOps.UnmergedFiles()
			if len(remaining) > 0 {
				fmt.Println(color.YellowString("\n  Still unresolved:"))
				for _, cf := range remaining {
					fmt.Printf("    %s\n", color.CyanString(cf))
				}
				continue
			}
			// All resolved — complete the merge
			_ = gitOps.AddAll(".")
			if err := gitOps.Commit(mergeMsg); err != nil {
				return fmt.Errorf("can't complete merge commit: %w", err)
			}
			fmt.Println(color.GreenString("  Merge complete."))
			fmt.Println()
			return nil

		case 'g':
			// Save user's work, then replace all exercise files from init branch
			_ = gitOps.CreateBranchFromHead(backupBranch)
			gitOps.PrintInfo(fmt.Sprintf("git branch %s", backupBranch))
			_ = gitOps.CheckoutFiles(initBranch, exerciseDir)
			_ = gitOps.AddAll(".")
			_ = gitOps.Commit(mergeMsg)
			fmt.Printf("  Your code saved to branch %s\n", color.MagentaString(backupBranch))
			fmt.Println("  Restore anytime with: " + color.CyanString("git checkout %s -- %s", backupBranch, exerciseDir))
			fmt.Println(color.GreenString("  All exercise files replaced with our versions."))
			fmt.Println()
			return nil

		case 'q':
			// Abort merge, restore pre-merge state
			_ = gitOps.MergeAbort()
			fmt.Println(color.YellowString("  Merge aborted, staying on current exercise."))
			return errMergeAborted
		}
	}
}
