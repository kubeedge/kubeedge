/*
Copyright 2026 The KubeEdge Authors.

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

package keeper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func TestMatchAndNotifyBeforeReceive(t *testing.T) {
	k := NewSyncKeeper()
	request := model.NewMessage("")
	channel := k.addKeepChannel(request.GetID())

	// The response arrives before anyone receives from the channel.
	response := model.NewMessage(request.GetID())
	require.True(t, k.MatchAndNotify(*response))

	select {
	case got := <-channel:
		assert.Equal(t, response.GetID(), got.GetID())
	default:
		t.Fatal("the response was dropped because nobody was receiving yet")
	}
}

func TestMatchAndNotifyWithoutWaiter(t *testing.T) {
	k := NewSyncKeeper()
	response := model.NewMessage("unknown-request-id")

	assert.False(t, k.Match(*response))
	assert.False(t, k.MatchAndNotify(*response))
}

func TestWaitResponse(t *testing.T) {
	k := NewSyncKeeper()
	request := model.NewMessage("")
	response := model.NewMessage(request.GetID())

	go func() {
		// Notify as soon as WaitResponse has registered the request.
		for !k.MatchAndNotify(*response) {
			time.Sleep(time.Millisecond)
		}
	}()

	got, err := k.WaitResponse(request, time.Now().Add(10*time.Second))
	require.NoError(t, err)
	assert.Equal(t, response.GetID(), got.GetID())
	assert.False(t, k.Match(*response), "the request should be removed after the response")
}

func TestWaitResponseTimeout(t *testing.T) {
	k := NewSyncKeeper()
	request := model.NewMessage("")

	_, err := k.WaitResponse(request, time.Now().Add(10*time.Millisecond))
	require.ErrorContains(t, err, "wait response timeout")
	assert.False(t, k.Match(*model.NewMessage(request.GetID())), "the request should be removed after the timeout")
}
