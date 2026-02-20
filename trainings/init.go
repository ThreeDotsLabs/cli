package trainings

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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

	// Git integration: init repo, configure preferences, initial commit
	// Skip git entirely in non-interactive mode (pipes, CI, E2E) — we can't prompt for preferences.
	// forceGit overrides the non-interactive check (used by E2E tests and scripted restore).
	gitOps := git.NewOps(trainingRootDir, noGit || (!forceGit && !internal.IsStdinTerminal()))

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
				cfg.GitConfigured = true
				cfg.GitEnabled = true
				cfg.GitAutoCommit = true
				cfg.GitGoldenSync = "always"
				cfg.GitGoldenMode = "compare"

				if err := h.config.WriteTrainingConfig(cfg, trainingRootFs); err != nil {
					return errors.Wrap(err, "can't update training config with git preferences")
				}
			}

			filesToCommit := []string{".tdl-training", ".gitignore"}
			if hasGoWorkspace(trainingRootDir) {
				filesToCommit = append(filesToCommit, "go.work")
			}
			if err := gitOps.AddFiles(filesToCommit...); err != nil {
				logrus.WithError(err).Warn("Could not stage initial files")
			}
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
	fmt.Println("  After passing, the official solution is saved for comparison:")
	fmt.Println(color.CyanString("    git diff <your-branch>..tdl/golden/<exercise> -- <exercise-dir>"))
	fmt.Println()
	fmt.Println("  Press g after passing to replace your solution with the official one.")
	fmt.Println("  Your work is saved to a backup branch first (never destructive).")
	fmt.Println()
	fmt.Printf("  Defaults: auto-commit on, golden sync always.\n")
	fmt.Printf("  To change: %s\n\n", color.CyanString("tdl training config"))
}

// promptGitPreferences runs interactive prompts for git settings.
// Used by "tdl training config" to let users change preferences.
func promptGitPreferences() (autoCommit bool, goldenSync string, goldenMode string) {
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
	fmt.Println("After passing an exercise, create a branch with the official solution for comparison?")

	goldenPrompt := internal.Prompt(
		internal.Actions{
			{Shortcut: '\n', Action: "always sync golden solutions (recommended)", ShortcutAliases: []rune{'\r'}},
			{Shortcut: 'a', Action: "ask each time"},
			{Shortcut: 'n', Action: "never sync golden solutions"},
		},
		os.Stdin,
		os.Stdout,
	)

	switch goldenPrompt {
	case '\n':
		goldenSync = "always"
	case 'a':
		goldenSync = "ask"
	default:
		goldenSync = "never"
	}

	goldenMode = "compare" // golden branch always uses compare; `g` action does override

	fmt.Println()
	return autoCommit, goldenSync, goldenMode
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
		ctxWithRequestHeader(ctx, h.cliMetadata),
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
		"# TDL exercise state (managed by CLI)",
		".tdl-exercise",
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
	fmt.Printf(
		"This command will clone training source code to %s directory.\n",
		trainingDir,
	)

	if !internal.ConfirmPromptDefaultYes("continue") {
		return ErrInterrupted
	}

	return nil
}
