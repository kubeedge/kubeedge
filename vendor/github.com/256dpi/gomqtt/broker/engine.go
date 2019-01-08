// Package broker implements an extensible MQTT broker.
package broker

import (
	"net"
	"sync"
	"time"

	"github.com/256dpi/gomqtt/transport"

	"gopkg.in/tomb.v2"
)

// The Engine handles incoming connections and connects them to the backend.
type Engine struct {
	// The Backend that will be passed to accepted clients.
	Backend Backend

	// ConnectTimeout defines the timeout to receive the first packet.
	ConnectTimeout time.Duration

	// The DefaultReadLimit defines the initial read limit.
	DefaultReadLimit int64

	// OnError can be used to receive errors from engine. If an error is received
	// the server should be restarted.
	OnError func(error)

	mutex sync.Mutex
	tomb  tomb.Tomb
}

// NewEngine returns a new Engine.
func NewEngine(backend Backend) *Engine {
	return &Engine{
		Backend:        backend,
		ConnectTimeout: 10 * time.Second,
	}
}

// Accept begins accepting connections from the passed server.
func (e *Engine) Accept(server transport.Server) {
	e.tomb.Go(func() error {
		for {
			// return if dying
			if !e.tomb.Alive() {
				return tomb.ErrDying
			}

			// accept next connection
			conn, err := server.Accept()
			if err != nil {
				// call error callback if available
				if e.OnError != nil {
					e.OnError(err)
				}

				return err
			}

			// handle connection
			if !e.Handle(conn) {
				return nil
			}
		}
	})
}

// Handle takes over responsibility and handles a transport.Conn. It returns
// false if the engine is closing and the connection has been closed.
func (e *Engine) Handle(conn transport.Conn) bool {
	// check conn
	if conn == nil {
		panic("passed conn is nil")
	}

	// acquire mutex
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// close conn immediately when dying
	if !e.tomb.Alive() {
		conn.Close()
		return false
	}

	// set default read limit
	conn.SetReadLimit(e.DefaultReadLimit)

	// set initial read timeout
	conn.SetReadTimeout(e.ConnectTimeout)

	// handle client
	NewClient(e.Backend, conn)

	return true
}

// Close will stop handling incoming connections and close all acceptors. The
// call will block until all acceptors returned.
//
// Note: All passed servers to Accept must be closed before calling this method.
func (e *Engine) Close() {
	// acquire mutex
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// stop acceptors
	e.tomb.Kill(nil)
	e.tomb.Wait()
}

// Run runs the passed engine on a random available port and returns a channel
// that can be closed to shutdown the engine. This method is intended to be used
// in testing scenarios.
func Run(engine *Engine, protocol string) (string, chan struct{}, chan struct{}) {
	// launch server
	server, err := transport.Launch(protocol + "://localhost:0")
	if err != nil {
		panic(err)
	}

	// prepare channels
	quit := make(chan struct{})
	done := make(chan struct{})

	// start accepting connections
	engine.Accept(server)

	// prepare shutdown
	go func() {
		// wait for signal
		<-quit

		// errors from close are ignored
		server.Close()

		// close broker
		engine.Close()

		close(done)
	}()

	// get random port
	_, port, _ := net.SplitHostPort(server.Addr().String())

	return port, quit, done
}
