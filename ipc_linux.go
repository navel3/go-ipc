package ipc

import "net"

// Listen announces on the pipe name.
func Listen(name string) (*Listener, error) {
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: name, Net: "unix"})
	if err != nil {
		return nil, err
	}

	return &Listener{l: l}, nil
}

// Accept implements the Accept method in the net.Listener interface; it waits
// for the next call and return a Conn.
func (l *Listener) Accept() (*Conn, error) {
	conn, err := l.l.Accept()
	if err != nil {
		return nil, err
	}

	return newConn(conn), nil
}

// Dial connects to the named pipe.
func Dial(name string) (*Conn, error) {
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: name, Net: "unix"})
	if err != nil {
		return nil, err
	}

	return newConn(conn), nil
}
