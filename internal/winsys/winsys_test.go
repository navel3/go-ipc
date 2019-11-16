package winsys

import (
	"os"
	"syscall"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWSASocket(t *testing.T) {
	t.Run("test good call", func(t *testing.T) {
		sock, err := WSASocket(windows.AF_INET, windows.SOCK_STREAM, windows.IPPROTO_TCP, nil, 0, 0)
		if sock == ^windows.Handle(0) || err != nil {
			t.Errorf("Failed to call API. retval=%v, err=[%v]", sock, err)
		}
	})

	t.Run("should return system error on failure", func(t *testing.T) {
		// exercise
		const badArg int32 = 9999
		sock, err := WSASocket(badArg, badArg, badArg, nil, 0, 0)

		// verify
		if got, want := sock, ^windows.Handle(0); got != want {
			t.Errorf("Bad return value: got=%v, want=[%v]", got, want)
		}
		if got, want := err, windows.WSAESOCKTNOSUPPORT; got != want {
			t.Errorf("Bad return value: got=%v, want=[%v]", got, want)
		}
	})
}

func TestDuplicateSocket(t *testing.T) {
	t.Run("test good call", func(t *testing.T) {
		// setup
		sock, err := WSASocket(windows.AF_INET, windows.SOCK_STREAM, windows.IPPROTO_TCP, nil, 0, 0)
		if sock == ^windows.Handle(0) || err != nil {
			t.Errorf("Failed to call API. retval=%v, err=[%v]", sock, err)
		}

		// exercise
		var info WSAPROTOCOL_INFO
		ret := WSADuplicateSocket(sock, uint32(os.Getpid()), &info)

		// verify
		if got, want := ret, error(nil); got != want {
			t.Errorf("got = %v, want = %v", got, want)
		}

	})

	t.Run("should return system error on failure", func(t *testing.T) {
		// setup
		var info WSAPROTOCOL_INFO
		badSock := windows.Handle(1234)
		var want syscall.Errno = 10038 // WSAENOTSOCK

		// exercise
		if got := WSADuplicateSocket(badSock, uint32(os.Getpid()), &info); got != want {
			t.Errorf("got = %v, want = %v", got, want)
		}
	})

}
