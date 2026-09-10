package forwarder

import (
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type Writer struct {
	*protocol.Writer
}

func (w *Writer) Time(t *time.Time){
	
}