package poll

import (
	"github.com/smallnest/epoller"
	"k8s.io/klog"
	"net"
	"sync"
)

type Epoll struct {
	fd       int
	callback func(conn net.Conn)
	poller   epoller.Poller
}

var (
	mu sync.RWMutex
)

const (
	eventNum    = 128
	waitForever = -1
)

// CreatePoll open a epoll file description
func CreatePoll(fn func(conn net.Conn)) (*Epoll, error) {
	poller, err := epoller.NewPollerWithBuffer(128)
	ep := &Epoll{
		callback: fn, poller: poller,
	}

	if err != nil {
		return nil, err
	}
	ep.fd = -1
	return ep, err
}

// Loop for epoll
func (ep *Epoll) Loop() {
	poll(ep)
}
func poll(ep *Epoll) {
	for {

		conns, err := ep.poller.WaitWithBuffer()
		if err != nil {
			if err.Error() != "bad file descriptor" {
				klog.Error("failed to poll: %v", err)
			}

			continue
		}
		for _, conn := range conns {
			ep.callback(conn)
		}
	}
}

// DestroyEpoll implement release the epoll file description
func (ep *Epoll) DestroyEpoll() {
	ep.poller.Close()
}

// EpollCtrlAdd: Add a socket to the epoll
func (ep *Epoll) EpollCtrlAdd(conn net.Conn) error {
	err := ep.poller.Add(conn)
	if err != nil {
		klog.Errorf("Add: %s , %d, %s", err, ep.fd, conn)
	}
	return err
}

// EpollCtrlDel: delete a socket to the epoll
func (ep *Epoll) EpollCtrlDel(conn net.Conn) error {
	err := ep.poller.Remove(conn)
	if err != nil {
		klog.Errorf("Del: %s , %d, %s", err, ep.fd, conn)
	}
	return err
}

// EpollCtrlMod: modify a socket to the epoll
func (ep *Epoll) EpollCtrlMod(conn net.Conn) error {
	err := ep.poller.Remove(conn)
	if err != nil {
		klog.Errorf("Del: %s , %d, %s", err, ep.fd, conn)
	}
	err = ep.poller.Add(conn)
	if err != nil {
		klog.Errorf("Del: %s , %d, %s", err, ep.fd, conn)
	}
	return err
}
