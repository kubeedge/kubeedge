/*
Copyright 2021 The KubeEdge Authors.

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

package metaserver

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	commontypes "github.com/kubeedge/kubeedge/common/types"
)

// Application record the resources that are in applying for requesting to be transferred down from the cloud, please:
// 0.use Agent.Generate to generate application
// 1.use Agent.Apply to apply application( generate msg and send it to cloud dynamiccontroller)
type Application struct {
	// ID is the SHA256 checksum generated from request information
	ID string

	// The following field defines the Application request information
	// Key format: group version resource namespaces name
	Key         string
	Verb        ApplicationVerb
	Nodename    string
	Option      []byte
	ReqBody     []byte
	Subresource string
	Token       string

	// The following field defines the Application response result
	RespBody []byte
	Status   ApplicationStatus
	Reason   string // why in this status
	Error    apierrors.StatusError

	ctx    context.Context // to end app.Wait
	cancel context.CancelFunc

	// count the number of current citations
	count     uint64
	countLock sync.Mutex
	// Timestamp record the last closing time of application, only make sense when count == 0
	Timestamp time.Time
}

func NewApplication(ctx context.Context, key string, verb ApplicationVerb, nodename, subresource string, option interface{}, reqBody interface{}) (*Application, error) {
	var v1 metav1.ListOptions
	if internal, ok := option.(metainternalversion.ListOptions); ok {
		err := metainternalversion.Convert_internalversion_ListOptions_To_v1_ListOptions(&internal, &v1, nil)
		if err != nil {
			// error here won't happen, log in case
			klog.Errorf("failed to transfer internalListOption to v1ListOption, force set to empty")
		}
		option = v1
	}
	token, ok := ctx.Value(commontypes.AuthorizationKey).(string)
	if !ok {
		klog.Errorf("unsupported Token type :%T", ctx.Value(commontypes.AuthorizationKey))
		return nil, fmt.Errorf("unsupported Token type :%T", ctx.Value(commontypes.AuthorizationKey))
	}
	ctx2, cancel := context.WithCancel(ctx)
	app := &Application{
		Key:         key,
		Verb:        verb,
		Nodename:    nodename,
		Subresource: subresource,
		Status:      PreApplying,
		Option:      ToBytes(option),
		ReqBody:     ToBytes(reqBody),
		Token:       token,
		ctx:         ctx2,
		cancel:      cancel,
		count:       0,
		countLock:   sync.Mutex{},
		Timestamp:   time.Time{},
	}
	app.Add()
	return app, nil
}

func (a *Application) Identifier() string {
	if a.ID != "" {
		return a.ID
	}
	b := []byte(a.Nodename)
	b = append(b, []byte(a.Key)...)
	b = append(b, []byte(a.Verb)...)
	b = append(b, a.Option...)
	b = append(b, a.ReqBody...)
	b = append(b, []byte(a.Subresource)...)
	b = append(b, []byte(a.Token)...)
	a.ID = fmt.Sprintf("%x", sha256.Sum256(b))
	return a.ID
}

func (a *Application) String() string {
	return fmt.Sprintf("(NodeName=%v;Key=%v;Verb=%v;Status=%v;Reason=%v)", a.Nodename, a.Key, a.Verb, a.Status, a.Reason)
}

func (a *Application) ReqContent() interface{} {
	return a.ReqBody
}

func (a *Application) RespContent() interface{} {
	return a.RespBody
}

// OptionTo convert application option. Remember `i` must be a pointer to the initialized variable
func (a *Application) OptionTo(i interface{}) error {
	err := json.Unmarshal(a.Option, i)
	if err != nil {
		return fmt.Errorf("failed to prase Option bytes, %v", err)
	}
	return nil
}

func (a *Application) ReqBodyTo(i interface{}) error {
	err := json.Unmarshal(a.ReqBody, i)
	if err != nil {
		return fmt.Errorf("failed to parse ReqBody bytes, %v", err)
	}
	return nil
}

func (a *Application) RespBodyTo(i interface{}) error {
	err := json.Unmarshal(a.RespBody, i)
	if err != nil {
		return fmt.Errorf("failed to parse RespBody bytes, %v", err)
	}
	return nil
}

func (a *Application) GVR() schema.GroupVersionResource {
	gvr, _, _ := ParseKey(a.Key)
	return gvr
}

func (a *Application) Namespace() string {
	_, ns, _ := ParseKey(a.Key)
	return ns
}

func (a *Application) Cancel() {
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *Application) GetStatus() ApplicationStatus {
	return a.Status
}

// Wait the result of application after it is applied by application agent
func (a *Application) Wait() {
	if a.ctx != nil {
		<-a.ctx.Done()
	}
}

func (a *Application) Reset() {
	if a.ctx != nil && a.cancel != nil {
		a.cancel()
	}
	a.ctx, a.cancel = context.WithCancel(beehiveContext.GetContext())
	a.Reason = ""
	a.RespBody = []byte{}
}

func (a *Application) Add() {
	a.countLock.Lock()
	a.count++
	a.countLock.Unlock()
}

func (a *Application) getCount() uint64 {
	a.countLock.Lock()
	c := a.count
	a.countLock.Unlock()
	return c
}

// Close must be called when applicant no longer using application
func (a *Application) Close() {
	a.countLock.Lock()
	defer a.countLock.Unlock()
	if a.count == 0 {
		return
	}

	a.Timestamp = time.Now()
	a.count--
	if a.count == 0 {
		a.Status = Completed
	}
}

func (a *Application) LastCloseTime() time.Time {
	a.countLock.Lock()
	defer a.countLock.Unlock()
	if a.count == 0 && !a.Timestamp.IsZero() {
		return a.Timestamp
	}
	return time.Time{}
}

func ToBytes(i interface{}) (bytes []byte) {
	if i == nil {
		return
	}

	if bytes, ok := i.([]byte); ok {
		return bytes
	}

	var err error
	if bytes, err = json.Marshal(i); err != nil {
		klog.Errorf("marshal content to []byte failed, err: %v", err)
	}
	return
}

// extract application in message's Content
func MsgToApplication(msg model.Message) (*Application, error) {
	var app = new(Application)
	contentData, err := msg.GetContentData()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(contentData, app)
	if err != nil {
		return nil, err
	}
	return app, nil
}
