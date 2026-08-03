package trainings

import (
	"os"
	"testing"
)

// The test binary's stdin is never a terminal, but internal.IsStdinTerminal() deliberately
// falls back to TERM/WT_SESSION/TERM_PROGRAM to detect mintty and Windows Terminal. Those
// are inherited from whatever shell runs `go test`, so without this the interactive branches
// would be taken and prompts would block on EOF — which fails differently depending on the
// developer's environment. Tests that need the interactive path can opt in with t.Setenv.
func TestMain(m *testing.M) {
	if _, set := os.LookupEnv("TDL_FORCE_INTERACTIVE"); !set {
		_ = os.Setenv("TDL_FORCE_INTERACTIVE", "false")
	}

	os.Exit(m.Run())
}
