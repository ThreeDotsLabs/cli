//go:build linux

package internal

import "golang.org/x/sys/unix"

// EnableOutputProcessing re-enables output processing (OPOST + ONLCR) on fd
// after term.MakeRaw has disabled it, so that \n is translated to \r\n and
// terminal output looks correct while input is still in raw mode.
func EnableOutputProcessing(fd int) error {
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return err
	}
	termios.Oflag |= unix.OPOST | unix.ONLCR
	return unix.IoctlSetTermios(fd, unix.TCSETS, termios)
}
