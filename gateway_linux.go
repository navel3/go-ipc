package ipc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

type serializer interface {
	serialize(w io.Writer) error
}

type deserializer interface {
	deserialize(r io.Reader) error
}

type gateway struct {
	wbuf bytes.Buffer
	rbuf []byte
}

func (gw *gateway) sendImpl(conn net.Conn, fd int, s serializer, msg []byte) (err error) {
	rawConn, err := conn.(*net.UnixConn).SyscallConn()
	if err != nil {
		return
	}

	gw.wbuf.Reset()
	writeWithLength(&gw.wbuf, func(b *bytes.Buffer) error {
		return s.serialize(b)
	})

	var n int
	rawConn.Control(func(connFd uintptr) {
		rights := unix.UnixRights(fd)
		for {
			n, err = unix.SendmsgN(int(connFd), gw.wbuf.Bytes(), rights, nil, 0)
			if err == nil || err != unix.EAGAIN {
				break
			}
			// TODO: There is no way to get write deadline of conn
			if ok, _ := waitIOEvent(waitWrite, fd, waitForever); !ok {
				break
			}
		}
	})

	if n < gw.wbuf.Len() {
		err = writeAll(conn, gw.wbuf.Bytes()[n:])
	}

	if len(msg) > 0 {
		err = writeData(msg, conn)
		if err != nil {
			return
		}
	}

	return
}

func (gw *gateway) receiveImpl(conn net.Conn, s deserializer) (fd int, err error) {
	rawConn, err := conn.(*net.UnixConn).SyscallConn()
	if err != nil {
		return
	}

	rights := unix.UnixRights(0)
	var dlen uint32
	rawConn.Control(func(connFd uintptr) {
		var buf [4]byte
		for {
			n, _, _, _, err := syscall.Recvmsg(int(connFd), buf[:], rights, 0)
			if err == nil {
				if n != 4 {
					panic(fmt.Sprintf("n must be 4 but was %v", n))
				}
				break
			}
			if err != unix.EAGAIN {
				break
			}
			// TODO: There is no way to get read deadline of conn
			if ok, _ := waitIOEvent(waitRead, fd, waitForever); !ok {
				break
			}
		}
		dlen = binary.BigEndian.Uint32(buf[:])
	})

	gw.rbuf = fitLength(gw.rbuf, int(dlen))
	err = readAll(conn, gw.rbuf)
	if err != nil {
		return
	}

	br := bytes.NewReader(gw.rbuf)

	err = s.deserialize(br)
	if err != nil {
		return
	}

	sockmsg, err := unix.ParseSocketControlMessage(rights)
	if err != nil {
		return
	}
	fds, err := unix.ParseUnixRights(&sockmsg[0])
	if err != nil {
		return
	}
	if len(fds) != 1 {
		panic("unexpected error. fds must be 1")
	}

	return fds[0], nil
}

const (
	waitRead = iota
	waitWrite
)

var waitForever *unix.Timeval = nil // dont update
var nowait = &unix.Timeval{}        // dont update

func waitIOEvent(mode, fd int, timeout *unix.Timeval) (bool, error) {
	fds := &unix.FdSet{}
	fds.Set(fd)

	var n int
	var err error
	if mode == waitWrite {
		n, err = unix.Select(fd+1, nil, fds, nil, timeout)
	} else {
		n, err = unix.Select(fd+1, fds, nil, nil, timeout)
	}
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	return true, nil
}
