//go:build !windows

package internal

import "golang.org/x/sys/unix"

// dirWritable returns nil if the calling process can create files in dir.
// Uses access(2), which the kernel evaluates against the process's
// credentials — covering ownership, supplementary groups, ACLs, and root.
func dirWritable(dir string) error {
	return unix.Access(dir, unix.W_OK)
}
