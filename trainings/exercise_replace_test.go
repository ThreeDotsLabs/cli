package trainings

import (
	"sort"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func TestMergeStartStateFiles(t *testing.T) {
	t.Run("first exercise (nil golden) returns scaffold as-is", func(t *testing.T) {
		scaffold := []*genproto.File{
			{Path: "a.txt", Content: "A"},
			{Path: "b.txt", Content: "B"},
		}
		merged := mergeStartStateFiles(nil, scaffold)
		assertFilesByPath(t, merged, map[string]string{
			"a.txt": "A",
			"b.txt": "B",
		})
	})

	t.Run("scaffold wins on path collision", func(t *testing.T) {
		golden := []*genproto.File{
			{Path: "shared.txt", Content: "from golden"},
			{Path: "golden-only.txt", Content: "G"},
		}
		scaffold := []*genproto.File{
			{Path: "shared.txt", Content: "from scaffold"}, // overrides golden
			{Path: "scaffold-only.txt", Content: "S"},
		}
		merged := mergeStartStateFiles(golden, scaffold)
		assertFilesByPath(t, merged, map[string]string{
			"shared.txt":        "from scaffold",
			"golden-only.txt":   "G",
			"scaffold-only.txt": "S",
		})
	})

	t.Run("regression: golden with filled-in placeholder is preserved when scaffold does not redeliver it", func(t *testing.T) {
		// This is the 0001_init_orders.up.sql scenario.
		// Earlier exercises scaffolded the file as empty; the user filled it in.
		// By a later exercise, the scaffold no longer includes that file — only
		// the prev-exercise golden does. The start state must preserve the
		// filled-in content.
		golden := []*genproto.File{
			{Path: "migrations/0001_init.sql", Content: "CREATE TABLE ..."},
			{Path: "common.go", Content: "package common"},
		}
		scaffold := []*genproto.File{
			{Path: "new_for_this_exercise.go", Content: "package new"},
		}
		merged := mergeStartStateFiles(golden, scaffold)
		assertFilesByPath(t, merged, map[string]string{
			"migrations/0001_init.sql": "CREATE TABLE ...", // survives from golden
			"common.go":                "package common",
			"new_for_this_exercise.go": "package new",
		})
	})
}

func TestReplaceExerciseFiles_is1to1(t *testing.T) {
	// The invariant: after replaceExerciseFiles, exerciseDir contains exactly
	// the replacement files — any extras are deleted. This is load-bearing
	// for the "sync with example" and "replace on conflict" UX: user's project
	// must never silently diverge from what the caller asked for.
	fs := afero.NewMemMapFs()
	rootFs := afero.NewBasePathFs(fs, "/").(*afero.BasePathFs)

	// Pre-populate exerciseDir with some files that should be removed.
	require.NoError(t, afero.WriteFile(fs, "/ex/stale_placeholder.sql", []byte("-- todo"), 0644))
	require.NoError(t, afero.WriteFile(fs, "/ex/user_scratch.txt", []byte("notes"), 0644))
	require.NoError(t, afero.WriteFile(fs, "/ex/keep_me.go", []byte("old content"), 0644))

	replacement := []*genproto.File{
		{Path: "keep_me.go", Content: "new content"},
		{Path: "fresh.go", Content: "fresh"},
	}
	require.NoError(t, replaceExerciseFiles(rootFs, replacement, "ex"))

	// keep_me.go is overwritten
	got, err := afero.ReadFile(fs, "/ex/keep_me.go")
	require.NoError(t, err)
	assert.Equal(t, "new content", string(got))

	// fresh.go is created
	got, err = afero.ReadFile(fs, "/ex/fresh.go")
	require.NoError(t, err)
	assert.Equal(t, "fresh", string(got))

	// stale_placeholder.sql and user_scratch.txt are DELETED (not in replacement).
	// This is the 1:1 invariant the fix enforces.
	_, err = fs.Stat("/ex/stale_placeholder.sql")
	assert.True(t, err != nil, "stale placeholder should have been deleted")
	_, err = fs.Stat("/ex/user_scratch.txt")
	assert.True(t, err != nil, "user scratch should have been deleted")
}

func assertFilesByPath(t *testing.T, got []*genproto.File, want map[string]string) {
	t.Helper()
	byPath := map[string]string{}
	for _, f := range got {
		byPath[f.Path] = f.Content
	}
	paths := make([]string, 0, len(byPath))
	for p := range byPath {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	assert.Equal(t, want, byPath, "merged files mismatch; paths present: %v", paths)
}
