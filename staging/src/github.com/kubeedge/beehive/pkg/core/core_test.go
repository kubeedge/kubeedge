/*
Copyright 2025 The KubeEdge Authors.

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

package core

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLocalModuleKeeper(t *testing.T) {
	t.Run("module restart policy is nil", func(t *testing.T) {
		var callStart int
		info := ModuleInfo{
			module: &SimpleModule{
				name:          "test",
				group:         "test",
				enable:        true,
				restartPolicy: nil,
				StartFunc: func() {
					callStart++
				},
			},
		}
		localModuleKeeper(&info)
		assert.Equal(t, 1, callStart)
	})

	t.Run("module restart policy is always", func(t *testing.T) {
		var callStart int
		info := ModuleInfo{
			module: &SimpleModule{
				name:   "test",
				group:  "test",
				enable: true,
				restartPolicy: &ModuleRestartPolicy{
					RestartType:    RestartTypeAlways,
					Retries:        2,
					IntervalSecond: 1,
				},
				StartFunc: func() {
					callStart++
				},
			},
		}
		localModuleKeeper(&info)
		assert.Equal(t, 3, callStart)
	})

	t.Run("module restart policy is on failure", func(t *testing.T) {
		var callStart int
		info := ModuleInfo{
			module: &SimpleModule{
				name:   "test",
				group:  "test",
				enable: true,
				restartPolicy: &ModuleRestartPolicy{
					RestartType:    RestartTypeOnFailure,
					Retries:        2,
					IntervalSecond: 1,
				},
				StartEFunc: func() error {
					callStart++
					// Error once, so it will only call Start() 2 times in total
					if callStart == 1 {
						return errors.New("test error")
					}
					return nil
				},
			},
		}
		localModuleKeeper(&info)
		assert.Equal(t, 2, callStart)
	})
}

func TestCalculateIntervalTime(t *testing.T) {
	cases := []struct {
		name       string
		curr       time.Duration
		limit      time.Duration
		growthRate float64
		want       time.Duration
	}{
		{
			name: "IntervalTimeGrowthRate is less than 1",
			curr: 1 * time.Second,
			want: 1 * time.Second,
		},
		{
			name:       "use default limit and interval time is equal to limit",
			curr:       30 * time.Second,
			growthRate: 2,
			want:       30 * time.Second,
		},
		{
			name:       "growth rate is 2 and calculated interval time is less than limit",
			curr:       4 * time.Second,
			limit:      10 * time.Second,
			growthRate: 2,
			want:       8 * time.Second,
		},
		{
			name:       "growth rate is 2 and calculated interval time is greater than limit",
			curr:       6 * time.Second,
			limit:      10 * time.Second,
			growthRate: 2,
			want:       10 * time.Second,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := calculateIntervalTime(c.curr, c.limit, c.growthRate)
			assert.Equal(t, c.want, res)
		})
	}
}
