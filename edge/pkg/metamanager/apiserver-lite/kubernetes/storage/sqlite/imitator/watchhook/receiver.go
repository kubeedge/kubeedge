package watchhook

import "k8s.io/apimachinery/pkg/watch"

type Receiver interface {
	Receive(event watch.Event) error
}
type ChanReceiver struct {
	ch chan<- watch.Event
}

func (hc *ChanReceiver) Receive(event watch.Event)error{
	//TODO: recover when hc.ch is closed if panic occurs
	hc.ch <- event
	return nil
}

func NewChanReceiver(ch chan<- watch.Event) *ChanReceiver {
	return &ChanReceiver{ch: ch}
}
