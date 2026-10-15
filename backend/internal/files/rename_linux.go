//go:build linux

package files

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
)

// Keep both parent directories open so neither symlink swaps nor destination
// races can redirect the move or silently replace an existing file.
func renameNoReplace(src *os.Root, from string, dst *os.Root, to string) error {
	source, err := src.Open(filepath.Dir(from))
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := dst.Open(filepath.Dir(to))
	if err != nil {
		return err
	}
	defer target.Close()
	return unix.Renameat2(int(source.Fd()), filepath.Base(from), int(target.Fd()), filepath.Base(to), unix.RENAME_NOREPLACE)
}

func isCrossDevice(err error) bool { return errors.Is(err, unix.EXDEV) }
