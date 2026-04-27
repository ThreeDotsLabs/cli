//go:build windows

package internal

import (
	"errors"
	"fmt"
	"os"
)

// dirWritable on Windows confirms the directory exists and is reachable.
// Go's Windows file mode mapping does not faithfully reflect ACLs, so a
// deeper check would require querying the security descriptor. We rely on
// the actual update to surface permission errors with a clear message.
func dirWritable(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("not a directory")
	}
	f, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", dir, err)
	}
	_ = f.Close()
	return nil
}
