package winsys

import "golang.org/x/sys/windows"

//go:generate go run golang.org/x/sys/windows/mkwinsyscall -output zwinsys.go winsys.go
//
//sys WSADuplicateSocket(socket windows.Handle, processID uint32, protocolInfo *WSAPROTOCOL_INFO) (err error) [failretval==0xffffffff] = ws2_32.WSADuplicateSocketW
//sys WSASocket(af int32, tp int32, protocol int32, protocolInfo *WSAPROTOCOL_INFO, g GROUP, flags uint32) (socket windows.Handle, err error) [failretval==invalid_socket] = ws2_32.WSASocketW
//sys WSACreateEvent() (event windows.Handle, err error) [failretval==0] = ws2_32.WSACreateEvent
//sys WSACloseEvent(event windows.Handle) (err error) [failretval==0]= ws2_32.WSACloseEvent
//sys WSAWaitForMultipleEvents(cEvents uint32, events *windows.Handle, waitAll bool, timeout uint32, alerttable bool) (result int32, err error) [failretval==-1] = ws2_32.WSAWaitForMultipleEvents
//sys WSAEventSelect(fd windows.Handle, event windows.Handle, networkevents int32) (err error) [failretval==socket_error] = ws2_32.WSAEventSelect

const (
	wsaprotocol_len    = 255
	max_protocol_chain = 7
	invalid_socket     = ^windows.Handle(0)
	socket_error       = uintptr(^uint32(0))
	FROM_PROTOCOL_INFO = -1
)

const FD_READ = 1 << 0
const FD_WRITE = 1 << 1
const FD_CLOSE = 1 << 5
const WSA_WAIT_EVENT_0 = 0
const WSA_WAIT_TIMEOUT = 0x102

type GROUP uint32

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type WSAPROTOCOLCHAIN struct {
	ChainLen     int32
	ChainEntries [max_protocol_chain]uint32
}

type WSAPROTOCOL_INFO struct {
	ServiceFlags1     uint32
	ServiceFlags2     uint32
	ServiceFlags3     uint32
	ServiceFlags4     uint32
	ProviderFlags     uint32
	ProviderID        GUID
	CatalogEntryID    uint32
	ProtocolChain     WSAPROTOCOLCHAIN
	Version           int32
	AddressFamily     int32
	MaxSockAddr       int32
	MinSockAddr       int32
	SocketType        int32
	Protocol          int32
	ProtocolMaxOffset int32
	NetworkByteOrder  int32
	SecurityScheme    int32
	MessageSize       uint32
	ProviderReserved  uint32
	Protocols         [wsaprotocol_len + 1]uint16
}
