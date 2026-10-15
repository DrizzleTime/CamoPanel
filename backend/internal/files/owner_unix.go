//go:build unix

package files

import (
	"io/fs"
	"os"
	"os/user"
	"strconv"
	"sync"
	"syscall"
)

var owners sync.Map
var groups sync.Map

func fileOwner(info fs.FileInfo) (string, string) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "—", "—"
	}
	uid, gid := strconv.FormatUint(uint64(stat.Uid), 10), strconv.FormatUint(uint64(stat.Gid), 10)
	owner := uid
	group := gid
	if cached, ok := owners.Load(uid); ok {
		owner = cached.(string)
	} else {
		if account, err := user.LookupId(uid); err == nil {
			owner = account.Username
		}
		owners.Store(uid, owner)
	}
	if cached, ok := groups.Load(gid); ok {
		group = cached.(string)
	} else {
		if account, err := user.LookupGroupId(gid); err == nil {
			group = account.Name
		}
		groups.Store(gid, group)
	}
	return owner, group
}

func preserveOwnership(file *os.File, info fs.FileInfo) error {
	original, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	created, err := file.Stat()
	if err != nil {
		return err
	}
	actual, ok := created.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	if actual.Uid == original.Uid && actual.Gid == original.Gid {
		return nil
	}
	return file.Chown(int(original.Uid), int(original.Gid))
}
