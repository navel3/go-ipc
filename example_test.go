package ipc_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/navel3/go-ipc"
)

func ExampleQueueListener() {
	ql := ipc.NewQueueListener(1)
	defer ql.Close()

	go func() {
		for {
			func() {
				conn, err := ql.Accept()
				if err != nil {
					return
				}
				defer conn.Close()

				// read data from conn
			}()
		}
	}()

	conn, err := ipc.Dial("pipename")
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		cmd, err := conn.ReceiveCommand()
		if err != nil {
			return
		}

		if cmd != ipc.TCPConnCommand {
			return
		}

		func() {
			tcpc, _, err := conn.ReceiveTCPConn()
			if err != nil {
				return
			}
			defer tcpc.Close()

			err = ql.Push(context.Background(), tcpc)
			if err != nil {
				return
			}
		}()
	}
}

func Example_server() {
	l, err := ipc.Listen("pipename")
	if err != nil {
		return
	}
	defer l.Close()

	conn, err := l.Accept()
	if err != nil {
		return
	}

	// send data
	err = conn.SendData([]byte{1, 2, 3})
	if err != nil {
		return
	}

	// send file
	f, err := ioutil.TempFile("", "example")
	err = conn.SendFile(f, []byte("message"))
	if err != nil {
		return
	}

	// send tcp conn
	netl, err := net.Listen("tcp", ":1234")
	if err != nil {
		return
	}
	defer netl.Close()

	netc, err := netl.Accept()
	if err != nil {
		return
	}
	defer netc.Close()

	var peeked [10]byte
	_, err = netc.Read(peeked[:])
	if err != nil {
		return
	}
	err = conn.SendTCPConn(netc.(*net.TCPConn), peeked[:], []byte("message"))
	if err != nil {
		return
	}
}

func Example_client() {
	conn, err := ipc.Dial("pipename")
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		cmd, err := conn.ReceiveCommand()
		if err != nil {
			return
		}

		switch cmd {
		case ipc.DataCommand:
			d, err := conn.ReceiveData()
			if err != nil {
				return
			}
			fmt.Println(d) // 1, 2, 3, 4
		case ipc.FileCommand:
			f, withData, err := conn.ReceiveFile()
			if err != nil {
				return
			}
			fmt.Println(f.Name(), withData) // /tempdir/example~, true
			f.Close()
			if withData {
				msg, err := conn.ReceiveData()
				if err != nil {
					return
				}
				fmt.Println(string(msg)) // "message"
			}
		case ipc.TCPConnCommand:
			tcpc, withData, err := conn.ReceiveTCPConn()
			if err != nil {
				return
			}
			var buf [20]byte
			tcpc.Read(buf[:])
			fmt.Println(buf, withData) // [peeked]+, true
			tcpc.Close()
			if withData {
				msg, err := conn.ReceiveData()
				if err != nil {
					return
				}
				fmt.Println(string(msg)) // "message"
			}
		}
	}
}
