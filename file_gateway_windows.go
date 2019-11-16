package ipc

import (
	"io"
	"net"
	"os"

	"golang.org/x/sys/windows"
)

type fileData struct {
	Handle   windows.Handle
	Name     string
	withData bool
}

func (f *fileData) serialize(w io.Writer) error {
	bw := &bytesWriter{w, nil}
	bw.write(uint64(f.Handle))
	bw.writeBytes([]byte(f.Name))
	bw.write(f.withData)
	return bw.err
}

func (f *fileData) deserialize(r io.Reader) error {
	var ui64 uint64
	br := &bytesReader{r, nil}
	br.read(&ui64)
	f.Handle = windows.Handle(ui64)
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
		err = gw.sendImpl(conn, msg, func() (s serializer, err error) {
			thisPHandle, err := windows.GetCurrentProcess()
			if err != nil {
				return
			}
			defer windows.CloseHandle(thisPHandle)

			targetPHandle, err := windows.OpenProcess(windows.PROCESS_DUP_HANDLE, false, gw.pid)
			if err != nil {
				return
			}
			defer windows.CloseHandle(targetPHandle)

			fdata := fileData{
				Name:     f.Name(),
				withData: len(msg) > 0,
			}

			err = windows.DuplicateHandle(
				thisPHandle,
				windows.Handle(fd),
				targetPHandle,
				&fdata.Handle,
				0,
				false,
				windows.DUPLICATE_SAME_ACCESS)
			if err != nil {
				return
			}
			return &fdata, nil
		})
	})
	if err == nil {
		f.Close()
	}
	return
}

func (gw *fileGateway) receive(conn net.Conn) (f *os.File, withData bool, err error) {
	var fdata fileData
	err = gw.receiveImpl(conn, &fdata)
	if err != nil {
		return
	}
	return os.NewFile(uintptr(fdata.Handle), fdata.Name), fdata.withData, nil
}

func newFileGateway() *fileGateway {
	return &fileGateway{}
}
