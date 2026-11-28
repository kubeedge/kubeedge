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

package dispatcher

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/api/apis/reliablesyncs/v1alpha1"
	"github.com/kubeedge/api/client/clientset/versioned/fake"
	synclisters "github.com/kubeedge/api/client/listers/reliablesyncs/v1alpha1"
	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common"
	tf "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/testing"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/session"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	mockcon "github.com/kubeedge/kubeedge/pkg/viaduct/pkg/conn/testing"
)

// policycontroller sends DELETE for an object to a node, then an UPDATE that adds
// it back. The UPDATE, or a newer sync loop resend that replaces it, must reach the node
// whether it arrives before or after the DELETE's ACK.
func TestUpdateAfterDeleteReachesEdge(t *testing.T) {
	tests := []struct {
		name            string
		updateBeforeAck bool
		syncLoopResend  bool
		wantLast        string
	}{
		{"update arrives while the delete waits for its ack", true, false, "update/6"},
		{"sync loop resend replaces the update while the delete waits for its ack", true, true, "update/7"},
		{"update arrives after the delete was acked", false, false, "update/6"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deleteMsg := tf.FromPolicyController(tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "4"), "delete"))
			updateMsg := tf.FromPolicyController(tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "6"), "update"))
			syncLoopMsg := tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "7"), "update")
			syncLoopMsg.Router.Source = modules.EdgeControllerModuleName

			client := &fake.Clientset{}
			initial := []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			}
			reactor := tf.NewObjectSyncReactor(client, tf.NoErrors)
			reactor.AddObjectSyncs(initial)
			objectSyncIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, objectSync := range initial {
				_ = objectSyncIndexer.Add(objectSync)
			}

			nmp := common.InitNodeMessagePool(tf.TestNodeID)
			mockController := gomock.NewController(t)
			defer mockController.Finish()
			mockConn := mockcon.NewMockConnection(mockController)
			mockConn.EXPECT().Close().AnyTimes()
			nodeSession := session.NewNodeSession(tf.TestNodeID, tf.TestProjectID, mockConn, tf.KeepaliveInterval, nmp, client)
			manager := session.NewSessionManager(10)
			manager.AddSession(nodeSession)

			dispatcher := &messageDispatcher{
				reliableClient:          client,
				SessionManager:          manager,
				objectSyncLister:        synclisters.NewObjectSyncLister(objectSyncIndexer),
				clusterObjectSyncLister: synclisters.NewClusterObjectSyncLister(cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})),
			}
			dispatcher.AddNodeMessagePool(tf.TestNodeID, nmp)

			var lock sync.Mutex
			var sent []string
			mockConn.EXPECT().WriteMessageAsync(gomock.Any()).DoAndReturn(func(msg *beehivemodel.Message) error {
				lock.Lock()
				sent = append(sent, msg.GetOperation()+"/"+msg.GetResourceVersion())
				lock.Unlock()
				if test.updateBeforeAck && msg.GetID() == deleteMsg.GetID() {
					dispatcher.enqueueAckMessage(tf.TestNodeID, updateMsg)
					if test.syncLoopResend {
						dispatcher.enqueueAckMessage(tf.TestNodeID, syncLoopMsg)
					}
				}
				nodeSession.ReceiveMessageAck(msg.GetID())
				return nil
			}).AnyTimes()

			wg := sync.WaitGroup{}
			stopCh := make(chan struct{})
			wg.Add(2)
			go func() {
				defer wg.Done()
				nodeSession.Start()
			}()
			go func() {
				defer wg.Done()
				ticker := time.NewTicker(tf.NormalSendKeepaliveInterval)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						nodeSession.KeepAliveMessage()
					case <-stopCh:
						return
					}
				}
			}()

			lastSent := func() string {
				lock.Lock()
				defer lock.Unlock()
				if len(sent) == 0 {
					return ""
				}
				return sent[len(sent)-1]
			}
			// Poll rather than sleep, so a slow runner does not fail the test.
			waitFor := func(what string, done func() bool) {
				err := wait.PollUntilContextTimeout(context.Background(), 20*time.Millisecond, 10*time.Second, true,
					func(context.Context) (bool, error) { return done(), nil })
				if err != nil {
					lock.Lock()
					defer lock.Unlock()
					t.Fatalf("%s: sent %v", what, sent)
				}
			}

			dispatcher.enqueueAckMessage(tf.TestNodeID, deleteMsg)
			if !test.updateBeforeAck {
				waitFor("delete not acknowledged", func() bool {
					return lastSent() == "delete/4" && len(nmp.AckMessageStore.List()) == 0
				})
				// The ACK deleted the ObjectSync; let the lister see that, as the informer would.
				for _, objectSync := range initial {
					_ = objectSyncIndexer.Delete(objectSync)
				}
				dispatcher.enqueueAckMessage(tf.TestNodeID, updateMsg)
			}
			waitFor(test.wantLast+" not sent last", func() bool { return lastSent() == test.wantLast })

			close(stopCh)
			nodeSession.Terminating()
			wg.Wait()
		})
	}
}
