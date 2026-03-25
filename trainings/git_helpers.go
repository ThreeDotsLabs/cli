package trainings

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/git"
)

// gitDefaultConfig sets the default git integration fields on a training config.
// Centralizes the 5 fields so init and clone can't drift.
func gitDefaultConfig(cfg *config.TrainingConfig) {
	cfg.GitConfigured = true
	cfg.GitEnabled = true
	cfg.GitAutoCommit = true
	cfg.GitAutoGolden = false
	cfg.GitGoldenMode = "compare"
}

// stageInitialFiles stages the base set of files for an initial commit:
// .tdl-training, .gitignore, and optionally go.work if a Go workspace exists.
// Extra files (e.g. a cloned exercise directory) can be appended.
func stageInitialFiles(gitOps *git.Ops, trainingRootDir string, extraFiles ...string) {
	files := []string{".tdl-training", ".gitignore"}
	if hasGoWorkspace(trainingRootDir) {
		files = append(files, "go.work")
	}
	files = append(files, extraFiles...)
	if err := gitOps.AddFiles(files...); err != nil {
		logrus.WithError(err).Warn("Could not stage initial files")
	}
}

// saveProgress performs the reset→stage→commit pattern used to save user work
// before switching exercises. It is a no-op if there are no changes to commit.
func saveProgress(gitOps *git.Ops, dir string, commitMsg string) {
	_ = gitOps.ResetStaging()
	if err := gitOps.AddAll(dir); err == nil && gitOps.HasStagedChanges() {
		if err := gitOps.Commit(commitMsg); err != nil {
			fmt.Println(formatGitWarning("Could not auto-commit your progress", err))
		}
	}
}
