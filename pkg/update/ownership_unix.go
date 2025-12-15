//go:build unix

package update

import (
	"os"
	"syscall"
)

// getFileOwnership extracts uid and gid from file info on Unix systems
func getFileOwnership(info os.FileInfo) (uid, gid int) {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return int(stat.Uid), int(stat.Gid)
	}
	return -1, -1
}

// chownFile changes the ownership of a file on Unix systems
// Returns nil if uid/gid are -1 (unknown) or if chown is not needed
func chownFile(path string, uid, gid int) error {
	if uid < 0 || gid < 0 {
		return nil
	}
	return os.Chown(path, uid, gid)
}
