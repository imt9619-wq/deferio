package forwarder

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const(
	Day = time.Hour * 24
)

type Forwarder struct {
	*Conn
    idMu            *sync.Mutex
    emptyIdSlot     []int
    idToPconn        []*PlayerConn
	timePoint       time.Time
	serverAddr      string
	isConnDown      *atomic.Bool
}

type ForwarderConfig struct{
	DialConfig
	ServerAddr string
    ShouldRedial  bool
    Redialretrys   int
    RedialIntervel time.Duration
}

func (conf ForwarderConfig) defaultForwarderConfig() ForwarderConfig{
	if conf.ShouldRedial && conf.RedialIntervel == 0{
		conf.RedialIntervel = 30 * time.Second
	}
	conf.DialConfig = conf.DialConfig.defaultDialConfig()
	return conf
}

func (conf ForwarderConfig) Dial() (*Forwarder, error){
	conf = conf.defaultForwarderConfig()
	if !conf.ShouldRedial{
		conn, err := conf.DialConfig.dial()
		if err != nil{
			return nil, err
		}
		fw := &Forwarder{
			Conn: conn,
			idMu: &sync.Mutex{},
			emptyIdSlot: make([]int, 0, 128),
			idToPconn: make([]*PlayerConn, 0, 32),
			timePoint: time.Now(),
			serverAddr: conf.ServerAddr,
			isConnDown: &atomic.Bool{},
		}	
		if err = fw.ondial(); err != nil{
			_ = fw.Conn.closeConn(false)
			return nil, err
		}
		return fw, nil
	}
	fw := &Forwarder{
		idMu: &sync.Mutex{},
		emptyIdSlot: make([]int, 0, 128),
		idToPconn: make([]*PlayerConn, 0, 32),
		timePoint: time.Now(),
		serverAddr: conf.ServerAddr,
		isConnDown: &atomic.Bool{},
	}
	var tryRedial func()
	tryRedial = func(){
		fw.isConnDown.Store(true)
		rt := 0
		for{
			if conf.Redialretrys > 0 && rt >= conf.Redialretrys{
				conf.Log.Error(fmt.Sprintf("gave up dialing AC after %d retries", rt), "Forwarder", "tryRedial")
				return
			}
			conn, err := conf.DialF(conf.Address)
			rt++
			if err != nil{
				conf.Log.Error(fmt.Sprintf("Failed redial on AC server (retry: %d): %s", rt, err), "Forwarder", "tryRedial")
				time.Sleep(conf.RedialIntervel)
				continue
			}
			fw.installConn(conn, tryRedial)
			fw.isConnDown.Store(false)
			if err = fw.ondial(); err != nil{
				conf.Log.Error(fmt.Sprintf("Failed to send newdial packet after redial: %s", err), "Forwarder", "tryRedial")
				fw.isConnDown.Store(true)
				_ = fw.Conn.closeConn(false)
				time.Sleep(conf.RedialIntervel)
				continue
			}
			return
		}
	}
	conf.fallback = tryRedial
	conn, err := conf.DialConfig.dial()
	if err != nil{
		fw.Conn = conf.getEmptyConn()
		fw.isConnDown.Store(true)
		go tryRedial()
		return fw, nil
	}
	fw.Conn = conn
	if err = fw.ondial(); err != nil{
		fw.isConnDown.Store(true)
		_ = fw.Conn.closeConn(false)
		go tryRedial()
	}
	return fw, nil
}

func (fw *Forwarder) installConn(nc net.Conn, fallback func()){
	if fw.Conn.loopStarted.Load(){
		_ = fw.Conn.closeConn(false)
	}
	fw.Conn.reset()
	fw.Conn.conf.fallback = fallback
	fw.Conn.Conn = nc
	fw.Conn.shieldID.Store(0)
	fw.Conn.shieldIDSet.Store(false)
	fw.Conn.loopStarted.Store(true)
	go fw.Conn.flushLoop()
}

func (fw *Forwarder) forwardPacket(pk ForwardPacket, id uint16, source uint8) error{
	if fw.isConnDown.Load(){
		return nil
	}
	hdr := Header{
		timeOffset: time.Since(fw.timePoint)%Day,
	}
	return fw.Conn.forwardPacket(&PacketWrapper{pk: pk, hdr: &hdr}, id, source)
}

func (fw *Forwarder) ondial() error{
	if err := fw.forwardPacket(&NewDialPacket{
		serverTime: fw.timePoint.UnixMicro(),
		serverAddr: fw.serverAddr,
	}, ServerID, SourceDeferioPacket); err != nil{
		return err
	}
	fw.idMu.Lock()
	players := append([]*PlayerConn(nil), fw.idToPconn...)
	fw.idMu.Unlock()
	for _, pconn := range players{
		if pconn != nil && pconn.data != nil{
			pconn.sendIncPlayer()
		}
	}
	return nil
}
