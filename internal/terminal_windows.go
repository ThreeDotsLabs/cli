//go:build windows

package internal

func FlushTerminalInput(fd int) error { return nil }
