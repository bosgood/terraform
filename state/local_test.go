package state

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestLocalState(t *testing.T) {
	ls := testLocalState(t)
	defer os.Remove(ls.Path)
	TestState(t, ls)
}

func TestLocalStateLocks(t *testing.T) {
	s := testLocalState(t)
	defer os.Remove(s.Path)

	// lock first
	if err := s.Lock("test"); err != nil {
		t.Fatal(err)
	}

	// second lock should fail
	if err := s.Lock("test"); err == nil {
		t.Fatal("expected lock failure")
	}

	if err := s.Unlock(); err != nil {
		t.Fatal(err)
	}

	// should be able to re-lock now
	if err := s.Lock("test"); err != nil {
		t.Fatal(err)
	}

	// Unlock should be repeatable
	if err := s.Unlock(); err != nil {
		t.Fatal(err)
	}
	if err := s.Unlock(); err != nil {
		t.Fatal(err)
	}

	// make sure both files are gone
	lockPath, lockInfoPath := s.lockPaths()
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatal("lock not removed")
	}
	if _, err := os.Stat(lockInfoPath); !os.IsNotExist(err) {
		t.Fatal("lock info not removed")
	}

}

func TestLocalState_pathOut(t *testing.T) {
	f, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	ls := testLocalState(t)
	ls.PathOut = f.Name()
	defer os.Remove(ls.Path)

	TestState(t, ls)
}

func TestLocalState_nonExist(t *testing.T) {
	ls := &LocalState{Path: "ishouldntexist"}
	if err := ls.RefreshState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if state := ls.State(); state != nil {
		t.Fatalf("bad: %#v", state)
	}
}

func TestLocalState_impl(t *testing.T) {
	var _ StateReader = new(LocalState)
	var _ StateWriter = new(LocalState)
	var _ StatePersister = new(LocalState)
	var _ StateRefresher = new(LocalState)
}

func testLocalState(t *testing.T) *LocalState {
	f, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	err = terraform.WriteState(TestStateInitial(), f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	ls := &LocalState{Path: f.Name()}
	if err := ls.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	return ls
}
