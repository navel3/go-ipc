package ipc

import (
	"bytes"
	"encoding/binary"
	"io"
)

type bytesWriter struct {
	w   io.Writer
	err error
}

func (w *bytesWriter) write(v interface{}) {
	if w.err == nil {
		w.err = binary.Write(w.w, binary.BigEndian, v)
	}
}

func (w *bytesWriter) writeBytes(b []byte) {
	if w.err != nil {
		return
	}
	w.err = binary.Write(w.w, binary.BigEndian, uint32(len(b)))
	if w.err != nil {
		return
	}
	w.err = binary.Write(w.w, binary.BigEndian, b)
}

type bytesReader struct {
	r   io.Reader
	err error
}

func (r *bytesReader) read(v interface{}) {
	if r.err == nil {
		r.err = binary.Read(r.r, binary.BigEndian, v)
	}
}

func (r *bytesReader) readBytes() []byte {
	if r.err != nil {
		return nil
	}

	var l uint32
	r.err = binary.Read(r.r, binary.BigEndian, &l)
	if r.err != nil {
		return nil
	}
	b := make([]byte, l)
	r.err = binary.Read(r.r, binary.BigEndian, &b)
	return b
}

func fitLength(b []byte, length int) []byte {
	if len(b) == length {
		return b
	}

	if cap(b) < length {
		return make([]byte, length)
	}
	return b[:length]
}

func writeAll(w io.Writer, b []byte) (err error) {
	siz := 0
	for {
		var n int
		n, err = w.Write(b[siz:])
		if err != nil {
			return
		}
		siz += n
		if siz == len(b) {
			return
		}
	}
}

func readAll(r io.Reader, b []byte) error {
	siz := 0
	for {
		n, err := r.Read(b[siz:])
		if err != nil {
			return err
		}
		siz += n
		if siz == len(b) {
			break
		}
	}

	return nil
}

func writeWithLength(b *bytes.Buffer, body func(b *bytes.Buffer) error) (err error) {
	// length place holder
	var zero4 [4]byte
	_, err = b.Write(zero4[:])
	if err != nil {
		return
	}

	err = body(b)
	if err != nil {
		return
	}

	// write length
	binary.BigEndian.PutUint32(b.Bytes(), uint32(b.Len()-4))
	return nil
}
