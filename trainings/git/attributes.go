package git

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

const (
	mergeAttributesFileName = "git-merge-attributes"
	mergeAttributes         = "go.sum merge=union\n"
)

// EnsureMergeAttributesFile writes the gitattributes file used by the CLI's merges into dir,
// unless it already has the expected content. It returns an absolute path with forward
// slashes, suitable for `git -c core.attributesFile=<path>` on every platform.
func EnsureMergeAttributesFile(dir string) (string, error) {
	path, err := filepath.Abs(filepath.Join(dir, mergeAttributesFileName))
	if err != nil {
		return "", fmt.Errorf("resolving merge attributes file path: %w", err)
	}

	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, []byte(mergeAttributes)) {
		return filepath.ToSlash(path), nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating directory for merge attributes file: %w", err)
	}
	if err := os.WriteFile(path, []byte(mergeAttributes), 0644); err != nil {
		return "", fmt.Errorf("writing merge attributes file %s: %w", path, err)
	}

	return filepath.ToSlash(path), nil
}
