package ipc

import (
	"encoding/binary"
	"net"
	"os"

	"github.com/Microsoft/go-winio"
)

// Accept implements the Accept method in the net.Listener interface; it waits
// for the next call and return a Conn.
func Listen(name string) (*Listener, error) {
	l, err := winio.ListenPipe(`\\.\pipe\`+name, &winio.PipeConfig{
		SecurityDescriptor: "",
		MessageMode:        false,
		InputBufferSize:    1024,
		OutputBufferSize:   1024,
	})
	if err != nil {
		return nil, err
	}

	return &Listener{l: l}, nil
}

// Dial connects to the named pipe.
func (l *Listener) Accept() (*Conn, error) {
	conn, err := l.l.Accept()
	if err != nil {
		return nil, err
	}

	c := newConn(conn)
	pid, err := recvSendPID(conn)
	if err != nil {
		return nil, err
	}

	c.fileGW.pid = pid
	c.socketGW.pid = pid

	return c, nil
}

func Dial(name string) (*Conn, error) {
	conn, err := winio.DialPipe(`\\.\pipe\`+name, nil)
	if err != nil {
		return nil, err
	}

	c := newConn(conn)
	pid, err := sendRecvPID(conn)
	if err != nil {
		return nil, err
	}

	c.fileGW.pid = pid
	c.socketGW.pid = pid

	return c, nil
}

func sendRecvPID(conn net.Conn) (pid uint32, err error) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(os.Getpid()))
	_, err = conn.Write(buf[:])
	if err != nil {
		return
	}

	_, err = conn.Read(buf[:])
	if err != nil {
		return
	}

	return binary.BigEndian.Uint32(buf[:]), nil
}

func recvSendPID(conn net.Conn) (pid uint32, err error) {
	var buf [4]byte
	_, err = conn.Read(buf[:])
	if err != nil {
		return
	}

	pid = binary.BigEndian.Uint32(buf[:])

	binary.BigEndian.PutUint32(buf[:], uint32(os.Getpid()))
	_, err = conn.Write(buf[:])
	return
}
