package main

import (
	"io"
	"log"
	"os"
	"syscall"
)

// Attempt to open and lock a terraform state file.
// Lock failure exits with 0 and writes "lock failed" to stderr.
func main() {
	if len(os.Args) != 2 {
		log.Fatal(os.Args[0], "statefile")
	}

	f, err := os.OpenFile(os.Args[1], os.O_RDWR, 0666)
	if err != nil {
		log.Fatal(err)
	}

	flock := &syscall.Flock_t{
		Type:   syscall.F_RDLCK | syscall.F_WRLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
	}

	if err := syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, flock); err != nil {
		// we check for this string rather than exit status so we no the
		// process executed
		io.WriteString(os.Stderr, "lock failed")
	}
}
