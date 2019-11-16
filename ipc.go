// Package ipc implements cross platform inter process communication.
//
// The main feature of this package is passing a tcp connection to other
// process. On windows, IPC is implemented with github.com/Microsoft/go-winio.
package ipc

import (
	"encoding/binary"
	"io"
	"net"
	"os"
)

// Command represents a IPC command.
type Command byte

// Kinds of Command. See also ReceiveCommand.
const (
	DataCommand Command = iota
	FileCommand
	TCPConnCommand
)

// Listener is a IPC listener; it implements net.Listener interface.
type Listener struct {
	l net.Listener
}

// Close implements the Close method in the net.Listener interface; it stop the
// listening.
func (l *Listener) Close() error {
	return l.l.Close()
}

// Conn is a IPC connection; it implements net.Conn interface.
type Conn struct {
	conn     net.Conn
	socketGW *socketGateway
	fileGW   *fileGateway
}

// Close implements the Close method in the net.Listener interface; it close the
// connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// SendData sends byte array to the peer. See also ReceiveData.
func (c *Conn) SendData(d []byte) error {
	b := [1]byte{byte(DataCommand)}
	if _, err := c.conn.Write(b[:]); err != nil {
		return err
	}

	return writeData(d, c)
}

// ReceiveDataLen receives data length from the peer.
//
// You generally do not need to use this method; use ReceiveData.
func (c *Conn) ReceiveDataLen() (int, error) {
	var buf [4]byte
	_, err := c.Read(buf[:])
	if err != nil {
		return 0, err
	}

	return int(binary.BigEndian.Uint32(buf[:])), nil
}

// ReceiveData receives byte array from the peer. See also SendData.
func (c *Conn) ReceiveData() ([]byte, error) {
	dlen, err := c.ReceiveDataLen()
	if err != nil {
		return nil, err
	}

	data := make([]byte, dlen)
	err = readAll(c, data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// SendFile passes the file handle to the peer. f is closed when the passing is
// succeeded but not if an error occurs.
//
// msg is an additional information; it is not mandatory.
//
// See also ReceiveFile.
func (c *Conn) SendFile(f *os.File, msg []byte) error {
	buf := [1]byte{byte(FileCommand)}
	if _, err := c.conn.Write(buf[:]); err != nil {
		return err
	}
	return c.fileGW.send(c.conn, f, msg)
}

// ReceiveFile receives a file handle from the peer. The second return value
// indicate trailing data exists; call ReceiveData to receive it.
// See also SendFile.
func (c *Conn) ReceiveFile() (*os.File, bool, error) {
	return c.fileGW.receive(c.conn)
}

// SendTCPConn passes a TCP connection to the peer. conn is closed when the
// passing is succeeded but not if an error occurs.
//
// peeked is data peeked from Conn; it will be injected to TCPConn in the peer.
// This is intended to implements reverse proxy like feature.
// Specify nil if there is no peeked data.
//
// msg is an additional information. Specify nil if nothing.
//
// See also ReceiveFile.
func (c *Conn) SendTCPConn(conn *net.TCPConn, peeked, msg []byte) error {
	buf := [1]byte{byte(TCPConnCommand)}
	if _, err := c.conn.Write(buf[:]); err != nil {
		return err
	}
	return c.socketGW.send(c.conn, conn, peeked, msg)
}

// ReceiveTCPConn receives a TCP connection from the peer. The second return
// value indicate trailing data exists; call ReceiveData to receive it.
// See also SendTCPConn
func (c *Conn) ReceiveTCPConn() (TCPConn, bool, error) {
	sock, laddr, raddr, ld, withData, err := c.socketGW.receive(c.conn)
	if err != nil {
		return nil, false, err
	}
	return newTCPConn(sock, laddr, raddr, ld), withData, nil
}

// ReceiveCommand receives a command from the peer; it bocks until receives any
// command or an error occurs.
//
// Possible commands are:
//   DataCommand: The peer called SendData
//   FileCommand: The peer called SendFile
//   TCPConnCommand: The peer called SendTCPConn
func (c *Conn) ReceiveCommand() (Command, error) {
	var b [1]byte
	if _, err := c.conn.Read(b[:]); err != nil {
		return 0, err
	}
	return Command(b[0]), nil
}

// Read implements the Read method in the net.Conn interface.
func (c *Conn) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

// Write implements the Write method in the net.Conn interface.
func (c *Conn) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}

func newConn(conn net.Conn) *Conn {
	return &Conn{
		conn:     conn,
		socketGW: newSocketGateway(),
		fileGW:   newFileGateway(),
	}
}

func writeData(d []byte, w io.Writer) (err error) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(len(d)))

	err = writeAll(w, buf[:])
	if err != nil {
		return
	}

	return writeAll(w, d)
}
