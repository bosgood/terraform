// +build windows
package state

// TODO: windows file locking
func (s *LocalState) lock(reason string) error {
	return nil
}

func (s *LocalState) Unlock() error {
	return nil
}
