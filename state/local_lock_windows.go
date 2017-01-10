// +build windows

package state

import (
	"fmt"
	"os"
)

func (s *LocalState) lock(reason string) error {
	lockPath, _ := s.lockPaths()

	// Use a normal file as the lock file on windows.
	// This won't be quite as robust as the symlink lock on unix, but it will
	// allow us to play along on a shared filesystem.
	// TODO: if we need mandatory locking on windows, this could be augmented
	//       with LockFileEx on the lockfile, but since this isn't supported by
	//       all unix systems, there may not be much benefit.
	lockFile, err := os.OpenFile(lockPath, os.O_RDWR|os.O_EXCL, 0600)
	if err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to lock state file %q: %s", s.Path, err)
		}

		lockInfo, err := s.lockInfo(lockPath)
		if err != nil {
			return err
		}

		return lockInfo.Err()
	}

	lockFile.Close()

	if err := s.writeLockInfo(reason, lockPath); err != nil {
		s.Unlock()
		return err
	}
	return nil
}
