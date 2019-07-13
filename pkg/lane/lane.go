package lane

import (
	"time"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/api"
)

type Lane interface {
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	ReadMessage(msg *model.Message) error
	WriteMessage(msg *model.Message) error
	Read(raw []byte) (int, error)
	Write(raw []byte) (int, error)
}

func NewLane(protoType string, van interface{}) Lane {
	switch protoType {
	case api.ProtocolTypeQuic:
		return NewQuicLane(van)
	case api.ProtocolTypeWS:
		return NewWSLaneWithoutPack(van)
	}
	log.LOGGER.Errorf("bad protocol type(%s)", protoType)
	return nil
}
