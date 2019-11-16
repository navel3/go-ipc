package ipc

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
)

type serializer interface {
	serialize(w io.Writer) error
}

type deserializer interface {
	deserialize(r io.Reader) error
}

type gateway struct {
	pid  uint32
	wbuf bytes.Buffer
	rbuf []byte
}

func (gw *gateway) sendImpl(conn net.Conn, msg []byte, makeSerializer func() (serializer, error)) (err error) {
	s, err := makeSerializer()
	if err != nil {
		return
	}

	gw.wbuf.Reset()
	writeWithLength(&gw.wbuf, func(b *bytes.Buffer) error {
		return s.serialize(b)
	})

	err = writeAll(conn, gw.wbuf.Bytes())
	if err != nil {
		return
	}

	if len(msg) > 0 {
		err = writeData(msg, conn)
		if err != nil {
			return
		}
	}
	return
}

func (gw *gateway) receiveImpl(conn net.Conn, s deserializer) (err error) {
	var dlen uint32
	err = binary.Read(conn, binary.BigEndian, &dlen)
	if err != nil {
		return
	}

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

	return nil
}
