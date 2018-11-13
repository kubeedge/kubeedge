package clients

import (
	"github.com/kubeedge/kubeedge/beehive/pkg/core/model"
)

type Adapter interface {
	Init() error
	Uninit()
	// async mode
	Send(message model.Message) error
	Receive() (model.Message, error)

	// notify auth info
	Notify(authInfo map[string]string)
}
