// +build !windows

package state

import (
	"fmt"
	"os"
)

// We use a symlink lock technique for unix systems so that we aren't dependent
// on filesystem settings or compatibility.
func (s *LocalState) lock(reason string) error {
	lockPath, lockInfoPath := s.lockPaths()

	// Create the broken symlink first, because that's the atomic operation.
	// Once we have a symlink, then we can fill in the info at our leisure.
	err := os.Symlink(lockInfoPath, lockPath)
	if err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to lock state file %q: %s", s.Path, err)
		}

		// Read any info through the symlink path, in case it was a regular
		// file created in windows.
		lockInfo, err := s.lockInfo(lockPath)
		if err != nil {
			return err
		}

		// TODO: Should we automatically unlock after expiration?
		//       This would align with lock implementations where the lock
		//       disappears after expiration.
		return lockInfo.Err()
	}

	if err := s.writeLockInfo(reason, lockInfoPath); err != nil {
		s.Unlock()
		return err
	}

	return nil
}
