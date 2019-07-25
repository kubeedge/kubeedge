package poll

import (
	"syscall"

	"k8s.io/klog"
)

type Epoll struct {
	fd       int
	callback func(fd int32)
}

const (
	eventNum    = 128
	waitForever = -1
)

// CreatePoll open a epoll file description
func CreatePoll(fn func(fd int32)) (*Epoll, error) {
	ep := &Epoll{
		callback: fn,
	}
	fd, err := syscall.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	ep.fd = fd
	return ep, err
}

// Loop for epoll
func (ep *Epoll) Loop() {
	events := make([]syscall.EpollEvent, eventNum)
	for {
		num, err := syscall.EpollWait(ep.fd, events, waitForever)
		if num < 0 || err != nil {
			klog.Warningf("[L4 Proxy] poll wait error : %s", err)
		}
		for _, ev := range events {
			if ev.Events&syscall.EPOLLIN == syscall.EPOLLIN {
				ep.callback(ev.Fd)
			}
		}
	}
}

// DestroyEpoll implement release the epoll file description
func (ep *Epoll) DestroyEpoll() {
	syscall.Close(ep.fd)
}

// EpollCtrlAdd: Add a socket to the epoll
func (ep *Epoll) EpollCtrlAdd(fd int32) error {
	err := syscall.EpollCtl(ep.fd, syscall.EPOLL_CTL_ADD, int(fd), &syscall.EpollEvent{
		Fd:     int32(fd),
		Events: syscall.EPOLLIN,
	})
	if err != nil {
		klog.Errorf("Add: %s , %d, %d", err, ep.fd, fd)
	}

	return err
}

// EpollCtrlDel: delete a socket to the epoll
func (ep *Epoll) EpollCtrlDel(fd int) error {
	err := syscall.EpollCtl(ep.fd, syscall.EPOLL_CTL_DEL, fd, &syscall.EpollEvent{
		Fd:     int32(fd),
		Events: uint32(syscall.EPOLLIN),
	})
	if err != nil {
		klog.Errorf("Del: %s , %d, %d", err, ep.fd, fd)
	}
	return err
}

// EpollCtrlMod: modify a socket to the epoll
func (ep *Epoll) EpollCtrlMod(fd int) error {
	err := syscall.EpollCtl(ep.fd, syscall.EPOLL_CTL_MOD, fd, &syscall.EpollEvent{
		Fd:     int32(fd),
		Events: syscall.EPOLLIN,
	})
	return err
}
