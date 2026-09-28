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

package dtmanager

import (
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/mocks"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

var (
	originalMembershipServiceFactory = MembershipServiceFactory
)

// newMockFactory returns a MembershipServiceFactory replacement backed by mockService.
func newMockFactory(mockService *mocks.MockDeviceService) func() interface {
	AddDeviceTrans(adds []models.Device, addAttrs []models.DeviceAttr, addTwins []models.DeviceTwin) error
	DeleteDeviceTrans(deletes []string) error
	QueryDevice(key string, condition string) ([]models.Device, error)
	QueryDeviceAttr(key, condition string) (*[]models.DeviceAttr, error)
	QueryDeviceTwin(key, condition string) (*[]models.DeviceTwin, error)
} {
	return func() interface {
		AddDeviceTrans(adds []models.Device, addAttrs []models.DeviceAttr, addTwins []models.DeviceTwin) error
		DeleteDeviceTrans(deletes []string) error
		QueryDevice(key string, condition string) ([]models.Device, error)
		QueryDeviceAttr(key, condition string) (*[]models.DeviceAttr, error)
		QueryDeviceTwin(key, condition string) (*[]models.DeviceTwin, error)
	} {
		return mockService
	}
}

// ---------- getRemoveList ----------

func TestGetRemoveList(t *testing.T) {
	dtc := &dtcontext.DTContext{DeviceList: &sync.Map{}}
	var device dttype.Device
	dtc.DeviceList.Store("DeviceB", &device)
	dArray := []dttype.Device{{ID: "123"}}
	value := getRemoveList(dtc, dArray)
	if len(value) != 1 {
		t.Fatalf("expected 1 item in remove list, got %d", len(value))
	}
	if value[0].ID != "DeviceB" {
		t.Errorf("expected DeviceB, got %v", value[0].ID)
	}
}

func TestGetRemoveListProperDeviceID(t *testing.T) {
	dtc := &dtcontext.DTContext{DeviceList: &sync.Map{}}
	var device dttype.Device
	dtc.DeviceList.Store("123", &device)
	dArray := []dttype.Device{{ID: "123"}}
	value := getRemoveList(dtc, dArray)
	if len(value) != 0 {
		t.Errorf("expected empty remove list, got %v", value)
	}
}

// ---------- initMemActionCallBack ----------

func TestInitMemActionCallBack(t *testing.T) {
	initMemActionCallBack()
	expectedActions := []string{dtcommon.MemGet, dtcommon.MemUpdated, dtcommon.MemDetailResult}
	for _, action := range expectedActions {
		if _, exists := memActionCallBack[action]; !exists {
			t.Errorf("expected callback for action %s not found", action)
		}
	}
}

// ---------- dealMembershipDetail ----------

func TestDealMembershipDetailInvalidEmptyMessage(t *testing.T) {
	dtc := &dtcontext.DTContext{DeviceList: &sync.Map{}, GroupID: "1"}
	err := dealMembershipDetail(dtc, "t", "invalid")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestDealMembershipDetailInvalidMsg(t *testing.T) {
	dtc := &dtcontext.DTContext{DeviceList: &sync.Map{}, GroupID: "1"}
	m := &model.Message{Content: "invalidmsg"}
	err := dealMembershipDetail(dtc, "t", m)
	if !reflect.DeepEqual(err, errors.New("assertion failed")) {
		t.Errorf("expected assertion failed, got %v", err)
	}
}

func TestDealMembershipDetailInvalidContent(t *testing.T) {
	dtc := &dtcontext.DTContext{DeviceList: &sync.Map{}, GroupID: "1"}
	var cnt []uint8
	cnt = append(cnt, 1)
	m := &model.Message{Content: cnt}
	err := dealMembershipDetail(dtc, "t", m)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestDealMembershipDetailValid(t *testing.T) {
	defer func() { MembershipServiceFactory = originalMembershipServiceFactory }()

	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}

	mockService := mocks.NewMockDeviceService()
	mockService.AddDeviceTransFunc = func(adds []models.Device, addAttrs []models.DeviceAttr, addTwins []models.DeviceTwin) error {
		return nil
	}
	MembershipServiceFactory = newMockFactory(mockService)

	payload := dttype.MembershipUpdate{
		AddDevices:  []dttype.Device{{ID: "DeviceA", Name: "Router", State: "unknown"}},
		BaseMessage: dttype.BaseMessage{EventID: "eventid"},
	}
	content, _ := json.Marshal(payload)
	m := &model.Message{Content: content}
	err := dealMembershipDetail(dtc, "t", m)
	if err != nil {
		t.Errorf("expected nil, got error: %v", err)
	}
}

// dealMembershipDetail with device to remove: device in context but not in cloud list
func TestDealMembershipDetailRemovesStaleDevice(t *testing.T) {
	defer func() { MembershipServiceFactory = originalMembershipServiceFactory }()

	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}
	// Pre-populate with a device that is NOT in the cloud detail payload
	dtc.DeviceList.Store("StaleDevice", &dttype.Device{ID: "StaleDevice"})
	dtc.DeviceMutex.Store("StaleDevice", &sync.Mutex{})

	mockService := mocks.NewMockDeviceService()
	mockService.DeleteDeviceTransFunc = func(deletes []string) error { return nil }
	MembershipServiceFactory = newMockFactory(mockService)

	// Cloud sends detail with different device
	payload := dttype.MembershipUpdate{
		AddDevices:  []dttype.Device{{ID: "DeviceA", Name: "Router", State: "unknown"}},
		BaseMessage: dttype.BaseMessage{EventID: "ev1"},
	}
	mockService.AddDeviceTransFunc = func(adds []models.Device, addAttrs []models.DeviceAttr, addTwins []models.DeviceTwin) error {
		return nil
	}
	content, _ := json.Marshal(payload)
	m := &model.Message{Content: content}
	err := dealMembershipDetail(dtc, "t", m)
	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
	// StaleDevice must have been removed
	if _, loaded := dtc.DeviceList.Load("StaleDevice"); loaded {
		t.Error("expected StaleDevice to be removed from DeviceList")
	}
}

// ---------- dealMembershipUpdate ----------

func TestDealMembershipUpdateEmptyMessage(t *testing.T) {
	dtc := &dtcontext.DTContext{DeviceList: &sync.Map{}, GroupID: "1"}
	err := dealMembershipUpdate(dtc, "t", "invalid")
	if !reflect.DeepEqual(err, errors.New("msg not Message type")) {
		t.Errorf("expected msg not Message type, got %v", err)
	}
}

func TestDealMembershipUpdateInvalidMsg(t *testing.T) {
	dtc := &dtcontext.DTContext{DeviceList: &sync.Map{}, GroupID: "1"}
	m := &model.Message{Content: "invalidmessage"}
	err := dealMembershipUpdate(dtc, "t", m)
	if !reflect.DeepEqual(err, errors.New("assertion failed")) {
		t.Errorf("expected assertion failed, got %v", err)
	}
}

func TestDealMembershipUpdateInvalidContent(t *testing.T) {
	dtc := &dtcontext.DTContext{DeviceList: &sync.Map{}, GroupID: "1"}
	var cnt []uint8
	cnt = append(cnt, 1)
	m := &model.Message{Content: cnt}
	err := dealMembershipUpdate(dtc, "t", m)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestDealMembershipUpdateValidAddedDevice(t *testing.T) {
	defer func() { MembershipServiceFactory = originalMembershipServiceFactory }()

	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}

	mockService := mocks.NewMockDeviceService()
	mockService.AddDeviceTransFunc = func(adds []models.Device, addAttrs []models.DeviceAttr, addTwins []models.DeviceTwin) error {
		return nil
	}
	MembershipServiceFactory = newMockFactory(mockService)

	payload := dttype.MembershipUpdate{
		AddDevices:  []dttype.Device{{ID: "DeviceA", Name: "Router", State: "unknown"}},
		BaseMessage: dttype.BaseMessage{EventID: "eventid"},
	}
	content, _ := json.Marshal(payload)
	m := &model.Message{Content: content}
	err := dealMembershipUpdate(dtc, "t", m)
	if err != nil {
		t.Errorf("expected nil, got error: %v", err)
	}
}

func TestDealMembershipUpdateValidRemovedDevice(t *testing.T) {
	defer func() { MembershipServiceFactory = originalMembershipServiceFactory }()

	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}
	// device must exist to be removed
	dtc.DeviceList.Store("DeviceA", &dttype.Device{ID: "DeviceA"})
	dtc.DeviceMutex.Store("DeviceA", &sync.Mutex{})

	mockService := mocks.NewMockDeviceService()
	mockService.DeleteDeviceTransFunc = func(deletes []string) error { return nil }
	MembershipServiceFactory = newMockFactory(mockService)

	payload := dttype.MembershipUpdate{
		RemoveDevices: []dttype.Device{{ID: "DeviceA", Name: "Router", State: "unknown"}},
		BaseMessage:   dttype.BaseMessage{EventID: "eventid"},
	}
	content, _ := json.Marshal(payload)
	m := &model.Message{Content: content}
	err := dealMembershipUpdate(dtc, "t", m)
	if err != nil {
		t.Errorf("expected nil, got error: %v", err)
	}
}

// AddDeviceTrans returns error — device must be cleaned from DeviceList, no panic.
func TestDealMembershipUpdateAddDeviceServiceError(t *testing.T) {
	defer func() { MembershipServiceFactory = originalMembershipServiceFactory }()

	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}

	mockService := mocks.NewMockDeviceService()
	mockService.AddDeviceTransFunc = func(adds []models.Device, addAttrs []models.DeviceAttr, addTwins []models.DeviceTwin) error {
		return errors.New("db write error")
	}
	MembershipServiceFactory = newMockFactory(mockService)

	payload := dttype.MembershipUpdate{
		AddDevices:  []dttype.Device{{ID: "DeviceA", Name: "Router", State: "unknown"}},
		BaseMessage: dttype.BaseMessage{EventID: "eventid"},
	}
	content, _ := json.Marshal(payload)
	m := &model.Message{Content: content}

	// Must not panic; function returns nil (error is logged, not propagated).
	err := dealMembershipUpdate(dtc, "t", m)
	if err != nil {
		t.Errorf("expected nil (error is logged), got: %v", err)
	}
	// Both DeviceList and DeviceMutex must be cleaned up on failure.
	if _, loaded := dtc.DeviceList.Load("DeviceA"); loaded {
		t.Error("expected DeviceA to be removed from DeviceList after AddDeviceTrans failure")
	}
	if _, loaded := dtc.DeviceMutex.Load("DeviceA"); loaded {
		t.Error("expected DeviceA to be removed from DeviceMutex after AddDeviceTrans failure")
	}
}

// DeleteDeviceTrans returns error — device is still removed from context.
func TestDealMembershipUpdateDeleteDeviceServiceError(t *testing.T) {
	defer func() { MembershipServiceFactory = originalMembershipServiceFactory }()

	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}
	dtc.DeviceList.Store("DeviceA", &dttype.Device{ID: "DeviceA"})
	dtc.DeviceMutex.Store("DeviceA", &sync.Mutex{})

	mockService := mocks.NewMockDeviceService()
	mockService.DeleteDeviceTransFunc = func(deletes []string) error {
		return errors.New("db delete error")
	}
	MembershipServiceFactory = newMockFactory(mockService)

	payload := dttype.MembershipUpdate{
		RemoveDevices: []dttype.Device{{ID: "DeviceA"}},
		BaseMessage:   dttype.BaseMessage{EventID: "eventid"},
	}
	content, _ := json.Marshal(payload)
	m := &model.Message{Content: content}
	err := dealMembershipUpdate(dtc, "t", m)
	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
	// Device must be removed from context even when the DB transaction fails.
	if _, loaded := dtc.DeviceList.Load("DeviceA"); loaded {
		t.Error("expected DeviceA to be removed from DeviceList even on DB error")
	}
	if _, loaded := dtc.DeviceMutex.Load("DeviceA"); loaded {
		t.Error("expected DeviceA to be removed from DeviceMutex even on DB error")
	}
}

// ---------- dealMembershipGet ----------

func TestDealMembershipGetEmptyMsg(t *testing.T) {
	dtc := &dtcontext.DTContext{DeviceList: &sync.Map{}, GroupID: "1"}
	err := dealMembershipGet(dtc, "t", "invalid")
	if !reflect.DeepEqual(err, errors.New("msg not Message type")) {
		t.Errorf("expected msg not Message type, got %v", err)
	}
}

func TestDealMembershipGetInvalidMsg(t *testing.T) {
	dtc := &dtcontext.DTContext{DeviceList: &sync.Map{}, GroupID: "1"}
	m := &model.Message{Content: "hello"}
	err := dealMembershipGet(dtc, "t", m)
	if !reflect.DeepEqual(err, errors.New("assertion failed")) {
		t.Errorf("expected assertion failed, got %v", err)
	}
}

func TestDealMembershipGetValid(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}
	payload := dttype.MembershipUpdate{
		BaseMessage: dttype.BaseMessage{EventID: "eventid"},
	}
	content, _ := json.Marshal(payload)
	m := &model.Message{Content: content}
	err := dealMembershipGet(dtc, "t", m)
	if !reflect.DeepEqual(err, errors.New("Not found chan to communicate")) {
		t.Errorf("expected Not found chan to communicate, got: %v", err)
	}
}

// dealMembershipGet with known device in DeviceList — response contains that device.
func TestDealMembershipGetKnownDevice(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}
	dtc.DeviceList.Store("DeviceA", &dttype.Device{ID: "DeviceA", Name: "Router"})

	payload := dttype.MembershipUpdate{BaseMessage: dttype.BaseMessage{EventID: "ev1"}}
	content, _ := json.Marshal(payload)
	m := &model.Message{Content: content}
	// No CommChan wired — expect the well-known channel error, not a logic error.
	err := dealMembershipGet(dtc, "t", m)
	if !reflect.DeepEqual(err, errors.New("Not found chan to communicate")) {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------- dealMembershipGetInner ----------

func TestDealMembershipGetInnerValid(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}
	payload := dttype.MembershipUpdate{BaseMessage: dttype.BaseMessage{EventID: "eventid"}}
	content, _ := json.Marshal(payload)
	err := dealMembershipGetInner(dtc, content)
	if !reflect.DeepEqual(err, errors.New("Not found chan to communicate")) {
		t.Errorf("expected Not found chan to communicate, got: %v", err)
	}
}

func TestDealMembershipGetInnerInValid(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}
	err := dealMembershipGetInner(dtc, []byte("invalid"))
	if !reflect.DeepEqual(err, errors.New("Not found chan to communicate")) {
		t.Errorf("expected Not found chan to communicate, got: %v", err)
	}
}

// ---------- SyncDeviceFromSqlite ----------

func TestSyncDeviceFromSqliteQueryDeviceError(t *testing.T) {
	defer func() { MembershipServiceFactory = originalMembershipServiceFactory }()

	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
	}

	mockService := mocks.NewMockDeviceService()
	mockService.QueryDeviceFunc = func(key, condition string) ([]models.Device, error) {
		return nil, errors.New("db error")
	}
	MembershipServiceFactory = newMockFactory(mockService)

	err := SyncDeviceFromSqlite(dtc, "device1")
	if err == nil || err.Error() != "db error" {
		t.Errorf("expected db error, got: %v", err)
	}
}

func TestSyncDeviceFromSqliteDeviceNotFound(t *testing.T) {
	defer func() { MembershipServiceFactory = originalMembershipServiceFactory }()

	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
	}

	mockService := mocks.NewMockDeviceService()
	mockService.QueryDeviceFunc = func(key, condition string) ([]models.Device, error) {
		return []models.Device{}, nil
	}
	MembershipServiceFactory = newMockFactory(mockService)

	err := SyncDeviceFromSqlite(dtc, "device1")
	if err == nil || err.Error() != "not found device" {
		t.Errorf("expected not found device, got: %v", err)
	}
}

func TestSyncDeviceFromSqliteQueryAttrError(t *testing.T) {
	defer func() { MembershipServiceFactory = originalMembershipServiceFactory }()

	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
	}

	mockService := mocks.NewMockDeviceService()
	mockService.QueryDeviceFunc = func(key, condition string) ([]models.Device, error) {
		return []models.Device{{ID: "device1", Name: "test"}}, nil
	}
	mockService.QueryDeviceAttrFunc = func(key, condition string) (*[]models.DeviceAttr, error) {
		return nil, errors.New("attr error")
	}
	MembershipServiceFactory = newMockFactory(mockService)

	err := SyncDeviceFromSqlite(dtc, "device1")
	if err == nil || err.Error() != "attr error" {
		t.Errorf("expected attr error, got: %v", err)
	}
}

func TestSyncDeviceFromSqliteQueryTwinError(t *testing.T) {
	defer func() { MembershipServiceFactory = originalMembershipServiceFactory }()

	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
	}

	mockService := mocks.NewMockDeviceService()
	mockService.QueryDeviceFunc = func(key, condition string) ([]models.Device, error) {
		return []models.Device{{ID: "device1", Name: "test"}}, nil
	}
	emptyAttrs := []models.DeviceAttr{}
	mockService.QueryDeviceAttrFunc = func(key, condition string) (*[]models.DeviceAttr, error) {
		return &emptyAttrs, nil
	}
	mockService.QueryDeviceTwinFunc = func(key, condition string) (*[]models.DeviceTwin, error) {
		return nil, errors.New("twin error")
	}
	MembershipServiceFactory = newMockFactory(mockService)

	err := SyncDeviceFromSqlite(dtc, "device1")
	if err == nil || err.Error() != "twin error" {
		t.Errorf("expected twin error, got: %v", err)
	}
}

func TestSyncDeviceFromSqliteSuccess(t *testing.T) {
	defer func() { MembershipServiceFactory = originalMembershipServiceFactory }()

	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
	}

	mockService := mocks.NewMockDeviceService()
	mockService.QueryDeviceFunc = func(key, condition string) ([]models.Device, error) {
		return []models.Device{{ID: "device1", Name: "Router", State: "online"}}, nil
	}
	emptyAttrs := []models.DeviceAttr{}
	mockService.QueryDeviceAttrFunc = func(key, condition string) (*[]models.DeviceAttr, error) {
		return &emptyAttrs, nil
	}
	emptyTwins := []models.DeviceTwin{}
	mockService.QueryDeviceTwinFunc = func(key, condition string) (*[]models.DeviceTwin, error) {
		return &emptyTwins, nil
	}
	MembershipServiceFactory = newMockFactory(mockService)

	err := SyncDeviceFromSqlite(dtc, "device1")
	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
	val, loaded := dtc.DeviceList.Load("device1")
	if !loaded {
		t.Fatal("expected device1 to be stored in DeviceList")
	}
	dev, ok := val.(*dttype.Device)
	if !ok {
		t.Fatal("expected *dttype.Device")
	}
	if dev.Name != "Router" {
		t.Errorf("expected Name=Router, got %s", dev.Name)
	}
}

// ---------- MemWorker.Start ----------

func TestMemWorkerStart(t *testing.T) {
	const processingDelay = 100 * time.Millisecond

	dtc, err := dtcontext.InitDTContext()
	if err != nil {
		t.Fatalf("Failed to initialize DTContext: %v", err)
	}

	tests := []struct {
		name    string
		message *dttype.DTMessage
	}{
		{
			name: "TestValidMemGet",
			message: &dttype.DTMessage{
				Action:   dtcommon.MemGet,
				Identity: "node1",
				Msg:      &model.Message{Content: []byte(`{"event_id":"1"}`)},
			},
		},
		{
			name: "TestInvalidAction",
			message: &dttype.DTMessage{
				Action:   "invalid",
				Identity: "node1",
				Msg:      &model.Message{Content: []byte(`{"event_id":"1"}`)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiverChan := make(chan interface{}, 1)
			heartBeatChan := make(chan interface{}, 1)

			worker := MemWorker{
				Worker: Worker{
					ReceiverChan:  receiverChan,
					HeartBeatChan: heartBeatChan,
					DTContexts:    dtc,
				},
				Group: "testGroup",
			}

			done := make(chan struct{})
			go func() {
				defer close(done)
				worker.Start()
			}()

			receiverChan <- tt.message
			heartBeatChan <- "ping"
			time.Sleep(processingDelay)
			close(receiverChan)
			close(heartBeatChan)
			<-done
		})
	}
}

// TestAddDeviceDeltaTrueUnlockOnError covers the delta=true Unlock path in
// addDevice() when AddDeviceTrans fails (lines 240-242 of membership.go).
// This is the specific code introduced by this PR to fix the mutex ordering bug:
// Unlock must be called before DeviceMutex.Delete so the lookup succeeds.
//
// Note: addDevice retries up to dtcommon.RetryTimes (5) times with a 1s sleep,
// so this test takes ~5s. It is run directly (not in a goroutine) so the
// MembershipServiceFactory mock stays in effect for the full duration.
func TestAddDeviceDeltaTrueUnlockOnError(t *testing.T) {
	MembershipServiceFactory = newMockFactory(mocks.NewMockDeviceService())
	mockService := mocks.NewMockDeviceService()
	mockService.AddDeviceTransFunc = func(adds []models.Device, addAttrs []models.DeviceAttr, addTwins []models.DeviceTwin) error {
		return errors.New("db write error")
	}
	MembershipServiceFactory = newMockFactory(mockService)
	defer func() { MembershipServiceFactory = originalMembershipServiceFactory }()

	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}

	devices := []dttype.Device{{ID: "DeviceA", Name: "Router", State: "unknown"}}
	baseMsg := dttype.BaseMessage{EventID: "ev1"}

	// delta=true triggers Lock before the DB call and Unlock in the error path.
	// This test covers the current cleanup behavior (Unlock before DeviceMutex.Delete).
	// The stronger concurrent in-flight-waiter lifecycle is not fully addressed here
	// and is deferred to a follow-up.
	addDevice(dtc, devices, baseMsg, true /* delta */)

	// Both DeviceList and DeviceMutex must be cleaned up.
	if _, loaded := dtc.DeviceList.Load("DeviceA"); loaded {
		t.Error("expected DeviceA removed from DeviceList after failure with delta=true")
	}
	if _, loaded := dtc.DeviceMutex.Load("DeviceA"); loaded {
		t.Error("expected DeviceA removed from DeviceMutex after failure with delta=true")
	}
}

// TestRemoveDeviceDeltaTrueUnlockPath covers the delta=true Unlock path in
// removeDevice() (lines 315-317 of membership.go), introduced by this PR's
// fix to ensure Unlock is called before DeviceMutex.Delete.
func TestRemoveDeviceDeltaTrueUnlockPath(t *testing.T) {
	mockService := mocks.NewMockDeviceService()
	mockService.DeleteDeviceTransFunc = func(deletes []string) error { return nil }
	MembershipServiceFactory = newMockFactory(mockService)
	defer func() { MembershipServiceFactory = originalMembershipServiceFactory }()

	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}
	dtc.DeviceList.Store("DeviceA", &dttype.Device{ID: "DeviceA"})
	dtc.DeviceMutex.Store("DeviceA", &sync.Mutex{})

	devices := []dttype.Device{{ID: "DeviceA"}}
	baseMsg := dttype.BaseMessage{EventID: "ev1"}

	// delta=true: Lock is acquired before the DB call, Unlock in cleanup.
	// This test covers the current cleanup behavior (Unlock before DeviceMutex.Delete).
	// The stronger concurrent in-flight-waiter lifecycle is deferred to a follow-up.
	removeDevice(dtc, devices, baseMsg, true /* delta */)

	// Device must be removed from both maps.
	if _, loaded := dtc.DeviceList.Load("DeviceA"); loaded {
		t.Error("expected DeviceA removed from DeviceList after removeDevice with delta=true")
	}
	if _, loaded := dtc.DeviceMutex.Load("DeviceA"); loaded {
		t.Error("expected DeviceA removed from DeviceMutex after removeDevice with delta=true")
	}
}

// TestMemWorkerStartHeartbeatChannelClosed verifies that MemWorker.Start
// returns cleanly when the heartbeat channel is closed (lines 65-67 of
// membership.go: the `ok == false` branch in the HeartBeatChan case).
func TestMemWorkerStartHeartbeatChannelClosed(t *testing.T) {
	dtc, err := dtcontext.InitDTContext()
	if err != nil {
		t.Fatalf("Failed to initialize DTContext: %v", err)
	}

	receiverChan := make(chan interface{}, 1)
	heartBeatChan := make(chan interface{})

	worker := MemWorker{
		Worker: Worker{
			ReceiverChan:  receiverChan,
			HeartBeatChan: heartBeatChan,
			DTContexts:    dtc,
		},
		Group: "testGroup",
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Start()
	}()

	// Closing heartBeatChan causes the `ok == false` return path to execute.
	close(heartBeatChan)
	select {
	case <-done:
		// Start() returned as expected.
	case <-time.After(2 * time.Second):
		t.Error("MemWorker.Start did not return after heartbeat channel was closed")
	}
	close(receiverChan)
}

// TestAddDeviceExistsWithDelta verifies that addDevice with delta=true logs
// an error and skips the device when it already exists in the context
// (lines 192-196 of membership.go).
func TestAddDeviceExistsWithDelta(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}
	// Pre-populate so the device already "exists".
	dtc.DeviceList.Store("DeviceA", &dttype.Device{ID: "DeviceA", Name: "Router"})
	dtc.DeviceMutex.Store("DeviceA", &sync.Mutex{})

	devices := []dttype.Device{{ID: "DeviceA", Name: "Router", State: "unknown"}}
	baseMsg := dttype.BaseMessage{EventID: "ev1"}

	// delta=true + existing device → should log error and continue without panicking.
	addDevice(dtc, devices, baseMsg, true /* delta */)

	// Device must still be present (we only skipped adding, not removed it).
	if _, loaded := dtc.DeviceList.Load("DeviceA"); !loaded {
		t.Error("expected DeviceA to remain in DeviceList when it already existed with delta=true")
	}
}

// TestRemoveDeviceNotExisted verifies that removeDevice skips devices that
// are not present in the context without panicking (lines 294-296 of
// membership.go).
func TestRemoveDeviceNotExisted(t *testing.T) {
	mockService := mocks.NewMockDeviceService()
	mockService.DeleteDeviceTransFunc = func(deletes []string) error { return nil }
	MembershipServiceFactory = newMockFactory(mockService)
	defer func() { MembershipServiceFactory = originalMembershipServiceFactory }()

	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}
	// DeviceX is NOT stored in context; removeDevice should log and skip it.
	devices := []dttype.Device{{ID: "DeviceX"}}
	baseMsg := dttype.BaseMessage{EventID: "ev1"}

	// Must not panic; the device is silently skipped.
	removeDevice(dtc, devices, baseMsg, false /* delta */)

	// Nothing should have been added by removeDevice either.
	if _, loaded := dtc.DeviceList.Load("DeviceX"); loaded {
		t.Error("DeviceX should not appear in DeviceList after a skipped removeDevice")
	}
}
