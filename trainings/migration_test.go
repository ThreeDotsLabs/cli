package trainings

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ThreeDotsLabs/cli/trainings/config"
	"github.com/ThreeDotsLabs/cli/trainings/git"
)

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestPrintInitNeedsFreshDir_GitAvailable(t *testing.T) {
	cfg := config.TrainingConfig{
		TrainingName:   "go-event-driven",
		GitConfigured:  true,
		GitEnabled:     false,
		GitUnavailable: true,
	}

	out := captureStdout(func() { printInitNeedsFreshDir(cfg) })

	assert.Contains(t, out, "go-event-driven")
	assert.Contains(t, out, "cd ..")
	assert.Contains(t, out, "mkdir my-training")
	assert.Contains(t, out, "tdl training init go-event-driven .")
	assert.Contains(t, out, "Git is now available")
}

func TestPrintInitNeedsFreshDir_GitNotAvailable(t *testing.T) {
	origPath := os.Getenv("PATH")
	t.Cleanup(func() {
		os.Setenv("PATH", origPath)
		git.ResetCheckVersion()
	})
	os.Setenv("PATH", "")
	git.ResetCheckVersion()

	cfg := config.TrainingConfig{
		TrainingName:  "go-event-driven",
		GitConfigured: false,
		GitEnabled:    false,
	}

	out := captureStdout(func() { printInitNeedsFreshDir(cfg) })

	assert.Contains(t, out, "go-event-driven")
	assert.Contains(t, out, "Once git is installed")
	assert.Contains(t, out, "tdl training init go-event-driven .")
	assert.NotContains(t, out, "Git is now available")
}
