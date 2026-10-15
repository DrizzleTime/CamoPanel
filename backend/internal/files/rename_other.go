//go:build !linux

package files

import (
	"fmt"
	"os"
)

func renameNoReplace(src *os.Root, from string, dst *os.Root, to string) error {
	if _, err := dst.Lstat(to); err == nil {
		return fmt.Errorf("目标已存在")
	} else if !os.IsNotExist(err) {
		return err
	}
	if src.Name() != dst.Name() {
		return fmt.Errorf("不支持跨文件根重命名")
	}
	return src.Rename(from, to)
}

func isCrossDevice(err error) bool { return false }
