package logger

import (
	"os"
	"syscall"
)

func fileOpen(name string, flag int, perm os.FileMode) (*os.File, error) {
	oldmask := syscall.Umask(0)
	fd, err := os.OpenFile(name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, perm)
	syscall.Umask(oldmask)
	return fd, err
}
