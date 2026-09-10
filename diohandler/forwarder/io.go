package forwarder

import (
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type IO interface{
	protocol.IO
	Time(*time.Time)
}