/*
Copyright 2022 The KubeEdge Authors.

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

package session

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"k8s.io/klog/v2"

	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common"
	tf "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/testing"
	"github.com/kubeedge/kubeedge/pkg/apis/reliablesyncs/v1alpha1"
	reliableclient "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/pkg/client/clientset/versioned/fake"
	mockcon "github.com/kubeedge/viaduct/pkg/conn/testing"
)

func TestNodeSessionKeepAliveCheck(t *testing.T) {
	tests := []struct {
		// Name of the test, for logging
		name                  string
		SendKeepaliveInterval time.Duration
		SimulateNormal        bool
	}{
		{
			name:                  "Send keepalive normally",
			SendKeepaliveInterval: tf.NormalSendKeepaliveInterval,
			SimulateNormal:        true,
		},
		{
			name:                  "Send keepalive exception",
			SendKeepaliveInterval: tf.AbnormalSendKeepaliveInterval,
			SimulateNormal:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize the node session
			client := &fake.Clientset{}

			wg := sync.WaitGroup{}
			stopCh := make(chan struct{})

			session, mockController, mockConn := openNodeSession(t, &wg, stopCh, tt.SendKeepaliveInterval, client)
			defer mockController.Finish()

			mockConn.EXPECT().Close().AnyTimes()

			if tt.SimulateNormal {
				after := time.After(5 * time.Second)
				checkTicker := time.NewTicker(tt.SendKeepaliveInterval)

			TEST:
				for {
					select {
					case <-after:
						close(stopCh)
						wg.Wait()
						break TEST
					case <-checkTicker.C:
						if session.GetTerminateErr() != NoErr {
							t.Errorf("Expected %d got %d", NoErr, session.terminateErr)
						}
					}
				}
			} else {
				time.Sleep(tt.SendKeepaliveInterval)

				close(stopCh)
				wg.Wait()

				if session.GetTerminateErr() != TransportErr {
					t.Errorf("Expected %d got %d", TransportErr, session.terminateErr)
				}
			}
		})
	}
}

func TestNodeSessionSendNoAckMessage(t *testing.T) {
	tests := []struct {
		// Name of the test, for logging
		name            string
		InitialMessages []*beehivemodel.Message
		SimulateNormal  bool
	}{
		{
			name:           "Send no ack message normally",
			SimulateNormal: true,
			InitialMessages: []*beehivemodel.Message{
				tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "response"),
			},
		},
		{
			name:           "Send no ack message exception",
			SimulateNormal: false,
			InitialMessages: []*beehivemodel.Message{
				tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "response"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize the node session
			client := &fake.Clientset{}

			wg := sync.WaitGroup{}
			stopCh := make(chan struct{})

			session, mockController, mockConn := openNodeSession(t, &wg, stopCh, tf.NormalSendKeepaliveInterval, client)
			defer mockController.Finish()

			if tt.SimulateNormal {
				mockConn.EXPECT().WriteMessageAsync(gomock.Any()).Return(nil).Times(1)
			} else {
				mockConn.EXPECT().WriteMessageAsync(gomock.Any()).Return(errors.New("write err")).Times(1)
			}

			mockConn.EXPECT().Close().Return(nil).AnyTimes()

			go func() {
				for _, message := range tt.InitialMessages {
					enqueueNoAckMessage(session.nodeMessagePool, message)
				}
			}()

			// sleep 2 second to wait the message send
			time.Sleep(2 * time.Second)

			close(stopCh)
			session.Terminating()
			wg.Wait()

			if !tt.SimulateNormal {
				gotErr := session.GetTerminateErr()
				if gotErr != TransportErr {
					t.Errorf("Test %q: expected %d, got %d", tt.name, TransportErr, gotErr)
				}
			}
		})
	}
}

// Test the real nodeSession SendAckMessage methods with a fake API server
// and a fake connection. we call func `enqueueAckMessage` to simulate
// resource update message that comes from edgeController module.
func TestNodeSessionSendAckMessage(t *testing.T) {
	tests := []tf.TestCase{
		{
			Name:               "create single pod normally",
			InitialObjectSyncs: tf.NoObjectSyncs,
			ReactorErrors:      tf.NoErrors,
			InitialMessages: []*beehivemodel.Message{
				tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "update"),
			},
			ExpectedObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			},
			SimulateMessageFunc: normalSimulateMessageFunc,
		},
		{
			Name:               "create multi resource normally",
			InitialObjectSyncs: tf.NoObjectSyncs,
			ReactorErrors:      tf.NoErrors,
			InitialMessages: []*beehivemodel.Message{
				tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "update"),
				tf.NewConfigMapMessage(tf.NewTestConfigMapResource(tf.TestConfigMapName, tf.TestConfigMapUID, "2"), "update"),
			},
			ExpectedObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
				tf.NewObjectSync(tf.NewTestConfigMapResource(tf.TestConfigMapName, tf.TestConfigMapUID, "2"), "ConfigMap"),
			},
			SimulateMessageFunc: normalSimulateMessageFunc,
		},
		{
			Name: "Update running pods normally",
			InitialObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			},
			ReactorErrors: tf.NoErrors,
			InitialMessages: []*beehivemodel.Message{
				tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "update"),
			},
			ExpectedObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			},
			SimulateMessageFunc: normalSimulateMessageFunc,
		},
		{
			Name: "Update multi resource normally",
			InitialObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
				tf.NewObjectSync(tf.NewTestConfigMapResource(tf.TestConfigMapName, tf.TestConfigMapUID, "2"), "ConfigMap"),
			},
			ReactorErrors: tf.NoErrors,
			InitialMessages: []*beehivemodel.Message{
				tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "3"), "update"),
				tf.NewConfigMapMessage(tf.NewTestConfigMapResource(tf.TestConfigMapName, tf.TestConfigMapUID, "4"), "update"),
			},
			ExpectedObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "3"), "Pod"),
				tf.NewObjectSync(tf.NewTestConfigMapResource(tf.TestConfigMapName, tf.TestConfigMapUID, "4"), "ConfigMap"),
			},
			SimulateMessageFunc: normalSimulateMessageFunc,
		},
		{
			Name: "delete pod normally",
			InitialObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			},
			ReactorErrors: tf.NoErrors,
			InitialMessages: []*beehivemodel.Message{
				tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "delete"),
			},
			ExpectedObjectSyncs: tf.NoObjectSyncs,
			SimulateMessageFunc: normalSimulateMessageFunc,
		},
		{
			Name: "delete multi resource normally",
			InitialObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
				tf.NewObjectSync(tf.NewTestConfigMapResource(tf.TestConfigMapName, tf.TestConfigMapUID, "2"), "ConfigMap"),
			},
			ReactorErrors: tf.NoErrors,
			InitialMessages: []*beehivemodel.Message{
				tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "3"), "delete"),
				tf.NewConfigMapMessage(tf.NewTestConfigMapResource(tf.TestConfigMapName, tf.TestConfigMapUID, "4"), "delete"),
			},
			ExpectedObjectSyncs: tf.NoObjectSyncs,
			SimulateMessageFunc: normalSimulateMessageFunc,
		},
		{
			Name: "update running pod but update objectSync fail, retry success",
			InitialObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			},
			ReactorErrors: []tf.ReactorError{{Verb: "update", Resource: "objectsyncs", Error: errors.New("update err")}},
			InitialMessages: []*beehivemodel.Message{
				tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "update"),
				// The second message simulate that comes from syncController.
				tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "update"),
			},
			ExpectedObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			},
			SimulateMessageFunc: func(pool *common.NodeMessagePool, messages []*beehivemodel.Message) {
				for _, message := range messages {
					enqueueAckMessage(pool, message)
					time.Sleep(1 * time.Second)
				}
			},
		},
		{
			// if node connection is lost and the node will reconnected, syncController will retry send the message
			Name: "Update running pods, but the node connection is lost",
			InitialObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			},
			ReactorErrors: tf.NoErrors,
			InitialMessages: []*beehivemodel.Message{
				tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "update"),
			},
			ExpectedObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			},
			SimulateMessageFunc:  normalSimulateMessageFunc,
			InjectedConnErrTimes: 1,
		},
	}

	executeTest := func(t *testing.T, test tf.TestCase) {
		// Initialize the node session
		client := &fake.Clientset{}

		wg := sync.WaitGroup{}
		stopCh := make(chan struct{})

		session, mockController, mockConn := openNodeSession(t, &wg, stopCh, tf.NormalSendKeepaliveInterval, client)
		defer mockController.Finish()

		// set mock expected
		mockConn.EXPECT().Close().AnyTimes()

		if test.InjectedConnErrTimes > 0 {
			mockConn.EXPECT().WriteMessageAsync(gomock.Any()).
				Return(errors.New("write err")).Times(test.InjectedConnErrTimes)
		}

		mockConn.EXPECT().WriteMessageAsync(gomock.Any()).DoAndReturn(func(msg *beehivemodel.Message) error {
			session.ReceiveMessageAck(msg.GetID())
			return nil
		}).AnyTimes()

		reactor := tf.NewObjectSyncReactor(client, test.ReactorErrors)
		reactor.AddObjectSyncs(test.InitialObjectSyncs)

		wg.Add(1)
		// Simulate downstream message from edge controller
		go func() {
			defer wg.Done()
			test.SimulateMessageFunc(session.nodeMessagePool, test.InitialMessages)
		}()

		// sleep 2 second to wait the message send
		time.Sleep(2 * time.Second)

		evaluateTestResults(reactor, test, t)

		close(stopCh)
		session.Terminating()
		wg.Wait()
	}

	for _, test := range tests {
		test := test
		t.Run(test.Name, func(t *testing.T) {
			executeTest(t, test)
		})
	}
}

func normalSimulateMessageFunc(pool *common.NodeMessagePool, messages []*beehivemodel.Message) {
	for _, message := range messages {
		enqueueAckMessage(pool, message)
	}
}

func openNodeSession(t *testing.T, wg *sync.WaitGroup, stopCh chan struct{}, sendKeepaliveInterval time.Duration,
	reliableClient reliableclient.Interface) (*NodeSession, *gomock.Controller, *mockcon.MockConnection) {
	nmp := common.InitNodeMessagePool(tf.TestNodeID)

	mockController := gomock.NewController(t)
	mockConn := mockcon.NewMockConnection(mockController)

	session := NewNodeSession(tf.TestNodeID, tf.TestProjectID, mockConn, tf.KeepaliveInterval, nmp, reliableClient)

	wg.Add(1)

	go func() {
		defer wg.Done()
		session.Start()
	}()

	wg.Add(1)

	// Simulate send keepalive message
	go func(stopCh chan struct{}) {
		defer wg.Done()
		ticker := time.NewTicker(sendKeepaliveInterval)
		for {
			select {
			case <-ticker.C:
				session.KeepAliveMessage()
			case <-stopCh:
				return
			}
		}
	}(stopCh)

	return session, mockController, mockConn
}

func enqueueAckMessage(nodeMessagePool *common.NodeMessagePool, msg *beehivemodel.Message) {
	messageKey, err := common.AckMessageKeyFunc(msg)
	if err != nil {
		klog.Errorf("fail to get key for message: %s", msg.String())
		return
	}

	if err := nodeMessagePool.AckMessageStore.Add(msg); err != nil {
		klog.Errorf("fail to add message %v nodeStore, err: %v", msg, err)
		return
	}

	nodeMessagePool.AckMessageQueue.Add(messageKey)
}

func enqueueNoAckMessage(nodeMessagePool *common.NodeMessagePool, msg *beehivemodel.Message) {
	messageKey, _ := common.NoAckMessageKeyFunc(msg)
	if err := nodeMessagePool.NoAckMessageStore.Add(msg); err != nil {
		klog.Errorf("failed to add msg: %v", err)
		return
	}
	nodeMessagePool.NoAckMessageQueue.Add(messageKey)
}

func evaluateTestResults(reactor *tf.ObjectSyncReactor, test tf.TestCase, t *testing.T) {
	// Evaluate results
	if err := reactor.CheckObjectSyncs(test.ExpectedObjectSyncs); err != nil {
		t.Errorf("Test %q: %v", test.Name, err)
	}
}
