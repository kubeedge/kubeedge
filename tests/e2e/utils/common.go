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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
	edgeclientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/tests/e2e/constants"
)

const (
	Namespace             = "default"
	DeviceETPrefix        = "$hw/events/device/"
	TwinETUpdateSuffix    = "/twin/update"
	TwinETGetSuffix       = "/twin/get"
	TwinETGetResultSuffix = "/twin/get/result"

	BlueTooth         = "bluetooth"
	ModBus            = "modbus"
	Led               = "led"
	IncorrectInstance = "incorrect-instance"
	Customized        = "customized"
)

var TokenClient Token
var ClientOpts *MQTT.ClientOptions
var Client MQTT.Client
var TwinResult DeviceTwinResult

var CRDTestTimerGroup = NewTestTimerGroup()

// Token interface to validate the MQTT connection.
type Token interface {
	Wait() bool
	WaitTimeout(time.Duration) bool
	Error() error
}

// BaseMessage the base struct of event message
type BaseMessage struct {
	EventID   string `json:"event_id"`
	Timestamp int64  `json:"timestamp"`
}

// TwinValue the struct of twin value
type TwinValue struct {
	Value    *string        `json:"value,omitempty"`
	Metadata *ValueMetadata `json:"metadata,omitempty"`
}

// ValueMetadata the meta of value
type ValueMetadata struct {
	Timestamp int64 `json:"timestamp,omitempty"`
}

// TypeMetadata the meta of value type
type TypeMetadata struct {
	Type string `json:"type,omitempty"`
}

// TwinVersion twin version
type TwinVersion struct {
	CloudVersion int64 `json:"cloud"`
	EdgeVersion  int64 `json:"edge"`
}

// MsgTwin the struct of device twin
type MsgTwin struct {
	Expected        *TwinValue    `json:"expected,omitempty"`
	Actual          *TwinValue    `json:"actual,omitempty"`
	Optional        *bool         `json:"optional,omitempty"`
	Metadata        *TypeMetadata `json:"metadata,omitempty"`
	ExpectedVersion *TwinVersion  `json:"expected_version,omitempty"`
	ActualVersion   *TwinVersion  `json:"actual_version,omitempty"`
}

// DeviceTwinUpdate the struct of device twin update
type DeviceTwinUpdate struct {
	BaseMessage
	Twin map[string]*MsgTwin `json:"twin"`
}

// DeviceTwinResult device get result
type DeviceTwinResult struct {
	BaseMessage
	Twin map[string]*MsgTwin `json:"twin"`
}

type ServicebusResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Body string `json:"body"`
}

func NewDeployment(name, imgURL string, replicas int32) *apps.Deployment {
	deployment := apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    map[string]string{"app": name},
			Namespace: Namespace,
		},
		Spec: apps.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":                 name,
					constants.E2ELabelKey: constants.E2ELabelValue,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                 name,
						constants.E2ELabelKey: constants.E2ELabelValue,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  name,
							Image: imgURL,
						},
					},
					NodeSelector: map[string]string{
						"node-role.kubernetes.io/edge": "",
					},
				},
			},
		},
	}
	return &deployment
}

func NewPod(podName, imgURL string) *v1.Pod {
	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: v1.NamespaceDefault,
			Labels: map[string]string{
				"app":                 podName,
				constants.E2ELabelKey: constants.E2ELabelValue,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  podName,
					Image: imgURL,
				},
			},
			NodeSelector: map[string]string{
				"node-role.kubernetes.io/edge": "",
			},
		},
	}
	return &pod
}

func GetDeployment(c clientset.Interface, ns, name string) (*apps.Deployment, error) {
	return c.AppsV1().Deployments(ns).Get(context.TODO(), name, metav1.GetOptions{})
}

func CreateDeployment(c clientset.Interface, deployment *apps.Deployment) (*apps.Deployment, error) {
	return c.AppsV1().Deployments(deployment.Namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
}

// DeleteDeployment to delete deployment
func DeleteDeployment(c clientset.Interface, ns, name string) error {
	err := c.AppsV1().Deployments(ns).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		return nil
	}

	return err
}

// HandleDeviceModel to handle DeviceModel operation to apiserver.
func HandleDeviceModel(c edgeclientset.Interface, operation string, UID string, protocolType string) error {
	switch operation {
	case http.MethodPost:
		body := newDeviceModelObject(protocolType, false)
		_, err := c.DevicesV1alpha2().DeviceModels("default").Create(context.TODO(), body, metav1.CreateOptions{})
		return err

	case http.MethodPatch:
		body := newDeviceModelObject(protocolType, true)
		reqBytes, err := json.Marshal(body)
		if err != nil {
			Fatalf("Marshalling body failed: %v", err)
		}

		_, err = c.DevicesV1alpha2().DeviceModels("default").Patch(context.TODO(), UID, types.MergePatchType, reqBytes, metav1.PatchOptions{})
		return err

	case http.MethodDelete:
		err := c.DevicesV1alpha2().DeviceModels("default").Delete(context.TODO(), UID, metav1.DeleteOptions{})
		if err != nil && apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}

// HandleDeviceInstance to handle app deployment/delete using pod spec.
func HandleDeviceInstance(c edgeclientset.Interface, operation string, nodeSelector string, UID string, protocolType string) error {
	switch operation {
	case http.MethodPost:
		body := newDeviceInstanceObject(nodeSelector, protocolType, false)
		_, err := c.DevicesV1alpha2().Devices("default").Create(context.TODO(), body, metav1.CreateOptions{})
		return err

	case http.MethodPatch:
		body := newDeviceInstanceObject(nodeSelector, protocolType, true)
		reqBytes, err := json.Marshal(body)
		if err != nil {
			Fatalf("Marshalling body failed: %v", err)
		}

		_, err = c.DevicesV1alpha2().Devices("default").Patch(context.TODO(), UID, types.MergePatchType, reqBytes, metav1.PatchOptions{})
		return err

	case http.MethodDelete:
		err := c.DevicesV1alpha2().Devices("default").Delete(context.TODO(), UID, metav1.DeleteOptions{})
		if err != nil && apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}

// newDeviceInstanceObject creates a new device instance object
func newDeviceInstanceObject(nodeSelector string, protocolType string, updated bool) *v1alpha2.Device {
	var deviceInstance v1alpha2.Device
	if !updated {
		switch protocolType {
		case BlueTooth:
			deviceInstance = NewBluetoothDeviceInstance(nodeSelector)
		case ModBus:
			deviceInstance = NewModbusDeviceInstance(nodeSelector)
		case Led:
			deviceInstance = NewLedDeviceInstance(nodeSelector)
		case Customized:
			deviceInstance = NewCustomizedDeviceInstance(nodeSelector)
		case IncorrectInstance:
			deviceInstance = IncorrectDeviceInstance()
		}
	} else {
		switch protocolType {
		case BlueTooth:
			deviceInstance = UpdatedBluetoothDeviceInstance(nodeSelector)
		case ModBus:
			deviceInstance = UpdatedModbusDeviceInstance(nodeSelector)
		case Led:
			deviceInstance = UpdatedLedDeviceInstance(nodeSelector)
		case IncorrectInstance:
			deviceInstance = IncorrectDeviceInstance()
		}
	}
	return &deviceInstance
}

// newDeviceModelObject creates a new device model object
func newDeviceModelObject(protocolType string, updated bool) *v1alpha2.DeviceModel {
	var deviceModel v1alpha2.DeviceModel
	if !updated {
		switch protocolType {
		case BlueTooth:
			deviceModel = NewBluetoothDeviceModel()
		case ModBus:
			deviceModel = NewModbusDeviceModel()
		case Led:
			deviceModel = NewLedDeviceModel()
		case Customized:
			deviceModel = NewCustomizedDeviceModel()
		case "incorrect-model":
			deviceModel = IncorrectDeviceModel()
		}
	} else {
		switch protocolType {
		case BlueTooth:
			deviceModel = UpdatedBluetoothDeviceModel()
		case ModBus:
			deviceModel = UpdatedModbusDeviceModel()
		case Led:
			deviceModel = UpdatedLedDeviceModel()
		case "incorrect-model":
			deviceModel = IncorrectDeviceModel()
		}
	}
	return &deviceModel
}

func ListDeviceModel(c edgeclientset.Interface, ns string) ([]v1alpha2.DeviceModel, error) {
	deviceModelList, err := c.DevicesV1alpha2().DeviceModels(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return deviceModelList.Items, nil
}

func ListDevice(c edgeclientset.Interface, ns string) ([]v1alpha2.Device, error) {
	deviceList, err := c.DevicesV1alpha2().Devices(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return deviceList.Items, nil
}

// CheckDeviceModelExists verify whether the contents of the device model matches with what is expected
func CheckDeviceModelExists(deviceModels []v1alpha2.DeviceModel, expectedDeviceModel *v1alpha2.DeviceModel) error {
	modelExists := false
	for _, deviceModel := range deviceModels {
		if expectedDeviceModel.ObjectMeta.Name == deviceModel.ObjectMeta.Name {
			modelExists = true
			if !reflect.DeepEqual(expectedDeviceModel.TypeMeta, deviceModel.TypeMeta) ||
				expectedDeviceModel.ObjectMeta.Namespace != deviceModel.ObjectMeta.Namespace ||
				!reflect.DeepEqual(expectedDeviceModel.Spec, deviceModel.Spec) {
				return fmt.Errorf("the device model is not matching with what was expected")
			}
			break
		}
	}
	if !modelExists {
		return fmt.Errorf("the requested device model is not found")
	}

	return nil
}

func CheckDeviceExists(deviceList []v1alpha2.Device, expectedDevice *v1alpha2.Device) error {
	deviceExists := false
	for _, device := range deviceList {
		if expectedDevice.ObjectMeta.Name == device.ObjectMeta.Name {
			deviceExists = true
			if !reflect.DeepEqual(expectedDevice.TypeMeta, device.TypeMeta) ||
				expectedDevice.ObjectMeta.Namespace != device.ObjectMeta.Namespace ||
				!reflect.DeepEqual(expectedDevice.ObjectMeta.Labels, device.ObjectMeta.Labels) ||
				!reflect.DeepEqual(expectedDevice.Spec, device.Spec) {
				return fmt.Errorf("the device is not matching with what was expected")
			}
			twinExists := false
			for _, expectedTwin := range expectedDevice.Status.Twins {
				for _, twin := range device.Status.Twins {
					if expectedTwin.PropertyName == twin.PropertyName {
						twinExists = true
						if !reflect.DeepEqual(expectedTwin.Desired, twin.Desired) {
							return fmt.Errorf("Status twin " + twin.PropertyName + " not as expected")
						}
						break
					}
				}
			}
			if !twinExists {
				return fmt.Errorf("status twin(s) not found")
			}
			break
		}
	}

	if !deviceExists {
		return fmt.Errorf("the requested device is not found")
	}

	return nil
}

// MqttClientInit create mqtt client config
func MqttClientInit(server, clientID, username, password string) *MQTT.ClientOptions {
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

// MqttConnect function felicitates the MQTT connection
func MqttConnect() error {
	// Initiate the MQTT connection
	ClientOpts = MqttClientInit("tcp://127.0.0.1:1884", "eventbus", "", "")
	Client = MQTT.NewClient(ClientOpts)
	if TokenClient = Client.Connect(); TokenClient.Wait() && TokenClient.Error() != nil {
		return fmt.Errorf("client.Connect() Error is %s" + TokenClient.Error().Error())
	}
	return nil
}

// ChangeTwinValue sends the updated twin value to the edge through the MQTT broker
func ChangeTwinValue(updateMessage DeviceTwinUpdate, deviceID string) error {
	twinUpdateBody, err := json.Marshal(updateMessage)
	if err != nil {
		return fmt.Errorf("Error in marshalling: %s" + err.Error())
	}
	deviceTwinUpdate := DeviceETPrefix + deviceID + TwinETUpdateSuffix
	TokenClient = Client.Publish(deviceTwinUpdate, 0, false, twinUpdateBody)
	if TokenClient.Wait() && TokenClient.Error() != nil {
		return fmt.Errorf("client.publish() Error in device twin update is %s" + TokenClient.Error().Error())
	}
	return nil
}

// GetTwin function is used to get the device twin details from the edge
func GetTwin(updateMessage DeviceTwinUpdate, deviceID string) error {
	getTwin := DeviceETPrefix + deviceID + TwinETGetSuffix
	twinUpdateBody, err := json.Marshal(updateMessage)
	if err != nil {
		return fmt.Errorf("Error in marshalling: %s" + err.Error())
	}
	TokenClient = Client.Publish(getTwin, 0, false, twinUpdateBody)
	if TokenClient.Wait() && TokenClient.Error() != nil {
		return fmt.Errorf("client.publish() Error in device twin get  is: %s " + TokenClient.Error().Error())
	}
	return nil
}

// subscribe function subscribes  the device twin information through the MQTT broker
func TwinSubscribe(deviceID string) {
	getTwinResult := DeviceETPrefix + deviceID + TwinETGetResultSuffix
	TokenClient = Client.Subscribe(getTwinResult, 0, OnTwinMessageReceived)
	if TokenClient.Wait() && TokenClient.Error() != nil {
		Errorf("subscribe() Error in device twin result get  is %v", TokenClient.Error().Error())
	}
	for {
		twin := DeviceTwinUpdate{}
		err := GetTwin(twin, deviceID)
		if err != nil {
			Errorf("Error in getting device twin: %v", err.Error())
		}
		time.Sleep(1 * time.Second)
		if TwinResult.Twin != nil {
			break
		}
	}
}

// OnTwinMessageReceived callback function which is called when message is received
func OnTwinMessageReceived(client MQTT.Client, message MQTT.Message) {
	err := json.Unmarshal(message.Payload(), &TwinResult)
	if err != nil {
		Errorf("Error in unmarshalling: %v", err.Error())
	}
}

// CompareConfigMaps is used to compare 2 config maps
func CompareConfigMaps(configMap, expectedConfigMap v1.ConfigMap) bool {
	Infof("expectedConfigMap.Data: %v", expectedConfigMap.Data)
	Infof("configMap.Data %v", configMap.Data)

	if expectedConfigMap.ObjectMeta.Namespace != configMap.ObjectMeta.Namespace || !reflect.DeepEqual(expectedConfigMap.Data, configMap.Data) {
		return false
	}
	return true
}

// CompareDeviceProfileInConfigMaps is used to compare 2 device profile in config maps
func CompareDeviceProfileInConfigMaps(configMap, expectedConfigMap v1.ConfigMap) bool {
	deviceProfile := configMap.Data["deviceProfile.json"]
	ExpectedDeviceProfile := expectedConfigMap.Data["deviceProfile.json"]
	var deviceProfileMap, expectedDeviceProfileMap map[string]interface{}
	_ = json.Unmarshal([]byte(deviceProfile), &deviceProfileMap)
	_ = json.Unmarshal([]byte(ExpectedDeviceProfile), &expectedDeviceProfileMap)
	return reflect.DeepEqual(expectedConfigMap.TypeMeta, configMap.TypeMeta)
}

// CompareTwin is used to compare 2 device Twins
func CompareTwin(deviceTwin map[string]*MsgTwin, expectedDeviceTwin map[string]*MsgTwin) bool {
	for key := range expectedDeviceTwin {
		if deviceTwin[key].Metadata != nil && deviceTwin[key].Expected.Value != nil {
			if *deviceTwin[key].Metadata != *expectedDeviceTwin[key].Metadata || *deviceTwin[key].Expected.Value != *expectedDeviceTwin[key].Expected.Value {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

func SendMsg(url string, message []byte, header map[string]string) (bool, int) {
	var req *http.Request
	var err error

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
	}
	req, err = http.NewRequest(http.MethodPost, url, bytes.NewBuffer(message))
	if err != nil {
		// handle error
		Fatalf("Frame HTTP request failed, request: %s, reason: %v", req.URL.String(), err)
		return false, 0
	}
	for k, v := range header {
		req.Header.Add(k, v)
	}
	t := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		// handle error
		Fatalf("HTTP request is failed: %v", err)
		return false, 0
	}
	defer resp.Body.Close()
	Infof("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Since(t))
	return true, resp.StatusCode
}

func StartEchoServer() (string, error) {
	r := make(chan string)
	echo := func(response http.ResponseWriter, request *http.Request) {
		b, _ := io.ReadAll(request.Body)
		r <- string(b)
		if _, err := response.Write([]byte("Hello World")); err != nil {
			Errorf("Echo server write failed. reason: %s", err.Error())
		}
	}
	url := func(response http.ResponseWriter, request *http.Request) {
		b, _ := io.ReadAll(request.Body)
		var buff bytes.Buffer
		buff.WriteString("Reply from server: ")
		buff.Write(b)
		buff.WriteString(" Header of the message: [user]: " + request.Header.Get("user") +
			", [passwd]: " + request.Header.Get("passwd"))
		if _, err := response.Write(buff.Bytes()); err != nil {
			Errorf("Echo server write failed. reason: %s", err.Error())
		}
		r <- buff.String()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/echo", echo)
	mux.HandleFunc("/url", url)
	server := &http.Server{Addr: "127.0.0.1:9000", Handler: mux}
	go func() {
		err := server.ListenAndServe()
		Errorf("Echo server stop. reason: %s", err.Error())
	}()
	t := time.NewTimer(time.Second * 30)
	select {
	case resp := <-r:
		err := server.Shutdown(context.TODO())
		return resp, err
	case <-t.C:
		err := server.Shutdown(context.TODO())
		close(r)
		return "", err
	}
}

// subscribe function subscribes  the device twin information through the MQTT broker
func SubscribeMqtt(topic string) (string, error) {
	r := make(chan string)
	TokenClient = Client.Subscribe(topic, 0, func(client MQTT.Client, message MQTT.Message) {
		r <- string(message.Payload())
	})
	if TokenClient.Wait() && TokenClient.Error() != nil {
		return "", fmt.Errorf("subscribe() Error in topic %s. reason: %s", topic, TokenClient.Error().Error())
	}
	t := time.NewTimer(time.Second * 30)
	select {
	case result := <-r:
		Infof("subscribe topic %s to get result: %s", topic, result)
		return result, nil
	case <-t.C:
		close(r)
		return "", fmt.Errorf("wait for MQTT message time out. ")
	}
}

func PublishMqtt(topic, message string) error {
	TokenClient = Client.Publish(topic, 0, false, message)
	if TokenClient.Wait() && TokenClient.Error() != nil {
		return fmt.Errorf("client.publish() Error in topic %s. reason: %s. ", topic, TokenClient.Error().Error())
	}
	Infof("publish topic %s message %s", topic, message)
	return nil
}

func CallServicebus() (response string, err error) {
	var servicebusResponse ServicebusResponse
	payload := strings.NewReader(`{"method":"POST","targetURL":"http://127.0.0.1:9000/echo","payload":""}`)
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodPost, "http://127.0.0.1:9060", payload)
	req.Header.Add("Content-Type", "application/json")
	resp, _ := client.Do(req)
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &servicebusResponse)
	response = servicebusResponse.Body
	return
}

func GetStatefulSet(c clientset.Interface, ns, name string) (*apps.StatefulSet, error) {
	return c.AppsV1().StatefulSets(ns).Get(context.TODO(), name, metav1.GetOptions{})
}

func CreateStatefulSet(c clientset.Interface, statefulSet *apps.StatefulSet) (*apps.StatefulSet, error) {
	return c.AppsV1().StatefulSets(statefulSet.Namespace).Create(context.TODO(), statefulSet, metav1.CreateOptions{})
}

// DeleteStatefulSet to delete statefulSet
func DeleteStatefulSet(c clientset.Interface, ns, name string) error {
	err := c.AppsV1().StatefulSets(ns).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		return nil
	}

	return err
}

// NewTestStatefulSet create statefulSet for test
func NewTestStatefulSet(name, imgURL string, replicas int32) *apps.StatefulSet {
	return &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: Namespace,
			Labels:    map[string]string{"app": name},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":                 name,
					constants.E2ELabelKey: constants.E2ELabelValue,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                 name,
						constants.E2ELabelKey: constants.E2ELabelValue,
					},
				},
				Spec: v1.PodSpec{
					NodeSelector: map[string]string{
						"node-role.kubernetes.io/edge": "",
					},
					Containers: []v1.Container{
						{
							Name:  "nginx",
							Image: imgURL,
						},
					},
				},
			},
		},
	}
}

// WaitForStatusReplicas waits for the ss.Status.Replicas to be equal to expectedReplicas
func WaitForStatusReplicas(c clientset.Interface, ss *apps.StatefulSet, expectedReplicas int32) {
	ns, name := ss.Namespace, ss.Name
	pollErr := wait.PollImmediate(5*time.Second, 240*time.Second,
		func() (bool, error) {
			ssGet, err := c.AppsV1().StatefulSets(ns).Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if ssGet.Status.ObservedGeneration < ss.Generation {
				return false, nil
			}
			if ssGet.Status.Replicas != expectedReplicas {
				klog.Infof("Waiting for stateful set status.replicas to become %d, currently %d", expectedReplicas, ssGet.Status.Replicas)
				return false, nil
			}
			return true, nil
		})
	if pollErr != nil {
		Fatalf("Failed waiting for stateful set status.replicas updated to %d: %v", expectedReplicas, pollErr)
	}
}
