// Package git provides git operations for the training CLI.
//
// PRINCIPLE: Never modify user's code without explicit permission.
//
// The CLI must not stash, revert, reset, or otherwise alter user files silently.
// Operations that prepare exercise scaffolds should use isolated worktrees, not
// the user's working tree. When a destructive operation is unavoidable (e.g.
// replacing exercise files with golden solution or resolving conflicts by
// accepting ours), we MUST:
//  1. Ask the user for confirmation first.
//  2. Save their current code to a backup branch (tdl/backup/...) before overwriting.
//  3. Tell the user how to restore their code from the backup branch.
//
// Read-only operations (status, diff, log, merge-tree) are always safe.
// Worktree operations are safe because they work in an isolated temp directory.
// Commits of user's own work (auto-commit on exercise completion) are safe because
// they preserve, not destroy, user code.
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

// Ops provides git operations for the training CLI.
// All methods are no-ops when enabled is false.
//
// Before adding a method that writes to the user's working tree, read the
// package-level doc comment above — it describes the backup-first principle.
type Ops struct {
	rootDir string
	enabled bool
	quiet   bool // suppress printCmd output (for internal worktree operations)
}

func NewOps(rootDir string, disabled bool) *Ops {
	return &Ops{
		rootDir: rootDir,
		enabled: !disabled,
	}
}

// NewQuietOps creates an Ops that suppresses user-visible output.
// Use for internal operations like worktree staging/committing.
func NewQuietOps(rootDir string) *Ops {
	return &Ops{
		rootDir: rootDir,
		enabled: true,
		quiet:   true,
	}
}

func (g *Ops) Enabled() bool {
	return g.enabled
}

func (g *Ops) RootDir() string {
	return g.rootDir
}

// printCmd displays a mimicked git command to the user.
func (g *Ops) printCmd(display string) {
	if g.quiet {
		return
	}
	fmt.Printf("%s %s\n", color.MagentaString("•••"), "git "+display)
}

// PrintInfo displays a caller-controlled informational message in git command style.
func (g *Ops) PrintInfo(display string) {
	if g.quiet {
		return
	}
	fmt.Printf("%s %s\n", color.MagentaString("•••"), display)
}

func (g *Ops) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.rootDir

	logrus.WithFields(logrus.Fields{
		"args": args,
		"dir":  g.rootDir,
	}).Debug("Running git command")

	out, err := cmd.CombinedOutput()
	output := strings.TrimRight(string(out), " \t\n\r")

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"args":   args,
			"output": output,
			"err":    err,
		}).Debug("Git command failed")
		return output, fmt.Errorf("git %s: %s: %w", args[0], output, err)
	}

	return output, nil
}

// Init initializes a git repo. Returns true if a new repo was created.
func (g *Ops) Init() (bool, error) {
	if !g.enabled {
		return false, nil
	}

	if g.IsRepo() {
		return false, nil
	}

	g.printCmd("init")
	_, err := g.run("init", "--initial-branch=main")
	if err != nil {
		return false, err
	}

	return true, nil
}

// IsRepo returns true if the rootDir is inside a git repository.
func (g *Ops) IsRepo() bool {
	if !g.enabled {
		return false
	}

	_, err := g.run("rev-parse", "--git-dir")
	return err == nil
}

// CurrentBranch returns the name of the current branch.
func (g *Ops) CurrentBranch() (string, error) {
	if !g.enabled {
		return "", nil
	}

	return g.run("branch", "--show-current")
}

// HasUncommittedChanges returns true if there are uncommitted changes in the given directory.
func (g *Ops) HasUncommittedChanges(dir string) bool {
	if !g.enabled {
		return false
	}

	output, err := g.run("status", "--porcelain", "--", dir)
	if err != nil {
		return false
	}

	return output != ""
}

// HasStagedChanges returns true if there are staged changes ready to commit.
func (g *Ops) HasStagedChanges() bool {
	if !g.enabled {
		return false
	}

	_, err := g.run("diff", "--cached", "--quiet")
	// exit code 1 means there are changes
	return err != nil
}

// AddAll stages all changes in the given directory.
func (g *Ops) AddAll(dir string) error {
	if !g.enabled {
		return nil
	}

	g.printCmd(fmt.Sprintf("add %s/", dir))
	_, err := g.run("add", "--all", "--", dir)
	return err
}

// AddFiles stages specific files.
func (g *Ops) AddFiles(paths ...string) error {
	if !g.enabled {
		return nil
	}

	g.printCmd(fmt.Sprintf("add %s", strings.Join(paths, " ")))
	args := append([]string{"add", "--"}, paths...)
	_, err := g.run(args...)
	return err
}

// Commit creates a commit with the given message.
// Disables gpg signing and hooks to avoid interference.
func (g *Ops) Commit(msg string) error {
	if !g.enabled {
		return nil
	}

	g.printCmd(fmt.Sprintf("commit -m %q", msg))
	_, err := g.run(
		"-c", "commit.gpgsign=false",
		"-c", "core.hooksPath=/dev/null",
		"commit", "-m", msg,
	)
	return err
}

// CommitAllowEmpty creates a commit even when there are no staged changes.
func (g *Ops) CommitAllowEmpty(msg string) error {
	if !g.enabled {
		return nil
	}

	g.printCmd(fmt.Sprintf("commit -m %q", msg))
	_, err := g.run(
		"-c", "commit.gpgsign=false",
		"-c", "core.hooksPath=/dev/null",
		"commit", "--allow-empty", "-m", msg,
	)
	return err
}

// CommitWithDate creates a commit with an explicit author/committer date.
func (g *Ops) CommitWithDate(msg string, date time.Time) error {
	if !g.enabled {
		return nil
	}

	g.printCmd(fmt.Sprintf("commit -m %q", msg))

	cmd := exec.Command("git",
		"-c", "commit.gpgsign=false",
		"-c", "core.hooksPath=/dev/null",
		"commit", "-m", msg,
	)
	cmd.Dir = g.rootDir
	dateStr := date.Format(time.RFC3339)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE="+dateStr,
		"GIT_COMMITTER_DATE="+dateStr,
	)

	logrus.WithFields(logrus.Fields{
		"args": []string{"commit", "-m", msg},
		"dir":  g.rootDir,
		"date": dateStr,
	}).Debug("Running git commit with date")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// CommitAll stages all changes and commits with the given message.
func (g *Ops) CommitAll(msg string) error {
	if !g.enabled {
		return nil
	}

	g.printCmd(fmt.Sprintf("commit -am %q", msg))
	_, err := g.run(
		"-c", "commit.gpgsign=false",
		"-c", "core.hooksPath=/dev/null",
		"commit", "-am", msg,
	)
	return err
}

// BranchExists returns true if the named branch exists.
func (g *Ops) BranchExists(name string) bool {
	if !g.enabled {
		return false
	}

	_, err := g.run("rev-parse", "--verify", "refs/heads/"+name)
	return err == nil
}

// DeleteBranch force-deletes the named branch.
func (g *Ops) DeleteBranch(name string) error {
	if !g.enabled {
		return nil
	}

	_, err := g.run("branch", "-D", name)
	return err
}

// WorktreeAdd creates a new worktree at dir on a new branch.
// This is an internal operation — no user-visible output.
func (g *Ops) WorktreeAdd(dir, branch string) error {
	if !g.enabled {
		return nil
	}

	_, err := g.run("worktree", "add", dir, "-b", branch)
	return err
}

// WorktreeRemove removes a worktree at dir.
func (g *Ops) WorktreeRemove(dir string) error {
	if !g.enabled {
		return nil
	}

	_, err := g.run("worktree", "remove", dir, "--force")
	return err
}

// CheckoutBranch switches to the named branch.
func (g *Ops) CheckoutBranch(name string) error {
	if !g.enabled {
		return nil
	}

	g.printCmd(fmt.Sprintf("checkout %s", name))
	_, err := g.run("checkout", name)
	return err
}

// Merge merges the named branch into the current branch.
func (g *Ops) Merge(branch, msg string) error {
	if !g.enabled {
		return nil
	}

	g.printCmd(fmt.Sprintf("merge %s", branch))
	_, err := g.run("merge", branch, "--no-edit", "-m", msg)
	return err
}

// MergeAutoResolve merges the named branch, auto-resolving conflicts by preferring
// the incoming (theirs) version. Use this ONLY during restore where the merged content
// is immediately overwritten by the user's saved solution.
func (g *Ops) MergeAutoResolve(branch, msg string) error {
	if !g.enabled {
		return nil
	}

	g.printCmd(fmt.Sprintf("merge %s", branch))
	_, err := g.run("merge", branch, "--no-edit", "-m", msg, "-X", "theirs")
	return err
}

// MergeAutoResolveWithDate merges with auto-resolve and an explicit commit date.
func (g *Ops) MergeAutoResolveWithDate(branch, msg string, date time.Time) error {
	if !g.enabled {
		return nil
	}

	g.printCmd(fmt.Sprintf("merge %s", branch))

	cmd := exec.Command("git", "merge", branch, "--no-edit", "-m", msg, "-X", "theirs")
	cmd.Dir = g.rootDir
	dateStr := date.Format(time.RFC3339)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE="+dateStr,
		"GIT_COMMITTER_DATE="+dateStr,
	)

	logrus.WithFields(logrus.Fields{
		"args": []string{"merge", branch, "-m", msg, "-X", "theirs"},
		"dir":  g.rootDir,
		"date": dateStr,
	}).Debug("Running git merge with date")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git merge: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// DiffStat returns a diff stat between two refs.
func (g *Ops) DiffStat(ref1, ref2 string) (string, error) {
	if !g.enabled {
		return "", nil
	}

	return g.run("diff", "--stat", ref1+".."+ref2)
}

// DiffStatPath returns a diff stat between two refs, restricted to a specific path.
func (g *Ops) DiffStatPath(ref1, ref2, path string) (string, error) {
	if !g.enabled {
		return "", nil
	}

	return g.run("diff", "--stat", ref1+".."+ref2, "--", path)
}

// Log returns the last n commit messages in oneline format.
func (g *Ops) Log(n int) (string, error) {
	if !g.enabled {
		return "", nil
	}

	return g.run("log", "--oneline", fmt.Sprintf("-%d", n))
}

// HasCommits returns true if the repository has at least one commit.
func (g *Ops) HasCommits() bool {
	if !g.enabled {
		return false
	}

	_, err := g.run("rev-parse", "HEAD")
	return err == nil
}

// RevParse resolves a git ref to a commit hash.
func (g *Ops) RevParse(ref string) (string, error) {
	if !g.enabled {
		return "", nil
	}

	return g.run("rev-parse", ref)
}

// WorktreeAddFrom creates a worktree at dir on a new branch from a specific start point.
// startPoint can be a branch name or commit hash.
// This is an internal operation — no user-visible output.
func (g *Ops) WorktreeAddFrom(dir, branch, startPoint string) error {
	if !g.enabled {
		return nil
	}

	_, err := g.run("worktree", "add", dir, "-b", branch, startPoint)
	return err
}

// MergeTreeCheck detects merge conflicts WITHOUT touching the working tree.
// Uses `git merge-tree --write-tree HEAD <branch>` (Git 2.38+).
// Returns nil conflictFiles for clean merge, or list of conflicting file paths.
func (g *Ops) MergeTreeCheck(branch string) (conflictFiles []string, err error) {
	if !g.enabled {
		return nil, nil
	}

	cmd := exec.Command("git", "-c", "core.preloadIndex=true", "merge-tree", "--write-tree", "HEAD", branch)
	cmd.Dir = g.rootDir
	// Force English output so we can parse "CONFLICT" lines regardless of locale
	cmd.Env = append(os.Environ(), "LC_ALL=C")

	logrus.WithFields(logrus.Fields{
		"args": []string{"merge-tree", "--write-tree", "HEAD", branch},
		"dir":  g.rootDir,
	}).Debug("Running git command")

	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if err == nil {
		// Exit code 0 = clean merge, no conflicts
		return nil, nil
	}

	// Exit code 1 with merge-tree output = conflicts detected
	// Parse "CONFLICT" lines for file paths
	var files []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "CONFLICT") {
			// Lines look like: "CONFLICT (content): Merge conflict in <path>"
			if idx := strings.Index(line, "Merge conflict in "); idx >= 0 {
				path := strings.TrimSpace(line[idx+len("Merge conflict in "):])
				files = append(files, path)
			}
		}
	}

	if len(files) > 0 {
		return files, nil
	}

	// If we got an error but no conflict lines, merge-tree may not be available
	return nil, fmt.Errorf("git merge-tree failed: %s: %w", output, err)
}

// --- Working tree mutations ---
// These methods modify files in the user's working tree. They must ONLY be called
// after saving user's code to a backup branch and obtaining user confirmation.
// See package-level doc for the full principle.

// CheckoutTheirs resolves a merge conflict by accepting the incoming (theirs) version.
func (g *Ops) CheckoutTheirs(path string) error {
	if !g.enabled {
		return nil
	}

	g.printCmd(fmt.Sprintf("checkout --theirs -- %s", path))
	_, err := g.run("checkout", "--theirs", "--", path)
	return err
}

// CheckoutFiles checks out files from a branch into the working tree.
func (g *Ops) CheckoutFiles(branch, dir string) error {
	if !g.enabled {
		return nil
	}

	g.printCmd(fmt.Sprintf("checkout %s -- %s", branch, dir))
	_, err := g.run("checkout", branch, "--", dir)
	return err
}

// UnmergedFiles returns paths of files with unresolved merge conflicts.
func (g *Ops) UnmergedFiles() ([]string, error) {
	if !g.enabled {
		return nil, nil
	}

	output, err := g.run("diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return nil, nil
	}

	var files []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// HasUnmergedFiles returns true if there are unresolved merge conflicts.
func (g *Ops) HasUnmergedFiles() bool {
	files, err := g.UnmergedFiles()
	if err != nil {
		return false
	}
	return len(files) > 0
}

// MergeAbort aborts an in-progress merge.
func (g *Ops) MergeAbort() error {
	if !g.enabled {
		return nil
	}

	_, err := g.run("merge", "--abort")
	return err
}

// CreateBranchFromHead creates a new branch pointing at the current HEAD.
func (g *Ops) CreateBranchFromHead(name string) error {
	if !g.enabled {
		return nil
	}

	_, err := g.run("branch", name)
	return err
}

// CreateBranchFrom creates a new branch pointing at the given ref.
func (g *Ops) CreateBranchFrom(name, ref string) error {
	if !g.enabled {
		return nil
	}

	_, err := g.run("branch", name, ref)
	return err
}

// InitBranchName returns the init branch name for the given exercise directory.
func InitBranchName(exerciseDir string) string {
	return "tdl/init/" + filepath.ToSlash(exerciseDir)
}

// GoldenBranchName returns the golden branch name for the given exercise directory.
func GoldenBranchName(exerciseDir string) string {
	return "tdl/golden/" + filepath.ToSlash(exerciseDir)
}

// BackupBranchName returns a timestamped backup branch name.
// Each call produces a unique name so backups accumulate.
func BackupBranchName(exerciseDir string) string {
	ts := time.Now().Format("2006-01-02T15-04-05")
	return "tdl/backup/" + filepath.ToSlash(exerciseDir) + "-" + ts
}
