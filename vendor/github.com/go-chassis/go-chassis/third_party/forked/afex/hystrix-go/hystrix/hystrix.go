package hystrix

// Forked from github.com/afex/hystrix-go/hystrix
// Some parts of this file have been modified to make it functional in this package
import (
	"fmt"
	"github.com/go-mesh/openlogging"
	"sync"
	"time"
)

type runFunc func() error
type fallbackFunc func(error) error

// A CircuitError is an error which models various failure states of execution,
// such as the circuit being open or a timeout.
type CircuitError struct {
	Message string
}

func (e CircuitError) Error() string {
	return e.Message
}

type FallbackNullError struct {
	Message string
}

func (e FallbackNullError) Error() string {
	return e.Message
}

// command models the state used for a single execution on a circuit. "hystrix command" is commonly
// used to describe the pairing of your run/fallback functions with a circuit.
type command struct {
	sync.Mutex

	ticket       *struct{}
	start        time.Time
	errChan      chan error
	finished     chan bool
	fallbackOnce *sync.Once
	circuit      *CircuitBreaker
	run          runFunc
	fallback     fallbackFunc
	runDuration  time.Duration
	events       []string
	timedOut     bool
}

var (
	// ErrMaxConcurrency occurs when too many of the same named command are executed at the same time.
	ErrMaxConcurrency = CircuitError{Message: "max concurrency"}
	// ErrCircuitOpen returns when an execution attempt "short circuits". This happens due to the circuit being measured as unhealthy.
	ErrCircuitOpen = CircuitError{Message: "circuit open"}
	// ErrForceFallback occurs when force fallback is true
	ErrForceFallback = CircuitError{Message: "force fallback"}
)

// Go runs your function while tracking the health of previous calls to it.
// If your function begins slowing down or failing repeatedly, we will block
// new calls to it for you to give the dependent service time to repair.
//
// Define a fallback function if you want to define some code to execute during outages.
func Go(name string, run runFunc, fallback fallbackFunc) chan error {
	cmd := &command{
		run:          run,
		fallback:     fallback,
		start:        time.Now(),
		errChan:      make(chan error, 1),
		finished:     make(chan bool, 1),
		fallbackOnce: &sync.Once{},
	}

	// dont have methods with explicit params and returns
	// let data come in and out naturally, like with any closure
	// explicit error return to give place for us to kill switch the operation (fallback)

	circuit, _, err := GetCircuit(name)
	if err != nil {
		cmd.errChan <- err
		return cmd.errChan
	}
	cmd.circuit = circuit

	go func() {
		defer func() {
			cmd.finished <- true
		}()

		//Forcefallback is true
		if getSettings(name).ForceFallback {
			cmd.errorWithFallback(ErrForceFallback)
			return
		}
		// Circuits get opened when recent executions have shown to have a high error rate.
		// Rejecting new executions allows backends to recover, and the circuit will allow
		// new traffic when it feels a healthly state has returned.
		if getSettings(name).CircuitBreakerEnabled {
			if !cmd.circuit.AllowRequest() {
				cmd.errorWithFallback(ErrCircuitOpen)
				return
			}
		}

		// As backends falter, requests take longer but don't always fail.
		//
		// When requests slow down but the incoming rate of requests stays the same, you have to
		// run more at a time to keep up. By controlling concurrency during these situations, you can
		// shed load which accumulates due to the increasing ratio of active commands to incoming requests.
		cmd.Lock()
		select {
		case cmd.ticket = <-circuit.executorPool.Tickets:
			cmd.Unlock()
		default:
			cmd.Unlock()
			cmd.errorWithFallback(ErrMaxConcurrency)
			return
		}

		runStart := time.Now()
		runErr := run()

		cmd.runDuration = time.Since(runStart)

		if runErr != nil {
			cmd.errorWithFallback(runErr)
			return
		}

		cmd.reportEvent("success")
	}()

	go func() {
		defer func() {
			cmd.Lock()
			cmd.circuit.executorPool.Return(cmd.ticket)
			cmd.Unlock()

			err := cmd.circuit.ReportEvent(cmd.events, cmd.start, cmd.runDuration)
			if err != nil {
				openlogging.GetLogger().Warnf("can not report Metrics [%s]", err.Error())
			}
		}()

		select {
		case <-cmd.finished:

		}
	}()

	return cmd.errChan
}

// Do runs your function in a synchronous manner, blocking until either your function succeeds
// or an error is returned, including hystrix circuit errors
func Do(name string, run runFunc, fallback fallbackFunc) error {
	done := make(chan struct{}, 1)

	r := func() error {
		err := run()
		if err != nil {
			return err
		}

		done <- struct{}{}
		return nil
	}

	f := func(e error) error {
		err := fallback(e)
		if err != nil {
			return err
		}

		done <- struct{}{}
		return nil
	}

	var errChan chan error
	if fallback == nil {
		errChan = Go(name, r, nil)
	} else {
		errChan = Go(name, r, f)
	}

	select {
	case <-done:
		return nil
	case err := <-errChan:
		return err
	}
}

func (c *command) reportEvent(eventType string) {
	c.Lock()
	defer c.Unlock()

	c.events = append(c.events, eventType)
}

// errorWithFallback triggers the fallback while reporting the appropriate metric events.
// If called multiple times for a single command, only the first will execute to insure
// accurate Metrics and prevent the fallback from executing more than once.
func (c *command) errorWithFallback(err error) {
	c.fallbackOnce.Do(func() {
		eventType := "failure"
		if err == ErrCircuitOpen {
			eventType = "short-circuit"
		} else if err == ErrMaxConcurrency {
			eventType = "rejected"
		} else if err == ErrForceFallback {
			eventType = "force fallback"
		}

		c.reportEvent(eventType)
		fallbackErr := c.tryFallback(err)
		if fallbackErr != nil {
			c.errChan <- fallbackErr
		}
	})
}

func (c *command) tryFallback(err error) error {
	if c.fallback == nil {

		// If we don't have a fallback return the original error.
		return err
	}

	fallbackErr := c.fallback(err)
	if fallbackErr != nil {
		c.reportEvent("fallback-failure")
		return fmt.Errorf("fallback failed with '%v'. run error was '%v'", fallbackErr, err)
	}

	c.reportEvent("fallback-success")

	return nil
}
