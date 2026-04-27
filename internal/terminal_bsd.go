//go:build darwin || freebsd || openbsd || netbsd

package internal

import "golang.org/x/sys/unix"

// FlushTerminalInput discards any bytes the kernel has buffered in the
// terminal's input queue. Called after term.MakeRaw at prompt entry so that
// keystrokes the user typed in cooked mode while the previous command was
// running don't carry over and trigger spurious actions.
func FlushTerminalInput(fd int) error {
	// On BSD/Darwin, TIOCFLUSH takes a pointer to an int with bitmask
	// FREAD (0x1) / FWRITE (0x2). TCIFLUSH happens to equal FREAD (0x1).
	return unix.IoctlSetPointerInt(fd, unix.TIOCFLUSH, unix.TCIFLUSH)
}
