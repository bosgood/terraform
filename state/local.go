package state

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform/terraform"
)

// lock metadata structure for local locks
type lockInfo struct {
	// Path to the state file
	Path string
	// The time the lock was taken
	Created time.Time
	// The time this lock expires
	Expires time.Time
	// The lock reason passed to State.Lock
	Reason string
}

// return the lock info formatted in an error
func (l *lockInfo) Err() error {
	return fmt.Errorf("state file %q locked. created:%s, expires:%s, reason:%s",
		l.Path, l.Created, l.Expires, l.Reason)
}

// LocalState manages a state storage that is local to the filesystem.
type LocalState struct {
	// Path is the path to read the state from. PathOut is the path to
	// write the state to. If PathOut is not specified, Path will be used.
	// If PathOut already exists, it will be overwritten.
	Path    string
	PathOut string

	state     *terraform.State
	readState *terraform.State
	written   bool

	lockPath     string
	lockInfoPath string
}

// SetState will force a specific state in-memory for this local state.
func (s *LocalState) SetState(state *terraform.State) {
	s.state = state
	s.readState = state
}

// StateReader impl.
func (s *LocalState) State() *terraform.State {
	return s.state.DeepCopy()
}

// Lock implements a local filesystem state.Locker.
func (s *LocalState) Lock(reason string) error {
	return s.lock(reason)
}

func (s *LocalState) Unlock() error {
	lockPath, lockInfoPath := s.lockPaths()
	os.Remove(lockPath)
	os.Remove(lockInfoPath)
	return nil
}

// WriteState for LocalState always persists the state as well.
//
// StateWriter impl.
func (s *LocalState) WriteState(state *terraform.State) error {
	s.state = state

	path := s.PathOut
	if path == "" {
		path = s.Path
	}

	// If we don't have any state, we actually delete the file if it exists
	if state == nil {
		err := os.Remove(path)
		if err != nil && os.IsNotExist(err) {
			return nil
		}

		return err
	}

	// Create all the directories
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	s.state.IncrementSerialMaybe(s.readState)
	s.readState = s.state

	if err := terraform.WriteState(s.state, f); err != nil {
		return err
	}

	s.written = true
	return nil
}

// PersistState for LocalState is a no-op since WriteState always persists.
//
// StatePersister impl.
func (s *LocalState) PersistState() error {
	return nil
}

// StateRefresher impl.
func (s *LocalState) RefreshState() error {
	// If we've never loaded before, read from Path, otherwise we
	// read from PathOut.
	path := s.Path
	if s.written && s.PathOut != "" {
		path = s.PathOut
	}

	f, err := os.Open(path)
	if err != nil {
		// It is okay if the file doesn't exist, we treat that as a nil state
		if !os.IsNotExist(err) {
			return err
		}

		f = nil
	}

	var state *terraform.State
	if f != nil {
		defer f.Close()
		state, err = terraform.ReadState(f)
		if err != nil {
			return err
		}
	}

	s.state = state
	s.readState = state
	return nil
}

// return the paths for a symlink lock and lockInfo metadata.
func (s *LocalState) lockPaths() (lockPath, lockInfoPath string) {
	stateDir, stateName := filepath.Split(s.Path)
	if stateName == "" {
		panic("empty state file path")
	}

	if stateName[0] == '.' {
		stateName = stateName[1:]
	}

	lockPath = filepath.Join(stateDir, fmt.Sprintf(".%s.lock", stateName))
	lockInfoPath = filepath.Join(stateDir, fmt.Sprintf(".%s.lock.info", stateName))
	return
}

// lockInfo unmarshals the specified file into a lockInfo structure.
func (s *LocalState) lockInfo(path string) (*lockInfo, error) {
	infoData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("state file %q locked, but no info found", s.Path)
	}

	info := lockInfo{}
	err = json.Unmarshal(infoData, &info)
	if err != nil {
		return nil, fmt.Errorf("state file %q locked, but could not unmarshal lock info: %s", s.Path, err)
	}
	return &info, nil
}

// write a new lock info file
func (s *LocalState) writeLockInfo(reason, path string) error {
	lockInfo := &lockInfo{
		Path:    s.Path,
		Created: time.Now().UTC(),
		Expires: time.Now().Add(time.Hour).UTC(),
		Reason:  reason,
	}

	infoData, err := json.Marshal(lockInfo)
	if err != nil {
		panic(fmt.Sprintf("could not marshal lock info: %#v", lockInfo))
	}

	err = ioutil.WriteFile(s.lockInfoPath, infoData, 0600)
	if err != nil {
		return fmt.Errorf("could not write lock info for %q: %s", s.Path, err)
	}
	return nil
}
