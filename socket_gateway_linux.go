package ipc

import (
	"io"
	"net"
	"time"

	"golang.org/x/sys/unix"
)

type socketGateway struct {
	gateway
}

func (gw *socketGateway) send(conn net.Conn, sock *net.TCPConn, peeked, msg []byte) (err error) {
	rawSock, err := sock.SyscallConn()
	if err != nil {
		return
	}

	rawSock.Control(func(fd uintptr) {
		err = gw.sendImpl(conn,
			int(fd),
			&socketData{
				laddr:    *sock.LocalAddr().(*net.TCPAddr),
				raddr:    *sock.RemoteAddr().(*net.TCPAddr),
				peeked:   peeked,
				withData: len(msg) > 0,
			},
			msg)
	})

	if err == nil {
		sock.Close()
	}
	return
}

func (gw *socketGateway) receive(conn net.Conn) (sk *sysSocket, laddr net.TCPAddr, raddr net.TCPAddr, peeked []byte, withMsg bool, err error) {
	var sd socketData
	fd, err := gw.receiveImpl(conn, &sd)
	if err != nil {
		return
	}

	return &sysSocket{fd: fd}, sd.laddr, sd.raddr, sd.peeked, sd.withData, nil
}

func newSocketGateway() *socketGateway {
	return &socketGateway{}
}

type socketData struct {
	laddr    net.TCPAddr
	raddr    net.TCPAddr
	peeked   []byte
	withData bool
}

func (sd *socketData) serialize(w io.Writer) error {
	bw := &bytesWriter{w, nil}
	// laddr
	bw.writeBytes(sd.laddr.IP)
	bw.write(int32(sd.laddr.Port))
	bw.writeBytes([]byte(sd.laddr.Zone))
	// raddr
	bw.writeBytes(sd.raddr.IP)
	bw.write(int32(sd.raddr.Port))
	bw.writeBytes([]byte(sd.raddr.Zone))
	// peeked
	bw.writeBytes(sd.peeked)
	// withData
	bw.write(sd.withData)
	return bw.err
}

func (sd *socketData) deserialize(r io.Reader) error {
	br := &bytesReader{r, nil}
	var i32 int32
	// laddr
	sd.laddr.IP = br.readBytes()
	br.read(&i32)
	sd.laddr.Port = int(i32)
	sd.laddr.Zone = string(br.readBytes())
	// raddr
	sd.raddr.IP = br.readBytes()
	br.read(&i32)
	sd.raddr.Port = int(i32)
	sd.raddr.Zone = string(br.readBytes())
	// peeked
	sd.peeked = br.readBytes()
	// withData
	br.read(&sd.withData)
	return br.err
}

type sysSocket struct {
	fd            int
	readDeadline  time.Time
	writeDeadline time.Time
}

func (s *sysSocket) read(b []byte) (n int, err error) {
	for {
		n, _, err = unix.Recvfrom(s.fd, b, 0)
		if err == nil || err != unix.EAGAIN {
			break
		}
		// EAGAIN
		var ok bool
		ok, err = s.waitUntilReadable()
		if err != nil {
			break
		}
		if !ok {
			err = ErrTimeout
			break
		}

	}
	if err != nil && n < 0 {
		n = 0
	}
	return
}

func (s *sysSocket) write(b []byte) (n int, err error) {
	for {
		n, err = unix.Write(s.fd, b)
		if err == nil || err != unix.EAGAIN {
			break
		}
		// EAGAIN
		var ok bool
		ok, err = s.waitUntilWritable()
		if err != nil {
			break
		}
		if !ok {
			err = ErrTimeout
			break
		}

	}
	if err != nil && n < 0 {
		n = 0
	}
	return
}

func (s *sysSocket) close() error {
	return unix.Close(s.fd)
}

func (s *sysSocket) closeRead() error {
	return unix.Shutdown(s.fd, unix.SHUT_RD)
}

func (s *sysSocket) closeWrite() error {
	return unix.Shutdown(s.fd, unix.SHUT_WR)
}

func (s *sysSocket) setDeadline(t time.Time) error {
	s.setReadDeadline(t)
	s.setWriteDeadline(t)
	return nil
}

func (s *sysSocket) setReadDeadline(t time.Time) error {
	s.readDeadline = t
	return nil
}

func (s *sysSocket) setWriteDeadline(t time.Time) error {
	s.writeDeadline = t
	return nil
}

func (s *sysSocket) isReadableState() (bool, error) {
	return waitIOEvent(waitRead, s.fd, nowait)
}

func (s *sysSocket) waitUntilReadable() (bool, error) {
	to, err := deadlineToTimeval(s.readDeadline)
	if err != nil {
		return false, err
	}
	return waitIOEvent(waitRead, s.fd, to)
}

func (s *sysSocket) waitUntilWritable() (bool, error) {
	to, err := deadlineToTimeval(s.writeDeadline)
	if err != nil {
		return false, err
	}
	return waitIOEvent(waitWrite, s.fd, to)
}

func deadlineToTimeval(t time.Time) (*unix.Timeval, error) {
	if t.IsZero() {
		return waitForever, nil
	}

	now := time.Now()
	if t.Before(now) {
		return nil, ErrTimeout
	}

	subt := t.Sub(now)
	tv := unix.Timeval{}
	tv.Sec = int64(subt.Seconds())
	tv.Usec = int64(subt/1000) - tv.Sec*1000*1000
	return &tv, nil
}
