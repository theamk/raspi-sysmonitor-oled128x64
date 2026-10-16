package main

import (
	"fmt"
	"os"
	"syscall"
)

// acquireLock takes an exclusive flock on path; the kernel drops it when the process exits, even on a crash.
func acquireLock(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if err == syscall.EWOULDBLOCK {
			return nil, fmt.Errorf("another instance is already running (lock %s)", path)
		}
		return nil, err
	}
	return f, nil
}
