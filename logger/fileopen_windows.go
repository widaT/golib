package logger

import (
	"os"
)

func fileOpen(name string, flag int, perm os.FileMode) (*os.File, error) {
	fd, err := os.OpenFile(name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, perm)
	return fd, err
}
