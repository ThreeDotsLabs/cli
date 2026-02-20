package trainings

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/ThreeDotsLabs/cli/trainings/git"
)

func (h *Handlers) Run(ctx context.Context, detached bool) error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return nil
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)

	if detached {
		return h.detachedRun(ctx, trainingRootFs)
	} else {
		return h.interactiveRun(ctx, trainingRootFs)
	}
}

func (h *Handlers) detachedRun(ctx context.Context, trainingRootFs *afero.BasePathFs) error {
	successful, err := h.run(ctx, trainingRootFs)
	if err != nil {
		return err
	}
	if !successful {
		os.Exit(1)
	}

	fmt.Println()

	promptResult := internal.Prompt(
		internal.Actions{
			{Shortcut: '\n', Action: "go to the next exercise", ShortcutAliases: []rune{'\r'}},
			{Shortcut: 'q', Action: "quit"},
		},
		os.Stdin,
		os.Stdout,
	)
	if promptResult == 'q' {
		os.Exit(0)
	}

	trainingRoot, err := h.config.FindTrainingRoot()
	if err != nil {
		return err
	}

	_, err = h.nextExercise(ctx, h.config.ExerciseConfig(trainingRootFs).ExerciseID, trainingRoot)
	if errors.Is(err, errMergeAborted) {
		return nil // clean exit
	}
	if err != nil {
		return err
	}

	return nil
}

func (h *Handlers) interactiveRun(ctx context.Context, trainingRootFs *afero.BasePathFs) error {
	retries := 0
	mergeAborted := false

	for {
		if !mergeAborted {
			successful, err := h.run(ctx, trainingRootFs)
			if err != nil && retries < 3 {
				retries++
				time.Sleep(time.Duration(retries) * time.Millisecond * 50)
				logrus.WithError(err).WithField("retry", retries).Info("execution failed, retrying")
				continue
			}
			retries = 0

			fmt.Println()

			if err != nil {
				fmt.Println(color.RedString("Failed to execute solution: %s", err.Error()))

				if !internal.ConfirmPromptDefaultYes("run solution again") {
					return err
				} else {
					continue
				}
			}

			if !successful {
				if !internal.ConfirmPromptDefaultYes("run solution again") {
					return nil
				} else {
					continue
				}
			}
		}
		mergeAborted = false

		// Build actions dynamically based on git config
		actions := internal.Actions{
			{Shortcut: '\n', Action: "go to the next exercise", ShortcutAliases: []rune{'\r'}},
		}

		gitOps := h.newGitOps()
		trainingRoot, err := h.config.FindTrainingRoot()
		if err != nil {
			return err
		}
		cfg := h.config.TrainingConfig(trainingRootFs)
		exerciseCfg := h.config.ExerciseConfig(trainingRootFs)

		if gitOps.Enabled() && !exerciseCfg.IsTextOnly {
			actions = append(actions, internal.Action{Shortcut: 'g', Action: "replace your solution with golden"})
		}
		actions = append(actions,
			internal.Action{Shortcut: 'r', Action: "re-run solution"},
			internal.Action{Shortcut: 'q', Action: "quit"},
		)

		promptResult := internal.Prompt(actions, os.Stdin, os.Stdout)
		if promptResult == 'q' {
			os.Exit(0)
		}
		if promptResult == 'r' {
			continue
		}
		if promptResult == 'g' {
			h.overrideWithGolden(ctx, trainingRootFs, gitOps, exerciseCfg)
			// Fall through to next exercise (golden already committed, no staged changes)
		}

		// Auto-commit on advancing to next exercise
		if gitOps.Enabled() && cfg.GitAutoCommit && !exerciseCfg.IsTextOnly {
			if err := gitOps.AddAll(exerciseCfg.Directory); err != nil {
				logrus.WithError(err).Warn("Could not stage solution files")
			}
			if gitOps.HasStagedChanges() {
				if err := gitOps.Commit(fmt.Sprintf("completed %s", exerciseCfg.ModuleExercisePath())); err != nil {
					logrus.WithError(err).Warn("Could not commit solution")
				}
			}
		} else if gitOps.Enabled() && !cfg.GitAutoCommit && !exerciseCfg.IsTextOnly {
			if gitOps.HasUncommittedChanges(exerciseCfg.Directory) {
				fmt.Printf("  Tip: %s\n", color.HiBlackString("git commit -am \"completed %s\"", exerciseCfg.ModuleExercisePath()))
			}
		}

		// Always create golden branch for comparison (skip if user already pressed 'g')
		if gitOps.Enabled() && !exerciseCfg.IsTextOnly {
			goldenBranch := git.GoldenBranchName(exerciseCfg.ModuleExercisePath())
			if !gitOps.BranchExists(goldenBranch) {
				h.syncGoldenSolution(ctx, trainingRootFs, gitOps, exerciseCfg, "compare")
			}
		}

		finished, err := h.nextExercise(ctx, exerciseCfg.ExerciseID, trainingRoot)
		if errors.Is(err, errMergeAborted) {
			mergeAborted = true
			continue // stay on current exercise, re-show prompt
		}
		if err != nil {
			return err
		}
		if finished {
			return nil
		}

		// this is refreshed config after nextExercise execution
		currentExerciseConfig := h.config.ExerciseConfig(trainingRootFs)

		if currentExerciseConfig.IsTextOnly && !currentExerciseConfig.IsOptional {
			continue
		}

		var continueText string
		if currentExerciseConfig.IsTextOnly {
			continueText = "continue"
		} else {
			continueText = "run your solution"
		}

		postActions := internal.Actions{
			{Shortcut: '\n', Action: continueText, ShortcutAliases: []rune{'\r'}},
		}

		if currentExerciseConfig.IsOptional {
			fmt.Println()
			_, _ = color.New(color.Bold, color.FgCyan).Print("This module is optional.")
			fmt.Printf(" You can skip it if you're already familiar with this topic.\n\n")

			postActions = append(postActions, internal.Action{Shortcut: 's', Action: "skip"})
		}

		postActions = append(postActions, internal.Action{Shortcut: 'q', Action: "quit"})

		promptResult = internal.Prompt(postActions, os.Stdin, os.Stdout)
		if promptResult == 'q' {
			os.Exit(0)
		}

		if promptResult == 's' {
			err = h.Skip(ctx)
			if err != nil {
				return err
			}
		}
	}
}

func (h *Handlers) run(ctx context.Context, trainingRootFs *afero.BasePathFs) (bool, error) {
	// todo - validate if exercise id == training exercise id? to ensure about consistency
	success, err := h.runExercise(trainingRootFs)

	if isExerciseNoLongerAvailable(err) {
		fmt.Println(color.YellowString("We did update of the exercise code. Your local workspace is out of sync."))

		if !internal.ConfirmPromptDefaultYes("update your local workspace") {
			os.Exit(0)
		}

		trainingRoot, err := h.config.FindTrainingRoot()
		if err != nil {
			return false, err
		}

		_, err = h.nextExercise(ctx, "", trainingRoot)
		return true, err
	}

	return success, err
}

func isExerciseNoLongerAvailable(err error) bool {
	return status.Code(errors.Cause(err)) == codes.NotFound
}

func (h *Handlers) runExercise(trainingRootFs *afero.BasePathFs) (bool, error) {
	exerciseConfig := h.config.ExerciseConfig(trainingRootFs)

	solutionFiles, err := files.NewFiles().ReadSolutionFiles(trainingRootFs, exerciseConfig.Directory)
	if err != nil {
		return false, err
	}

	if len(solutionFiles) == 0 && !exerciseConfig.IsTextOnly {
		solutionFilesRealPath, err := trainingRootFs.RealPath(exerciseConfig.Directory)
		if err != nil {
			logrus.WithField("exercise_dir", exerciseConfig.Directory).Warn("Can't get realpath of solution")
		}

		hintCommand := "tdl training init " + h.config.TrainingConfig(trainingRootFs).TrainingName
		return false, UserFacingError{
			Msg:          fmt.Sprintf("No solution files found in %s.", solutionFilesRealPath),
			SolutionHint: "Please run " + color.CyanString(hintCommand) + " to init exercise files.",
		}
	}

	req := &genproto.VerifyExerciseRequest{
		ExerciseId: exerciseConfig.ExerciseID,
		Files:      solutionFiles,
		Token:      h.config.GlobalConfig().Token,
	}

	reqStr := strings.ReplaceAll(req.String(), h.config.GlobalConfig().Token, "[token]")
	logrus.WithField("req", reqStr).Info("Request prepared")

	runCtx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	stream, err := h.newGrpcClient().VerifyExercise(runCtx, req)
	if err != nil {
		return false, err
	}

	terminalPath := h.generateRunTerminalPath(trainingRootFs)

	successful := false
	finished := false
	verificationID := ""

	for {
		response, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, errors.Wrap(err, "error response from server")
		}

		if verificationID == "" && response.VerificationId != "" {
			verificationID = response.VerificationId
			logrus.
				WithField("verification_id", verificationID).
				WithField("metadata", response.Metadata).
				Debug("Verification started")
		}

		if len(response.Command) > 0 {
			printCommandWithPath(terminalPath, response.Command)
		}
		if len(response.Stdout) > 0 {
			fmt.Print(response.Stdout)
		}
		if len(response.Stderr) > 0 {
			_, _ = fmt.Fprint(os.Stderr, response.Stderr)
		}
		// todo - support stderr and commands

		if response.Finished {
			if len(response.GetSuiteResult().GetScenarios()) > 0 {
				PrintScenarios(response.GetSuiteResult().GetScenarios())
			}

			fmt.Println("--------")

			if response.Successful {
				if !exerciseConfig.IsTextOnly {
					fmt.Println(color.GreenString("SUCCESS"))
					fmt.Println("\nYou can now see an example solution on the website.")
				}
				successful = true
				finished = true
			} else {
				fmt.Println(color.RedString("FAIL"))
				finished = true
			}
		}

		if response.Finished {
			if response.Notification != "" {
				_, ok := h.notifications[response.Notification]
				if !ok {
					fmt.Println(color.HiYellowString("\n%s", response.Notification))
					h.notifications[response.Notification] = struct{}{}
				}
			} else if !h.solutionHintDisplayed && !response.Successful && response.SolutionAvailable {
				// Legacy behavior
				fmt.Println(color.HiYellowString("\nFeeling stuck? Don't give up! If you want to check the solution, you can now do it on the website."))
				h.solutionHintDisplayed = true
			}
		}
	}

	if !finished {
		return false, errors.New("execution didn't finish")
	} else {
		return successful, nil
	}
}

func printCommand(command string) {
	fmt.Print(color.CyanString("••• ") + command)
}

func printCommandWithPath(root string, command string) {
	fmt.Print(color.CyanString(fmt.Sprintf("••• %s ➜ ", root)) + command)
}

func printlnCommand(command string) {
	printCommand(command + "\n")
}

func (h *Handlers) generateRunTerminalPath(trainingRootFs *afero.BasePathFs) string {
	exerciseConfig := h.config.ExerciseConfig(trainingRootFs)

	wd, err := syscall.Getwd()
	if err != nil {
		logrus.WithError(err).Warn("Can't get wd")
		return "???"
	}

	exerciseDir, err := trainingRootFs.RealPath(exerciseConfig.Directory)
	if err != nil {
		logrus.WithError(err).Warn("Can't get exercise real path")
		return "???"
	}

	terminalPath, err := filepath.Rel(wd, exerciseDir)
	if err != nil {
		logrus.WithError(err).Warn("Can't get relative exercise path")
		return wd
	}

	if terminalPath == exerciseConfig.Directory {
		terminalPath = "./" + terminalPath
	}

	return terminalPath
}

// overrideWithGolden replaces the user's exercise files with the golden solution.
// Unlike syncGoldenSolution (which uses worktrees for branch-based comparison),
// this writes golden files directly to the exercise directory.
func (h *Handlers) overrideWithGolden(ctx context.Context, trainingRootFs *afero.BasePathFs, gitOps *git.Ops, exerciseCfg config.ExerciseConfig) {
	exerciseDir := exerciseCfg.Directory
	moduleExercisePath := exerciseCfg.ModuleExercisePath()

	// Commit uncommitted changes before overriding
	if gitOps.HasUncommittedChanges(exerciseDir) {
		if err := gitOps.AddAll(exerciseDir); err != nil {
			logrus.WithError(err).Warn("Could not stage solution files before override")
		}
		if gitOps.HasStagedChanges() {
			if err := gitOps.Commit(fmt.Sprintf("completed %s", moduleExercisePath)); err != nil {
				logrus.WithError(err).Warn("Could not commit solution before override")
			}
		}
	}

	// Fetch golden solution via gRPC
	resp, err := h.newGrpcClient().GetGoldenSolution(
		ctxWithRequestHeader(ctx, h.cliMetadata),
		&genproto.GetGoldenSolutionRequest{
			TrainingName: h.config.TrainingConfig(trainingRootFs).TrainingName,
			ExerciseId:   exerciseCfg.ExerciseID,
			Token:        h.config.GlobalConfig().Token,
		},
	)
	if err != nil {
		logrus.WithError(err).Warn("Could not fetch golden solution")
		fmt.Println(color.YellowString("  Could not fetch golden solution"))
		return
	}

	// Save user's solution to a timestamped backup branch
	backupBranch := git.BackupBranchName(moduleExercisePath)
	if err := gitOps.CreateBranchFromHead(backupBranch); err != nil {
		logrus.WithError(err).Warn("Could not save solution to backup branch")
	}
	gitOps.PrintInfo(fmt.Sprintf("git branch %s", backupBranch))

	// Write golden files directly to exercise directory
	f := files.NewFilesSilentDeleteUnused()
	if err := f.WriteExerciseFiles(resp.Files, trainingRootFs, exerciseDir); err != nil {
		logrus.WithError(err).Warn("Could not write golden files")
		fmt.Println(color.YellowString("  Could not write golden solution files"))
		return
	}

	// Stage and commit
	if err := gitOps.AddAll(exerciseDir); err != nil {
		logrus.WithError(err).Warn("Could not stage golden override")
		return
	}
	if gitOps.HasStagedChanges() {
		if err := gitOps.Commit(fmt.Sprintf("override with golden solution for %s", moduleExercisePath)); err != nil {
			logrus.WithError(err).Warn("Could not commit golden override")
			return
		}
	}

	fmt.Printf("  Your code saved to branch %s\n", color.MagentaString(backupBranch))
	fmt.Println("  Restore anytime with: " + color.CyanString("git checkout %s -- %s", backupBranch, exerciseDir))
	fmt.Println(color.GreenString("  Your code replaced with golden solution."))
}

// syncGoldenSolution creates a branch with the official solution for comparison.
// Uses git worktree to avoid touching the user's working tree.
// Golden branch is based on HEAD (user's completed commit) so that
// `git diff master..golden -- <dir>` only shows exercise-specific changes.
func (h *Handlers) syncGoldenSolution(ctx context.Context, trainingRootFs *afero.BasePathFs, gitOps *git.Ops, exerciseCfg config.ExerciseConfig, modeOverride string) {
	h.syncGoldenSolutionImpl(ctx, trainingRootFs, gitOps, exerciseCfg, modeOverride, false, time.Time{})
}

func (h *Handlers) syncGoldenSolutionQuiet(ctx context.Context, trainingRootFs *afero.BasePathFs, gitOps *git.Ops, exerciseCfg config.ExerciseConfig, commitDate time.Time) {
	h.syncGoldenSolutionImpl(ctx, trainingRootFs, gitOps, exerciseCfg, "compare", true, commitDate)
}

// syncGoldenSolutionImpl creates a branch with the official solution for comparison.
// Uses git worktree to avoid touching the user's working tree.
// Golden branch is based on HEAD (user's completed commit) so that
// `git diff master..golden -- <dir>` only shows exercise-specific changes.
// When quiet is true, all user-facing output is suppressed (for restore mode).
func (h *Handlers) syncGoldenSolutionImpl(ctx context.Context, trainingRootFs *afero.BasePathFs, gitOps *git.Ops, exerciseCfg config.ExerciseConfig, modeOverride string, quiet bool, commitDate time.Time) {
	if !gitOps.Enabled() {
		return
	}

	exerciseDir := exerciseCfg.Directory
	moduleExercisePath := exerciseCfg.ModuleExercisePath()
	currentBranch, _ := gitOps.CurrentBranch()

	// Ensure solution is committed before creating golden branch
	if gitOps.HasUncommittedChanges(exerciseDir) {
		if err := gitOps.AddAll(exerciseDir); err != nil {
			logrus.WithError(err).Warn("Could not stage solution files for golden sync")
		}
		if gitOps.HasStagedChanges() {
			if err := gitOps.Commit(fmt.Sprintf("completed %s", moduleExercisePath)); err != nil {
				logrus.WithError(err).Warn("Could not commit solution before golden sync")
			}
		}
	}

	// Fetch golden solution via gRPC
	resp, err := h.newGrpcClient().GetGoldenSolution(
		ctxWithRequestHeader(ctx, h.cliMetadata),
		&genproto.GetGoldenSolutionRequest{
			TrainingName: h.config.TrainingConfig(trainingRootFs).TrainingName,
			ExerciseId:   exerciseCfg.ExerciseID,
			Token:        h.config.GlobalConfig().Token,
		},
	)
	if err != nil {
		logrus.WithError(err).Warn("Could not fetch golden solution")
		if !quiet {
			fmt.Println(color.YellowString("  Could not fetch golden solution"))
		}
		return
	}

	// Create golden branch via worktree — based on HEAD for clean diffs.
	goldenBranch := git.GoldenBranchName(moduleExercisePath)
	if gitOps.BranchExists(goldenBranch) {
		if quiet {
			// Quiet mode: silently recreate golden branch
			if err := gitOps.DeleteBranch(goldenBranch); err != nil {
				logrus.WithError(err).Warn("Could not delete existing golden branch")
				return
			}
		} else if !internal.ConfirmPromptDefaultYes(fmt.Sprintf("Branch %s already exists. Delete and recreate?", goldenBranch)) {
			return
		} else {
			if err := gitOps.DeleteBranch(goldenBranch); err != nil {
				logrus.WithError(err).Warn("Could not delete existing golden branch")
				return
			}
		}
	}

	tmpDir, err := os.MkdirTemp("", "tdl-golden-")
	if err != nil {
		logrus.WithError(err).Warn("Could not create temp dir for golden worktree")
		return
	}
	defer os.RemoveAll(tmpDir)

	if err := gitOps.WorktreeAdd(tmpDir, goldenBranch); err != nil {
		logrus.WithError(err).Warn("Could not create golden worktree")
		return
	}
	defer gitOps.WorktreeRemove(tmpDir)

	worktreeExercisePath := filepath.Join(tmpDir, exerciseDir)
	os.MkdirAll(worktreeExercisePath, 0755)

	// Write golden files silently (worktree is internal)
	worktreeFs := afero.NewBasePathFs(afero.NewOsFs(), tmpDir).(*afero.BasePathFs)
	f := files.NewFilesSilent()
	if err := f.WriteExerciseFiles(resp.Files, worktreeFs, exerciseDir); err != nil {
		logrus.WithError(err).Warn("Could not write golden files")
		return
	}

	// Commit on golden branch (quiet — internal operation)
	worktreeOps := git.NewQuietOps(tmpDir)
	if err := worktreeOps.AddAll(exerciseDir); err != nil {
		logrus.WithError(err).Warn("Could not stage golden files")
		return
	}
	goldenCommitMsg := fmt.Sprintf("golden solution for %s", moduleExercisePath)
	if !worktreeOps.HasStagedChanges() {
		// Golden identical to user's solution — create empty commit so the branch
		// exists for comparison (git diff shows nothing, which is correct).
		if err := worktreeOps.CommitAllowEmpty(goldenCommitMsg); err != nil {
			logrus.WithError(err).Warn("Could not create golden commit")
			return
		}
	} else if !commitDate.IsZero() {
		if err := worktreeOps.CommitWithDate(goldenCommitMsg, commitDate); err != nil {
			logrus.WithError(err).Warn("Could not commit golden solution")
			return
		}
	} else {
		if err := worktreeOps.Commit(goldenCommitMsg); err != nil {
			logrus.WithError(err).Warn("Could not commit golden solution")
			return
		}
	}

	if quiet {
		// In quiet mode, just create the branch — no output, no mode application
		return
	}

	// Show simulated fetch for golden (user sees "git fetch" instead of internal worktree details)
	gitOps.PrintInfo(fmt.Sprintf("git fetch cli %s", goldenBranch))
	fmt.Println()

	// Apply golden sync mode
	var goldenMode string
	if modeOverride != "" {
		goldenMode = modeOverride
	} else {
		goldenMode = h.config.TrainingConfig(trainingRootFs).GitGoldenMode
		if goldenMode == "" {
			goldenMode = "compare"
		}
	}

	switch goldenMode {
	case "merge":
		if err := gitOps.Merge(goldenBranch, fmt.Sprintf("merge golden solution for %s", moduleExercisePath)); err != nil {
			fmt.Println(color.YellowString("  Golden merge has conflicts. Resolve them with:"))
			fmt.Println(color.CyanString("    git add -A && git commit"))
		} else {
			fmt.Println(color.GreenString("  Golden solution merged into your branch."))
		}
	default: // "compare"
		// Show diff stat between current branch and golden, restricted to exercise dir
		if stat, err := gitOps.DiffStatPath(currentBranch, goldenBranch, exerciseDir); err == nil && stat != "" {
			fmt.Println(stat)
			fmt.Println()
		}

		fmt.Printf("Compare with our solution: %s\n\n", color.CyanString("git diff %s..%s -- %s", currentBranch, goldenBranch, exerciseDir))
	}
}
