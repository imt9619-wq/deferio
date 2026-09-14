package detection

import "github.com/deferio/forwarder"

const (
	Killaura = iota + 1
	Scaffolding
)

type DetectionHandler interface {
	HandleDetection(player *forwarder.PlayerGameData, offence uint8, confidence float32)
}

type NopDetectionHandler struct{}

func (NopDetectionHandler) HandleDetection(player *forwarder.PlayerGameData, offence uint8, confidence float32) {}

type DetectionCache struct{
	
}