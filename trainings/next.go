package trainings

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"golang.org/x/term"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/ThreeDotsLabs/cli/trainings/git"
	mcppkg "github.com/ThreeDotsLabs/cli/trainings/mcp"
)

// promptRune displays a prompt for the given actions and reads a single valid keypress.
// Uses h.stdinCh when MCP is active, otherwise reads os.Stdin directly.
// This is the MCP-safe replacement for internal.Prompt when called from within interactiveRun.
func (h *Handlers) promptRune(actions internal.Actions) rune {
	defer fmt.Println()

	printPrompt(actions)

	termState, rawErr := term.MakeRaw(0)
	if rawErr == nil {
		defer term.Restore(0, termState)
	}

	if h.stdinCh == nil {
		reader := bufio.NewReader(os.Stdin)
		for {
			ch, _, err := reader.ReadRune()
			if err != nil {
				return 'q'
			}
			if string(ch) == "\x03" {
				if rawErr == nil {
					term.Restore(0, termState)
				}
				os.Exit(0)
			}
			if key, ok := actions.ReadKeyFromInput(ch); ok {
				return key
			}
		}
	}

	drainChannel(h.stdinCh)

	for {
		ch, ok := <-h.stdinCh
		if !ok {
			if rawErr == nil {
				// Reset terminal to cooked mode so the shell works normally after exit.
				term.Restore(0, termState)
			}
			logrus.Debug("stdin closed, exiting")
			fmt.Println(color.HiBlackString("Input closed — exiting."))
			os.Exit(0)
		}
		if string(ch) == "\x03" {
			if rawErr == nil {
				term.Restore(0, termState)
			}
			os.Exit(0)
		}
		if key, ok := actions.ReadKeyFromInput(ch); ok {
			return key
		}
	}
}

var errMergeAborted = fmt.Errorf("merge aborted by user")

func (h *Handlers) nextExercise(ctx context.Context, currentExerciseID string, trainingRoot string) (finished bool, err error) {
	return h.nextExerciseWithSkipped(ctx, currentExerciseID, trainingRoot, nil)
}

func (h *Handlers) nextExerciseWithSkipped(ctx context.Context, currentExerciseID string, trainingRoot string, skipExerciseIDs []string) (finished bool, err error) {
	h.solutionHintDisplayed = false
	h.solutionAvailable = false
	h.stuckRunCount = 0
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

	return h.setExercise(ctx, trainingRootFs, resp, trainingRoot, writeFiles)
}

func (h *Handlers) setExercise(ctx context.Context, fs *afero.BasePathFs, exercise *genproto.NextExerciseResponse, trainingRoot string, writeFiles bool) (finished bool, err error) {
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
			// Read current (soon-to-be-previous) exercise config for init chain continuity.
			// "golden of prev" for the 'g' merge-conflict path is now resolved server-side
			// via GetExerciseStartState(currentExerciseID), so we no longer thread it here.
			var prevModuleExercisePath string
			if files.DirOrFileExists(fs, ".tdl-exercise") {
				prevCfg := h.config.ExerciseConfig(fs)
				prevModuleExercisePath = prevCfg.ModuleExercisePath()
			}

			if err := h.setExerciseWithGit(ctx, gitOps, fs, exercise, trainingRoot, prevModuleExercisePath); err != nil {
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
		if h.stdinCh != nil {
			f = f.WithStdinCh(h.stdinCh)
		}
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
	// exercise.md is gitignored (copyright), so git-based flows won't deliver it.
	// Write it directly regardless of which path we took.
	writeExerciseMd(exercise.FilesToCreate, fs, exercise.Dir)

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
		ctx,
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
	ctx context.Context,
	gitOps *git.Ops,
	fs *afero.BasePathFs,
	exercise *genproto.NextExerciseResponse,
	trainingRoot string,
	prevModuleExercisePath string,
) error {
	exerciseDir := exercise.Dir
	moduleExercisePath := moduleExercisePathFromResponse(exercise)

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
			fmt.Println("  with our example solution (only this exercise is affected).")
			fmt.Printf("  Your code will be saved to %s: you can restore it anytime.\n", color.MagentaString(previewBackupBranch))
			fmt.Println()
			if h.loopState != nil {
				h.loopState.SetPendingAction("Merge conflict decision needed. Go to CLI.")
			}
			// Unblock MCP client immediately — the conflict prompt only accepts stdin input,
			// so the MCP tool call would hang until the 5-minute timeout otherwise.
			h.sendPendingMCPResult(mcppkg.MCPResult{
				Error: "Advancing is paused: merge conflicts detected in files you modified. " +
					"The user must resolve this in the CLI terminal. " +
					"Call training_get_exercise_info to check when resolved.",
			})
			conflictPrompt = h.promptRune(
				internal.Actions{
					{Shortcut: '\n', Action: "merge (resolve in editor)", ShortcutAliases: []rune{'\r'}},
					{Shortcut: 'g', Action: "replace exercise files with our example solution"},
					{Shortcut: 'q', Action: "abort: stay on current exercise"},
				},
			)
			if h.loopState != nil {
				h.loopState.ClearPendingAction()
			}
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
			if h.loopState != nil {
				h.loopState.SetPendingAction("Uncommitted changes blocking next exercise. Go to CLI to commit or stash.")
			}
			retryChoice := h.promptRune(internal.Actions{
				{Shortcut: '\n', Action: "retry", ShortcutAliases: []rune{'\r'}},
				{Shortcut: 'q', Action: "quit"},
			})
			if retryChoice != '\n' {
				if h.loopState != nil {
					h.loopState.ClearPendingAction()
				}
				return fmt.Errorf("merge blocked by uncommitted changes")
			}
			if h.loopState != nil {
				h.loopState.ClearPendingAction()
			}
			continue // retry after user commits
		}

		// Merge conflict — working tree has conflict markers
		trainingName := h.config.TrainingConfig(fs).TrainingName
		if conflictPrompt == 'g' && len(previewedConflictFiles) > 0 {
			// User chose "replace all exercise files". INVARIANT: after replace,
			// exerciseDir must be 1:1 with the exercise's start state
			// (golden(prev) merged with scaffold(current)) — no stale user files.
			// Start state comes from server's GetExerciseStartState.
			if err := h.replaceExerciseFilesOnMergeConflict(
				ctx, gitOps, fs,
				exercise.ExerciseId,
				exerciseDir, previewBackupBranch, mergeMsg, trainingName,
			); err != nil {
				return err
			}
		} else if internal.IsStdinTerminal() {
			// Interactive conflict resolution loop
			if err := h.resolveConflictsInteractive(
				ctx, gitOps, fs,
				initBranch, mergeMsg, moduleExercisePath, exerciseDir, trainingName,
				exercise.ExerciseId,
			); err != nil {
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

	if h.loopState != nil {
		var content strings.Builder
		// Preserve any content already captured earlier in this advance (e.g. sync diff
		// from overrideWithGolden) and append the new-exercise scaffold stat to it.
		// For the new-exercise scaffold we only include the diffstat — the full diff
		// for a fresh exercise is pure noise (imports, boilerplate, new go.mod) that
		// the student is about to read anyway in the files themselves.
		if existing := h.loopState.GetTransitionContent(); existing != "" {
			content.WriteString(existing)
			content.WriteString("\n")
		}
		if stat, err := gitOps.DiffStatPathPlain(oldHead, "HEAD", exerciseDir); err == nil && stat != "" {
			content.WriteString("## Files changed\n")
			content.WriteString(stat)
			content.WriteString("\n")
		}
		h.loopState.SetTransitionContent(content.String())
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

// moduleExercisePathFromResponse derives the "module/exercise" path from a NextExerciseResponse.
// Falls back to resp.Dir if module/exercise names are missing.
func moduleExercisePathFromResponse(resp *genproto.NextExerciseResponse) string {
	if resp.GetExercise() != nil {
		moduleName := resp.GetExercise().GetModule().GetName()
		exerciseName := resp.GetExercise().GetName()
		if moduleName != "" && exerciseName != "" {
			return moduleName + "/" + exerciseName
		}
	}
	return resp.Dir
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

// writeExerciseMd writes exercise.md directly to the filesystem.
// exercise.md is gitignored (copyright), so git-based flows (merge, checkout)
// won't deliver it. Call after any git-based file delivery.
func writeExerciseMd(allFiles []*genproto.File, fs afero.Fs, exerciseDir string) {
	var mdFiles []*genproto.File
	for _, f := range allFiles {
		if f.Path == files.ExerciseFile {
			mdFiles = append(mdFiles, f)
		}
	}
	if len(mdFiles) > 0 {
		if err := files.NewFilesSilent().WriteExerciseFiles(mdFiles, fs, exerciseDir); err != nil {
			logrus.WithError(err).Warn("Could not write exercise.md")
		}
	}
}

// writeServerFiles writes all files from the server response to the exercise directory,
// then stages and commits any changes. Used after git-based reset to ensure the latest
// scaffold (and golden files for EASY mode) is applied on top of the init branch checkout.
func writeServerFiles(
	allFiles []*genproto.File,
	trainingRootFs *afero.BasePathFs,
	exerciseDir string,
	gitOps *git.Ops,
	moduleExercisePath string,
) {
	if len(allFiles) == 0 {
		return
	}

	fw := files.NewFilesSilent()
	if err := fw.WriteExerciseFiles(allFiles, trainingRootFs, exerciseDir); err != nil {
		logrus.WithError(err).Warn("Could not write exercise files from server")
		return
	}

	if err := gitOps.AddAll(exerciseDir); err != nil {
		logrus.WithError(err).Warn("Could not stage exercise files")
		return
	}
	if gitOps.HasStagedChanges() {
		if err := gitOps.Commit(fmt.Sprintf("update exercise files for %s", moduleExercisePath)); err != nil {
			logrus.WithError(err).Warn("Could not commit exercise files")
		}
	}
}

// resolveConflictsInteractive runs a loop where the user can fix conflicts in their editor,
// replace all exercise files with the start state (saving to a backup branch), or quit.
//
// IMPORTANT: The 'g' (replace) path is destructive — it overwrites user files.
// We MUST save their code to a backup branch before replacing. The user explicitly
// confirms this action.
//
// INVARIANT on 'g': exerciseDir ends 1:1 with golden(prev)+scaffold(current) —
// routed through replaceExerciseFilesOnMergeConflict → replaceExerciseFilesAndCommit.
func (h *Handlers) resolveConflictsInteractive(
	ctx context.Context,
	gitOps *git.Ops,
	fs *afero.BasePathFs,
	initBranch, mergeMsg, moduleExercisePath, exerciseDir, trainingName string,
	currentExerciseID string,
) error {
	conflictFiles, _ := gitOps.UnmergedFiles()
	fmt.Println(color.YellowString("\n  Merge conflict detected."))
	fmt.Println(color.YellowString("  Files with conflicts:"))
	for _, cf := range conflictFiles {
		fmt.Printf("    %s\n", color.CyanString(cf))
	}

	backupBranch := git.BackupBranchName(moduleExercisePath)

	fmt.Println()
	fmt.Println("  Resolve the conflicts in your editor, or press 'g' to replace all exercise")
	fmt.Println("  files with our example solution (only this exercise is affected).")
	fmt.Printf("  Your code will be saved to %s: you can restore it anytime.\n", color.MagentaString(backupBranch))

	for {
		if h.loopState != nil {
			h.loopState.SetPendingAction("Merge conflicts need resolution. Go to CLI.")
		}
		choice := h.promptRune(
			internal.Actions{
				{Shortcut: '\n', Action: "confirm (conflicts resolved)", ShortcutAliases: []rune{'\r'}},
				{Shortcut: 'g', Action: "replace exercise files with our example solution"},
				{Shortcut: 'q', Action: "abort: cancel merge"},
			},
		)
		if h.loopState != nil {
			h.loopState.ClearPendingAction()
		}

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
			_ = gitOps.AddAll(exerciseDir)
			if err := gitOps.Commit(mergeMsg); err != nil {
				return fmt.Errorf("can't complete merge commit: %w", err)
			}
			fmt.Println(color.GreenString("  Merge complete."))
			fmt.Println()
			return nil

		case 'g':
			return h.replaceExerciseFilesOnMergeConflict(
				ctx, gitOps, fs,
				currentExerciseID,
				exerciseDir, backupBranch, mergeMsg, trainingName,
			)

		case 'q':
			// Abort merge, restore pre-merge state
			_ = gitOps.MergeAbort()
			fmt.Println(color.YellowString("  Merge aborted, staying on current exercise."))
			return errMergeAborted
		}
	}
}

// replaceExerciseFilesOnMergeConflict completes an in-progress merge by replacing
// exerciseDir with the start state (golden(prev) merged with scaffold(current)).
// User's pre-merge state is saved to backupBranch.
//
// INVARIANT: on success, exerciseDir is 1:1 with the start state — no stale user
// files. Routes through replaceExerciseFilesAndCommit → replaceExerciseFiles;
// see exercise_replace.go for why this invariant is load-bearing.
//
// Start state composition lives server-side in GetExerciseStartState.
// If backup creation fails and the user aborts, the merge is aborted and
// errMergeAborted is returned so callers can treat it as a clean cancellation.
func (h *Handlers) replaceExerciseFilesOnMergeConflict(
	ctx context.Context,
	gitOps *git.Ops,
	fs *afero.BasePathFs,
	currentExerciseID string,
	exerciseDir, backupBranch, mergeMsg, trainingName string,
) error {
	resp, err := h.newGrpcClient().GetExerciseStartState(ctx, &genproto.GetExerciseStartStateRequest{
		TrainingName: h.config.TrainingConfig(fs).TrainingName,
		Token:        h.config.GlobalConfig().Token,
		ExerciseId:   currentExerciseID,
	})
	if err != nil {
		// Abort the in-progress merge to leave a clean state.
		_ = gitOps.MergeAbort()
		fmt.Println(formatGitError("Could not fetch exercise start state", err, trainingName))
		return err
	}
	if len(resp.Files) == 0 {
		_ = gitOps.MergeAbort()
		fmt.Println(color.YellowString("  Server returned no files for exercise start state — aborting to protect your workspace."))
		fmt.Println(color.YellowString("  Please update your CLI or contact support if the problem persists."))
		return fmt.Errorf("empty exercise start state")
	}

	_, err = replaceExerciseFilesAndCommit(gitOps, fs, resp.Files, exerciseDir, backupBranch, mergeMsg)
	if err != nil {
		if errors.Is(err, errBackupAborted) {
			_ = gitOps.MergeAbort()
			fmt.Println(color.YellowString("  Merge aborted, staying on current exercise."))
			return errMergeAborted
		}
		fmt.Println(formatGitError("Could not replace exercise files", err, trainingName))
		fmt.Println(recoveryHint(trainingName))
		return err
	}
	fmt.Printf("  Your code saved to branch %s\n", color.MagentaString(backupBranch))
	fmt.Println("  Restore anytime with: " + color.CyanString("git checkout %s -- %s", backupBranch, exerciseDir))
	fmt.Println(color.GreenString("  All exercise files replaced with our example solution."))
	fmt.Println()
	return nil
}
