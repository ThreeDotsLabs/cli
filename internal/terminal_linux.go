//go:build linux

package internal

import "golang.org/x/sys/unix"

// FlushTerminalInput discards any bytes the kernel has buffered in the
// terminal's input queue. Called after term.MakeRaw at prompt entry so that
// keystrokes the user typed in cooked mode while the previous command was
// running don't carry over and trigger spurious actions.
func FlushTerminalInput(fd int) error {
	return unix.IoctlSetInt(fd, unix.TCFLSH, unix.TCIFLUSH)
}
