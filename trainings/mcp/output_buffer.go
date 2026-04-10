package mcp

import (
	"sync"
)

const DefaultMaxOutputSize = 50 * 1024 * 1024 // 50MB

// OutputBuffer is a thread-safe, bounded buffer that captures execution output.
// When the buffer exceeds maxSize, oldest data is dropped.
// Only the last execution's output is stored (call Reset between runs).
type OutputBuffer struct {
	mu      sync.Mutex
	buf     []byte
	maxSize int
}

func NewOutputBuffer(maxSize int) *OutputBuffer {
	return &OutputBuffer{
		maxSize: maxSize,
	}
}

func (b *OutputBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.buf = append(b.buf, p...)

	// Drop oldest data if over limit
	if len(b.buf) > b.maxSize {
		excess := len(b.buf) - b.maxSize
		b.buf = b.buf[excess:]
	}

	return len(p), nil
}

func (b *OutputBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return string(b.buf)
}

func (b *OutputBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return len(b.buf)
}

func (b *OutputBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.buf = b.buf[:0]
}
