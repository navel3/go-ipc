package ipc

import (
	"bytes"
	"encoding/gob"
	"net"
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/navel3/go-ipc/internal/winsys"
)

func BenchmarkSocketDataSerializeWithMethod(b *testing.B) {
	sd := newTestingSocketData()
	var buf bytes.Buffer
	for i := 0; i < b.N; i++ {
		buf.Reset()
		err := sd.serialize(&buf)
		if err != nil {
			b.Fatal(err)
		}

		err = sd.deserialize(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSocketDataSerializeWithGob(b *testing.B) {
	sd := newTestingSocketData()
	var buf bytes.Buffer
	for i := 0; i < b.N; i++ {
		buf.Reset()
		enc := gob.NewEncoder(&buf)
		err := enc.Encode(sd)
		if err != nil {
			b.Fatal(err)
		}

		dec := gob.NewDecoder(&buf)
		err = dec.Decode(sd)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestSocketDataSerialize(t *testing.T) {
	var tests = []struct {
		name   string
		IP     net.IP
		Zone   string
		peeked []byte
	}{
		{"empty", []byte{}, "", []byte{}},
		{"one", []byte{1}, "a", []byte{1}},
		{"multi", []byte{1, 2, 3}, "abc", []byte{2, 3, 4}},
	}
	for _, tt := range tests {
		sd := newTestingSocketData()
		sd.laddr.IP = tt.IP
		sd.laddr.Zone = tt.Zone
		sd.raddr.IP = tt.IP
		sd.raddr.Zone = tt.Zone
		sd.peeked = tt.peeked
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := sd.serialize(&buf)
			if err != nil {
				t.Errorf("serialize error: %s", err)
				return
			}
			var got socketData
			err = got.deserialize(&buf)
			if err != nil {
				t.Errorf("deserialize error: %s", err)
				return
			}
			if !reflect.DeepEqual(got.laddr.IP, sd.laddr.IP) {
				t.Errorf("error %s", pretty.Compare(got, *sd))
			}
		})
	}
}

func newTestingSocketData() *socketData {
	return &socketData{
		ProtocolInfo: winsys.WSAPROTOCOL_INFO{
			ServiceFlags1: 1,
			ServiceFlags2: 1,
			ServiceFlags3: 1,
			ServiceFlags4: 1,
			ProviderFlags: 1,
			ProviderID: winsys.GUID{
				Data1: 1,
				Data2: 1,
				Data3: 1,
				Data4: [8]byte{
					1,
					1,
					1,
					1,
					1,
					1,
					1,
					1,
				},
			},
			CatalogEntryID: 1,
			ProtocolChain: winsys.WSAPROTOCOLCHAIN{
				ChainLen: 1,
				ChainEntries: [7]uint32{
					1,
					1,
					1,
					1,
					1,
					1,
					1,
				},
			},
			Version:           1,
			AddressFamily:     1,
			MaxSockAddr:       1,
			MinSockAddr:       1,
			SocketType:        1,
			Protocol:          1,
			ProtocolMaxOffset: 1,
			NetworkByteOrder:  1,
			SecurityScheme:    1,
			MessageSize:       1,
			ProviderReserved:  1,
			Protocols: [256]uint16{
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
				5,
			},
		},
		laddr: net.TCPAddr{
			IP:   []byte{1, 2, 3, 4},
			Port: 1,
			Zone: "abc",
		},
		raddr: net.TCPAddr{
			IP:   []byte{2, 3, 4, 5},
			Port: 1,
			Zone: "cde",
		},
		peeked:   []byte{3, 4, 5, 6},
		withData: true,
	}
}
