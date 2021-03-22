package rule

import (
	"time"

	"k8s.io/klog/v2"
)

type ExecResult struct {
	RuleID    string
	ProjectID string
	Status    string
	Error     ErrorMsg
}

type ErrorMsg struct {
	Detail    string
	Timestamp time.Time
}

var (
	StopC         = make(chan struct{})
	ResultChannel = make(chan ExecResult, 1024)
)

func init() {
	go handleResult(ResultChannel, StopC)
}

func handleResult(resultReceiver <-chan ExecResult, stopC chan struct{}) {
	ruleStatus := make(map[string][2]int)
	errResults := []ExecResult{}
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	handleStatusAndReset := func() {
		if !timer.Stop() {
			<-timer.C
		}
		timer.Reset(30 * time.Second)

		go handleStatus(ruleStatus, errResults)

		ruleStatus = make(map[string][2]int)
		errResults = []ExecResult{}
	}

	for {
		select {
		case <-stopC:
			return
		case <-timer.C:
			// handle status once time is up
			handleStatusAndReset()
		case r, ok := <-resultReceiver:
			if !ok {
				klog.Info("result recevier is closed")
				return
			}

			stat, exist := ruleStatus[r.RuleID]
			if !exist {
				stat = [2]int{0, 0}
			}

			switch r.Status {
			case "SUCCESS":
				stat[0]++
			case "FAIL":
				stat[1]++
				errResults = append(errResults, r)
			}

			ruleStatus[r.RuleID] = stat

			// handle status if we have got enough error messages
			if len(errResults) >= 50 {
				handleStatusAndReset()
			}
		}
	}
}

func handleStatus(ruleStatus map[string][2]int, errResults []ExecResult) {
	for k, v := range ruleStatus {
		recordStatus(k, v[0], v[1])
	}
	recordErrorResults(errResults)
}

func recordStatus(rule string, succCount int, failCount int) {

}

func recordErrorResults(results []ExecResult) {
	//Check total error msg number for this rule

}
