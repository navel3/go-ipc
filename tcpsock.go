package ipc

import (
	"net"
	"runtime"
	"time"
)

// TCPConn is a TCP connection received from IPC peer.
type TCPConn interface {
	net.Conn
	CloseRead() error
	CloseWrite() error
}

type tcpConn struct {
	sysSocket *sysSocket
	laddr     net.TCPAddr
	raddr     net.TCPAddr
	peeked    []byte
}

func (c *tcpConn) Read(b []byte) (n int, err error) {
	if len(c.peeked) > 0 {
		n = copy(b, c.peeked)
		c.peeked = c.peeked[n:]
		if yes, _ := c.sysSocket.isReadableState(); !yes {
			return
		}
	}

	b = b[n:]
	if len(b) == 0 {
		return
	}

	nn, err := c.sysSocket.read(b)
	runtime.KeepAlive(c)
	n += nn
	return
}

func (c *tcpConn) Write(b []byte) (n int, err error) {
	n, err = c.sysSocket.write(b)
	runtime.KeepAlive(c)
	return
}

func (c *tcpConn) Close() error {
	runtime.SetFinalizer(c, nil)
	return c.sysSocket.close()
}

func (c *tcpConn) CloseRead() error {
	err := c.sysSocket.closeRead()
	runtime.KeepAlive(c)
	return err
}

func (c *tcpConn) CloseWrite() error {
	err := c.sysSocket.closeWrite()
	runtime.KeepAlive(c)
	return err
}

func (c *tcpConn) SetDeadline(t time.Time) error {
	return c.sysSocket.setDeadline(t)
}

func (c *tcpConn) SetReadDeadline(t time.Time) error {
	return c.sysSocket.setReadDeadline(t)
}

func (c *tcpConn) SetWriteDeadline(t time.Time) error {
	return c.sysSocket.setWriteDeadline(t)
}

func (c *tcpConn) LocalAddr() net.Addr {
	return &c.laddr
}

func (c *tcpConn) RemoteAddr() net.Addr {
	return &c.raddr
}

func newTCPConn(ss *sysSocket, laddr, raddr net.TCPAddr, peeked []byte) TCPConn {
	tcp := &tcpConn{ss, laddr, raddr, peeked}
	runtime.SetFinalizer(tcp, (*tcpConn).Close)
	return tcp
}
