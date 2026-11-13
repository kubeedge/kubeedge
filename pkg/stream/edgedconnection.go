package stream

import (
	"fmt"
	"sync"
)

// EdgedConnection indicate the connection request to the edged
type EdgedConnection interface {
	CreateConnectMessage() (*Message, error)
	Serve(tunnel SafeWriteTunneler) error
	CacheTunnelMessage(msg *Message)
	GetMessageID() uint64
	CloseReadChannel()
	CleanChannel()
	fmt.Stringer
}

// readChannelGuard serializes delivering tunnel messages to a connection's read channel with closing that channel.
//
// The tunnel session's reader goroutine delivers messages while the goroutine that served the connection removes it
// when the exec/attach/logs/metrics session ends. Without the guard a delivery can look the connection up, lose the
// race to the removal and then send on the closed channel, which panics and crashes edgecore. Holding a lock across the
// send is not enough on its own: when the channel is full the sender would wait forever while holding it, and the
// removal would wait for the lock. Closing therefore first signals done, which releases a waiting sender.
type readChannelGuard struct {
	mu        sync.RWMutex
	closed    bool
	doneOnce  sync.Once
	closeOnce sync.Once
	done      chan struct{}
}

func (g *readChannelGuard) doneChannel() chan struct{} {
	g.doneOnce.Do(func() { g.done = make(chan struct{}) })
	return g.done
}

// send delivers msg to ch unless the connection is closing; it reports whether msg was delivered.
func (g *readChannelGuard) send(ch chan *Message, msg *Message) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.closed {
		return false
	}
	select {
	case ch <- msg:
		return true
	case <-g.doneChannel():
		return false
	}
}

// close closes ch once, after releasing any sender waiting for room in it.
func (g *readChannelGuard) close(ch chan *Message) {
	done := g.doneChannel()
	g.closeOnce.Do(func() { close(done) })
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.closed {
		g.closed = true
		close(ch)
	}
}
