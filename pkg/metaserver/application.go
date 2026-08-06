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
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
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

	// The following field defines the Application response result
	// mu protects Status, Reason, Error, RespBody, ctx, and cancel
	// from concurrent reads and writes across goroutines.
	mu       *sync.RWMutex
	RespBody []byte
	Status   ApplicationStatus
	Reason   string // why in this status
	Error    apierrors.StatusError

	ctx    context.Context // to end app.Wait
	cancel context.CancelFunc

	// count the number of current citations
	count     uint64
	countLock *sync.Mutex
	// Timestamp record the last closing time of application, only make sense when count == 0
	Timestamp time.Time
}

func (a *Application) lock() {
	if a.mu == nil {
		a.mu = &sync.RWMutex{}
	}
	a.mu.Lock()
}

func (a *Application) unlock() {
	if a.mu != nil {
		a.mu.Unlock()
	}
}

func (a *Application) rLock() {
	if a.mu == nil {
		a.mu = &sync.RWMutex{}
	}
	a.mu.RLock()
}

func (a *Application) rUnlock() {
	if a.mu != nil {
		a.mu.RUnlock()
	}
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
	ctx2, cancel := context.WithCancel(ctx)
	app := &Application{
		Key:         key,
		Verb:        verb,
		Nodename:    nodename,
		Subresource: subresource,
		Status:      PreApplying,
		Option:      ToBytes(option),
		ReqBody:     ToBytes(reqBody),
		mu:          &sync.RWMutex{},
		ctx:         ctx2,
		cancel:      cancel,
		count:       0,
		countLock:   &sync.Mutex{},
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
	a.ID = fmt.Sprintf("%x", sha256.Sum256(b))
	return a.ID
}

func (a *Application) String() string {
	a.rLock()
	defer a.rUnlock()
	return fmt.Sprintf("(NodeName=%v;Key=%v;Verb=%v;Status=%v;Reason=%v)", a.Nodename, a.Key, a.Verb, a.Status, a.Reason)
}

func (a *Application) ReqContent() interface{} {
	return a.ReqBody
}

func (a *Application) RespContent() interface{} {
	a.rLock()
	defer a.rUnlock()
	return a.RespBody
}

// OptionTo convert application option. Remember `i` must be a pointer to the initialized variable
func (a *Application) OptionTo(i interface{}) error {
	err := json.Unmarshal(a.Option, i)
	if err != nil {
		return fmt.Errorf("failed to parse Option bytes, %v", err)
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
	a.rLock()
	body := a.RespBody
	a.rUnlock()
	err := json.Unmarshal(body, i)
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
	a.lock()
	defer a.unlock()
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *Application) GetStatus() ApplicationStatus {
	a.rLock()
	defer a.rUnlock()
	return a.Status
}

// SetStatus atomically updates the application status under the write lock.
func (a *Application) SetStatus(s ApplicationStatus) {
	a.lock()
	defer a.unlock()
	a.Status = s
}

// SetReason atomically updates the application Reason field under the write lock.
func (a *Application) SetReason(reason string) {
	a.lock()
	defer a.unlock()
	a.Reason = reason
}

// GetReason atomically reads the application Reason field.
func (a *Application) GetReason() string {
	a.rLock()
	defer a.rUnlock()
	return a.Reason
}

// GetError atomically reads the application Error field.
func (a *Application) GetError() apierrors.StatusError {
	a.rLock()
	defer a.rUnlock()
	return a.Error
}

// UpdateFromResponse atomically applies all response fields from a cloud reply.
func (a *Application) UpdateFromResponse(resp *Application) {
	a.lock()
	defer a.unlock()
	a.Status = resp.Status
	a.Reason = resp.Reason
	a.Error = resp.Error
	a.RespBody = resp.RespBody
}

// Wait the result of application after it is applied by application agent
func (a *Application) Wait() {
	a.rLock()
	ctx := a.ctx
	a.rUnlock()
	if ctx != nil {
		<-ctx.Done()
	}
}

// Reset prepares the application for reuse after it has reached Completed status.
// It cancels the old context, creates a new one, clears transient response fields,
// and transitions Status to PreApplying so that concurrent Apply calls do not
// mistake the in-flight re-apply as already finished.
func (a *Application) Reset() {
	a.lock()
	defer a.unlock()
	if a.ctx != nil && a.cancel != nil {
		a.cancel()
	}
	a.ctx, a.cancel = context.WithCancel(beehiveContext.GetContext())
	a.Reason = ""
	a.RespBody = []byte{}
	// Transition back to PreApplying so concurrent Apply calls wait
	// rather than seeing Completed and launching duplicate doApply goroutines.
	a.Status = PreApplying
}

func (a *Application) Add() {
	if a.countLock == nil {
		a.countLock = &sync.Mutex{}
	}
	a.countLock.Lock()
	a.count++
	a.countLock.Unlock()
}

func (a *Application) getCount() uint64 {
	if a.countLock == nil {
		a.countLock = &sync.Mutex{}
	}
	a.countLock.Lock()
	c := a.count
	a.countLock.Unlock()
	return c
}

// Close must be called when applicant no longer using application
func (a *Application) Close() {
	if a.countLock == nil {
		a.countLock = &sync.Mutex{}
	}
	a.countLock.Lock()
	defer a.countLock.Unlock()
	if a.count == 0 {
		return
	}

	a.Timestamp = time.Now()
	a.count--
	if a.count == 0 {
		a.lock()
		a.Status = Completed
		a.unlock()
	}
}

func (a *Application) LastCloseTime() time.Time {
	if a.countLock == nil {
		a.countLock = &sync.Mutex{}
	}
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

	nodeID, err := messagelayer.GetNodeID(msg)
	if err != nil {
		nodeID = app.Nodename
	}
	app.Nodename = nodeID
	app.mu = &sync.RWMutex{}
	app.countLock = &sync.Mutex{}
	return app, nil
}

// MsgToApplications extract applications in message's Content
func MsgToApplications(msg model.Message) (map[string]Application, error) {
	contentData, err := msg.GetContentData()
	if err != nil {
		return nil, err
	}

	applications := make(map[string]Application)

	err = json.Unmarshal(contentData, &applications)
	if err != nil {
		return nil, err
	}
	return applications, nil
}
