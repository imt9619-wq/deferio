package main

import (
	"errors"

	dioplayer "github.com/deferio/au/player"
	"github.com/sandertv/gophertunnel/minecraft"
)

type Session struct {
	front, back *minecraft.Conn
	listener *minecraft.Listener
	p *dioplayer.Player
}

func NewSession(front, back *minecraft.Conn, listener *minecraft.Listener) *Session{
	s := &Session{}
	s.front, s.back = front, back
	s.listener = listener
	return s 
}

func (s *Session) InitializePlayer() error{
	errC := make(chan error, 2)
	data := s.back.GameData()
	go func(){errC <- s.front.StartGame(data)}()
	go func(){errC <- s.back.DoSpawn()}()
	for range 2{
		if err := <-errC; err != nil{
			return err
		}
	}
	s.p = dioplayer.NewPlayer(&data)
	defer s.back.Close()
	defer s.listener.Disconnect(s.front, "connection lost")
	go func(){errC <- s.startFrontLoop()}()
	go func(){errC <- s.startBackLoop()}()
	for range 2{
		if err := <-errC; err != nil{
			return err
		}
	}
	return nil
}

func (s *Session) startFrontLoop() error{
	for {
		pk, err := s.front.ReadPacket()
		if err != nil {
			return err
		}
		s.p.HandleFrontPacket(pk)
		if err := s.back.WritePacket(pk); err != nil {
			var disc minecraft.DisconnectError
			if ok := errors.As(err, &disc); ok {
				_ = s.listener.Disconnect(s.front, disc.Error())
			}
			return err
		}
	}
}

func (s *Session) startBackLoop() error{
	for {
		pk, err := s.back.ReadPacket()
		if err != nil {
			var disc minecraft.DisconnectError
			if ok := errors.As(err, &disc); ok {
				_ = s.listener.Disconnect(s.front, disc.Error())
			}
			return err
		}
		s.p.HandleBackPacket(pk)
		if err := s.front.WritePacket(pk); err != nil {
			return err
		}
	}
}