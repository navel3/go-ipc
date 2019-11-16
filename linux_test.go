// +build linux

package ipc

import (
	"golang.org/x/sys/unix"
)

const (
	SoSndbuf = unix.SO_SNDBUF
	SoRcvbuf = unix.SO_RCVBUF
)

type SocketType int

// minimize sock send & recv buffer size
func minimizeSocketBuffer(fd SocketType, optname int) (int, error) {
	if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, optname, 1); err != nil {
		return 0, err
	}
	return unix.GetsockoptInt(int(fd), unix.SOL_SOCKET, optname)
}
