package forwarder

import (
	"iter"
	"sync"
)

type sendBuffer struct {
	sendBuf, sendBufSpare [][]byte
	*sync.Mutex
}

// mostly copied from gophertunnel/minecraft.(*Conn).Flush()
func (buf *sendBuffer) swapAndSend() iter.Seq[[]byte]{
	return func(yield func([]byte) bool) {
		buf.Lock()
		if len(buf.sendBuf) == 0{
			buf.Unlock()
			return 
		}
		send := buf.sendBuf
		buf.sendBuf = buf.sendBufSpare[:0]
		buf.sendBufSpare = nil
		buf.Unlock()
		defer func() {
			for i := range send {
				send[i] = nil
			}
			buf.Lock()
			buf.sendBufSpare = send[:0]
			buf.Unlock()
		}()
		for _, pk := range send {
			if !yield(pk) {
				return
			}
		}
	}
}

func (buf *sendBuffer) append(b []byte){
	buf.Lock()
	buf.sendBuf = append(buf.sendBuf, b)
	buf.Unlock()
}

func newSendBuf() *sendBuffer{
	return &sendBuffer{
		Mutex: &sync.Mutex{},
		sendBuf: make([][]byte, 0, 4096),
		sendBufSpare: make([][]byte, 0, 4096),
	}
}