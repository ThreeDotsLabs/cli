package trainings

import (
	"sort"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

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
