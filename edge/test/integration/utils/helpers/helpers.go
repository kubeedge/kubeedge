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

package helpers

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/common"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/edge"
)

//DeviceUpdate device update
type DeviceUpdate struct {
	State      string                     `json:"state,omitempty"`
	Attributes map[string]*dttype.MsgAttr `json:"attributes"`
}

//Device the struct of device
type Device struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	State       string `json:"state,omitempty"`
	LastOnline  string `json:"last_online,omitempty"`
}

//Attribute Structure to read data from DB (Should match with the DB-table 'device_attr' schema)
type Attribute struct {
	ID          string `json:"id,omitempty"`
	DeviceID    string `json:"deviceid,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Value       string `json:"value,omitempty"`
	Optional    bool   `json:"optional,omitempty"`
	Type        string `json:"attr_type,omitempty"`
	MetaData    string `json:"metadata,omitempty"`
}

//Twin Structure to read data from DB (Should match with the DB-table 'device_twin' schema)
type TwinAttribute struct {
	ID           string `json:"id,omitempty"`
	DeviceID     string `json:"deviceid,omitempty"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	Expected     string `json:"expected,omitempty"`
	Actual       string `json:"actual,omitempty"`
	ExpectedMeta string `json:"expected_meta,omitempty"`
	ActualMeta   string `json:"actual_meta,omitempty"`
	ExpectedVer  string `json:"expected_version,omitempty"`
	ActualVer    string `json:"actual_version,omitempty"`
	Optional     bool   `json:"optional,omitempty"`
	Type         string `json:"attr_type,omitempty"`
	MetaData     string `json:"metadata,omitempty"`
}

func GenerateDeviceID(deviceSuffix string) string {
	return deviceSuffix + edge.GetRandomString(10)
}

//Function to Generate Device
func CreateDevice(deviceID string, deviceName string, deviceState string) dttype.Device {
	device := dttype.Device{
		ID:          deviceID,
		Name:        deviceName,
		Description: "IntegrationTest",
		State:       deviceState,
		Attributes:  make(map[string]*dttype.MsgAttr),
		Twin:        make(map[string]*dttype.MsgTwin),
	}
	return device
}

//Function to add Device attribute to existing device
func AddDeviceAttribute(device dttype.Device, attributeName string, attributeValue string, attributeType string) {
	var optional = true
	var typeMeta = dttype.TypeMetadata{Type: attributeType}
	var attribute = dttype.MsgAttr{Value: attributeValue, Optional: &optional, Metadata: &typeMeta}
	device.Attributes[attributeName] = &attribute
}

//Function to add Twin attribute to existing device
func AddTwinAttribute(device dttype.Device, attributeName string, attributeValue string, attributeType string) {
	value := attributeValue
	optional := true
	valueMeta := dttype.ValueMetadata{Timestamp: time.Now().Unix()}
	typeMeta := dttype.TypeMetadata{Type: attributeType}
	twinVersion := dttype.TwinVersion{CloudVersion: 1.0, EdgeVersion: 1.0}
	twinValue := dttype.TwinValue{Value: &value, Metadata: &valueMeta}
	msgTwin := dttype.MsgTwin{Expected: &twinValue,
		Actual:          &twinValue,
		Optional:        &optional,
		Metadata:        &typeMeta,
		ExpectedVersion: &twinVersion,
		ActualVersion:   &twinVersion,
	}

	device.Twin[attributeName] = &msgTwin
}

//Function to access the edgecore DB and return the device state.
func GetDeviceStateFromDB(deviceID string) string {
	var device Device
	db, err := sql.Open("sqlite3", utils.DBFile)
	if err != nil {
		common.Fatalf("Open Sqlite DB failed : %v", err)
	}
	defer db.Close()
	row, err := db.Query("SELECT * FROM device")
	if err != nil {
		common.Fatalf("Query Sqlite DB failed: %v", err)
	}
	defer row.Close()
	for row.Next() {
		err = row.Scan(&device.ID, &device.Name, &device.Description, &device.State, &device.LastOnline)
		if err != nil {
			common.Fatalf("Failed to scan DB rows: %v", err)
		}
		if string(device.ID) == deviceID {
			break
		}
	}
	return device.State
}

func GetTwinAttributesFromDB(deviceID string, Name string) TwinAttribute {
	var twinAttribute TwinAttribute
	db, err := sql.Open("sqlite3", utils.DBFile)
	if err != nil {
		common.Fatalf("Open Sqlite DB failed : %v", err)
	}
	defer db.Close()
	row, err := db.Query("SELECT * FROM device_twin")
	defer row.Close()

	for row.Next() {
		err = row.Scan(&twinAttribute.ID,
			&twinAttribute.DeviceID,
			&twinAttribute.Name,
			&twinAttribute.Description,
			&twinAttribute.Expected,
			&twinAttribute.Actual,
			&twinAttribute.ExpectedMeta,
			&twinAttribute.ActualMeta,
			&twinAttribute.ExpectedVer,
			&twinAttribute.ActualVer,
			&twinAttribute.Optional,
			&twinAttribute.Type,
			&twinAttribute.MetaData)

		if err != nil {
			common.Fatalf("Failed to scan DB rows: %v", err)
		}
		if string(twinAttribute.DeviceID) == deviceID && twinAttribute.Name == Name {
			break
		}
	}
	return twinAttribute
}

func GetDeviceAttributesFromDB(deviceID string, Name string) Attribute {
	var attribute Attribute

	db, err := sql.Open("sqlite3", utils.DBFile)
	if err != nil {
		common.Fatalf("Open Sqlite DB failed : %v", err)
	}
	defer db.Close()
	row, err := db.Query("SELECT * FROM device_attr")
	defer row.Close()

	for row.Next() {
		err = row.Scan(&attribute.ID, &attribute.DeviceID, &attribute.Name, &attribute.Description, &attribute.Value, &attribute.Optional, &attribute.Type, &attribute.MetaData)
		if err != nil {
			common.Fatalf("Failed to scan DB rows: %v", err)
		}
		if string(attribute.DeviceID) == deviceID && attribute.Name == Name {
			break
		}
	}
	return attribute
}

// HubclientInit create mqtt client config
func HubClientInit(server, clientID, username, password string) *MQTT.ClientOptions {
	opts := MQTT.NewClientOptions().AddBroker(server).SetClientID(clientID).SetCleanSession(true)
	if username != "" {
		opts.SetUsername(username)
		if password != "" {
			opts.SetPassword(password)
		}
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	opts.SetTLSConfig(tlsConfig)
	return opts
}

//function to handle device addition and deletion.
func HandleAddAndDeleteDevice(operation, testMgrEndPoint string, device dttype.Device) bool {
	var httpMethod string
	var payload dttype.MembershipUpdate
	switch operation {
	case "PUT":
		httpMethod = http.MethodPut
		payload = dttype.MembershipUpdate{AddDevices: []dttype.Device{
			device,
		}}
	case "DELETE":
		httpMethod = http.MethodDelete
		payload = dttype.MembershipUpdate{RemoveDevices: []dttype.Device{
			device,
		}}
	default:
		common.Fatalf("operation %q is invalid", operation)
		return false
	}

	respbytes, err := json.Marshal(payload)
	if err != nil {
		common.Fatalf("Payload marshalling failed: %v", err)
		return false
	}

	req, err := http.NewRequest(httpMethod, testMgrEndPoint, bytes.NewBuffer(respbytes))
	if err != nil {
		// handle error
		common.Fatalf("Frame HTTP request failed: %v", err)
		return false
	}

	client := &http.Client{}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	t := time.Now()
	resp, err := client.Do(req)

	if err != nil {
		// handle error
		common.Fatalf("HTTP request is failed :%v", err)
		return false
	}
	common.Infof("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Since(t))
	return true
}

//HandleAddAndDeletePods is function to handle app deployment/delete deployment.
func HandleAddAndDeletePods(operation string, edgedpoint string, UID string, container []v1.Container, restartPolicy v1.RestartPolicy) bool {
	var httpMethod string
	switch operation {
	case "PUT":
		httpMethod = http.MethodPut
	case "DELETE":
		httpMethod = http.MethodDelete
	default:
		common.Fatalf("operation %q is invalid", operation)
		return false
	}

	payload := &v1.Pod{
		TypeMeta:   metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: UID, Namespace: metav1.NamespaceDefault, UID: types.UID(UID)},
		Spec:       v1.PodSpec{RestartPolicy: restartPolicy, Containers: container},
	}
	respbytes, err := json.Marshal(payload)
	if err != nil {
		common.Fatalf("Payload marshalling failed: %v", err)
		return false
	}

	req, err := http.NewRequest(httpMethod, edgedpoint, bytes.NewBuffer(respbytes))
	if err != nil {
		// handle error
		common.Fatalf("Frame HTTP request failed: %v", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	t := time.Now()
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// handle error
		common.Fatalf("HTTP request is failed :%v", err)
		return false
	}
	common.Infof("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Since(t))
	return true
}

//Function to get the pods from Edged
func GetPods(EdgedEndpoint string) (v1.PodList, error) {
	var pods v1.PodList
	var bytes io.Reader
	client := &http.Client{}
	t := time.Now()
	req, err := http.NewRequest(http.MethodGet, EdgedEndpoint, bytes)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		common.Fatalf("Frame HTTP request failed: %v", err)
		return pods, nil
	}
	resp, err := client.Do(req)
	if err != nil {
		common.Fatalf("Sending HTTP request failed: %v", err)
		return pods, nil
	}
	common.Infof("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Since(t))
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		common.Fatalf("HTTP Response reading has failed: %v", err)
		return pods, nil
	}
	err = json.Unmarshal(contents, &pods)
	if err != nil {
		common.Fatalf("Unmarshal HTTP Response has failed: %v", err)
		return pods, nil
	}
	return pods, nil
}

//CheckPodRunningState is function to check the Pod state
func CheckPodRunningState(EdgedEndPoint, podname string) {
	gomega.Eventually(func() string {
		var status string
		pods, _ := GetPods(EdgedEndPoint)
		for index := range pods.Items {
			pod := &pods.Items[index]
			if podname == pod.Name {
				status = string(pod.Status.Phase)
				common.Infof("PodName: %s PodStatus: %s", pod.Name, pod.Status.Phase)
			}
		}
		return status
	}, "240s", "2s").Should(gomega.Equal("Running"), "Application Deployment is Unsuccessful, Pod has not come to Running State")
}

//CheckPodDeletion is function to check pod deletion
func CheckPodDeletion(EdgedEndPoint, UID string) {
	gomega.Eventually(func() bool {
		var IsExist = false
		pods, _ := GetPods(EdgedEndPoint)
		if len(pods.Items) > 0 {
			for index := range pods.Items {
				pod := &pods.Items[index]
				common.Infof("PodName: %s PodStatus: %s", pod.Name, pod.Status.Phase)
				if pod.Name == UID {
					IsExist = true
				}
			}
		}
		return IsExist
	}, "30s", "4s").Should(gomega.Equal(false), "Delete Application deployment is Unsuccessful, Pod has not come to Running State")
}
