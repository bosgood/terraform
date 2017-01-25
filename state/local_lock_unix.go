// +build !windows

package state

import (
	"fmt"
	"log"
	"os"
	"syscall"
)

// use fcntl POSIX locks for the most consistent behavior across platforms, and
// hopefully some campatibility over NFS and CIFS.
func (s *LocalState) lock() error {
	flock := &syscall.Flock_t{
		Type:   syscall.F_RDLCK | syscall.F_WRLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
	}

	fd := s.stateFile.Fd()
	err := syscall.FcntlFlock(fd, syscall.F_SETLK, flock)
	if err != nil {
		lockInfo, err := s.lockInfo()
		if err != nil {
			// we can't get any lock info, at least get the PID if we can.
			if err := syscall.FcntlFlock(fd, syscall.F_GETLK, flock); err != nil {
				log.Fatalf("state file locked by pid: %d", flock.Pid)
			}
			// the error is only going to be "resource temporarily unavailable"
			return fmt.Errorf("state file locked")
		}

		return lockInfo.Err()
	}

	return nil
}

func (s *LocalState) unlock() error {
	flock := &syscall.Flock_t{
		Type:   syscall.F_UNLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
	}

	fd := s.stateFile.Fd()
	return syscall.FcntlFlock(fd, syscall.F_SETLK, flock)
}
