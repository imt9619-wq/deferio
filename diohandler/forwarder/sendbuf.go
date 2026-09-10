package forwarder

import (
	"iter"
	"sync"
)

type sendBuffer struct {
	sendBuf, sendBufSpare [][]byte
	mu                    *sync.Mutex
}

// mostly copied from gophertunnel/minecraft.(*Conn).Flush()
func (buf *sendBuffer) swapAndSend() iter.Seq[[]byte]{
	return func(yield func([]byte) bool) {
		buf.mu.Lock()
		if len(buf.sendBuf) == 0{
			buf.mu.Unlock()
			return 
		}
		send := buf.sendBuf
		buf.sendBuf = buf.sendBufSpare[:0]
		buf.sendBufSpare = nil
		buf.mu.Unlock()
		defer func() {
			for i := range send {
				send[i] = nil
			}
			buf.mu.Lock()
			buf.sendBufSpare = send[:0]
			buf.mu.Unlock()
		}()
		for _, pk := range send {
			if !yield(pk) {
				return
			}
		}
	}
}

func (buf *sendBuffer) append(b []byte){
	buf.mu.Lock()
	buf.sendBuf = append(buf.sendBuf, b)
	buf.mu.Unlock()
}

func newSendBuf() *sendBuffer{
	return &sendBuffer{
		mu: &sync.Mutex{},
		sendBuf: make([][]byte, 0, 4096),
		sendBufSpare: make([][]byte, 0, 4096),
	}
}