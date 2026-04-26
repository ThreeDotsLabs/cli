package trainings

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

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

func (h *Handlers) Init(ctx context.Context, trainingName string, dir string, noGit bool, forceGit bool) error {
	logrus.WithFields(logrus.Fields{
		"training_name": trainingName,
	}).Debug("Starting training")

	wd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}
	trainingRootDir := path.Join(wd, dir)

	// trainingRootDir may be different when doing init in already existing workspace
	trainingRootDir, alreadyInitialized, previousSolutionsAvailable, err := h.startTraining(ctx, trainingName, trainingRootDir)
	if errors.Is(err, ErrInterrupted) {
		fmt.Println("Interrupted")
		return nil
	} else if err != nil {
		return err
	}

	if alreadyInitialized {
		trainingRootFs := newTrainingRootFs(trainingRootDir)
		if files.DirOrFileExists(trainingRootFs, ".tdl-exercise") {
			fmt.Println("Training is already initialised, nothing to do.")
			return nil
		}
		// Partial init: exercise not yet set up, fall through to nextExercise.
	}

	// Git integration: init repo, configure preferences, initial commit.
	// Always attempt git detection; terminal is only needed for the "git missing" prompt.
	gitOps := git.NewOps(trainingRootDir, noGit)
	gitWasUnavailable := false

	if gitOps.Enabled() {
		detectedVersion, err := git.CheckVersion()
		if err != nil {
			var notInstalled *git.GitNotInstalledError
			var tooOld *git.GitTooOldError
			if errors.As(err, &notInstalled) {
				if internal.IsStdinTerminal() {
					printGitUnavailableNotice("Git is not installed.", git.InstallHint(runtime.GOOS))
					_ = internal.RecordGitInstallNoticeShown()
					if !promptContinueWithoutGit() {
						return nil
					}
				}
				gitOps = git.NewOps(trainingRootDir, true)
				gitWasUnavailable = true
			} else if errors.As(err, &tooOld) {
				if internal.IsStdinTerminal() {
					reason := fmt.Sprintf("Your git version (%s) is too old: %s or newer is required.", tooOld.Detected, tooOld.Required)
					printGitUnavailableNotice(reason, git.InstallHint(runtime.GOOS))
					_ = internal.RecordGitInstallNoticeShown()
					if !promptContinueWithoutGit() {
						return nil
					}
				}
				gitOps = git.NewOps(trainingRootDir, true)
				gitWasUnavailable = true
			} else {
				// Unparseable version — warn in logs, don't block
				logrus.WithError(err).Warn("Could not verify git version")
			}
		} else if !detectedVersion.AtLeast(git.RecommendedVersion) {
			// Above minimum but below recommended — soft info
			fmt.Printf("  %s Git %s detected (minimum: %s). Upgrade to %s for conflict preview when loading exercises.\n\n",
				color.YellowString("ℹ"),
				detectedVersion, git.MinVersion, git.RecommendedVersion,
			)
		}
	}

	if !alreadyInitialized {
		if gitOps.Enabled() {
			_, err := gitOps.Init()
			if err != nil {
				logrus.WithError(err).Warn("Could not initialize git repository")
			}

			trainingRootFs := newTrainingRootFs(trainingRootDir)
			cfg := h.config.TrainingConfig(trainingRootFs)

			if !cfg.GitConfigured {
				showGitDefaults()
				gitDefaultConfig(&cfg)

				if err := h.config.WriteTrainingConfig(cfg, trainingRootFs); err != nil {
					return errors.Wrap(err, "can't update training config with git preferences")
				}
			}

			stageInitialFiles(gitOps, trainingRootDir)
			initMsg := fmt.Sprintf("initialize %s", trainingName)

			if gitOps.HasStagedChanges() && !previousSolutionsAvailable {
				// No restore coming — commit now with current time.
				if err := gitOps.Commit(initMsg); err != nil {
					logrus.WithError(err).Warn("Could not create initial commit")
					fmt.Println(formatGitWarning("Could not create initial git commit", err))
					fmt.Println(color.YellowString("  Your training will work normally, but git history may be incomplete."))
				}
			}
			// When previousSolutionsAvailable, staged changes remain uncommitted
			// so restore() can create the initialize commit with the correct date.
		} else {
			// --no-git or git unavailable: mark as configured (git disabled)
			trainingRootFs := newTrainingRootFs(trainingRootDir)
			cfg := h.config.TrainingConfig(trainingRootFs)
			if !cfg.GitConfigured {
				cfg.GitConfigured = true
				cfg.GitEnabled = false
				cfg.GitUnavailable = gitWasUnavailable
				if err := h.config.WriteTrainingConfig(cfg, trainingRootFs); err != nil {
					return errors.Wrap(err, "can't update training config")
				}
			}
		}

		var previousSolutions []string

		if previousSolutionsAvailable {
			fmt.Println("\nIt looks like you have already started this training and have existing exercises.")
			fmt.Println("You can clone your existing solutions to this directory.")

			// forceGit implies scripted/non-interactive mode — auto-accept restore
			ok := forceGit || promptForPastSolutions()

			if ok {
				previousSolutions, err = h.restore(ctx, trainingRootDir, gitOps)
				if err != nil {
					var ufe UserFacingError
					if errors.As(err, &ufe) {
						return ufe
					}
					return errors.Wrap(err, "can't restore existing exercises")
				}
			}

			// If user declined restore (or restore found nothing), commit now.
			if gitOps.Enabled() && gitOps.HasStagedChanges() {
				initMsg := fmt.Sprintf("initialize %s", trainingName)
				if err := gitOps.Commit(initMsg); err != nil {
					logrus.WithError(err).Warn("Could not create initial commit")
					fmt.Println(formatGitWarning("Could not create initial git commit", err))
					fmt.Println(color.YellowString("  Your training will work normally, but git history may be incomplete."))
				}
			}
		}

		_, err = h.nextExerciseWithSkipped(ctx, "", trainingRootDir, previousSolutions)
		if err != nil {
			return err
		}
	} else {
		// Partial init (alreadyInitialized but no .tdl-exercise): fetch the first exercise.
		_, err = h.nextExerciseWithSkipped(ctx, "", trainingRootDir, nil)
		if err != nil {
			return err
		}
	}

	if !isInTrainingRoot(trainingRootDir) {
		relDir, err := filepath.Rel(wd, trainingRootDir)
		if err != nil {
			return errors.Wrap(err, "can't get relative path")
		}

		fmt.Println("\nNow run " + color.CyanString("cd "+relDir+"/") + " to enter the training workspace")
	}

	return nil
}

func showGitDefaults() {
	fmt.Println()
	fmt.Println(color.New(color.Bold).Sprint("Git integration"))
	fmt.Println()
	fmt.Println("  Your progress will be tracked with git. Here's what happens automatically:")
	fmt.Println()
	fmt.Println("  When you complete an exercise:")
	fmt.Println(color.CyanString("    git add <exercise-dir> && git commit -m \"completed <exercise>\""))
	fmt.Println()
	fmt.Println("  When loading the next exercise:")
	fmt.Printf("    %s git fetch cli tdl/init/<exercise>\n", color.MagentaString("•••"))
	fmt.Printf("    %s git merge tdl/init/<exercise>\n", color.MagentaString("•••"))
	fmt.Println()
	fmt.Println("  After passing, the example solution is saved for comparison:")
	fmt.Println(color.CyanString("    git diff <your-branch>..tdl/example/<exercise> -- <exercise-dir>"))
	fmt.Println()
	fmt.Println("  Press s after passing to sync with the example solution.")
	fmt.Println("  Your work is saved to a backup branch first (never destructive).")
	fmt.Println()
	fmt.Printf("  Defaults: auto-commit on, auto-sync off.\n")
	fmt.Printf("  To change: %s\n\n", color.CyanString("tdl training settings"))
}

// printGitUnavailableNotice shows a recommendation banner when git is missing or too old.
func printGitUnavailableNotice(reason string, installHint string) {
	sep := color.HiBlackString(strings.Repeat("─", internal.TerminalWidth()))
	title := color.New(color.Bold, color.FgHiYellow).Sprint("  *** Git Recommended ***")

	fmt.Println(sep)
	fmt.Println(title)
	fmt.Println()
	fmt.Println("  " + reason)
	fmt.Println()
	fmt.Println("  Git integration gives you:")
	fmt.Println("  • Progress tracking: each completed exercise is committed automatically")
	fmt.Println("  • Solution comparison: diff your code with the example solution")
	fmt.Println("  • Safe exercise loading: preview conflicts before merging new exercises")
	fmt.Println()
	for _, line := range strings.Split(installHint, "\n") {
		if line == "" {
			fmt.Println()
		} else {
			fmt.Println("  " + line)
		}
	}
	fmt.Println()
	fmt.Println("  You can continue without git: you can always migrate later.")
	fmt.Println(sep)
	fmt.Println()
}

// promptContinueWithoutGit asks the user whether to continue without git or quit to install.
// Returns true if the user chose to continue.
func promptContinueWithoutGit() bool {
	choice := internal.Prompt(
		internal.Actions{
			{Shortcut: '\n', Action: "continue without git", ShortcutAliases: []rune{'\r'}},
			{Shortcut: 'q', Action: "quit and install git first"},
		},
		os.Stdin,
		os.Stdout,
	)
	return choice == '\n'
}

// promptGitPreferences runs interactive prompts for git settings.
// Used by "tdl training settings" to let users change preferences.
func promptGitPreferences() (autoCommit bool, autoGolden bool) {
	fmt.Println()
	fmt.Println(color.New(color.Bold).Sprint("Git settings"))
	fmt.Println("Automatically commit your progress when you pass each exercise?")

	autoCommitPrompt := internal.Prompt(
		internal.Actions{
			{Shortcut: '\n', Action: "enable auto-commit (recommended)", ShortcutAliases: []rune{'\r'}},
			{Shortcut: 'n', Action: "skip auto-commit (you'll commit manually)"},
		},
		os.Stdin,
		os.Stdout,
	)
	autoCommit = autoCommitPrompt == '\n'

	fmt.Println()
	fmt.Println("After passing, automatically sync with the example solution?")

	autoGoldenPrompt := internal.Prompt(
		internal.Actions{
			{Shortcut: '\n', Action: "skip (you can press s manually)", ShortcutAliases: []rune{'\r'}},
			{Shortcut: 'y', Action: "enable auto-sync"},
		},
		os.Stdin,
		os.Stdout,
	)
	autoGolden = autoGoldenPrompt == 'y'

	fmt.Println()
	return autoCommit, autoGolden
}

func promptForPastSolutions() bool {
	promptValue := internal.Prompt(
		internal.Actions{
			{Shortcut: '\n', Action: "download your latest solution FOR EACH EXERCISE", ShortcutAliases: []rune{'\r'}},
			{Shortcut: 'n', Action: "cancel"},
		},
		os.Stdin,
		os.Stdout,
	)
	return promptValue == '\n'
}

func isInTrainingRoot(trainingRoot string) bool {
	pwd, err := os.Getwd()
	if err != nil {
		logrus.WithError(err).Warn("Can't get current working directory")
		return false
	}

	absPwd, err := filepath.Abs(pwd)
	if err != nil {
		logrus.WithError(err).Warn("Can't get absolute path of current working directory")
		return false
	}

	absTrainingRoot, err := filepath.Abs(trainingRoot)
	if err != nil {
		logrus.WithError(err).Warn("Can't get absolute path of training root")
		return false
	}

	return absPwd == absTrainingRoot
}

var ErrInterrupted = errors.New("interrupted")

func (h *Handlers) startTraining(
	ctx context.Context,
	trainingName string,
	trainingRootDir string,
) (string, bool, bool, error) {
	ctx = withSubAction(ctx, "init")

	alreadyExistingTrainingRoot, err := h.config.FindTrainingRoot()
	if err == nil {
		fmt.Println(color.BlueString("Training was already initialised. Training root:" + alreadyExistingTrainingRoot))
		trainingRootDir = alreadyExistingTrainingRoot
	} else if !errors.Is(err, config.TrainingRootNotFoundError) {
		return "", false, false, errors.Wrap(err, "can't check if training root exists")
	} else {
		if err := h.showTrainingStartPrompt(trainingRootDir); err != nil {
			return "", false, false, err
		}

		// we will create training root in current working directory
		logrus.Debug("No training root yet")
	}

	alreadyInitialized := alreadyExistingTrainingRoot != ""
	trainingRootFs := newTrainingRootFs(trainingRootDir)

	if alreadyInitialized {
		cfg := h.config.TrainingConfig(trainingRootFs)
		if cfg.TrainingName != trainingName {
			return "", false, false, fmt.Errorf(
				"training %s was already started in this directory, please go to other directory and run `tdl training init`",
				cfg.TrainingName,
			)
		}
	} else {
		err := os.MkdirAll(trainingRootDir, 0755)
		if err != nil {
			return "", false, false, errors.Wrap(err, "can't create training root dir")
		}

		err = createGoWorkspace(trainingRootDir)
		if err != nil {
			logrus.WithError(err).Warn("Could not create go workspace")
		}
	}

	resp, err := h.newGrpcClient().StartTraining(
		ctx,
		&genproto.StartTrainingRequest{
			TrainingName: trainingName,
			Token:        h.config.GlobalConfig().Token,
		},
	)
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return "", false, false, UserFacingError{
				Msg:          fmt.Sprintf("Training '%v' not found", trainingName),
				SolutionHint: "Please check the correct training name on the website.\n\nIf you wanted to init the training in a separate directory, use this format:\n\n\ttdl training init <name> <directory>",
			}
		}
		return "", false, false, errors.Wrap(err, "start training gRPC call failed")
	}

	if !alreadyInitialized {
		if err := h.config.WriteTrainingConfig(config.TrainingConfig{TrainingName: trainingName}, trainingRootFs); err != nil {
			return "", false, false, errors.Wrap(err, "can't write training config")
		}

		if err := writeGitignore(trainingRootFs); err != nil {
			return "", false, false, err
		}
	}

	return trainingRootDir, alreadyInitialized, resp.PreviousSolutionsAvailable, nil
}

var gitignore = strings.Join(
	[]string{
		"# Exercise content is subject to Three Dots Labs' copyright.",
		"**/" + files.ExerciseFile,
		"",
		"# TDL local state (managed by CLI)",
		".tdl-training",
		".tdl-exercise",
		"",
		"# AI coding tool configs (managed by CLI)",
		"CLAUDE.md",
		"AGENTS.md",
		".mcp.json",
		"",
	},
	"\n",
)

func writeGitignore(trainingRootFs *afero.BasePathFs) error {
	if !files.DirOrFileExists(trainingRootFs, ".gitignore") {
		f, err := trainingRootFs.Create(".gitignore")
		if err != nil {
			return errors.Wrap(err, "can't create .gitignore")
		}

		if _, err := f.Write([]byte(gitignore)); err != nil {
			return errors.Wrap(err, "can't write .gitignore")
		}
	}

	return nil
}

func createGoWorkspace(trainingRoot string) error {
	if !hasGo() {
		return nil
	}

	cmd := exec.Command("go", "work", "init")
	cmd.Dir = trainingRoot

	printlnCommand("go work init")

	out, err := cmd.CombinedOutput()
	if strings.Contains(string(out), "already exists") {
		logrus.Debug("go.work already exists")
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "can't run go work init: %s", string(out))
	}

	return nil
}

func hasGoWorkspace(trainingRoot string) bool {
	_, err := os.Stat(path.Join(trainingRoot, "go.work"))
	return err == nil
}

func addModuleToWorkspace(trainingRoot string, modulePath string) error {
	return addModuleToWorkspaceQuiet(trainingRoot, modulePath, true)
}

func addModuleToWorkspaceQuiet(trainingRoot string, modulePath string, quiet bool) error {
	if !hasGo() {
		return nil
	}

	if !hasGoWorkspace(trainingRoot) {
		return nil
	}

	cmd := exec.Command("go", "work", "use", modulePath)
	cmd.Dir = trainingRoot

	if !quiet {
		printlnCommand(fmt.Sprintf("go work use %v", modulePath))
	}

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "can't run go work use")
	}

	return nil
}

func hasGo() bool {
	_, err := exec.LookPath("go")
	return err == nil
}

func (h *Handlers) showTrainingStartPrompt(trainingDir string) error {
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		absTrainingDir, err := filepath.Abs(trainingDir)
		if err == nil && absTrainingDir == homeDir {
			if !internal.IsStdinTerminal() {
				return fmt.Errorf("refusing to init in home directory %s — create a subdirectory first", homeDir)
			}

			fmt.Println()
			fmt.Println(color.YellowString("  ⚠ Warning: you are about to clone training files directly into your home directory (%s).", homeDir))
			fmt.Println(color.YellowString("  This is almost never what you want. We recommend creating a subdirectory instead."))
			fmt.Println()

			if !internal.FConfirmPrompt(color.YellowString("Are you sure you want to continue?"), os.Stdin, os.Stdout) {
				return ErrInterrupted
			}
			return nil
		}
	}

	fmt.Printf(
		"This command will clone training source code to %s directory.\n",
		trainingDir,
	)

	if !internal.ConfirmPromptDefaultYes("continue") {
		return ErrInterrupted
	}

	return nil
}
