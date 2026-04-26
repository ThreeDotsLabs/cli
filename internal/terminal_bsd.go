//go:build darwin || freebsd || openbsd || netbsd

package internal

import "golang.org/x/sys/unix"

func EnableOutputProcessing(fd int) error {
	termios, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		return err
	}
	termios.Oflag |= unix.OPOST | unix.ONLCR
	return unix.IoctlSetTermios(fd, unix.TIOCSETA, termios)
}
