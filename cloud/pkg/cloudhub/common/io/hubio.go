package io

import (
	"k8s.io/klog"
	"sync"
	"time"

	"github.com/kubeedge/beehive/pkg/core/model"
	commonmodel "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	"github.com/kubeedge/viaduct/pkg/conn"
)

// CloudHubIO handle the IO operation from connection
type CloudHubIO interface {
	sync.Locker
	GetHubInfo() *commonmodel.HubInfo
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	ReadData(*model.Message) (int, error)
	WriteData(*model.Message) error
	Close() error
	KeepaliveChannel() chan struct{}
}

// JSONIO address the json data from connection
type JSONIO struct {
	hubInfo commonmodel.HubInfo
	sync.Mutex
	Connection       conn.Connection
	keepaliveChannel chan struct{}
}

func NewJSONIO(nodeId, projectId string, conn conn.Connection) CloudHubIO {
	return &JSONIO{
		hubInfo:          commonmodel.HubInfo{},
		Connection:       conn,
		keepaliveChannel: make(chan struct{}, 1),
	}
}

//GetHubInfo get HubInfo
func (io *JSONIO) GetHubInfo() *commonmodel.HubInfo {
	return &io.hubInfo
}

// SetReadDeadline set read operation dead line
func (io *JSONIO) SetReadDeadline(time time.Time) error {
	return io.Connection.SetReadDeadline(time)
}

// SetWriteDeadline set write operation dead line
func (io *JSONIO) SetWriteDeadline(time time.Time) error {
	return io.Connection.SetWriteDeadline(time)
}

// ReadData read data from connection
func (io *JSONIO) ReadData(msg *model.Message) (int, error) {
	return 0, io.Connection.ReadMessage(msg)
}

// WriteData write data to connection
func (io *JSONIO) WriteData(msg *model.Message) error {
	err := io.Connection.WriteMessageAsync(msg)
	if err != nil {
		return err
	}
	return nil
}

// Close close the IO operation
func (io *JSONIO) Close() error {
	klog.Infof("websocket connection close, nodeID: %s", io.hubInfo.NodeID)
	return io.Connection.Close()
}

//KeepaliveChannel get KeepaliveChannel
func (io *JSONIO) KeepaliveChannel() chan struct{} {
	return io.keepaliveChannel
}
