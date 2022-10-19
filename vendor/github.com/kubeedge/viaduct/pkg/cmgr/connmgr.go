package cmgr

import (
	"sync"

	"github.com/kubeedge/viaduct/pkg/conn"
)

// the callback for getting connection key
type ConnKey func(connection conn.Connection) string

// connection instances management
type ConnectionManager struct {
	connKey     ConnKey
	connections sync.Map
}

// new connection manager instance
// you the conn key like this:
//func getConnKey(conn conn.Connection) string {
//	return conn.ConnectionState().Headers.Get("node_id")
//}
func NewManager(connKey ConnKey) *ConnectionManager {
	keyFunc := getConnKeyDefault
	if connKey != nil {
		keyFunc = connKey
	}
	return &ConnectionManager{
		connKey: keyFunc,
	}
}

// get conn key default
func getConnKeyDefault(conn conn.Connection) string {
	return conn.RemoteAddr().String()
}

// add connection into store
func (mgr *ConnectionManager) AddConnection(conn conn.Connection) {
	mgr.connections.Store(mgr.connKey(conn), conn)
}

// delete connection from store
func (mgr *ConnectionManager) DelConnection(conn conn.Connection) {
	mgr.connections.Delete(mgr.connKey(conn))
}

// get connection for store
func (mgr *ConnectionManager) GetConnection(key string) (conn.Connection, bool) {
	obj, exist := mgr.connections.Load(key)
	if exist {
		return obj.(conn.Connection), true
	}
	return nil, false
}

func (mgr *ConnectionManager) Range(f func(key, value interface{}) bool) {
	mgr.connections.Range(f)
}
