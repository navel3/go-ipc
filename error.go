package ipc

import (
	"errors"
)

var (
	// ErrClosedQueue is returned from queue operation on QueueListener
	// that have been closed.
	ErrClosedQueue = errors.New("closed queue")

	// ErrTimeout is returned when read or write operation is not completed
	// before the deadline.
	ErrTimeout = errors.New("timeout")
)
