//go:build unix

package tty

import (
	"os"

	"golang.org/x/sys/unix"
)

func disableEcho() (func(), error) {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	fd := int(f.Fd())
	t, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		f.Close()
		return nil, err
	}
	old := *t
	t.Lflag &^= unix.ECHO
	if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, t); err != nil {
		f.Close()
		return nil, err
	}
	return func() {
		_ = unix.IoctlSetTermios(fd, ioctlWriteTermios, &old)
		_ = f.Close()
	}, nil
}
