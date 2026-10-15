//go:build !unix

package files

import (
	"io/fs"
	"os"
)

func fileOwner(info fs.FileInfo) (string, string) { return "—", "—" }

func preserveOwnership(file *os.File, info fs.FileInfo) error { return nil }
