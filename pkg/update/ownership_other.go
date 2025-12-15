//go:build !unix

package update

import (
	"os"
)

// getFileOwnership is a no-op on non-Unix systems
func getFileOwnership(info os.FileInfo) (uid, gid int) {
	return -1, -1
}

// chownFile is a no-op on non-Unix systems
func chownFile(path string, uid, gid int) error {
	return nil
}
