package common

import "sync"

type LogByteBuffer struct {
	data []byte
	size int
	lock sync.Mutex
}

func NewLogByteBuffer(cap uint) *LogByteBuffer {
	res := LogByteBuffer{
		data: make([]byte, cap),
		size: int(cap),
	}
	return &res
}

func (b *LogByteBuffer) Write(p []byte) (int, error) {
	b.lock.Lock()
	defer func() {
		b.lock.Unlock()
	}()
	if len(p) >= b.size {
		// input is too big only the last portion will be written.
		b.data = p[len(p)-b.size:]
		return b.size, nil
	} else {
		b.data = append(b.data[len(p):], p...)
		return len(p), nil
	}
}

// Does not consume data just outputs the entire buffer
func (b *LogByteBuffer) Read(p []byte) (int, error) {
	b.lock.Lock()
	defer func() {
		b.lock.Unlock()
	}()
	if len(p) <= b.size {
		copy(p, b.data[b.size-len(p):])
		return len(p), nil
	} else {
		copy(p, b.data)
		return len(p), nil
	}
}
