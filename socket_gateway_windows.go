package ipc

import (
	"io"
	"net"
	"time"
	"unsafe"

	"github.com/navel3/go-ipc/internal/winsys"
	"golang.org/x/sys/windows"
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
		sd := socketData{
			laddr:    *sock.LocalAddr().(*net.TCPAddr),
			raddr:    *sock.RemoteAddr().(*net.TCPAddr),
			peeked:   peeked,
			withData: len(msg) > 0,
		}

		err = gw.sendImpl(conn, msg, func() (s serializer, err error) {
			err = winsys.WSADuplicateSocket(windows.Handle(fd), uint32(gw.pid), &sd.ProtocolInfo)
			if err != nil {
				return
			}
			return &sd, nil
		})
	})
	if err == nil {
		sock.Close()
	}
	return
}

func (gw *socketGateway) receive(conn net.Conn) (sk *sysSocket, laddr net.TCPAddr, raddr net.TCPAddr, peeked []byte, withMsg bool, err error) {
	var sd socketData

	err = gw.receiveImpl(conn, &sd)
	if err != nil {
		return
	}

	fd, err := winsys.WSASocket(winsys.FROM_PROTOCOL_INFO,
		winsys.FROM_PROTOCOL_INFO,
		winsys.FROM_PROTOCOL_INFO,
		&sd.ProtocolInfo,
		0,
		0)
	if err != nil {
		return
	}

	// set non-blocking mode to enable deadline
	const finbio = uint32(0x8004667e)
	on := uint32(1)
	var retsize uint32
	err = windows.WSAIoctl(fd, finbio, (*byte)(unsafe.Pointer(&on)), 4, nil, 0, &retsize, nil, 0)
	if err != nil {
		return
	}

	return &sysSocket{fd: fd}, sd.laddr, sd.raddr, sd.peeked, sd.withData, nil
}

func newSocketGateway() *socketGateway {
	return &socketGateway{}
}

type socketData struct {
	ProtocolInfo winsys.WSAPROTOCOL_INFO
	laddr        net.TCPAddr
	raddr        net.TCPAddr
	peeked       []byte
	withData     bool
}

func (sd *socketData) serialize(w io.Writer) error {
	bw := &bytesWriter{w, nil}
	// ProtocolInfo
	pi := &sd.ProtocolInfo
	bw.write(pi.ServiceFlags1)
	bw.write(pi.ServiceFlags2)
	bw.write(pi.ServiceFlags3)
	bw.write(pi.ServiceFlags4)
	bw.write(pi.ProviderFlags)
	bw.write(pi.ProviderID.Data1)
	bw.write(pi.ProviderID.Data2)
	bw.write(pi.ProviderID.Data3)
	bw.write(pi.ProviderID.Data4)
	bw.write(pi.CatalogEntryID)
	bw.write(pi.ProtocolChain.ChainLen)
	bw.write(pi.ProtocolChain.ChainEntries)
	bw.write(pi.Version)
	bw.write(pi.AddressFamily)
	bw.write(pi.MaxSockAddr)
	bw.write(pi.MinSockAddr)
	bw.write(pi.SocketType)
	bw.write(pi.Protocol)
	bw.write(pi.ProtocolMaxOffset)
	bw.write(pi.NetworkByteOrder)
	bw.write(pi.SecurityScheme)
	bw.write(pi.MessageSize)
	bw.write(pi.ProviderReserved)
	bw.write(pi.Protocols)
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
	// ProtocolInfo
	pi := &sd.ProtocolInfo
	br.read(&pi.ServiceFlags1)
	br.read(&pi.ServiceFlags2)
	br.read(&pi.ServiceFlags3)
	br.read(&pi.ServiceFlags4)
	br.read(&pi.ProviderFlags)
	br.read(&pi.ProviderID.Data1)
	br.read(&pi.ProviderID.Data2)
	br.read(&pi.ProviderID.Data3)
	br.read(&pi.ProviderID.Data4)
	br.read(&pi.CatalogEntryID)
	br.read(&pi.ProtocolChain.ChainLen)
	br.read(&pi.ProtocolChain.ChainEntries)
	br.read(&pi.Version)
	br.read(&pi.AddressFamily)
	br.read(&pi.MaxSockAddr)
	br.read(&pi.MinSockAddr)
	br.read(&pi.SocketType)
	br.read(&pi.Protocol)
	br.read(&pi.ProtocolMaxOffset)
	br.read(&pi.NetworkByteOrder)
	br.read(&pi.SecurityScheme)
	br.read(&pi.MessageSize)
	br.read(&pi.ProviderReserved)
	br.read(&pi.Protocols)

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
	fd            windows.Handle
	readDeadline  time.Time
	writeDeadline time.Time
}

func (s *sysSocket) read(b []byte) (n int, err error) {
	var read uint32
	var flags uint32
	buf := windows.WSABuf{
		Len: uint32(len(b)),
		Buf: &b[0],
	}
	for {
		err = windows.WSARecv(s.fd, &buf, 1, &read, &flags, nil, nil)
		if err == nil || err != windows.WSAEWOULDBLOCK {
			break
		}
		// WSAEWOULDBLOCK
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
	n = int(read)

	if err != nil && n < 0 {
		n = 0
	}
	return
}

func (s *sysSocket) write(b []byte) (n int, err error) {
	buf := windows.WSABuf{
		Len: uint32(len(b)),
		Buf: &b[0],
	}
	var written uint32
	var flags uint32
	for {
		err = windows.WSASend(s.fd, &buf, 1, &written, flags, nil, nil)
		if err == nil || err != windows.WSAEWOULDBLOCK {
			break
		}
		// WSAEWOULDBLOCK
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
	n = int(written)
	if err != nil && n < 0 {
		n = 0
	}
	return
}

func (s *sysSocket) close() error {
	return windows.Closesocket(s.fd)
}

func (s *sysSocket) closeRead() error {
	err := windows.Shutdown(s.fd, windows.SHUT_RD)
	return err
}

func (s *sysSocket) closeWrite() error {
	err := windows.Shutdown(s.fd, windows.SHUT_WR)
	return err
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
	ev, err := winsys.WSACreateEvent()
	if err != nil {
		return false, err
	}
	defer winsys.WSACloseEvent(ev)

	if err := winsys.WSAEventSelect(s.fd, ev, winsys.FD_READ|winsys.FD_CLOSE); err != nil {
		return false, err
	}

	events := [1]windows.Handle{ev}

	ret, err := winsys.WSAWaitForMultipleEvents(1, &events[0], false, 0, false)
	if err != nil {
		return false, err
	}

	return ret == winsys.WSA_WAIT_EVENT_0, nil
}

func (s *sysSocket) waitUntilReadable() (bool, error) {
	to, err := deadlineToTimeout(s.readDeadline)
	if err != nil {
		return false, err
	}
	return waitIOEvent(waitRead, s.fd, to)
}

func (s *sysSocket) waitUntilWritable() (bool, error) {
	to, err := deadlineToTimeout(s.writeDeadline)
	if err != nil {
		return false, err
	}
	return waitIOEvent(waitWrite, s.fd, to)
}

const (
	waitRead = iota
	waitWrite
)

func waitIOEvent(mode int, fd windows.Handle, timeout uint32) (bool, error) {
	ev, err := winsys.WSACreateEvent()
	if err != nil {
		return false, err
	}
	defer winsys.WSACloseEvent(ev)

	networkevents := int32(winsys.FD_CLOSE)
	if mode == waitWrite {
		networkevents |= winsys.FD_WRITE
	} else {
		networkevents |= winsys.FD_READ
	}

	if err := winsys.WSAEventSelect(fd, ev, networkevents); err != nil {
		return false, err
	}

	events := [1]windows.Handle{ev}

	ret, err := winsys.WSAWaitForMultipleEvents(1, &events[0], false, timeout, false)
	if err != nil {
		return false, err
	}

	return ret == winsys.WSA_WAIT_EVENT_0, nil
}

func deadlineToTimeout(t time.Time) (uint32, error) {
	if t.IsZero() {
		return windows.INFINITE, nil
	}

	now := time.Now()
	if t.Before(now) {
		return 0, ErrTimeout
	}

	return uint32(t.Sub(now).Milliseconds()), nil
}
