package ipc

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
	"golang.org/x/sync/errgroup"
)

func TestListenAndDial(t *testing.T) {
	t.Run("Listen unused pipe", func(t *testing.T) {
		pipename := "a"
		l, err := Listen(pipename)
		if err != nil {
			t.Fatalf("Failed to listen: %v", err)
		}
		if l == nil {
			t.Fatal("Listen returned nil")
		}
		defer l.Close()

		acceptAndDial := func(info string) (*Conn, *Conn) {
			t.Log(info)
			ch := make(chan *Conn)
			go func() {
				conn, err := l.Accept()
				if err != nil {
					t.Fatalf("Failed to accept: %v", err)
				}
				if conn == nil {
					t.Fatal("Accept returned nil")
				}
				ch <- conn
			}()

			conn, err := Dial(pipename)
			if err != nil {
				t.Fatalf("Failed to dial: %v", err)
			}
			if conn == nil {
				t.Fatal("Dial returned nil")
			}
			return <-ch, conn
		}

		// establish 2 connections
		c1a, c1d := acceptAndDial("one")
		defer c1a.Close()
		defer c1d.Close()
		c2a, c2d := acceptAndDial("two")
		defer c2a.Close()
		defer c2d.Close()

	})

	t.Run("Listen pipe already used", func(t *testing.T) {
		l1, err := Listen("dup")
		if err != nil {
			t.Fatalf("Failed to listen: %v", err)
		}
		if l1 == nil {
			t.Fatal("Listen returned nil")
		}
		defer l1.Close()

		l2, err := Listen("dup")
		if err == nil {
			t.Errorf("Expected error but returned nil")
		}
		if l2 != nil {
			t.Errorf("Expected nil but returned object: %v", l2)
		}
	})

	t.Run("Dial to invalid pipe", func(t *testing.T) {
		conn, err := Dial("none")
		if err == nil {
			t.Errorf("Expected error but returned nil")
		}
		if conn != nil {
			t.Errorf("Expected nil but returned object: %v", conn)
		}
	})
}

func TestSendTCP(t *testing.T) {
	const pipename = "a"
	const address = ":1234"
	const headerLen = 2 // N:
	const body = "this-is-body"
	const wholeLen = headerLen + len(body)

	var eg errgroup.Group

	// tcp client
	tcpClientFn := func(header string) error {
		conn, err := net.Dial("tcp", address)
		if err != nil {
			return newErr(err)
		}
		defer conn.Close()

		sendData := []byte(header + body)
		if _, err := conn.Write(sendData); err != nil {
			return newErr(err)
		}

		recvData := make([]byte, wholeLen)
		n, err := conn.Read(recvData)
		if n != wholeLen {
			return newErr(err)
		}
		if err != nil {
			return newErr(err)
		}

		if !reflect.DeepEqual(recvData, sendData) {
			return newErr(fmt.Errorf("echo backed data differes: got: %v but want: %v", recvData, sendData))
		}

		return nil
	}

	// ipc client
	ipcClientFn := func() error {
		conn, err := Dial(pipename)
		if err != nil {
			return newErr(err)
		}
		defer conn.Close()

		// receive message from server
		cmd, err := conn.ReceiveCommand()
		if err != nil {
			return newErr(fmt.Errorf("ReceiveCommand error: %v", err))
		}
		if got, want := cmd, TCPConnCommand; got != want {
			return newErr(fmt.Errorf("got command %v, but want %v", got, want))
		}

		tcpConn, withData, err := conn.ReceiveTCPConn()
		if err != nil {
			return newErr(err)
		}
		defer tcpConn.Close()

		if got, want := withData, true; got != want {
			return newErr(fmt.Errorf("got withData=%v but want %v", got, want))
		}

		data, err := conn.ReceiveData()
		if err != nil {
			newErr(fmt.Errorf("ReceiveData error: %v", err))
		}

		// verify data
		if got, want := string(data), "message"; got != want {
			return newErr(fmt.Errorf("message err: got %v but want %v", got, want))
		}

		// read data
		buf := make([]byte, headerLen+len(body))
		n, err := tcpConn.Read(buf)
		if err != nil {
			return newErr(err)
		}
		if n != wholeLen {
			return newErr(fmt.Errorf("data len error: got %v but want %v", n, wholeLen))
		}

		// echo back
		_, err = tcpConn.Write(buf)
		if err != nil {
			return newErr(err)
		}
		return nil
	}

	// ipc server
	eg.Go(func() error {
		l, err := Listen(pipename)
		if err != nil {
			return newErr(err)
		}
		defer l.Close()

		// listen & accept tcp
		tcpl, err := net.Listen("tcp", address)
		if err != nil {
			return newErr(err)
		}
		defer tcpl.Close()

		// two clients
		eg.Go(ipcClientFn)
		eg.Go(ipcClientFn)

		// two tcp clients
		eg.Go(func() error { return tcpClientFn("1:") })
		eg.Go(func() error { return tcpClientFn("2:") })

		for i := 0; i < 2; i++ {
			// accept ipc client
			err = func() error {
				conn, err := l.Accept()
				if err != nil {
					return newErr(err)
				}
				defer conn.Close()

				tcpConn, err := tcpl.Accept()
				if err != nil {
					return newErr(err)
				}
				defer tcpConn.Close()

				// read header
				leadBuf := make([]byte, headerLen)
				if _, err := tcpConn.Read(leadBuf); err != nil {
					return newErr(err)
				}

				if err := conn.SendTCPConn(tcpConn.(*net.TCPConn), leadBuf, []byte("message")); err != nil {
					return newErr(err)
				}
				return nil
			}()
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		t.Errorf("error: %v", err)
	}
}

func TestSendData(t *testing.T) {
	type Syn struct{}
	syn := make(chan Syn)

	pipename := "a"
	l, err := Listen(pipename)
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer l.Close()

	testData := []byte("test data")
	// client
	go func() {
		conn, err := Dial(pipename)
		if err != nil {
			t.Fatalf("Failed to dial: %v", err)
		}
		defer conn.Close()

		err = conn.SendData(testData)
		if err != nil {
			t.Fatalf("Failed to send data: %v", err)
		}

		syn <- struct{}{}
	}()
	defer func() { <-syn }()

	conn, err := l.Accept()
	if err != nil {
		t.Fatalf("Failed to accept: %v", err)
	}
	defer conn.Close()

	cmd, err := conn.ReceiveCommand()
	if err != nil {
		t.Fatalf("ReceiveCommand error: %v", err)
	}
	if got, want := cmd, DataCommand; got != want {
		t.Fatalf("got command %v, but want %v", got, want)
	}

	data, err := conn.ReceiveData()
	if err != nil {
		t.Fatalf("ReadData error: %v", err)
	}

	if !reflect.DeepEqual(data, testData) {
		t.Errorf("read data differs: %s", pretty.Compare(data, testData))
	}
}

func TestSendFile(t *testing.T) {
	type Syn struct{}
	syn := make(chan Syn)

	pipename := "a"
	l, err := Listen(pipename)
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer l.Close()

	const contentLen = 10
	const content = "1234567890"
	var filename string

	sentData := []byte{1, 2, 3, 4}

	// client
	go func() {
		conn, err := Dial(pipename)
		if err != nil {
			t.Fatalf("Failed to dial: %v", err)
		}
		defer conn.Close()

		f, err := ioutil.TempFile("", "ipc-test")
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}
		filename = f.Name()

		_, err = f.Write([]byte(content))
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		err = conn.SendFile(f, sentData)
		if err != nil {
			t.Fatalf("Failed to send data: %v", err)
		}

		syn <- struct{}{}
	}()
	defer func() { <-syn }()

	conn, err := l.Accept()
	if err != nil {
		t.Fatalf("Failed to accept: %v", err)
	}
	defer conn.Close()

	cmd, err := conn.ReceiveCommand()
	if err != nil {
		t.Fatalf("ReceiveCommand error: %v", err)
	}
	if got, want := cmd, FileCommand; got != want {
		t.Fatalf("got command %v, but want %v", got, want)
	}

	f, withData, err := conn.ReceiveFile()
	if err != nil {
		t.Fatalf("ReceiveFile error: %v", err)
	}
	if !withData {
		t.Fatalf("ReceivedFile should returns withData=true but was false")
	}

	receivedData, err := conn.ReceiveData()
	if err != nil {
		newErr(fmt.Errorf("ReceiveData error: %v", err))
	}

	if !reflect.DeepEqual(receivedData, sentData) {
		t.Fatalf("data differs: %s", pretty.Compare(receivedData, sentData))
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		t.Fatalf("seek error: %v", err)
	}
	defer f.Close()

	var gotContent [contentLen]byte
	_, err = f.Read(gotContent[:])
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if f.Name() != filename {
		t.Errorf("file name differs: got `%s`, but want `%s`", f.Name(), filename)

	}

	if string(gotContent[:]) != content {
		t.Errorf("file content differs: got `%s`, but want `%s`", string(gotContent[:]), content)
	}
}

func TestTCPConn(t *testing.T) {
	setup := func() (src net.Conn, sink net.Conn, err error) {
		const pipename = "p"
		const address = ":1234"

		eg := errgroup.Group{}

		var phase = []chan struct{}{
			make(chan struct{}),
			make(chan struct{}),
		}
		defer func() {
			for _, c := range phase {
				close(c)
			}
		}()

		// tcp client
		eg.Go(func() error {
			<-phase[1]
			var err error
			src, err = net.Dial("tcp", address)
			if err != nil {
				return err
			}
			return nil
		})

		// ipc server
		eg.Go(func() error {
			ipcl, err := Listen(pipename)
			if err != nil {
				return newErr(err)
			}
			defer ipcl.Close()

			phase[0] <- struct{}{}

			// listen & accept tcp
			tcpl, err := net.Listen("tcp", address)
			if err != nil {
				return newErr(err)
			}
			defer tcpl.Close()
			phase[1] <- struct{}{}

			conn, err := ipcl.Accept()
			if err != nil {
				return newErr(err)
			}
			defer conn.Close()

			tcpConn, err := tcpl.Accept()
			if err != nil {
				return err
			}

			err = conn.SendTCPConn(tcpConn.(*net.TCPConn), nil, nil)
			if err != nil {
				return err
			}
			return nil
		})

		// ipc client
		eg.Go(func() error {
			<-phase[0]

			conn, err := Dial(pipename)
			if err != nil {
				return newErr(err)
			}
			defer conn.Close()

			_, err = conn.ReceiveCommand()
			if err != nil {
				return newErr(err)
			}
			sink, _, err = conn.ReceiveTCPConn()
			if err != nil {
				return newErr(err)
			}
			return nil
		})

		if err = eg.Wait(); err != nil {
			return
		}

		if src == nil || sink == nil {
			t.Fatalf("src(%v) or sink(%v) is nil", src, sink)
		}

		return src, sink, nil
	}

	teardown := func(src, sink net.Conn) {
		src.Close()
		sink.Close()
	}

	// exercise
	t.Run("Read() block until timeout when the socket buffer is empty", func(t *testing.T) {
		src, sink, err := setup()
		if err != nil {
			t.Fatal(err)
		}
		defer teardown(src, sink)

		if err = sink.SetReadDeadline(time.Now().Add(10 * time.Millisecond)); err != nil {
			t.Fatal(err)
		}

		var buf [1]byte
		st := time.Now()
		n, err := sink.Read(buf[:])
		elapsed := time.Since(st)
		if elapsed.Milliseconds() < 10 {
			t.Errorf("returned before deadline: elapsed %v", elapsed)
		}
		if n != 0 {
			t.Errorf("got n=%v but want 0", n)
		}
		if got, want := err, ErrTimeout; got != want {
			t.Errorf("got error `%v` but want `%v`", got, want)
		}
	})

	t.Run("Write() block until timeout when the socket buffer is full", func(t *testing.T) {
		src, sink, err := setup()
		if err != nil {
			t.Fatal(err)
		}
		defer teardown(src, sink)

		sndBufSize, err := minimizeSocketBuffer(SocketType(sink.(*tcpConn).sysSocket.fd), SoSndbuf)
		if err != nil {
			t.Fatal(err)
		}

		var rcvBufSize int
		rawSrc, _ := src.(*net.TCPConn).SyscallConn()
		rawSrc.Control(func(fd uintptr) {
			rcvBufSize, err = minimizeSocketBuffer(SocketType(fd), SoRcvbuf)
		})
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("Bufsize: snd:%d, rcv:%d", sndBufSize, rcvBufSize)

		setDeadline := func() {
			if err = sink.SetWriteDeadline(time.Now().Add(10 * time.Millisecond)); err != nil {
				t.Fatal(err)
			}
		}
		setDeadline()

		// write until window full
		var buf [65534]byte
		st := time.Now()
		var n int
		for {
			n, err = sink.Write(buf[:])
			if n == 0 {
				break
			}
			setDeadline()
		}
		elapsed := time.Since(st)
		if elapsed.Milliseconds() < 10 {
			t.Errorf("returned before deadline: elapsed %v", elapsed)
		}
		if got, want := err, ErrTimeout; got != want {
			t.Errorf("got error `%v` but want `%v`", got, want)
		}
	})
}

func BenchmarkTCPDirect(b *testing.B) {
	tcpl, err := net.Listen("tcp", ":1234")
	if err != nil {
		b.Fatal(err)
	}
	defer tcpl.Close()

	// TCP Server
	go func() {
		r := bufio.NewReader(nil)
		w := bufio.NewWriter(nil)
		for {
			func() {
				conn, err := tcpl.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				r.Reset(conn)
				w.Reset(conn)

				// Read HTTP request header
				for {
					line, _, err := r.ReadLine()
					if err != nil {
						b.Fatal(err)
					}
					if len(line) == 0 {
						break
					}
				}
				// Write HTTP response header
				w.WriteString("HTTP/1.1 200 OK\r\n")
				w.Flush()
				conn.Close()
			}()
		}
	}()

	raddr, _ := net.ResolveTCPAddr("tcp", ":1234")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := func() error {
			conn, err := net.DialTCP("tcp", nil, raddr)
			if err != nil {
				return newErr(err)
			}
			defer conn.Close()

			_, err = conn.Write([]byte("GET / HTTP/1/1\r\nHOST: a.com\r\n\r\n"))
			if err != nil {
				return newErr(err)
			}
			r := bufio.NewReader(conn)
			_, _, err = r.ReadLine()
			if err != nil {
				return newErr(err)
			}
			return nil
		}()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTCPIPC(b *testing.B) {
	const pipename = "p"

	eg := errgroup.Group{}

	ipcl, e := Listen(pipename)
	if e != nil {
		b.Fatal(e)
	}
	defer ipcl.Close()

	queuel := NewQueueListener(1)
	defer queuel.Close()

	tcpl, e := net.Listen("tcp", ":1234")
	if e != nil {
		b.Fatal(e)
	}
	defer tcpl.Close()

	// IPC Server
	eg.Go(func() error {
		ipcconn, err := ipcl.Accept()
		if err != nil {
			return newErr(err)
		}
		defer ipcconn.Close()

		for {
			err = func() error {
				conn, err := tcpl.Accept()
				if err != nil {
					return newErr(err)
				}
				defer conn.Close()

				// Read some bytes
				var buf [10]byte
				_, err = conn.Read(buf[:])
				if err != nil {
					return newErr(err)
				}
				return ipcconn.SendTCPConn(conn.(*net.TCPConn), buf[:], nil)
			}()
			if err != nil {
				break
			}
		}
		return err
	})

	// IPC Client
	eg.Go(func() error {
		conn, err := Dial(pipename)
		if err != nil {
			return err
		}
		defer conn.Close()

		for {
			_, err = conn.ReceiveCommand() // cmd is TCPConn
			if err != nil {
				err = newErr(err)
				break
			}

			tcpconn, _, err := conn.ReceiveTCPConn()
			if err != nil {
				err = newErr(err)
				break
			}
			err = queuel.Push(nil, tcpconn)
			if err != nil {
				err = newErr(err)
				break
			}
		}
		return err
	})

	// TCP Client
	eg.Go(func() error {
		r := bufio.NewReader(nil)
		w := bufio.NewWriter(nil)
		var err error
		for {
			err = func() error {
				conn, err := queuel.Accept()
				if err != nil {
					return newErr(err)
				}
				defer conn.Close()
				r.Reset(conn)
				w.Reset(conn)

				// Read HTTP request header
				for {
					line, _, err := r.ReadLine()
					if err != nil {
						return newErr(err)
					}
					if len(line) == 0 {
						break
					}
				}
				// Write HTTP response header
				w.WriteString("HTTP/1.1 200 OK\r\n")
				w.Flush()
				return nil
			}()
			if err != nil {
				break
			}
		}
		return err
	})

	raddr, _ := net.ResolveTCPAddr("tcp", ":1234")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := func() error {
			conn, err := net.DialTCP("tcp", nil, raddr)
			if err != nil {
				return newErr(err)
			}
			defer conn.Close()

			_, err = conn.Write([]byte("GET / HTTP/1/1\r\nHOST: a.com\r\n\r\n"))
			if err != nil {
				return newErr(err)
			}

			r := bufio.NewReader(conn)
			_, _, err = r.ReadLine()
			if err != nil {
				return newErr(err)
			}
			return nil
		}()
		if err != nil {
			b.Fatal(e)
		}
	}

	tcpl.Close()
	queuel.Close()
	ipcl.Close()
	eg.Wait()
}

func newErr(err error) error {
	_, _, line, _ := runtime.Caller(1)
	return fmt.Errorf("(line:%v), %v", line, err)
}
