package leaderelection

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"k8s.io/client-go/tools/leaderelection"
)

// ReadyzAdaptor associates the /readyz endpoint with the LeaderElection object.
// It helps deal with the /readyz endpoint being set up prior to the LeaderElection.
// This contains the code needed to act as an adaptor between the leader
// election code the health check code. It allows us to provide readyz
// status about the leader election. Most specifically about if the leader
// has failed to renew without exiting the process. In that case we should
// report not healthy and rely on the kubelet to take down the process.
type ReadyzAdaptor struct {
	pointerLock sync.Mutex
	le          *leaderelection.LeaderElector
	timeout     time.Duration
}

// Name returns the name of the health check we are implementing.
func (l *ReadyzAdaptor) Name() string {
	return "leaderElection"
}

// Check is called by the readyz endpoint handler.
// It fails (returns an error) if we own the lease but had not been able to renew it.
func (l *ReadyzAdaptor) Check(req *http.Request) error {
	l.pointerLock.Lock()
	defer l.pointerLock.Unlock()
	if l.le == nil {
		return fmt.Errorf("leaderElection is not setting")
	}
	if !l.le.IsLeader() {
		return fmt.Errorf("not yet a leader")
	}
	return l.le.Check(l.timeout)
}

// SetLeaderElection ties a leader election object to a ReadyzAdaptor
func (l *ReadyzAdaptor) SetLeaderElection(le *leaderelection.LeaderElector) {
	l.pointerLock.Lock()
	defer l.pointerLock.Unlock()
	l.le = le
}

// NewLeaderReadyzAdaptor creates a basic healthz adaptor to monitor a leader election.
// timeout determines the time beyond the lease expiry to be allowed for timeout.
// checks within the timeout period after the lease expires will still return healthy.
func NewLeaderReadyzAdaptor(timeout time.Duration) *ReadyzAdaptor {
	result := &ReadyzAdaptor{
		timeout: timeout,
	}
	return result
}
