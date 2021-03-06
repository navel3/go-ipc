// Code generated by 'go generate'; DO NOT EDIT.

package winsys

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var _ unsafe.Pointer

// Do the interface allocations only once for common
// Errno values.
const (
	errnoERROR_IO_PENDING = 997
)

var (
	errERROR_IO_PENDING error = syscall.Errno(errnoERROR_IO_PENDING)
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case errnoERROR_IO_PENDING:
		return errERROR_IO_PENDING
	}
	// TODO: add more here, after collecting data on the common
	// error values see on Windows. (perhaps when running
	// all.bat?)
	return e
}

var (
	modws2_32 = windows.NewLazySystemDLL("ws2_32.dll")

	procWSADuplicateSocketW      = modws2_32.NewProc("WSADuplicateSocketW")
	procWSASocketW               = modws2_32.NewProc("WSASocketW")
	procWSACreateEvent           = modws2_32.NewProc("WSACreateEvent")
	procWSACloseEvent            = modws2_32.NewProc("WSACloseEvent")
	procWSAWaitForMultipleEvents = modws2_32.NewProc("WSAWaitForMultipleEvents")
	procWSAEventSelect           = modws2_32.NewProc("WSAEventSelect")
)

func WSADuplicateSocket(socket windows.Handle, processID uint32, protocolInfo *WSAPROTOCOL_INFO) (err error) {
	r1, _, e1 := syscall.Syscall(procWSADuplicateSocketW.Addr(), 3, uintptr(socket), uintptr(processID), uintptr(unsafe.Pointer(protocolInfo)))
	if r1 == 0xffffffff {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func WSASocket(af int32, tp int32, protocol int32, protocolInfo *WSAPROTOCOL_INFO, g GROUP, flags uint32) (socket windows.Handle, err error) {
	r0, _, e1 := syscall.Syscall6(procWSASocketW.Addr(), 6, uintptr(af), uintptr(tp), uintptr(protocol), uintptr(unsafe.Pointer(protocolInfo)), uintptr(g), uintptr(flags))
	socket = windows.Handle(r0)
	if socket == invalid_socket {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func WSACreateEvent() (event windows.Handle, err error) {
	r0, _, e1 := syscall.Syscall(procWSACreateEvent.Addr(), 0, 0, 0, 0)
	event = windows.Handle(r0)
	if event == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func WSACloseEvent(event windows.Handle) (err error) {
	r1, _, e1 := syscall.Syscall(procWSACloseEvent.Addr(), 1, uintptr(event), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func WSAWaitForMultipleEvents(cEvents uint32, events *windows.Handle, waitAll bool, timeout uint32, alerttable bool) (result int32, err error) {
	var _p0 uint32
	if waitAll {
		_p0 = 1
	} else {
		_p0 = 0
	}
	var _p1 uint32
	if alerttable {
		_p1 = 1
	} else {
		_p1 = 0
	}
	r0, _, e1 := syscall.Syscall6(procWSAWaitForMultipleEvents.Addr(), 5, uintptr(cEvents), uintptr(unsafe.Pointer(events)), uintptr(_p0), uintptr(timeout), uintptr(_p1), 0)
	result = int32(r0)
	if result == -1 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func WSAEventSelect(fd windows.Handle, event windows.Handle, networkevents int32) (err error) {
	r1, _, e1 := syscall.Syscall(procWSAEventSelect.Addr(), 3, uintptr(fd), uintptr(event), uintptr(networkevents))
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
