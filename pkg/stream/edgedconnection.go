package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/avast/retry-go"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	"k8s.io/klog/v2"
)

const (
	sendRemoveMesaageRetryAttempts = 3
	sendRemoveMesaageRetryDelay    = 1 * time.Second
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

// httpClientCustomization sets up custom processing for serveByClient()
type httpClientCustomization struct {
	name                   string
	handleRequest          func(*http.Request)
	receiveFromCloudStream func()
	write2CloudStream      func(tunnel SafeWriteTunneler, resp *http.Response)
}

// roundTripperCustomization sets up custom processing for serveByRoundTripper()
type roundTripperCustomization struct {
	name                   string
	handleRequest          func(*http.Request)
	receiveFromCloudStream func(con net.Conn)
	write2CloudStream      func(tunnel SafeWriteTunneler, con net.Conn)
}

type BaseEdgedConnection struct {
	MessID    uint64          `json:"messID"`
	Method    string          `json:"method"`
	Header    http.Header     `json:"header"`
	URL       url.URL         `json:"url"`
	ctx       context.Context `json:"-"`
	cancel    func()          `json:"-"`
	ReadChan  chan *Message   `json:"-"`
	closeOnce sync.Once       `json:"-"`
	servering atomic.Bool     `json:"-"`
	Stop      chan struct{}   `json:"-"`
}

func (c *BaseEdgedConnection) InitContext(parent context.Context) {
	if parent == nil {
		parent = context.Background()
	}
	c.ctx, c.cancel = context.WithCancel(parent)
}

func (c *BaseEdgedConnection) GetMessageID() uint64 {
	return c.MessID
}

func (c *BaseEdgedConnection) CacheTunnelMessage(msg *Message) {
	select {
	case <-c.ctx.Done():
		return
	default:
		c.ReadChan <- msg
	}
}

func (c *BaseEdgedConnection) CloseReadChannel() {
	c.closeOnce.Do(func() {
		c.cancel()
		close(c.ReadChan)
	})
}

func (c *BaseEdgedConnection) CleanChannel() {
	for {
		select {
		case <-c.Stop:
		default:
			return
		}
	}
}

func (c *BaseEdgedConnection) createConnectMessage(t MessageType, impl EdgedConnection) (*Message, error) {
	data, err := json.Marshal(impl)
	if err != nil {
		return nil, err
	}
	return NewMessage(c.MessID, t, data), nil
}

// serveByClient provides a kubelet proxy service implemented by http.Client
func (c *BaseEdgedConnection) serveByClient(
	tunnel SafeWriteTunneler,
	cus httpClientCustomization,
) error {
	if !c.servering.CompareAndSwap(false, true) {
		return errors.New("this instance is already serving")
	}
	if cus.name == "" {
		return errors.New("name of server customization must be not empty")
	}
	if cus.receiveFromCloudStream == nil {
		return errors.New("receiveFromCloudStream function of server customization must be not nil")
	}
	if cus.write2CloudStream == nil {
		return errors.New("write2CloudStream function of server customization must be not nil")
	}

	if c.ctx == nil {
		c.InitContext(nil)
	}
	// connect edged
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, c.URL.String(), nil)
	if err != nil {
		klog.Errorf("create new logs request error %v", err)
		return err
	}
	req.Header = c.Header
	if cus.handleRequest != nil {
		cus.handleRequest(req)
	}
	resp, err := client.Do(req)
	if err != nil {
		klog.Errorf("request logs error %v", err)
		return err
	}
	defer func() {
		if !resp.Close {
			if err := resp.Body.Close(); err != nil {
				klog.Warningf("failed to close response, err: %v", err)
			}
		}
	}()

	defer func() {
		err := retry.Do(func() error {
			msg := NewMessage(c.MessID, MessageTypeRemoveConnect, nil)
			if err := tunnel.WriteMessage(msg); err != nil {
				return fmt.Errorf("%s send %s message error, err: %v", cus.name, msg.MessageType, err)
			}
			return nil
		}, retry.Attempts(sendRemoveMesaageRetryAttempts), retry.Delay(sendRemoveMesaageRetryDelay))
		if err != nil {
			klog.Warningf("failed to write remove connection message to cloud, err: %v", err)
		}
	}()

	go cus.receiveFromCloudStream()
	go cus.write2CloudStream(tunnel, resp)
	if c.Stop == nil {
		c.Stop = make(chan struct{}, 2)
	}
	<-c.Stop
	klog.Infof("receive stop signal, stop %s ...", cus.name)
	return nil
}

// serveByClient provides a kubelet proxy service implemented by RoundTripper
func (c *BaseEdgedConnection) serveByRoundTripper(
	tunnel SafeWriteTunneler,
	cus roundTripperCustomization,
) error {
	if !c.servering.CompareAndSwap(false, true) {
		return errors.New("this instance is already serving")
	}
	if cus.name == "" {
		return errors.New("name of server customization must be not empty")
	}
	if cus.receiveFromCloudStream == nil {
		return errors.New("receiveFromCloudStream function of server customization must be not nil")
	}
	if cus.write2CloudStream == nil {
		return errors.New("write2CloudStream function of server customization must be not nil")
	}

	if c.ctx == nil {
		c.InitContext(nil)
	}
	tripper, err := spdy.NewRoundTripper(nil)
	if err != nil {
		return fmt.Errorf("failed to creates a new tripper, err: %v", err)
	}
	req, err := http.NewRequest(c.Method, c.URL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create attach request, err: %v", err)
	}
	req.Header = c.Header
	if cus.handleRequest != nil {
		cus.handleRequest(req)
	}
	con, err := tripper.Dial(req)
	if err != nil {
		klog.Errorf("failed to dial, err: %v", err)
		return err
	}
	defer func() {
		if err := con.Close(); err != nil {
			klog.Warningf("failed to close round tropper connection, err: %v", err)
		}
	}()

	defer func() {
		err := retry.Do(func() error {
			msg := NewMessage(c.MessID, MessageTypeRemoveConnect, nil)
			if err := tunnel.WriteMessage(msg); err != nil {
				return fmt.Errorf("%s send %s message error, err: %v", cus.name, msg.MessageType, err)
			}
			return nil
		}, retry.Attempts(sendRemoveMesaageRetryAttempts), retry.Delay(sendRemoveMesaageRetryDelay))
		if err != nil {
			klog.Warningf("failed to write remove connection message to cloud, err: %v", err)
		}
	}()

	go cus.receiveFromCloudStream(con)
	go cus.write2CloudStream(tunnel, con)
	if c.Stop == nil {
		c.Stop = make(chan struct{}, 2)
	}
	<-c.Stop
	klog.Infof("receive stop signal, so stop %s ...", cus.name)
	return nil
}
