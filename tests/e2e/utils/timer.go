/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"sync"
	"time"
)

// TestTimer represents a test timer
type TestTimer struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
}

// End is used to end the test timer
func (testTimer *TestTimer) End() {
	if !testTimer.IsEnded() {
		testTimer.EndTime = time.Now()
	}
}

// IsEnded represents if the test timer is ended
func (testTimer *TestTimer) IsEnded() bool {
	return !testTimer.EndTime.IsZero()
}

// Duration is used to calculate the duration
func (testTimer *TestTimer) Duration() time.Duration {
	endTime := testTimer.EndTime
	if !testTimer.IsEnded() {
		endTime = time.Now()
	}
	return endTime.Sub(testTimer.StartTime)
}

// PrintResult prints the result of the test timer
func (testTimer *TestTimer) PrintResult() {
	if testTimer.IsEnded() {
		Infof("Test case name: %s start time: %v duration: %v\n",
			testTimer.Name, testTimer.StartTime, testTimer.Duration())
	} else {
		Infof("Test case name: %s start time: %v duration: %v so far\n",
			testTimer.Name, testTimer.StartTime, testTimer.Duration())
	}
}

// TestTimerGroup includes one or more test timers
type TestTimerGroup struct {
	mutex      sync.Mutex
	testTimers []*TestTimer
}

// NewTestTimerGroup creates a new test timer group
func NewTestTimerGroup() *TestTimerGroup {
	return &TestTimerGroup{}
}

// NewTestTimer creates a new test timer
func (group *TestTimerGroup) NewTestTimer(name string) *TestTimer {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	testTimer := &TestTimer{Name: name, StartTime: time.Now()}
	group.testTimers = append(group.testTimers, testTimer)
	return testTimer
}

// GetTestTimers returns test timers
func (group *TestTimerGroup) GetTestTimers() []*TestTimer {
	return group.testTimers
}

// PrintResult prints the results of all test timers.
func (group *TestTimerGroup) PrintResult() {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	for _, testTimer := range group.testTimers {
		testTimer.PrintResult()
	}
}
