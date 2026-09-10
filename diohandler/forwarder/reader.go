package forwarder

import (
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type Reader struct {
	*protocol.Reader
}

func (r *Reader) Time(t *time.Time){
}