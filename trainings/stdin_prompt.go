package trainings

import (
	"bufio"
	"io"
	"os"

	"golang.org/x/term"

	"github.com/ThreeDotsLabs/cli/internal"
)

// enterPromptMode puts the terminal into raw input mode for the duration of a
// single prompt and flushes any input the user typed in cooked mode while the
// previous command was running. Returns a cleanup function the caller must
// defer. If MakeRaw fails, the cleanup is a no-op and reads will be in cooked
// mode (Enter required, less responsive but functional).
func (h *Handlers) enterPromptMode() func() {
	state, err := term.MakeRaw(0)
	if err != nil {
		return func() {}
	}
	h.sessionTermState = state
	_ = internal.FlushTerminalInput(0)
	return func() {
		_ = term.Restore(0, state)
		h.sessionTermState = nil
	}
}

// startScopedStdinReader spawns a goroutine that reads runes from os.Stdin and
// sends them on the returned channel. The goroutine exits when done is closed
// or stdin reaches EOF. On Ctrl+C (\x03), it restores the terminal and exits
// the process — matching pre-MCP internal.Prompt semantics.
//
// The caller MUST close(done) when finished (typically via defer) so the
// goroutine doesn't outlive the prompt indefinitely. Note: the goroutine may
// remain blocked in read() for one more keystroke after done is closed; this
// is bounded (one stuck read at a time) and benign — it exits on the next
// byte or stdin close.
func (h *Handlers) startScopedStdinReader(done <-chan struct{}) <-chan rune {
	runeCh := make(chan rune)
	go func() {
		defer close(runeCh)
		reader := bufio.NewReader(os.Stdin)
		for {
			r, _, err := reader.ReadRune()
			if err != nil {
				if err == io.EOF && internal.IsStdinTerminal() {
					// Spurious EOF from raw/cooked mode flips — recreate and retry.
					reader = bufio.NewReader(os.Stdin)
					continue
				}
				return
			}
			if r == '\x03' {
				h.restoreTerminal()
				os.Exit(0)
			}
			select {
			case runeCh <- r:
			case <-done:
				return
			}
		}
	}()
	return runeCh
}
