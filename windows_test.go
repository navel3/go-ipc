// +build windows

package ipc

import (
	"golang.org/x/sys/windows"
)

const (
	SoSndbuf = windows.SO_SNDBUF
	SoRcvbuf = windows.SO_RCVBUF
)

type SocketType windows.Handle

// minimize sock send & recv buffer size
func minimizeSocketBuffer(fd SocketType, optname int) (int, error) {
	if err := windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, optname, 1); err != nil {
		return 0, err
	}
	return 1234, nil // Getsockopt is not supported by windows
	//return windows.GetsockoptInt(fd, windows.SOL_SOCKET, optname)
}
