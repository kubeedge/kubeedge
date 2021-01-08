package rule

import (
	"k8s.io/klog/v2"
	"time"
)

type ExecResult struct {
	RuleID        string
	ProjectID     string
	Status        string
	Error         ErrorMsg
}

type ErrorMsg struct {
	Detail    string
	Timestamp time.Time
}

var ResultChannel chan ExecResult
var StopChan chan bool

func init() {
	StopChan = make(chan bool)
	go do(StopChan)
}

func do(stop chan bool) {
	ResultChannel = make(chan ExecResult, 1024)
	ruleStatus := make(map[string][2]int)
	errorMsgs := []ExecResult{}
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	for {
		select {
		case r := <-ResultChannel:
			stat, exist := ruleStatus[r.RuleID]
			if !exist {
				stat = [2]int{0, 0}
			}

			if r.Status == "SUCCESS" {
				stat[0]++
			} else if r.Status == "FAIL" {
				stat[1]++
				errorMsgs = append(errorMsgs, r)
			}
			ruleStatus[r.RuleID] = stat
			//Commit to DB if we have got enough error messages
			if len(errorMsgs) >= 50 {
				timer.Reset(30 * time.Second)
				rs := ruleStatus
				er := errorMsgs

				go handleStatus(rs, er)

				//cleanMap(ruleStatus)
				ruleStatus = make(map[string][2]int)
				errorMsgs = []ExecResult{}
			}
		case <-timer.C:
			//Record to DB once time is up
			timer.Reset(30 * time.Second)
			rs := ruleStatus
			er := errorMsgs

			go handleStatus(rs, er)

			//cleanMap(ruleStatus)
			ruleStatus = make(map[string][2]int)
			errorMsgs = []ExecResult{}
		case _, ok := <-stop:
			if !ok {
				klog.Warningf("do stop channel is closed")
			}
			return
		}
	}
}

func handleStatus(ruleStatus map[string][2]int, msg []ExecResult) {
	for k, v := range ruleStatus {
		recordStatus(k, v[0], v[1])
	}
	recordErrorMsg(msg)
}

func recordStatus(rule string, succCount int, failCount int) {

}

func recordErrorMsg(results []ExecResult) {
	//Check total error msg number for this rule

}

func cleanMap(m map[string][2]int) {
	for k := range m {
		delete(m, k)
	}
}
