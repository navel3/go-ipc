package ipc

import (
	"io"
	"net"
	"os"
)

type fileData struct {
	Name     string
	withData bool
}

func (f *fileData) serialize(w io.Writer) error {
	bw := &bytesWriter{w, nil}
	bw.writeBytes([]byte(f.Name))
	bw.write(f.withData)
	return bw.err
}

func (f *fileData) deserialize(r io.Reader) error {
	br := &bytesReader{r, nil}
	if b := br.readBytes(); b != nil {
		f.Name = string(b)
	}
	br.read(&f.withData)
	return br.err
}

type fileGateway struct {
	gateway
}

func (gw *fileGateway) send(conn net.Conn, f *os.File, msg []byte) (err error) {
	rawFile, err := f.SyscallConn()
	if err != nil {
		return
	}

	rawFile.Control(func(fd uintptr) {
		err = gw.sendImpl(conn,
			int(fd),
			&fileData{
				Name:     f.Name(),
				withData: len(msg) > 0,
			},
			msg)
	})

	if err != nil {
		f.Close()
	}
	return
}

func (gw *fileGateway) receive(conn net.Conn) (f *os.File, withData bool, err error) {
	var fdata fileData
	fd, err := gw.receiveImpl(conn, &fdata)
	if err != nil {
		return
	}

	return os.NewFile(uintptr(fd), fdata.Name), fdata.withData, nil
}

func newFileGateway() *fileGateway {
	return &fileGateway{}
}
