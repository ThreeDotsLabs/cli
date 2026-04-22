package trainings

import (
	"fmt"

	"github.com/spf13/afero"

	"github.com/ThreeDotsLabs/cli/trainings/files"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
	"github.com/ThreeDotsLabs/cli/trainings/git"
)

// replaceExerciseFiles writes the given files into exerciseDir and DELETES any
// working-tree file in that dir not in the list.
//
// INVARIANT: after this call, exerciseDir is 1:1 with `files` — no stale user
// content. This is the SINGLE POINT that enforces the "sync/replace must not
// leave stale user files" invariant. Every code path that replaces the user's
// solution with example or start-state files MUST route through this function
// (via replaceExerciseFilesAndCommit).
//
// Do NOT inline this call with files.NewFilesSilent — that path is additive and
// will reintroduce the project-style-training staleness bug (empty placeholders
// from earlier scaffolds overwriting the user's filled-in work; see the
// 0001_init_orders.up.sql regression).
func replaceExerciseFiles(
	trainingRootFs *afero.BasePathFs,
	replacementFiles []*genproto.File,
	exerciseDir string,
) error {
	return files.NewFilesSilentDeleteUnused().WriteExerciseFiles(replacementFiles, trainingRootFs, exerciseDir)
}

// replaceExerciseFilesAndCommit is the complete orchestration for replacing
// the user's exercise files with a given file list: save backup → write files
// (1:1, deletes extras) → stage → commit. ALL callers that replace the user's
// solution with example / start-state content MUST go through this function:
//   - overrideWithGolden ('s' action):       files = golden(current) via GetGoldenSolution
//   - g during next/merge-conflict:          files = start state via GetExerciseStartState
//   - resetCleanFiles:                       files = start state via GetExerciseStartState
//
// The backup branch is REQUIRED — destructive ops must always be recoverable.
// If saveToBackupBranch returns errBackupAborted, that error is returned directly
// so callers can distinguish user-abort from other failures.
//
// Returns (committed=true, nil) on success with a commit created, (false, nil)
// if the resulting state matched HEAD with nothing to commit, or (false, err)
// on any failure.
func replaceExerciseFilesAndCommit(
	gitOps *git.Ops,
	trainingRootFs *afero.BasePathFs,
	replacementFiles []*genproto.File,
	exerciseDir string,
	backupBranch string,
	commitMsg string,
) (committed bool, err error) {
	if err := saveToBackupBranch(gitOps, backupBranch); err != nil {
		return false, err
	}
	if err := replaceExerciseFiles(trainingRootFs, replacementFiles, exerciseDir); err != nil {
		return false, fmt.Errorf("could not replace exercise files: %w", err)
	}
	if err := gitOps.AddAll(exerciseDir); err != nil {
		return false, fmt.Errorf("could not stage replaced files: %w", err)
	}
	if !gitOps.HasStagedChanges() {
		return false, nil
	}
	if err := gitOps.Commit(commitMsg); err != nil {
		return false, fmt.Errorf("could not commit replaced files: %w", err)
	}
	return true, nil
}
