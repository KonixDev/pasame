//go:build !windows

package lifecycle

import (
	"os"
	"syscall"
)

func tryLock(f *os.File) error { return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) }
func unlock(f *os.File)        { syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }
