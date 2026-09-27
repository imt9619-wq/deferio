package forwarder

import (
	"fmt"
	"net"
)

type DialConfig struct { // size=96 (0x60)
    ConnConfig
    DialF          func(address string) (net.Conn, error)
}

func (d DialConfig) defaultDialConfig() DialConfig {
	if d.DialF == nil {
		d.DialF = func(address string) (net.Conn, error) {
			return net.Dial("unix", address)
		}
	}
	d.ConnConfig = d.defaultConnConfig()
	return d
}

func (d DialConfig) dial() (*Conn, error) {
	conn, err := d.DialF(d.Address)
	if err != nil {
		return nil, fmt.Errorf("Forwarder: Failed to dial: %v", err)
	}
	return d.newConn(conn), nil
}