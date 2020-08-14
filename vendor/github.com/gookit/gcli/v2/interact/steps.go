package interact

import (
	"context"
	"fmt"
)

// StepHandler for steps run
type StepHandler func(ctx context.Context) error

// StepsRun follow the steps to run
type StepsRun struct {
	// mark is stopped
	stopped bool
	// steps length
	length int
	// current step index
	current int
	// Steps step name and handler define.
	// {
	// 	// step 1
	// 	func(ctx context.Context) { do something.}
	// 	// step 2
	// 	func(ctx context.Context) { do something.}
	// }
	Steps []StepHandler
	// record error
	err error
}

// Run all steps
func (s *StepsRun) Run() {
	if s.stopped {
		return
	}

	s.length = len(s.Steps)
	if s.length == 0 {
		s.err = fmt.Errorf("no step need to running")
		return
	}

	ctx := context.Background()

	for i, handler := range s.Steps {
		s.current = i

		err := handler(ctx)
		if err != nil {
			s.err = err
			return
		}
	}
}

// Stop set stop run
func (s *StepsRun) Stop() {
	s.stopped = true
}

// Err get error
func (s *StepsRun) Err() error {
	return s.err
}
