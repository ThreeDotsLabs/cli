package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	// Create initial commit so we have a HEAD
	writeFile(t, dir, "README.md", "# test\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "initial")

	return dir
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, string(out))
	return string(out)
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	fullPath := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
	require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
}

func TestInit_NewRepo(t *testing.T) {
	dir := t.TempDir()
	ops := NewOps(dir, false)

	created, err := ops.Init()
	require.NoError(t, err)
	assert.True(t, created)
	assert.True(t, ops.IsRepo())
}

func TestInit_ExistingRepo(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	created, err := ops.Init()
	require.NoError(t, err)
	assert.False(t, created, "should not create a new repo if one exists")
	assert.True(t, ops.IsRepo())
}

func TestAllOps_Disabled(t *testing.T) {
	ops := NewOps(filepath.Join(t.TempDir(), "nonexistent"), true)

	assert.False(t, ops.Enabled())
	assert.False(t, ops.IsRepo())
	assert.False(t, ops.HasUncommittedChanges("."))
	assert.False(t, ops.HasStagedChanges())
	assert.False(t, ops.BranchExists("main"))
	assert.False(t, ops.HasCommits())

	created, err := ops.Init()
	assert.NoError(t, err)
	assert.False(t, created)

	branch, err := ops.CurrentBranch()
	assert.NoError(t, err)
	assert.Empty(t, branch)

	assert.NoError(t, ops.AddAll("."))
	assert.NoError(t, ops.AddFiles("foo"))
	assert.NoError(t, ops.Commit("msg"))
	assert.NoError(t, ops.CommitAll("msg"))
	assert.NoError(t, ops.DeleteBranch("b"))
	wtDir := filepath.Join(t.TempDir(), "wt")
	assert.NoError(t, ops.WorktreeAdd(wtDir, "b"))
	assert.NoError(t, ops.WorktreeRemove(wtDir))
	assert.NoError(t, ops.CheckoutBranch("b"))
	assert.NoError(t, ops.Merge("b", "msg"))
	assert.NoError(t, ops.MergeAbort())
	assert.NoError(t, ops.CreateBranchFromHead("b2"))
	assert.False(t, ops.HasUnmergedFiles())

	unmerged, err := ops.UnmergedFiles()
	assert.NoError(t, err)
	assert.Empty(t, unmerged)

	stat, err := ops.DiffStat("a", "b")
	assert.NoError(t, err)
	assert.Empty(t, stat)

	log, err := ops.Log(1)
	assert.NoError(t, err)
	assert.Empty(t, log)
}

func TestBranchLifecycle(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	branchName := "tdl/init/01-module/01-exercise"
	assert.False(t, ops.BranchExists(branchName))

	// Create branch via worktree
	tmpDir := t.TempDir()
	worktreeDir := filepath.Join(tmpDir, "wt")
	require.NoError(t, ops.WorktreeAdd(worktreeDir, branchName))
	require.NoError(t, ops.WorktreeRemove(worktreeDir))

	assert.True(t, ops.BranchExists(branchName))

	require.NoError(t, ops.DeleteBranch(branchName))
	assert.False(t, ops.BranchExists(branchName))
}

func TestWorktree_CreateAndRemove(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	tmpDir := t.TempDir()
	worktreeDir := filepath.Join(tmpDir, "wt")

	require.NoError(t, ops.WorktreeAdd(worktreeDir, "test-branch"))

	// Worktree should be writable
	writeFile(t, worktreeDir, "test.go", "package main\n")
	assert.FileExists(t, filepath.Join(worktreeDir, "test.go"))

	require.NoError(t, ops.WorktreeRemove(worktreeDir))

	// Worktree dir should be gone
	_, err := os.Stat(worktreeDir)
	assert.True(t, os.IsNotExist(err))
}

func TestWorktree_CommitOnBranch(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	tmpDir := t.TempDir()
	worktreeDir := filepath.Join(tmpDir, "wt")

	require.NoError(t, ops.WorktreeAdd(worktreeDir, "feature-branch"))

	// Write and commit a file in the worktree
	writeFile(t, worktreeDir, "feature.go", "package feature\n")
	worktreeOps := NewOps(worktreeDir, false)
	require.NoError(t, worktreeOps.AddAll("."))
	require.NoError(t, worktreeOps.Commit("add feature"))

	require.NoError(t, ops.WorktreeRemove(worktreeDir))

	// File should NOT be on main branch
	_, err := os.Stat(filepath.Join(dir, "feature.go"))
	assert.True(t, os.IsNotExist(err))

	// But branch should exist with our commit
	assert.True(t, ops.BranchExists("feature-branch"))
}

func TestMerge_FastForward(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	tmpDir := t.TempDir()
	worktreeDir := filepath.Join(tmpDir, "wt")

	require.NoError(t, ops.WorktreeAdd(worktreeDir, "new-files"))

	exerciseDir := filepath.Join(worktreeDir, "01-module", "01-exercise")
	require.NoError(t, os.MkdirAll(exerciseDir, 0755))
	writeFile(t, exerciseDir, "main.go", "package main\n")
	writeFile(t, exerciseDir, "main_test.go", "package main\n")

	worktreeOps := NewOps(worktreeDir, false)
	require.NoError(t, worktreeOps.AddAll("."))
	require.NoError(t, worktreeOps.Commit("exercise files"))

	require.NoError(t, ops.WorktreeRemove(worktreeDir))

	// Merge should fast-forward
	require.NoError(t, ops.Merge("new-files", "merge exercise"))

	// Files should now be in main working tree
	assert.FileExists(t, filepath.Join(dir, "01-module", "01-exercise", "main.go"))
	assert.FileExists(t, filepath.Join(dir, "01-module", "01-exercise", "main_test.go"))
}

func TestMerge_ThreeWay(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Make a change on main first (diverge from the branch point)
	writeFile(t, dir, "user-file.txt", "user work\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "user work")

	// Create branch from parent commit (diverged)
	tmpDir := t.TempDir()
	worktreeDir := filepath.Join(tmpDir, "wt")

	// Create branch from HEAD~1 (the initial commit)
	runGit(t, dir, "branch", "server-branch", "HEAD~1")
	runGit(t, dir, "worktree", "add", worktreeDir, "server-branch")

	exerciseDir := filepath.Join(worktreeDir, "01-module", "01-exercise")
	require.NoError(t, os.MkdirAll(exerciseDir, 0755))
	writeFile(t, exerciseDir, "main.go", "package main\n")

	worktreeOps := NewOps(worktreeDir, false)
	require.NoError(t, worktreeOps.AddAll("."))
	require.NoError(t, worktreeOps.Commit("exercise files"))

	runGit(t, dir, "worktree", "remove", worktreeDir, "--force")

	// Three-way merge
	require.NoError(t, ops.Merge("server-branch", "merge exercise"))

	// Both changes should exist
	assert.FileExists(t, filepath.Join(dir, "user-file.txt"))
	assert.FileExists(t, filepath.Join(dir, "01-module", "01-exercise", "main.go"))
}

func TestMerge_Conflict(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Write conflicting content on main
	writeFile(t, dir, "shared.txt", "main version\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "main change")

	// Create a branch from the initial commit with conflicting content
	runGit(t, dir, "branch", "conflict-branch", "HEAD~1")
	tmpDir := t.TempDir()
	worktreeDir := filepath.Join(tmpDir, "wt")
	runGit(t, dir, "worktree", "add", worktreeDir, "conflict-branch")

	writeFile(t, worktreeDir, "shared.txt", "branch version\n")
	runGit(t, worktreeDir, "add", ".")
	runGit(t, worktreeDir, "-c", "commit.gpgsign=false", "commit", "-m", "branch change")
	runGit(t, dir, "worktree", "remove", worktreeDir, "--force")

	// Merge should fail
	err := ops.Merge("conflict-branch", "merge conflict")
	assert.Error(t, err)

	// Clean up merge state
	runGit(t, dir, "merge", "--abort")
}

func TestScopedAdd(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Create files in two different directories
	writeFile(t, dir, "dir-a/file.go", "package a\n")
	writeFile(t, dir, "dir-b/file.go", "package b\n")

	// Only add dir-a
	require.NoError(t, ops.AddAll("dir-a"))

	// dir-a should be staged, dir-b should not
	assert.True(t, ops.HasStagedChanges())

	output := runGit(t, dir, "status", "--porcelain")

	// Verify only dir-a is staged (A = staged, ?? = untracked)
	assert.Contains(t, output, "A  dir-a/file.go")
	assert.Contains(t, output, "?? dir-b")
}

func TestHasStagedChanges(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	assert.False(t, ops.HasStagedChanges())

	writeFile(t, dir, "new.go", "package new\n")
	runGit(t, dir, "add", "new.go")

	assert.True(t, ops.HasStagedChanges())
}

func TestHasUncommittedChanges(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	assert.False(t, ops.HasUncommittedChanges("."))

	writeFile(t, dir, "somedir/new.go", "package new\n")

	assert.True(t, ops.HasUncommittedChanges("somedir"))
	assert.False(t, ops.HasUncommittedChanges("otherdir"))
}

func TestBranchNames(t *testing.T) {
	assert.Equal(t, "tdl/init/01-module/01-exercise", InitBranchName("01-module/01-exercise"))
	assert.Equal(t, "tdl/example/01-module/01-exercise", GoldenBranchName("01-module/01-exercise"))
}

func TestCurrentBranch(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	branch, err := ops.CurrentBranch()
	require.NoError(t, err)
	// Could be "main" or "master" depending on git config
	assert.NotEmpty(t, branch)
}

func TestLog(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	log, err := ops.Log(1)
	require.NoError(t, err)
	assert.Contains(t, log, "initial")
}

func TestHasCommits(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)
	assert.True(t, ops.HasCommits())

	emptyDir := t.TempDir()
	emptyOps := NewOps(emptyDir, false)
	runGit(t, emptyDir, "init")
	assert.False(t, emptyOps.HasCommits())
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	require.NoError(t, err)
	return string(data)
}

func TestWorktreeAddFrom(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Create commit A (initial already exists), then commit B
	writeFile(t, dir, "a.txt", "from commit A\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "commit A")

	writeFile(t, dir, "b.txt", "from commit B\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "commit B")

	// Get commit A hash (HEAD~1)
	commitA := strings.TrimSpace(runGit(t, dir, "rev-parse", "HEAD~1"))

	// Create worktree from commit A (not HEAD)
	tmpDir := t.TempDir()
	worktreeDir := filepath.Join(tmpDir, "wt")
	require.NoError(t, ops.WorktreeAddFrom(worktreeDir, "from-a-branch", commitA))

	// Worktree should have a.txt but NOT b.txt
	assert.FileExists(t, filepath.Join(worktreeDir, "a.txt"))
	_, err := os.Stat(filepath.Join(worktreeDir, "b.txt"))
	assert.True(t, os.IsNotExist(err), "b.txt should not exist in worktree based on commit A")

	require.NoError(t, ops.WorktreeRemove(worktreeDir))
}

func TestRevParse(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Add another commit
	writeFile(t, dir, "extra.txt", "extra\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "extra")

	hash, err := ops.RevParse("HEAD")
	require.NoError(t, err)
	assert.Len(t, hash, 40, "should be a 40-char hex hash")

	prevHash, err := ops.RevParse("HEAD~1")
	require.NoError(t, err)
	assert.Len(t, prevHash, 40)
	assert.NotEqual(t, hash, prevHash)
}

// TestInitChain_UserChangesPreserved simulates the exact user scenario:
// Exercise 1 provides main.go, user modifies line 5 and adds ctx.go,
// exercise 2 (based on init chain) changes line 10 of main.go.
// With the init chain, git preserves the user's line 5 change via 3-way merge.
func TestInitChain_UserChangesPreserved(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Exercise 1: provide main.go with 10 lines
	exercise1Content := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"line5-original\")\n\tfmt.Println(\"line6\")\n\tfmt.Println(\"line7\")\n\tfmt.Println(\"line8\")\n\tfmt.Println(\"line9\")\n\tfmt.Println(\"line10-original\")\n}\n"

	// Create init/01-mod/01-ex from HEAD
	tmpDir1 := t.TempDir()
	wt1 := filepath.Join(tmpDir1, "wt")
	initBranch1 := "tdl/init/01-mod/01-ex"
	require.NoError(t, ops.WorktreeAdd(wt1, initBranch1))

	writeFile(t, wt1, "01-mod/01-ex/main.go", exercise1Content)
	wt1Ops := NewQuietOps(wt1)
	require.NoError(t, wt1Ops.AddAll("01-mod/01-ex"))
	require.NoError(t, wt1Ops.Commit("init files for 01-mod/01-ex"))
	require.NoError(t, ops.WorktreeRemove(wt1))

	// Merge init branch into main
	require.NoError(t, ops.Merge(initBranch1, "start 01-mod/01-ex"))

	// User modifies line 5 and adds ctx.go
	userContent := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"line5-USER-MODIFIED\")\n\tfmt.Println(\"line6\")\n\tfmt.Println(\"line7\")\n\tfmt.Println(\"line8\")\n\tfmt.Println(\"line9\")\n\tfmt.Println(\"line10-original\")\n}\n"
	writeFile(t, dir, "01-mod/01-ex/main.go", userContent)
	writeFile(t, dir, "01-mod/01-ex/ctx.go", "package main\n\nimport \"context\"\n\nvar ctx = context.Background()\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "user solution")

	// Exercise 2: changes line 10 (based on PREVIOUS INIT BRANCH — the chain)
	exercise2Content := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"line5-original\")\n\tfmt.Println(\"line6\")\n\tfmt.Println(\"line7\")\n\tfmt.Println(\"line8\")\n\tfmt.Println(\"line9\")\n\tfmt.Println(\"line10-EXERCISE2\")\n}\n"

	tmpDir2 := t.TempDir()
	wt2 := filepath.Join(tmpDir2, "wt")
	initBranch2 := "tdl/init/01-mod/02-ex"
	// KEY: base on previous init branch, not HEAD
	require.NoError(t, ops.WorktreeAddFrom(wt2, initBranch2, initBranch1))

	writeFile(t, wt2, "01-mod/01-ex/main.go", exercise2Content)
	wt2Ops := NewQuietOps(wt2)
	require.NoError(t, wt2Ops.AddAll("01-mod/01-ex"))
	require.NoError(t, wt2Ops.Commit("init files for 01-mod/02-ex"))
	require.NoError(t, ops.WorktreeRemove(wt2))

	// Merge exercise 2's init branch — should be a 3-way merge
	require.NoError(t, ops.Merge(initBranch2, "start 01-mod/02-ex"))

	// Assert: user's line 5 change preserved, exercise 2 line 10 change applied, ctx.go exists
	result := readFile(t, dir, "01-mod/01-ex/main.go")
	assert.Contains(t, result, "line5-USER-MODIFIED", "user's line 5 change should be preserved")
	assert.Contains(t, result, "line10-EXERCISE2", "exercise 2's line 10 change should be applied")
	assert.FileExists(t, filepath.Join(dir, "01-mod/01-ex/ctx.go"), "user's ctx.go should still exist")
}

// TestInitChain_ConflictWhenSameLinesModified verifies that when both user
// and exercise modify the same line, a proper merge conflict occurs.
func TestInitChain_ConflictWhenSameLinesModified(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	exercise1Content := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"shared-line-original\")\n}\n"

	// Create init/01-mod/01-ex
	tmpDir1 := t.TempDir()
	wt1 := filepath.Join(tmpDir1, "wt")
	initBranch1 := "tdl/init/01-mod/01-ex"
	require.NoError(t, ops.WorktreeAdd(wt1, initBranch1))

	writeFile(t, wt1, "01-mod/01-ex/main.go", exercise1Content)
	wt1Ops := NewQuietOps(wt1)
	require.NoError(t, wt1Ops.AddAll("01-mod/01-ex"))
	require.NoError(t, wt1Ops.Commit("init files for 01-mod/01-ex"))
	require.NoError(t, ops.WorktreeRemove(wt1))
	require.NoError(t, ops.Merge(initBranch1, "start 01-mod/01-ex"))

	// User modifies the SAME line
	userContent := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"shared-line-USER\")\n}\n"
	writeFile(t, dir, "01-mod/01-ex/main.go", userContent)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "user solution")

	// Exercise 2 also modifies the SAME line
	exercise2Content := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"shared-line-EXERCISE2\")\n}\n"

	tmpDir2 := t.TempDir()
	wt2 := filepath.Join(tmpDir2, "wt")
	initBranch2 := "tdl/init/01-mod/02-ex"
	require.NoError(t, ops.WorktreeAddFrom(wt2, initBranch2, initBranch1))

	writeFile(t, wt2, "01-mod/01-ex/main.go", exercise2Content)
	wt2Ops := NewQuietOps(wt2)
	require.NoError(t, wt2Ops.AddAll("01-mod/01-ex"))
	require.NoError(t, wt2Ops.Commit("init files for 01-mod/02-ex"))
	require.NoError(t, ops.WorktreeRemove(wt2))

	// Merge should FAIL with conflict
	err := ops.Merge(initBranch2, "start 01-mod/02-ex")
	assert.Error(t, err, "should produce a merge conflict when both sides modify the same line")

	// Clean up merge state
	runGit(t, dir, "merge", "--abort")
}

// TestInitChain_WithoutChain_UserChangesLost documents the bug we're fixing.
// Without the init chain (init branch from HEAD), user changes are silently lost.
func TestInitChain_WithoutChain_UserChangesLost(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	exercise1Content := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"line5-original\")\n\tfmt.Println(\"line6\")\n\tfmt.Println(\"line7\")\n\tfmt.Println(\"line8\")\n\tfmt.Println(\"line9\")\n\tfmt.Println(\"line10-original\")\n}\n"

	// Exercise 1 from HEAD
	tmpDir1 := t.TempDir()
	wt1 := filepath.Join(tmpDir1, "wt")
	require.NoError(t, ops.WorktreeAdd(wt1, "init-ex1"))

	writeFile(t, wt1, "01-mod/01-ex/main.go", exercise1Content)
	wt1Ops := NewQuietOps(wt1)
	require.NoError(t, wt1Ops.AddAll("01-mod/01-ex"))
	require.NoError(t, wt1Ops.Commit("exercise 1"))
	require.NoError(t, ops.WorktreeRemove(wt1))
	require.NoError(t, ops.Merge("init-ex1", "start ex1"))

	// User modifies line 5
	userContent := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"line5-USER-MODIFIED\")\n\tfmt.Println(\"line6\")\n\tfmt.Println(\"line7\")\n\tfmt.Println(\"line8\")\n\tfmt.Println(\"line9\")\n\tfmt.Println(\"line10-original\")\n}\n"
	writeFile(t, dir, "01-mod/01-ex/main.go", userContent)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "user solution")

	// Exercise 2 from HEAD (NOT from init chain — this is the bug)
	exercise2Content := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"line5-original\")\n\tfmt.Println(\"line6\")\n\tfmt.Println(\"line7\")\n\tfmt.Println(\"line8\")\n\tfmt.Println(\"line9\")\n\tfmt.Println(\"line10-EXERCISE2\")\n}\n"

	tmpDir2 := t.TempDir()
	wt2 := filepath.Join(tmpDir2, "wt")
	require.NoError(t, ops.WorktreeAdd(wt2, "init-ex2"))

	writeFile(t, wt2, "01-mod/01-ex/main.go", exercise2Content)
	wt2Ops := NewQuietOps(wt2)
	require.NoError(t, wt2Ops.AddAll("01-mod/01-ex"))
	require.NoError(t, wt2Ops.Commit("exercise 2"))
	require.NoError(t, ops.WorktreeRemove(wt2))

	// This merge is a fast-forward — user's changes silently overwritten
	require.NoError(t, ops.Merge("init-ex2", "start ex2"))

	result := readFile(t, dir, "01-mod/01-ex/main.go")
	// This documents the BUG: user's line 5 change is LOST
	assert.Contains(t, result, "line5-original", "BUG: user's line 5 change should be lost without init chain")
	assert.NotContains(t, result, "line5-USER-MODIFIED", "BUG: user's change is overwritten by fast-forward")
	assert.Contains(t, result, "line10-EXERCISE2", "exercise 2's change is applied")
}

func TestMergeTreeCheck_NoConflict(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Create a branch with new files (no overlap)
	tmpDir := t.TempDir()
	wt := filepath.Join(tmpDir, "wt")
	runGit(t, dir, "branch", "no-conflict-branch", "HEAD")
	runGit(t, dir, "worktree", "add", wt, "no-conflict-branch")

	writeFile(t, wt, "new-file.txt", "new content\n")
	runGit(t, wt, "add", ".")
	runGit(t, wt, "-c", "commit.gpgsign=false", "commit", "-m", "new file")
	runGit(t, dir, "worktree", "remove", wt, "--force")

	conflictFiles, err := ops.MergeTreeCheck("no-conflict-branch")
	if err != nil {
		t.Skipf("git merge-tree --write-tree not available: %v", err)
	}
	assert.Empty(t, conflictFiles, "should have no conflicts for non-overlapping files")
}

func TestMergeTreeCheck_Conflict(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Modify main.go on main
	writeFile(t, dir, "shared.txt", "main version\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "main change")

	// Create branch from HEAD~1 with conflicting change
	runGit(t, dir, "branch", "conflict-check", "HEAD~1")
	tmpDir := t.TempDir()
	wt := filepath.Join(tmpDir, "wt")
	runGit(t, dir, "worktree", "add", wt, "conflict-check")

	writeFile(t, wt, "shared.txt", "branch version\n")
	runGit(t, wt, "add", ".")
	runGit(t, wt, "-c", "commit.gpgsign=false", "commit", "-m", "branch change")
	runGit(t, dir, "worktree", "remove", wt, "--force")

	// Save current state of working tree
	statBefore, _ := os.Stat(filepath.Join(dir, "shared.txt"))

	conflictFiles, err := ops.MergeTreeCheck("conflict-check")
	if err != nil {
		t.Skipf("git merge-tree --write-tree not available: %v", err)
	}
	assert.Equal(t, []string{"shared.txt"}, conflictFiles, "should report shared.txt as conflicting")

	// Verify working tree NOT modified (merge-tree is read-only)
	statAfter, _ := os.Stat(filepath.Join(dir, "shared.txt"))
	assert.Equal(t, statBefore.ModTime(), statAfter.ModTime(), "working tree should not be modified")
	content := readFile(t, dir, "shared.txt")
	assert.Equal(t, "main version\n", content, "file content should be unchanged")
}

func TestCheckoutTheirs(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Set up merge conflict
	writeFile(t, dir, "conflict.txt", "ours version\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "ours")

	runGit(t, dir, "branch", "theirs-branch", "HEAD~1")
	tmpDir := t.TempDir()
	wt := filepath.Join(tmpDir, "wt")
	runGit(t, dir, "worktree", "add", wt, "theirs-branch")
	writeFile(t, wt, "conflict.txt", "theirs version\n")
	runGit(t, wt, "add", ".")
	runGit(t, wt, "-c", "commit.gpgsign=false", "commit", "-m", "theirs")
	runGit(t, dir, "worktree", "remove", wt, "--force")

	// Start merge (will conflict)
	err := ops.Merge("theirs-branch", "merge theirs")
	require.Error(t, err)

	// Resolve with CheckoutTheirs
	require.NoError(t, ops.CheckoutTheirs("conflict.txt"))

	content := readFile(t, dir, "conflict.txt")
	assert.Equal(t, "theirs version\n", content, "should have theirs version after checkout --theirs")

	// Complete the merge
	require.NoError(t, ops.AddFiles("conflict.txt"))
	require.NoError(t, ops.Commit("resolved"))
}

func TestGoldenBasedOnHEAD(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Create init branch and merge
	tmpDir1 := t.TempDir()
	wt1 := filepath.Join(tmpDir1, "wt")
	initBranch := "tdl/init/01-mod/01-ex"
	require.NoError(t, ops.WorktreeAdd(wt1, initBranch))
	writeFile(t, wt1, "01-mod/01-ex/main.go", "package main\n")
	writeFile(t, wt1, "01-mod/01-ex/main_test.go", "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n")
	wt1Ops := NewQuietOps(wt1)
	require.NoError(t, wt1Ops.AddAll("01-mod/01-ex"))
	require.NoError(t, wt1Ops.Commit("init exercise"))
	require.NoError(t, ops.WorktreeRemove(wt1))
	require.NoError(t, ops.Merge(initBranch, "start exercise"))

	// User commits solution + extra file
	writeFile(t, dir, "01-mod/01-ex/main.go", "package main\n\nfunc main() {}\n")
	writeFile(t, dir, "01-mod/01-ex/helper.go", "package main\n\nfunc helper() {}\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "user solution")

	// Create example solution branch FROM HEAD (user's completed commit)
	goldenBranch := "tdl/example/01-mod/01-ex"
	tmpDir2 := t.TempDir()
	wt2 := filepath.Join(tmpDir2, "wt")
	require.NoError(t, ops.WorktreeAdd(wt2, goldenBranch))

	// No directory cleaning — syncGoldenSolution writes example solution files over the worktree.
	exerciseDir := filepath.Join(wt2, "01-mod", "01-ex")
	os.MkdirAll(exerciseDir, 0755)

	// Write example solution files — commit with an explicit date
	// (mirrors production code where callers pass time.Now().Add(1s))
	writeFile(t, wt2, "01-mod/01-ex/main.go", "package main\n\nfunc main() {\n\t// example solution\n}\n")
	wt2Ops := NewQuietOps(wt2)
	require.NoError(t, wt2Ops.AddAll("01-mod/01-ex"))
	goldenCommitDate := time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)
	require.NoError(t, wt2Ops.CommitWithDate("example solution", goldenCommitDate))
	require.NoError(t, ops.WorktreeRemove(wt2))

	// Verify example solution is based on HEAD (user's commit)
	mainHash, err := ops.RevParse("HEAD")
	require.NoError(t, err)

	mergeBase := strings.TrimSpace(runGit(t, dir, "merge-base", "HEAD", goldenBranch))
	assert.Equal(t, mainHash, mergeBase, "example solution should be based on HEAD (user's completed commit)")

	// Verify diff restricted to exercise dir only shows exercise-specific files
	diffOutput := strings.TrimSpace(runGit(t, dir, "diff", "--name-only", "HEAD.."+goldenBranch))
	for _, f := range strings.Split(diffOutput, "\n") {
		if f == "" {
			continue
		}
		assert.True(t, strings.HasPrefix(f, "01-mod/01-ex/"),
			"example solution diff should only contain exercise dir files, got: %s", f)
	}

	// Only main.go should appear in diff (the actual example solution change).
	// Without directory cleaning, unchanged files (helper.go, main_test.go)
	// persist from HEAD on the example solution branch — they shouldn't appear in diff.
	assert.Contains(t, diffOutput, "01-mod/01-ex/main.go",
		"example solution diff should include the changed solution file")
	assert.NotContains(t, diffOutput, "01-mod/01-ex/helper.go",
		"user's extra files should not appear in diff (preserved from HEAD)")
	assert.NotContains(t, diffOutput, "01-mod/01-ex/main_test.go",
		"shared files (like test files) should not appear in diff (preserved from HEAD)")

	// Verify all files still exist on the example solution branch
	goldenTreeOutput := strings.TrimSpace(runGit(t, dir, "ls-tree", "--name-only", goldenBranch, "--", "01-mod/01-ex/"))
	assert.Contains(t, goldenTreeOutput, "main.go",
		"example solution branch should have solution file")
	assert.Contains(t, goldenTreeOutput, "main_test.go",
		"example solution branch should preserve test files from init")
	assert.Contains(t, goldenTreeOutput, "helper.go",
		"example solution branch should preserve user files from HEAD")

	// Verify example solution commit has the explicit date we passed.
	goldenDateStr := strings.TrimSpace(runGit(t, dir, "log", "-1", "--format=%aI", goldenBranch))
	goldenDate, err := time.Parse(time.RFC3339, goldenDateStr)
	require.NoError(t, err)
	assert.Equal(t, goldenCommitDate, goldenDate.UTC(),
		"example solution commit should have the explicit date we passed")
}

func TestDiffStat_Alignment(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Create a branch with some changes
	runGit(t, dir, "branch", "stat-branch")
	tmpDir := t.TempDir()
	wt := filepath.Join(tmpDir, "wt")
	runGit(t, dir, "worktree", "add", wt, "stat-branch")

	writeFile(t, wt, "file-a.go", "package main\n\nfunc a() {}\n")
	writeFile(t, wt, "file-b.go", "package main\n\nfunc b() {}\n")
	runGit(t, wt, "add", ".")
	runGit(t, wt, "-c", "commit.gpgsign=false", "commit", "-m", "add files")
	runGit(t, dir, "worktree", "remove", wt, "--force")

	stat, err := ops.DiffStat("HEAD", "stat-branch")
	require.NoError(t, err)
	require.NotEmpty(t, stat)

	// All non-summary lines should have consistent leading space
	lines := strings.Split(stat, "\n")
	for _, line := range lines {
		if line == "" || strings.Contains(line, "changed") {
			continue // skip summary line and empty lines
		}
		assert.True(t, strings.HasPrefix(line, " "),
			"diff stat line should have leading space for alignment, got: %q", line)
	}
}

func TestDiffStatPath(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Create a branch with files in two directories
	runGit(t, dir, "branch", "stat-path-branch")
	tmpDir := t.TempDir()
	wt := filepath.Join(tmpDir, "wt")
	runGit(t, dir, "worktree", "add", wt, "stat-path-branch")

	writeFile(t, wt, "dir-a/file.go", "package a\n")
	writeFile(t, wt, "dir-b/file.go", "package b\n")
	runGit(t, wt, "add", ".")
	runGit(t, wt, "-c", "commit.gpgsign=false", "commit", "-m", "add files in two dirs")
	runGit(t, dir, "worktree", "remove", wt, "--force")

	// Full diff should include both directories
	fullStat, err := ops.DiffStat("HEAD", "stat-path-branch")
	require.NoError(t, err)
	assert.Contains(t, fullStat, "dir-a")
	assert.Contains(t, fullStat, "dir-b")

	// Path-restricted diff should only include dir-a
	pathStat, err := ops.DiffStatPath("HEAD", "stat-path-branch", "dir-a")
	require.NoError(t, err)
	assert.Contains(t, pathStat, "dir-a")
	assert.NotContains(t, pathStat, "dir-b")
}

func TestCheckoutFiles(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Create a branch with different file content
	runGit(t, dir, "branch", "other-branch")
	tmpDir := t.TempDir()
	wt := filepath.Join(tmpDir, "wt")
	runGit(t, dir, "worktree", "add", wt, "other-branch")

	require.NoError(t, os.MkdirAll(filepath.Join(wt, "exercise"), 0755))
	writeFile(t, wt, "exercise/main.go", "package main // from other branch\n")
	runGit(t, wt, "add", ".")
	runGit(t, wt, "-c", "commit.gpgsign=false", "commit", "-m", "other content")
	runGit(t, dir, "worktree", "remove", wt, "--force")

	// Write different content on main
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "exercise"), 0755))
	writeFile(t, dir, "exercise/main.go", "package main // from main\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "main content")

	// Checkout files from other branch
	require.NoError(t, ops.CheckoutFiles("other-branch", "exercise"))

	content := readFile(t, dir, "exercise/main.go")
	assert.Equal(t, "package main // from other branch\n", content, "should have content from other branch")
}

func TestUnmergedFiles_NoConflict(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	files, err := ops.UnmergedFiles()
	require.NoError(t, err)
	assert.Empty(t, files, "should have no unmerged files in clean repo")
}

func TestUnmergedFiles_WithConflict(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Create a merge conflict
	writeFile(t, dir, "conflict.txt", "main version\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "main")

	runGit(t, dir, "branch", "conflict-branch", "HEAD~1")
	tmpDir := t.TempDir()
	wt := filepath.Join(tmpDir, "wt")
	runGit(t, dir, "worktree", "add", wt, "conflict-branch")
	writeFile(t, wt, "conflict.txt", "branch version\n")
	runGit(t, wt, "add", ".")
	runGit(t, wt, "-c", "commit.gpgsign=false", "commit", "-m", "branch")
	runGit(t, dir, "worktree", "remove", wt, "--force")

	// Start merge (will conflict)
	err := ops.Merge("conflict-branch", "merge")
	require.Error(t, err)

	files, err := ops.UnmergedFiles()
	require.NoError(t, err)
	assert.Equal(t, []string{"conflict.txt"}, files)

	// Clean up
	runGit(t, dir, "merge", "--abort")
}

func TestUnmergedFiles_AfterResolution(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Create a merge conflict
	writeFile(t, dir, "conflict.txt", "main version\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "main")

	runGit(t, dir, "branch", "resolve-branch", "HEAD~1")
	tmpDir := t.TempDir()
	wt := filepath.Join(tmpDir, "wt")
	runGit(t, dir, "worktree", "add", wt, "resolve-branch")
	writeFile(t, wt, "conflict.txt", "branch version\n")
	runGit(t, wt, "add", ".")
	runGit(t, wt, "-c", "commit.gpgsign=false", "commit", "-m", "branch")
	runGit(t, dir, "worktree", "remove", wt, "--force")

	// Start merge (will conflict)
	err := ops.Merge("resolve-branch", "merge")
	require.Error(t, err)

	// Resolve with checkout --theirs + add
	require.NoError(t, ops.CheckoutTheirs("conflict.txt"))
	require.NoError(t, ops.AddFiles("conflict.txt"))

	files, err := ops.UnmergedFiles()
	require.NoError(t, err)
	assert.Empty(t, files, "should have no unmerged files after resolution")

	// Complete merge
	require.NoError(t, ops.Commit("resolved"))
}

func TestMergeAbort(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Create a merge conflict
	writeFile(t, dir, "abort-test.txt", "main version\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "main")

	runGit(t, dir, "branch", "abort-branch", "HEAD~1")
	tmpDir := t.TempDir()
	wt := filepath.Join(tmpDir, "wt")
	runGit(t, dir, "worktree", "add", wt, "abort-branch")
	writeFile(t, wt, "abort-test.txt", "branch version\n")
	runGit(t, wt, "add", ".")
	runGit(t, wt, "-c", "commit.gpgsign=false", "commit", "-m", "branch")
	runGit(t, dir, "worktree", "remove", wt, "--force")

	err := ops.Merge("abort-branch", "merge")
	require.Error(t, err)

	// Abort should restore clean state
	require.NoError(t, ops.MergeAbort())

	content := readFile(t, dir, "abort-test.txt")
	assert.Equal(t, "main version\n", content, "should be restored to pre-merge state")
	assert.False(t, ops.HasUnmergedFiles(), "should have no unmerged files after abort")
}

func TestCreateBranchFromHead(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	require.NoError(t, ops.CreateBranchFromHead("test-wip-branch"))
	assert.True(t, ops.BranchExists("test-wip-branch"))

	// Verify branch points at HEAD
	headHash, err := ops.RevParse("HEAD")
	require.NoError(t, err)
	branchHash, err := ops.RevParse("test-wip-branch")
	require.NoError(t, err)
	assert.Equal(t, headHash, branchHash, "new branch should point at HEAD")
}

func TestBackupBranchName(t *testing.T) {
	name := BackupBranchName("01-module/01-exercise")
	assert.True(t, strings.HasPrefix(name, "tdl/backup/01-module/01-exercise-"))

	// Verify timestamp format (2006-01-02T15-04-05)
	parts := strings.SplitN(name, "01-module/01-exercise-", 2)
	require.Len(t, parts, 2)
	_, err := time.Parse("2006-01-02T15-04-05", parts[1])
	assert.NoError(t, err, "timestamp should parse as 2006-01-02T15-04-05, got: %s", parts[1])
}

func TestOverrideWithBackupSave(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	exerciseDir := "01-module/01-exercise"

	// Create exercise files (user's solution)
	writeFile(t, dir, filepath.Join(exerciseDir, "main.go"), "package main // user solution\n")
	require.NoError(t, ops.AddAll(exerciseDir))
	require.NoError(t, ops.Commit("completed 01-module/01-exercise"))

	userHead, err := ops.RevParse("HEAD")
	require.NoError(t, err)

	// Create example solution branch via worktree (simulating syncGoldenSolution)
	goldenBranch := GoldenBranchName("01-module/01-exercise")
	goldenDir := t.TempDir()
	require.NoError(t, ops.WorktreeAdd(goldenDir, goldenBranch))

	goldenExercisePath := filepath.Join(goldenDir, exerciseDir)
	require.NoError(t, os.MkdirAll(goldenExercisePath, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(goldenExercisePath, "main.go"),
		[]byte("package main // example solution\n"),
		0644,
	))

	goldenOps := NewQuietOps(goldenDir)
	require.NoError(t, goldenOps.AddAll(exerciseDir))
	require.NoError(t, goldenOps.Commit("example solution for 01-module/01-exercise"))
	require.NoError(t, ops.WorktreeRemove(goldenDir))

	// --- Override flow (what `g` does) ---

	// 1. Save user's solution to a timestamped backup branch
	backupBranch := BackupBranchName("01-module/01-exercise")
	require.NoError(t, ops.CreateBranchFromHead(backupBranch))

	// 2. Checkout example solution files into working tree
	require.NoError(t, ops.CheckoutFiles(goldenBranch, exerciseDir))

	// 3. Stage + commit
	require.NoError(t, ops.AddAll(exerciseDir))
	assert.True(t, ops.HasStagedChanges(), "example solution files should differ from user's")
	require.NoError(t, ops.Commit("override with example solution for 01-module/01-exercise"))

	// --- Verify ---

	// Backup branch points at user's original HEAD
	backupHash, err := ops.RevParse(backupBranch)
	require.NoError(t, err)
	assert.Equal(t, userHead, backupHash, "backup branch should point at user's original HEAD")

	// Working tree has example solution content
	content := readFile(t, dir, filepath.Join(exerciseDir, "main.go"))
	assert.Equal(t, "package main // example solution\n", content, "working tree should have example solution content")

	// HEAD commit message matches expected pattern
	log, err := ops.Log(1)
	require.NoError(t, err)
	assert.Contains(t, log, "override with example solution for 01-module/01-exercise")
}

func TestInit_DefaultBranchMain(t *testing.T) {
	dir := t.TempDir()
	ops := NewOps(dir, false)

	created, err := ops.Init()
	require.NoError(t, err)
	require.True(t, created)

	// Create an initial commit so the branch name is visible
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	writeFile(t, dir, "README.md", "hello")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "initial")

	branch, err := ops.CurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "main", branch, "Init should create repo with default branch 'main'")
}

func TestMergeAutoResolveWithDate(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir, false)

	// Create diverging history so the merge produces a real merge commit.
	// On current branch: add one file
	writeFile(t, dir, "main-only.txt", "main content\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "main commit")

	currentBranch, err := ops.CurrentBranch()
	require.NoError(t, err)

	// Create a side branch from HEAD~1 with a conflicting change
	runGit(t, dir, "branch", "side", "HEAD~1")
	tmpDir := t.TempDir()
	wt := filepath.Join(tmpDir, "wt")
	runGit(t, dir, "worktree", "add", wt, "side")
	writeFile(t, wt, "side-only.txt", "side content\n")
	runGit(t, wt, "add", ".")
	runGit(t, wt, "-c", "commit.gpgsign=false", "commit", "-m", "side commit")
	runGit(t, dir, "worktree", "remove", wt, "--force")

	// Ensure we're on the main branch
	require.NoError(t, ops.CheckoutBranch(currentBranch))

	// Merge with a specific date
	mergeDate := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	err = ops.MergeAutoResolveWithDate("side", "merge with date", mergeDate)
	require.NoError(t, err)

	// Verify the merge commit date
	output := strings.TrimSpace(runGit(t, dir, "log", "-1", "--format=%aI"))
	assert.True(t, strings.HasPrefix(output, "2024-06-15T12:00:00"),
		"author date should match, got: %s", output)

	// Verify both files exist (merge succeeded)
	assert.FileExists(t, filepath.Join(dir, "main-only.txt"))
	assert.FileExists(t, filepath.Join(dir, "side-only.txt"))
}

func TestBackupBranchName_UniquePerCall(t *testing.T) {
	name1 := BackupBranchName("mod/ex")
	time.Sleep(time.Second)
	name2 := BackupBranchName("mod/ex")
	assert.NotEqual(t, name1, name2, "successive calls should produce different names")
}
