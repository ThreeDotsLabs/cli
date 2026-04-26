package trainings

import (
	"bufio"
	"context"
	"encoding/json"
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
	"golang.org/x/term"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/ThreeDotsLabs/cli/trainings/git"
	mcppkg "github.com/ThreeDotsLabs/cli/trainings/mcp"
)

func (h *Handlers) Run(ctx context.Context, detached bool) error {
	trainingRoot, err := h.config.FindTrainingRoot()
	if errors.Is(err, config.TrainingRootNotFoundError) {
		h.printNotInATrainingDirectory()
		return nil
	}

	trainingRootFs := newTrainingRootFs(trainingRoot)
	printGitNotices(h.config.TrainingConfig(trainingRootFs))

	agentInstructions, err := h.fetchAgentInstructions(ctx, trainingRootFs)
	if err != nil {
		return err
	}

	// MCP auto-detection: prompt to enable if AI coding tools are found.
	// Skipped for detached mode and when --mcp-port 0 explicitly disables MCP.
	// Gated on the server providing agent instructions for this training:
	// if the training has no instructions, skip MCP setup entirely.
	if !detached && h.mcpPort != 0 {
		if len(agentInstructions) == 0 {
			h.mcpPort = 0
			h.loopState = nil
		} else {
			h.configureMCPIfNeeded(trainingRootFs, agentInstructions)
		}
	}

	if detached {
		return h.detachedRun(ctx, trainingRootFs)
	} else {
		return h.interactiveRun(ctx, trainingRootFs)
	}
}

// fetchAgentInstructions asks the server for training-specific agent instructions
// (content for CLAUDE.md / AGENTS.md). Returns nil when the training has none.
// On codes.Unimplemented (old server) it logs a warning and returns nil.
// On codes.NotFound (training has no instructions registered) it silently returns nil.
func (h *Handlers) fetchAgentInstructions(ctx context.Context, trainingRootFs *afero.BasePathFs) ([]byte, error) {
	cfg := h.config.TrainingConfig(trainingRootFs)

	resp, err := h.newGrpcClient().GetAgentInstructions(ctx, &genproto.GetAgentInstructionsRequest{
		TrainingName: cfg.TrainingName,
		Token:        h.config.GlobalConfig().Token,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.Unimplemented:
			logrus.Warn("Server does not support agent instructions, skipping AI companion setup")
			return nil, nil
		case codes.NotFound:
			// Training has no agent instructions registered — that's fine, they're optional.
			return nil, nil
		}
		return nil, errors.Wrap(err, "fetching agent instructions")
	}

	if resp.AgentInstructions == "" {
		return nil, nil
	}
	return []byte(resp.AgentInstructions), nil
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

	_, err = h.nextExercise(withSubAction(ctx, "next"), h.config.ExerciseConfig(trainingRootFs).ExerciseID, trainingRoot)
	if errors.Is(err, errMergeAborted) {
		return nil // clean exit
	}
	if err != nil {
		return err
	}

	return nil
}

func (h *Handlers) interactiveRun(ctx context.Context, trainingRootFs *afero.BasePathFs) error {
	// Start MCP server if enabled
	if h.loopState != nil && h.mcpPort > 0 {
		h.setLoopExerciseInfo(trainingRootFs)

		mcpCtx, mcpCancel := context.WithCancel(ctx)
		defer mcpCancel()

		srv := mcppkg.NewServer(h.loopState, h.mcpPort)
		if err := srv.Start(mcpCtx); err != nil {
			logrus.WithError(err).Warn("Failed to start MCP server")
		} else {
			fmt.Printf("%s\n", color.HiBlackString("MCP server listening on %s", srv.Addr()))
		}
	}

	// Single stdin reader goroutine for the entire interactive session.
	// Only needed when MCP is active (to select between stdin and MCP commands).
	if h.loopState != nil {
		// Set terminal to raw mode once for the entire session so the goroutine
		// always reads in raw mode (ICANON off). Without this, the goroutine
		// can block in a cooked-mode read between prompts, and term.MakeRaw
		// called later in waitForAction may not interrupt that blocked syscall.
		// Re-enable output processing (OPOST) so test output is not garbled.
		if sessState, err := term.MakeRaw(0); err == nil {
			_ = internal.EnableOutputProcessing(0)
			h.sessionTermState = sessState
			defer func() {
				_ = term.Restore(0, sessState)
				h.sessionTermState = nil
			}()
		}

		ch := make(chan rune, 1)
		go func() {
			defer close(ch)
			reader := bufio.NewReader(os.Stdin)
			for {
				r, _, err := reader.ReadRune()
				if err != nil {
					if err == io.EOF && internal.IsStdinTerminal() {
						reader = bufio.NewReader(os.Stdin)
						continue
					}
					return
				}
				if r == '\x03' {
					// Ctrl+C: restore terminal before exiting so the shell is
					// not left in raw mode (os.Exit bypasses deferred restores).
					h.restoreTerminal()
					os.Exit(0)
				}
				ch <- r
			}
		}()
		h.stdinCh = ch
		defer func() { h.stdinCh = nil }()
	}

	// Background poller: long `tr run` sessions (days/weeks) can outlive the
	// one-shot update check that runs at CLI startup. This goroutine keeps
	// checking periodically so a newer release is still surfaced mid-session.
	// Dev builds skip this unless --force-update-prompt is set for testing.
	if h.cliMetadata.Version != "" && (h.cliMetadata.Version != "dev" || h.cliMetadata.ForceUpdatePrompt) {
		go h.backgroundUpdateCheck(ctx)
	}

	retries := 0
	mergeAborted := false

	for {
		if !mergeAborted {
			h.setLoopState(mcppkg.StateRunning)
			if h.loopState != nil {
				h.loopState.ClearLastError()
			}

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
				userErr := formatServerError(err)
				fmt.Println(color.RedString("Failed to execute solution: %s", userErr))

				if h.loopState != nil {
					errMsg := fmt.Sprintf("Training CLI error: %s", userErr)
					h.loopState.SetLastError(errMsg)
					fmt.Fprintln(h.loopState.OutputBuffer(), errMsg)
				}

				h.setLoopState(mcppkg.StateFailed)
				failActions, failActionMap := h.augmentActionsWithUpdate(
					internal.Actions{
						{Shortcut: '\n', Action: "run solution again", ShortcutAliases: []rune{'\r'}},
						{Shortcut: 'q', Action: "quit"},
					},
					map[rune]loopAction{
						'\n': loopActionRun,
						'q':  loopActionQuit,
					},
				)
				action, fromMCP := h.waitForAction(
					failActions,
					failActionMap,
					map[mcppkg.CommandType]loopAction{
						mcppkg.CmdRunSolution:   loopActionRun,
						mcppkg.CmdResetExercise: loopActionResetExercise,
					},
				)
				ctx = withMCPTriggered(ctx, fromMCP)
				if action == loopActionQuit {
					return userErr
				}
				if action == loopActionUpdate {
					h.handleUpdateAction(ctx)
					continue
				}
				if action == loopActionResetExercise {
					if err := h.resetExerciseFromLoop(ctx, trainingRootFs); err != nil {
						if errors.Is(err, errBackupAborted) {
							h.sendPendingMCPResult(mcppkg.MCPResult{Error: "Reset aborted: backup branch creation failed and user declined to continue"})
						} else {
							h.sendPendingMCPResult(mcppkg.MCPResult{Error: fmt.Sprintf("Reset failed: %v", err)})
						}
					}
					continue
				}
				continue
			}

			if !successful {
				// When stuck (10+ failures), create example solution branch or remind about it
				if h.solutionAvailable {
					gitOps := h.newGitOps()
					exerciseCfg := h.config.ExerciseConfig(trainingRootFs)
					if gitOps.Enabled() && !exerciseCfg.IsTextOnly {
						goldenBranch := git.GoldenBranchName(exerciseCfg.ModuleExercisePath())
						if !gitOps.BranchExists(goldenBranch) {
							h.syncGoldenSolution(withSubAction(ctx, "sync-golden-stuck"), trainingRootFs, gitOps, exerciseCfg, "compare", time.Now().Add(1*time.Second))
						} else {
							// Branch already exists — remind the user how to compare
							currentBranch, _ := gitOps.CurrentBranch()
							fmt.Printf("\nCompare with our solution: %s\n\n", color.CyanString("git diff %s..%s -- %s", currentBranch, goldenBranch, compareDir(gitOps, exerciseCfg.Directory)))
						}
					}
				}

				h.setLoopState(mcppkg.StateFailed)
				failActions, failActionMap := h.augmentActionsWithUpdate(
					internal.Actions{
						{Shortcut: '\n', Action: "run solution again", ShortcutAliases: []rune{'\r'}},
						{Shortcut: 'q', Action: "quit"},
					},
					map[rune]loopAction{
						'\n': loopActionRun,
						'q':  loopActionQuit,
					},
				)
				action, fromMCP := h.waitForAction(
					failActions,
					failActionMap,
					map[mcppkg.CommandType]loopAction{
						mcppkg.CmdRunSolution:   loopActionRun,
						mcppkg.CmdResetExercise: loopActionResetExercise,
					},
				)
				ctx = withMCPTriggered(ctx, fromMCP)
				if action == loopActionQuit {
					return nil
				}
				if action == loopActionUpdate {
					h.handleUpdateAction(ctx)
					continue
				}
				if action == loopActionResetExercise {
					if err := h.resetExerciseFromLoop(ctx, trainingRootFs); err != nil {
						if errors.Is(err, errBackupAborted) {
							h.sendPendingMCPResult(mcppkg.MCPResult{Error: "Reset aborted: backup branch creation failed and user declined to continue"})
						} else {
							h.sendPendingMCPResult(mcppkg.MCPResult{Error: fmt.Sprintf("Reset failed: %v", err)})
						}
					}
					continue
				}
				continue
			}
		}
		mergeAborted = false

		// Build actions dynamically based on git config
		actions := internal.Actions{
			{Shortcut: '\n', Action: "go to the next exercise", ShortcutAliases: []rune{'\r'}},
		}
		actionMap := map[rune]loopAction{
			'\n': loopActionNextExercise,
		}

		gitOps := h.newGitOps()
		trainingRoot, err := h.config.FindTrainingRoot()
		if err != nil {
			return err
		}
		cfg := h.config.TrainingConfig(trainingRootFs)
		exerciseCfg := h.config.ExerciseConfig(trainingRootFs)

		if gitOps.Enabled() && !exerciseCfg.IsTextOnly && !cfg.GitAutoGolden {
			actions = append(actions, internal.Action{Shortcut: 's', Action: "sync with example solution"})
			actionMap['s'] = loopActionSyncSolution
		}
		actions = append(actions,
			internal.Action{Shortcut: 'r', Action: "re-run solution"},
			internal.Action{Shortcut: 'q', Action: "quit"},
		)
		actionMap['r'] = loopActionRun
		actionMap['q'] = loopActionQuit

		h.setLoopState(mcppkg.StateSucceeded)
		actions, actionMap = h.augmentActionsWithUpdate(actions, actionMap)
		chosenAction, fromMCP := h.waitForAction(
			actions,
			actionMap,
			map[mcppkg.CommandType]loopAction{
				mcppkg.CmdRunSolution:         loopActionRun,
				mcppkg.CmdNextExercise:        loopActionNextExercise,
				mcppkg.CmdSyncAndNextExercise: loopActionSyncSolution,
				mcppkg.CmdResetExercise:       loopActionResetExercise,
			},
		)
		ctx = withMCPTriggered(ctx, fromMCP)

		if chosenAction == loopActionQuit {
			h.restoreTerminal()
			os.Exit(0)
		}
		if chosenAction == loopActionUpdate {
			h.handleUpdateAction(ctx)
			continue
		}
		if chosenAction == loopActionRun {
			continue
		}
		if chosenAction == loopActionResetExercise {
			if err := h.resetExerciseFromLoop(ctx, trainingRootFs); err != nil {
				if errors.Is(err, errBackupAborted) {
					h.sendPendingMCPResult(mcppkg.MCPResult{Error: "Reset aborted: backup branch creation failed and user declined to continue"})
				} else {
					h.sendPendingMCPResult(mcppkg.MCPResult{Error: fmt.Sprintf("Reset failed: %v", err)})
				}
			}
			continue
		}

		// Starting an advance cycle — clear any stale transition content so the
		// sync/override step (below) and nextExercise both build on a clean slate.
		// Must happen before overrideWithGolden; otherwise its captured sync diff
		// would be wiped before nextExercise can append the scaffold diff.
		if h.loopState != nil {
			h.loopState.ClearTransitionContent()
		}

		if chosenAction == loopActionSyncSolution {
			h.overrideWithGolden(withSubAction(ctx, "sync-golden-manual"), trainingRootFs, gitOps, exerciseCfg)
			// Fall through to next exercise (example solution already committed, no staged changes)
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

		// Auto-sync: override with example solution automatically after passing
		if cfg.GitAutoGolden && gitOps.Enabled() && !exerciseCfg.IsTextOnly && chosenAction != loopActionSyncSolution {
			h.overrideWithGolden(withSubAction(ctx, "sync-golden-auto"), trainingRootFs, gitOps, exerciseCfg)
		}

		// Create example solution branch for comparison (skip if user synced or auto-sync ran)
		if gitOps.Enabled() && !exerciseCfg.IsTextOnly && !cfg.GitAutoGolden && chosenAction != loopActionSyncSolution {
			goldenBranch := git.GoldenBranchName(exerciseCfg.ModuleExercisePath())
			if !gitOps.BranchExists(goldenBranch) {
				h.syncGoldenSolution(withSubAction(ctx, "sync-golden-auto"), trainingRootFs, gitOps, exerciseCfg, "compare", time.Now().Add(1*time.Second))
			}

			// Capture comparison diffstat for MCP when not syncing
			if h.loopState != nil && gitOps.BranchExists(goldenBranch) {
				if stat, err := gitOps.DiffStatPathPlain("HEAD", goldenBranch, exerciseCfg.Directory); err == nil && stat != "" {
					var content strings.Builder
					content.WriteString("## Files updated in your workspace\n")
					content.WriteString(stat)
					content.WriteString("\n")
					h.loopState.SetTransitionContent(content.String())
				}
			}
		}

		h.setLoopState(mcppkg.StateAdvancing)

		finished, err := h.nextExercise(withSubAction(ctx, "next"), exerciseCfg.ExerciseID, trainingRoot)
		if errors.Is(err, errMergeAborted) {
			h.sendPendingMCPResult(mcppkg.MCPResult{Error: "Merge aborted by user. Staying on current exercise."})
			mergeAborted = true
			continue // stay on current exercise, re-show prompt
		}
		if err != nil {
			h.sendPendingMCPResult(mcppkg.MCPResult{Error: fmt.Sprintf("Failed to advance: %v", err)})
			return err
		}
		if finished {
			h.sendPendingMCPResult(mcppkg.MCPResult{Success: true, Message: "Training finished!"})
			return nil
		}

		h.setLoopExerciseInfo(trainingRootFs)
		h.sendPendingMCPResult(h.buildAdvanceResult())

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

		if h.loopState != nil {
			// MCP mode: auto-continue. The MCP client already triggered
			// the advance — no reason to block before running the solution.
			continue
		}
		promptResult := internal.Prompt(postActions, os.Stdin, os.Stdout)
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
	success, err := h.runExercise(ctx, trainingRootFs)

	if isExerciseNoLongerAvailable(err) {
		fmt.Println(color.YellowString("We did update of the exercise code. Your local workspace is out of sync."))

		if h.loopState == nil {
			if !internal.ConfirmPromptDefaultYes("update your local workspace") {
				os.Exit(0)
			}
		}
		// MCP mode: auto-accept update — exercise must be refreshed.

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

func (h *Handlers) runExercise(ctx context.Context, trainingRootFs *afero.BasePathFs) (bool, error) {
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

	runCtx, cancel := context.WithTimeout(withSubAction(ctx, "verify"), time.Second*30)
	defer cancel()

	stream, err := h.newGrpcClient().VerifyExercise(runCtx, req)
	if err != nil {
		return false, err
	}

	terminalPath := h.generateRunTerminalPath(trainingRootFs)

	// Set up output capture for MCP log buffer
	stdoutW := io.Writer(os.Stdout)
	stderrW := io.Writer(os.Stderr)
	if h.loopState != nil {
		h.loopState.OutputBuffer().Reset()
		stdoutW = io.MultiWriter(os.Stdout, h.loopState.OutputBuffer())
		stderrW = io.MultiWriter(os.Stderr, h.loopState.OutputBuffer())
	}

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
			cmdStr := fmt.Sprintf("%s%s", color.CyanString(fmt.Sprintf("••• %s ➜ ", terminalPath)), response.Command)
			fmt.Fprint(stdoutW, cmdStr)
		}
		if len(response.Stdout) > 0 {
			fmt.Fprint(stdoutW, response.Stdout)
		}
		if len(response.Stderr) > 0 {
			fmt.Fprint(stderrW, response.Stderr)
		}

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
			if response.SolutionAvailable {
				h.solutionAvailable = true
				h.stuckRunCount++
			}

			if response.Notification != "" {
				if response.SolutionAvailable {
					// Show periodically when stuck (first time + every 3 attempts)
					if h.stuckRunCount%3 == 1 {
						fmt.Println(color.HiYellowString("\n%s", response.Notification))
					}
				} else {
					_, ok := h.notifications[response.Notification]
					if !ok {
						fmt.Println(color.HiYellowString("\n%s", response.Notification))
						h.notifications[response.Notification] = struct{}{}
					}
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

// loopAction represents what the interactive loop should do next.
type loopAction int

const (
	loopActionRun           loopAction = iota // Run (or re-run) the solution
	loopActionNextExercise                    // Advance to next exercise
	loopActionSyncSolution                    // Sync with example solution
	loopActionQuit                            // Quit the loop
	loopActionResetExercise                   // Reset exercise to clean files
	loopActionUpdate                          // Update the CLI binary and exit
)

// backgroundUpdateCheck polls for a newer CLI release while the interactive
// loop is alive. Runs until ctx is cancelled (e.g. user quits the loop).
func (h *Handlers) backgroundUpdateCheck(ctx context.Context) {
	check := func() {
		available, version, notes := internal.CheckUpdateAvailable(h.cliMetadata.Version, h.cliMetadata.ForceUpdatePrompt)
		if available {
			h.setUpdateAvailable(version, notes)
			logrus.WithField("version", version).Debug("Background update check: update available")
		}
	}

	check() // immediate check so a user who launched right after a release sees it quickly

	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}

// augmentActionsWithUpdate appends a 'u' action to the given prompt when a
// background update check has flagged a newer release. The yellow notice
// line is printed above the prompt only on the first prompt after detection
// (per ShouldShowUpdateNoticeCLI); the 'u' action stays in the list on
// every subsequent prompt so the user can trigger the update anytime.
//
// Returns fresh action/map copies; the caller's originals are not mutated.
func (h *Handlers) augmentActionsWithUpdate(actions internal.Actions, actionMap map[rune]loopAction) (internal.Actions, map[rune]loopAction) {
	available, version, releaseNotes := h.getUpdateAvailable()
	if !available {
		return actions, actionMap
	}

	if h.shouldShowUpdateNoticeCLI() {
		fmt.Println()
		_, _ = color.New(color.FgHiYellow).Printf(
			"A new CLI version is available: %s → %s. Press 'u' to update.\n",
			h.cliMetadata.Version, version,
		)
		if formatted := internal.FormatReleaseNotes(releaseNotes, 15); formatted != "" {
			fmt.Println()
			fmt.Println("Release notes:")
			fmt.Println(formatted)
		}
		h.markUpdateNoticeShownCLI()
	}

	newActions := make(internal.Actions, 0, len(actions)+1)
	newActions = append(newActions, actions...)
	newActions = append(newActions, internal.Action{Shortcut: 'u', Action: "update CLI (will exit)"})

	newMap := make(map[rune]loopAction, len(actionMap)+1)
	for k, v := range actionMap {
		newMap[k] = v
	}
	newMap['u'] = loopActionUpdate

	return newActions, newMap
}

// handleUpdateAction runs the self-update flow and exits on success.
// On failure it prints the error and returns, letting the caller continue
// the interactive loop with the current CLI version.
func (h *Handlers) handleUpdateAction(ctx context.Context) {
	fmt.Println()
	if err := internal.RunUpdate(ctx, h.cliMetadata.Version, internal.UpdateOptions{
		SkipConfirm: true,
		ForceUpdate: true,
	}); err != nil {
		fmt.Println(color.RedString("Update failed: %v", err))
		fmt.Println(color.HiBlackString("Continuing with current version..."))
		return
	}
	fmt.Println()
	fmt.Println("Please re-run your command.")
	h.restoreTerminal()
	os.Exit(0)
}

// waitForAction prints a prompt and waits for input from either stdin or the MCP command channel.
// When MCP is disabled (loopState == nil), it reads stdin synchronously (like internal.Prompt).
// When MCP is enabled, stdinCh must be a long-lived channel fed by a single goroutine in interactiveRun.
func (h *Handlers) waitForAction(
	actions internal.Actions,
	actionMap map[rune]loopAction,
	validMCPCmds map[mcppkg.CommandType]loopAction,
) (loopAction, bool) {
	if h.loopState == nil {
		// No MCP — delegate to promptRune for the actual reading.
		key := h.promptRune(actions)
		if action, ok := actionMap[key]; ok {
			return action, false
		}
		return loopActionQuit, false
	}

	// MCP mode — need to select on both stdinCh and MCP commands.
	defer fmt.Println()
	printPrompt(actions)

	termState, rawErr := term.MakeRaw(0)
	if rawErr == nil {
		defer term.Restore(0, termState)
	}

	drainChannel(h.stdinCh)

	for {
		select {
		case ch, ok := <-h.stdinCh:
			if !ok {
				h.restoreTerminal()
				logrus.Debug("stdin closed, exiting")
				fmt.Println(color.HiBlackString("Input closed — exiting."))
				os.Exit(0)
			}
			if key, ok := actions.ReadKeyFromInput(ch); ok {
				if action, mapped := actionMap[key]; mapped {
					return action, false
				}
			}
		case cmd := <-h.loopState.CommandCh():
			if mappedAction, ok := validMCPCmds[cmd.Type]; ok {
				if cmd.Type == mcppkg.CmdNextExercise || cmd.Type == mcppkg.CmdSyncAndNextExercise || cmd.Type == mcppkg.CmdResetExercise {
					// Defer result until the full operation completes (blocking for MCP client).
					h.pendingMCPResultCh = cmd.ResultCh
				} else {
					cmd.ResultCh <- mcppkg.MCPResult{Success: true, Message: "command accepted"}
				}
				return mappedAction, true
			}
			cmd.ResultCh <- mcppkg.MCPResult{
				Error: fmt.Sprintf("command '%s' not valid in current state (%s)", cmd.Type, h.loopState.GetState()),
			}
		}
	}
}

// printPrompt formats and prints the action prompt line.
func printPrompt(actions internal.Actions) {
	var actionsStr []string
	for _, action := range actions {
		actionsStr = append(actionsStr, fmt.Sprintf(
			"%s to %s",
			color.New(color.Bold).Sprint(action.KeyString()),
			action.Action,
		))
	}
	fmt.Printf("%s", "Press "+formatActionsMessage(actionsStr)+" ")
}

// drainChannel discards any buffered values from a channel.
func drainChannel(ch <-chan rune) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

func (h *Handlers) sendPendingMCPResult(result mcppkg.MCPResult) {
	if h.pendingMCPResultCh != nil {
		h.pendingMCPResultCh <- result
		h.pendingMCPResultCh = nil
	}
}

func (h *Handlers) buildAdvanceResult() mcppkg.MCPResult {
	if h.loopState == nil {
		return mcppkg.MCPResult{}
	}
	info := h.loopState.GetExerciseInfo()
	content := h.loopState.GetTransitionContent()

	resp := struct {
		Status       string `json:"status"`
		ExerciseID   string `json:"exercise_id"`
		Directory    string `json:"directory"`
		ModuleName   string `json:"module_name"`
		ExerciseName string `json:"exercise_name"`
		Content      string `json:"content,omitempty"`
	}{
		Status:       "advanced",
		ExerciseID:   info.ExerciseID,
		Directory:    info.Directory,
		ModuleName:   info.ModuleName,
		ExerciseName: info.ExerciseName,
		Content:      content,
	}

	data, _ := json.MarshalIndent(resp, "", "  ")
	return mcppkg.MCPResult{Success: true, Message: string(data)}
}

// resetExerciseFromLoop performs a clean-files reset of the current exercise,
// then sends the deferred MCP result. Called from the interactive loop when
// CmdResetExercise is received.
func (h *Handlers) resetExerciseFromLoop(ctx context.Context, trainingRootFs *afero.BasePathFs) error {
	exerciseCfg := h.config.ExerciseConfig(trainingRootFs)
	gitOps := h.newGitOps()

	if !gitOps.Enabled() || exerciseCfg.IsTextOnly || exerciseCfg.Directory == "" {
		return fmt.Errorf("cannot reset: git is not enabled or exercise is text-only")
	}

	moduleExercisePath := exerciseCfg.ModuleExercisePath()
	initBranch := git.InitBranchName(moduleExercisePath)

	if !gitOps.BranchExists(initBranch) {
		return fmt.Errorf("cannot reset: init branch %s does not exist", initBranch)
	}

	// Auto-commit uncommitted changes before reset
	if gitOps.HasUncommittedChanges(exerciseCfg.Directory) {
		saveProgress(gitOps, exerciseCfg.Directory, fmt.Sprintf("save progress on %s", moduleExercisePath))
	}

	// Perform the clean-files reset — fetches fresh scaffold+golden and writes the start state.
	backupBranch, err := h.resetCleanFiles(ctx, gitOps, trainingRootFs, exerciseCfg.ExerciseID, moduleExercisePath, exerciseCfg.Directory)
	if err != nil {
		return err
	}

	// Send deferred MCP result with reset details
	h.sendPendingMCPResult(h.buildResetResult(backupBranch, exerciseCfg))

	return nil
}

func (h *Handlers) buildResetResult(backupBranch string, exerciseCfg config.ExerciseConfig) mcppkg.MCPResult {
	resp := struct {
		Status       string `json:"status"`
		ExerciseID   string `json:"exercise_id"`
		Directory    string `json:"directory"`
		ModuleName   string `json:"module_name"`
		ExerciseName string `json:"exercise_name"`
		BackupBranch string `json:"backup_branch,omitempty"`
	}{
		Status:       "reset",
		ExerciseID:   exerciseCfg.ExerciseID,
		Directory:    exerciseCfg.Directory,
		ModuleName:   exerciseCfg.ModuleName,
		ExerciseName: exerciseCfg.ExerciseName,
		BackupBranch: backupBranch,
	}

	data, _ := json.MarshalIndent(resp, "", "  ")
	return mcppkg.MCPResult{Success: true, Message: string(data)}
}

func formatActionsMessage(actionsStr []string) string {
	switch len(actionsStr) {
	case 0:
		return ""
	case 1:
		return actionsStr[0]
	default:
		return strings.Join(actionsStr[:len(actionsStr)-1], ", ") + " or " + actionsStr[len(actionsStr)-1]
	}
}

func (h *Handlers) setLoopExerciseInfo(trainingRootFs *afero.BasePathFs) {
	if h.loopState == nil {
		return
	}
	cfg := h.config.ExerciseConfig(trainingRootFs)
	h.loopState.SetExerciseInfo(mcppkg.ExerciseInfo{
		ExerciseID:   cfg.ExerciseID,
		Directory:    cfg.Directory,
		IsTextOnly:   cfg.IsTextOnly,
		IsOptional:   cfg.IsOptional,
		ModuleName:   cfg.ModuleName,
		ExerciseName: cfg.ExerciseName,
	})
}

func (h *Handlers) setLoopState(state mcppkg.ExerciseState) {
	if h.loopState == nil {
		return
	}
	h.loopState.SetState(state)
}

// compareDir returns exerciseDir relative to the user's cwd, so that the
// displayed "git diff ... -- <path>" command works when copied from any directory.
func compareDir(gitOps *git.Ops, exerciseDir string) string {
	wd, err := os.Getwd()
	if err != nil {
		return exerciseDir
	}
	rel, err := filepath.Rel(wd, filepath.Join(gitOps.RootDir(), exerciseDir))
	if err != nil {
		return exerciseDir
	}
	return rel
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

// overrideWithGolden replaces the user's exercise files with the example solution.
// Unlike syncGoldenSolution (which uses worktrees for branch-based comparison),
// this writes example solution files directly to the exercise directory.
//
// IMPORTANT: This is a destructive operation — user's code is overwritten.
// We MUST save their work to a backup branch before replacing files.
// The user explicitly chose this action ('s' key).
//
// INVARIANT: after this returns, exerciseDir is 1:1 with the example solution —
// any stale user files not in the golden are deleted. Enforced via
// replaceExerciseFilesAndCommit → replaceExerciseFiles. Do not replace this
// with a direct WriteExerciseFiles call; see exercise_replace.go for why.
func (h *Handlers) overrideWithGolden(ctx context.Context, trainingRootFs *afero.BasePathFs, gitOps *git.Ops, exerciseCfg config.ExerciseConfig) {
	exerciseDir := exerciseCfg.Directory
	moduleExercisePath := exerciseCfg.ModuleExercisePath()

	// Commit uncommitted changes before overriding so they end up in git history.
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

	// Fetch example solution via gRPC
	goldenFiles, err := h.fetchGoldenFiles(
		ctx,
		h.config.TrainingConfig(trainingRootFs).TrainingName,
		exerciseCfg.ExerciseID,
		h.config.GlobalConfig().Token,
	)
	if err != nil {
		logrus.WithError(err).Warn("Could not fetch golden solution")
		fmt.Println(color.YellowString("  Could not fetch example solution"))
		return
	}

	backupBranch := git.BackupBranchName(moduleExercisePath)
	goldenCommitted, err := replaceExerciseFilesAndCommit(
		gitOps, trainingRootFs, goldenFiles, exerciseDir, backupBranch,
		fmt.Sprintf("override with example solution for %s", moduleExercisePath),
	)
	if err != nil {
		if errors.Is(err, errBackupAborted) {
			fmt.Println(color.YellowString("  Aborting example solution override to protect your code."))
			return
		}
		logrus.WithError(err).Warn("Could not override with example solution")
		fmt.Println(color.YellowString("  Could not write example solution files"))
		return
	}
	fmt.Printf("  Your code saved to branch %s\n", color.MagentaString(backupBranch))
	fmt.Println("  Restore anytime with: " + color.CyanString("git checkout %s -- %s", backupBranch, exerciseDir))

	fmt.Println(color.GreenString("  Your code replaced with example solution."))

	// Capture the sync diffstat (student → golden) so the MCP client can surface it.
	// Only the stat — the full diff is noise; the student can run `git diff` themselves
	// if they want the full body.
	if h.loopState != nil && goldenCommitted {
		var content strings.Builder
		if stat, err := gitOps.DiffStatPathPlain(backupBranch, "HEAD", exerciseDir); err == nil && stat != "" {
			content.WriteString("## Sync: your code vs example solution\n")
			content.WriteString(stat)
			content.WriteString("\n")
		}
		if s := content.String(); s != "" {
			h.loopState.SetTransitionContent(s)
			logrus.WithField("bytes", len(s)).Debug("Sync diff captured")
		} else {
			logrus.Info("Sync diff was empty (no changes between student code and golden)")
		}
	}
}

// syncGoldenSolution creates a branch with the example solution for comparison.
// Uses git worktree to avoid touching the user's working tree.
// Example solution branch is based on HEAD (user's completed commit) so that
// `git diff master..example -- <dir>` only shows exercise-specific changes.
func (h *Handlers) syncGoldenSolution(ctx context.Context, trainingRootFs *afero.BasePathFs, gitOps *git.Ops, exerciseCfg config.ExerciseConfig, modeOverride string, commitDate time.Time) {
	h.syncGoldenSolutionImpl(ctx, trainingRootFs, gitOps, exerciseCfg, modeOverride, false, commitDate)
}

func (h *Handlers) syncGoldenSolutionQuiet(ctx context.Context, trainingRootFs *afero.BasePathFs, gitOps *git.Ops, exerciseCfg config.ExerciseConfig, commitDate time.Time) {
	h.syncGoldenSolutionImpl(ctx, trainingRootFs, gitOps, exerciseCfg, "compare", true, commitDate)
}

// syncGoldenSolutionImpl creates a branch with the example solution for comparison.
// Uses git worktree to avoid touching the user's working tree.
// Example solution branch is based on HEAD (user's completed commit) so that
// `git diff master..example -- <dir>` only shows exercise-specific changes.
// When quiet is true, all user-facing output is suppressed (for restore mode).
func (h *Handlers) syncGoldenSolutionImpl(ctx context.Context, trainingRootFs *afero.BasePathFs, gitOps *git.Ops, exerciseCfg config.ExerciseConfig, modeOverride string, quiet bool, commitDate time.Time) {
	if !gitOps.Enabled() {
		return
	}

	exerciseDir := exerciseCfg.Directory
	moduleExercisePath := exerciseCfg.ModuleExercisePath()
	currentBranch, _ := gitOps.CurrentBranch()

	// Ensure solution is committed before creating example solution branch
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

	// Fetch example solution via gRPC
	resp, err := h.newGrpcClient().GetGoldenSolution(
		ctx,
		&genproto.GetGoldenSolutionRequest{
			TrainingName: h.config.TrainingConfig(trainingRootFs).TrainingName,
			ExerciseId:   exerciseCfg.ExerciseID,
			Token:        h.config.GlobalConfig().Token,
		},
	)
	if err != nil {
		logrus.WithError(err).Warn("Could not fetch golden solution")
		if !quiet {
			fmt.Println(color.YellowString("  Could not fetch example solution"))
		}
		return
	}

	// Create example solution branch via worktree — based on HEAD for clean diffs.
	goldenBranch := git.GoldenBranchName(moduleExercisePath)
	if gitOps.BranchExists(goldenBranch) {
		if quiet {
			// Quiet mode: silently recreate example solution branch
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

	// Clean exercise directory to prevent stale files from previous exercises
	// leaking onto the golden branch (e.g., files from later modules that
	// share the same project directory).
	os.RemoveAll(worktreeExercisePath)
	os.MkdirAll(worktreeExercisePath, 0755)

	// Restore base state from init branch (accumulated scaffold files: tests,
	// go.mod, config, etc.) so the golden branch has a complete project state.
	initBranch := git.InitBranchName(moduleExercisePath)
	if gitOps.BranchExists(initBranch) {
		worktreeInitOps := git.NewQuietOps(tmpDir)
		if err := worktreeInitOps.CheckoutPathFrom(initBranch, exerciseDir); err != nil {
			logrus.WithError(err).Debug("Could not restore init files for golden branch")
		}
	}

	// Write example solution files silently (worktree is internal)
	worktreeFs := afero.NewBasePathFs(afero.NewOsFs(), tmpDir).(*afero.BasePathFs)
	f := files.NewFilesSilent()
	if err := f.WriteExerciseFiles(resp.Files, worktreeFs, exerciseDir); err != nil {
		logrus.WithError(err).Warn("Could not write golden files")
		return
	}

	// Commit on example solution branch (quiet — internal operation)
	worktreeOps := git.NewQuietOps(tmpDir)
	if err := worktreeOps.AddAll(exerciseDir); err != nil {
		logrus.WithError(err).Warn("Could not stage golden files")
		return
	}
	goldenCommitMsg := fmt.Sprintf("example solution for %s", moduleExercisePath)

	if !worktreeOps.HasStagedChanges() {
		// Example solution identical to user's solution — create empty commit so the branch
		// exists for comparison (git diff shows nothing, which is correct).
		if err := worktreeOps.CommitAllowEmptyWithDate(goldenCommitMsg, commitDate); err != nil {
			logrus.WithError(err).Warn("Could not create golden commit")
			return
		}
	} else {
		if err := worktreeOps.CommitWithDate(goldenCommitMsg, commitDate); err != nil {
			logrus.WithError(err).Warn("Could not commit golden solution")
			return
		}
	}

	if quiet {
		// In quiet mode, just create the branch — no output, no mode application
		return
	}

	// Show simulated fetch for example solution (user sees "git fetch" instead of internal worktree details)
	gitOps.PrintInfo(fmt.Sprintf("git fetch cli %s", goldenBranch))
	fmt.Println()

	// Apply sync mode
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
		if err := gitOps.Merge(goldenBranch, fmt.Sprintf("merge example solution for %s", moduleExercisePath)); err != nil {
			fmt.Println(color.YellowString("  Example solution merge has conflicts. Resolve them with:"))
			fmt.Println(color.CyanString("    git add -A && git commit"))
		} else {
			fmt.Println(color.GreenString("  Example solution merged into your branch."))
		}
	default: // "compare"
		fmt.Printf("Compare with our solution: %s\n\n", color.CyanString("git diff %s..%s -- %s", currentBranch, goldenBranch, compareDir(gitOps, exerciseDir)))
	}
}
