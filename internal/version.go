package internal

import (
	"runtime/debug"
	"strings"
)

// BinaryVersion returns the resolved binary version. It reads from build info
// (set by go install / goreleaser), falling back to "dev" for local builds.
func BinaryVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return strings.TrimPrefix(bi.Main.Version, "v")
	}
	return "dev"
}
