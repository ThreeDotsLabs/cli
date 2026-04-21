package trainings

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/git"
)

// --- test helpers (mirrors git/git_test.go pattern) ---

func initTestRepoTrainings(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	runGitTrainings(t, dir, "init")
	runGitTrainings(t, dir, "config", "user.email", "test@test.com")
	runGitTrainings(t, dir, "config", "user.name", "Test")

	writeFileTrainings(t, dir, "README.md", "# test\n")
	runGitTrainings(t, dir, "add", ".")
	runGitTrainings(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "initial")

	return dir
}

func runGitTrainings(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, string(out))
	return string(out)
}

func writeFileTrainings(t *testing.T, dir, name, content string) {
	t.Helper()
	fullPath := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
	require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
}

func commitCount(t *testing.T, dir string) int {
	t.Helper()
	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	var count int
	_, err = fmt.Sscanf(string(out), "%d", &count)
	require.NoError(t, err)
	return count
}

// --- gitDefaultConfig tests ---

func TestGitDefaultConfig_SetsAllFields(t *testing.T) {
	cfg := &config.TrainingConfig{}
	gitDefaultConfig(cfg)

	assert.True(t, cfg.GitConfigured)
	assert.True(t, cfg.GitEnabled)
	assert.True(t, cfg.GitAutoCommit)
	assert.False(t, cfg.GitAutoGolden)
	assert.Equal(t, "compare", cfg.GitGoldenMode)
}

func TestGitDefaultConfig_PreservesExistingFields(t *testing.T) {
	cfg := &config.TrainingConfig{
		TrainingName: "my-training",
	}
	gitDefaultConfig(cfg)

	assert.Equal(t, "my-training", cfg.TrainingName, "TrainingName should be preserved")
	assert.True(t, cfg.GitConfigured)
	assert.True(t, cfg.GitEnabled)
}

// --- stageInitialFiles tests ---

func TestStageInitialFiles_BaseCase(t *testing.T) {
	dir := initTestRepoTrainings(t)
	gitOps := git.NewOps(dir, false)

	// Create the base files
	writeFileTrainings(t, dir, ".tdl-training", "training_name = \"test\"\n")
	writeFileTrainings(t, dir, ".gitignore", "*.tmp\n")

	stageInitialFiles(gitOps, dir)

	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(out), ".gitignore")
	// .tdl-training is local state; it must not land in the initial commit
	// even when present on disk.
	assert.NotContains(t, string(out), ".tdl-training")
}

func TestStageInitialFiles_WithGoWork(t *testing.T) {
	dir := initTestRepoTrainings(t)
	gitOps := git.NewOps(dir, false)

	writeFileTrainings(t, dir, ".tdl-training", "training_name = \"test\"\n")
	writeFileTrainings(t, dir, ".gitignore", "*.tmp\n")
	writeFileTrainings(t, dir, "go.work", "go 1.21\n")

	stageInitialFiles(gitOps, dir)

	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(out), "go.work")
}

func TestStageInitialFiles_WithExtraFiles(t *testing.T) {
	dir := initTestRepoTrainings(t)
	gitOps := git.NewOps(dir, false)

	writeFileTrainings(t, dir, ".tdl-training", "training_name = \"test\"\n")
	writeFileTrainings(t, dir, ".gitignore", "*.tmp\n")
	writeFileTrainings(t, dir, "exercise/main.go", "package main\n")

	stageInitialFiles(gitOps, dir, "exercise")

	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(out), "exercise/main.go")
}

// --- saveProgress tests ---

func TestSaveProgress_WithChanges(t *testing.T) {
	dir := initTestRepoTrainings(t)
	gitOps := git.NewOps(dir, false)

	// Create and commit an exercise directory first
	writeFileTrainings(t, dir, "exercise/main.go", "package main\n")
	runGitTrainings(t, dir, "add", ".")
	runGitTrainings(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "add exercise")

	countBefore := commitCount(t, dir)

	// Make a change
	writeFileTrainings(t, dir, "exercise/main.go", "package main\n\nfunc hello() {}\n")

	saveProgress(gitOps, "exercise", "save progress on 01-module/01-exercise")

	countAfter := commitCount(t, dir)
	assert.Equal(t, countBefore+1, countAfter, "should create one new commit")

	// Verify commit message
	cmd := exec.Command("git", "log", "-1", "--format=%s")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(out), "save progress on 01-module/01-exercise")
}

func TestSaveProgress_WithoutChanges(t *testing.T) {
	dir := initTestRepoTrainings(t)
	gitOps := git.NewOps(dir, false)

	countBefore := commitCount(t, dir)

	saveProgress(gitOps, "exercise", "should not appear")

	countAfter := commitCount(t, dir)
	assert.Equal(t, countBefore, countAfter, "should not create a commit when there are no changes")
}

func TestSaveProgress_ResetsStaging(t *testing.T) {
	dir := initTestRepoTrainings(t)
	gitOps := git.NewOps(dir, false)

	// Create two directories
	writeFileTrainings(t, dir, "exercise-a/main.go", "package a\n")
	writeFileTrainings(t, dir, "exercise-b/main.go", "package b\n")
	runGitTrainings(t, dir, "add", ".")
	runGitTrainings(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "add exercises")

	// Modify both, but stage only exercise-a via git add
	writeFileTrainings(t, dir, "exercise-a/main.go", "package a\n\nfunc changed() {}\n")
	writeFileTrainings(t, dir, "exercise-b/main.go", "package b\n\nfunc changed() {}\n")
	runGitTrainings(t, dir, "add", "exercise-a")

	// saveProgress for exercise-b should reset staging first,
	// so only exercise-b ends up in the commit
	saveProgress(gitOps, "exercise-b", "save exercise-b")

	// Check that the last commit only touches exercise-b
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(out), "exercise-b/main.go")
	assert.NotContains(t, string(out), "exercise-a/main.go")
}
