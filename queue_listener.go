package ipc

import (
	"context"
	"net"
	"sync"
)

// QueueListener is a Listener accept from queue; it implements net.Listener
// interface.
// TCPConn received from the IPC peer should be pushed to this.
type QueueListener struct {
	ch       chan net.Conn
	m        sync.Mutex // guard ch
	isClosed bool
	addr     queueAddr
}

// Push pushes a net.Conn to the queue.
//
// The queue has capacity specified at the creation; Push blocks the call if
// there is no space.
func (l *QueueListener) Push(ctx context.Context, c net.Conn) error {
	l.m.Lock()
	defer l.m.Unlock()

	if l.isClosed {
		return ErrClosedQueue
	}

	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case l.ch <- c:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Accept implements the Accept method in the net.Listener interface; it waits
// for the next push to the queue.
//
// If the listener closed while waiting, Accept returns ErrClosedQueue.
func (l *QueueListener) Accept() (net.Conn, error) {
	if l.isClosed {
		return nil, ErrClosedQueue
	}

	tcp, ok := <-l.ch
	if !ok {
		return nil, ErrClosedQueue
	}
	return tcp, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *QueueListener) Close() error {
	l.m.Lock()
	defer l.m.Unlock()
	if l.isClosed {
		return nil
	}
	l.isClosed = true

	if len(l.ch) > 0 {
		// vacuume channel and close connections
		for c := range l.ch {
			c.Close()
		}
	}

	close(l.ch)
	return nil
}

// Addr returns the listener's network address.
func (l *QueueListener) Addr() net.Addr {
	return &l.addr
}

// NewQueueListener creates and initialize a new QueueListener.
//
// The backlog is the capacity of the queue like listen(2). When the queue is
// filled with connections, Push is blocked until a connection is dequeued with
// Accept.
func NewQueueListener(backlog uint) *QueueListener {
	return &QueueListener{
		ch:       make(chan net.Conn, backlog),
		isClosed: false,
		addr:     queueAddr{},
	}
}

type queueAddr struct {
}

func (i *queueAddr) Network() string {
	return "queue"
}

func (i *queueAddr) String() string {
	return "queue"
}
