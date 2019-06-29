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
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"reflect"
	"strings"
	"time"

	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/apis/devices/v1alpha1"
	"github.com/kubeedge/viaduct/pkg/api"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	Namespace             = "default"
	DeviceETPrefix        = "$hw/events/device/"
	TwinETUpdateSuffix    = "/twin/update"
	TwinETGetSuffix       = "/twin/get"
	TwinETGetResultSuffix = "/twin/get/result"
)

var (
	ProtocolQuic      bool
	ProtocolWebsocket bool
)

var TokenClient Token
var ClientOpts *MQTT.ClientOptions
var Client MQTT.Client
var TwinResult DeviceTwinResult

//Token interface to validate the MQTT connection.
type Token interface {
	Wait() bool
	WaitTimeout(time.Duration) bool
	Error() error
}

//BaseMessage the base struct of event message
type BaseMessage struct {
	EventID   string `json:"event_id"`
	Timestamp int64  `json:"timestamp"`
}

//TwinValue the struct of twin value
type TwinValue struct {
	Value    *string        `json:"value, omitempty"`
	Metadata *ValueMetadata `json:"metadata,omitempty"`
}

//ValueMetadata the meta of value
type ValueMetadata struct {
	Timestamp int64 `json:"timestamp, omitempty"`
}

//TypeMetadata the meta of value type
type TypeMetadata struct {
	Type string `json:"type,omitempty"`
}

//TwinVersion twin version
type TwinVersion struct {
	CloudVersion int64 `json:"cloud"`
	EdgeVersion  int64 `json:"edge"`
}

//MsgTwin the struct of device twin
type MsgTwin struct {
	Expected        *TwinValue    `json:"expected,omitempty"`
	Actual          *TwinValue    `json:"actual,omitempty"`
	Optional        *bool         `json:"optional,omitempty"`
	Metadata        *TypeMetadata `json:"metadata,omitempty"`
	ExpectedVersion *TwinVersion  `json:"expected_version,omitempty"`
	ActualVersion   *TwinVersion  `json:"actual_version,omitempty"`
}

//DeviceTwinUpdate the struct of device twin update
type DeviceTwinUpdate struct {
	BaseMessage
	Twin map[string]*MsgTwin `json:"twin"`
}

//DeviceTwinResult device get result
type DeviceTwinResult struct {
	BaseMessage
	Twin map[string]*MsgTwin `json:"twin"`
}

//Function to get nginx deployment spec
func nginxDeploymentSpec(imgUrl, selector string, replicas int) *apps.DeploymentSpec {
	var nodeselector map[string]string
	if selector == "" {
		nodeselector = map[string]string{}
	} else {
		nodeselector = map[string]string{"disktype": selector}
	}
	deplObj := apps.DeploymentSpec{
		Replicas: func() *int32 { i := int32(replicas); return &i }(),
		Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "nginx"}},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"app": "nginx"},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "nginx",
						Image: imgUrl,
					},
				},
				NodeSelector: nodeselector,
			},
		},
	}

	return &deplObj
}

//Function to get edgecore deploymentspec object
func edgecoreDeploymentSpec(imgURL, configmap string, replicas int) *apps.DeploymentSpec {
	IsSecureCtx := true
	deplObj := apps.DeploymentSpec{
		Replicas: func() *int32 { i := int32(replicas); return &i }(),
		Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "edgecore"}},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"app": "edgecore"},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:            "edgecore",
						Image:           imgURL,
						SecurityContext: &v1.SecurityContext{Privileged: &IsSecureCtx},
						ImagePullPolicy: v1.PullPolicy("IfNotPresent"),
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceName(v1.ResourceCPU):    resource.MustParse("200m"),
								v1.ResourceName(v1.ResourceMemory): resource.MustParse("100Mi"),
							},
							Limits: v1.ResourceList{
								v1.ResourceName(v1.ResourceCPU):    resource.MustParse("200m"),
								v1.ResourceName(v1.ResourceMemory): resource.MustParse("100Mi"),
							},
						},
						Env:          []v1.EnvVar{{Name: "DOCKER_HOST", Value: "tcp://localhost:2375"}},
						VolumeMounts: []v1.VolumeMount{{Name: "cert", MountPath: "/etc/kubeedge/certs"}, {Name: "conf", MountPath: "/etc/kubeedge/edge/conf"}},
					}, {
						Name:            "dind-daemon",
						SecurityContext: &v1.SecurityContext{Privileged: &IsSecureCtx},
						Image:           "docker:dind",
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceName(v1.ResourceCPU):    resource.MustParse("20m"),
								v1.ResourceName(v1.ResourceMemory): resource.MustParse("256Mi"),
							},
						},
						VolumeMounts: []v1.VolumeMount{{Name: "docker-graph-storage", MountPath: "/var/lib/docker"}},
					},
				},
				NodeSelector: map[string]string{"k8snode": "kb-perf-node"},
				Volumes: []v1.Volume{
					{Name: "cert", VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/etc/kubeedge/certs"}}},
					{Name: "conf", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{LocalObjectReference: v1.LocalObjectReference{Name: configmap}}}},
					{Name: "docker-graph-storage", VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}}},
				},
			},
		},
	}
	return &deplObj
}

//Function to create cloudcore deploymentspec object
func cloudcoreDeploymentSpec(imgURL, configmap string, replicas int) *apps.DeploymentSpec {
	var portInfo []v1.ContainerPort
	portInfo = []v1.ContainerPort{{ContainerPort: 10000, Protocol: "TCP", Name: "websocket"}, {ContainerPort: 10001, Protocol: "UDP", Name: "quic"}}

	deplObj := apps.DeploymentSpec{
		Replicas: func() *int32 { i := int32(replicas); return &i }(),
		Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "edgecontroller"}},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"app": "edgecontroller"},
			},
			Spec: v1.PodSpec{
				HostNetwork:   true,
				RestartPolicy: "Always",
				Containers: []v1.Container{
					{
						Name:            "edgecontroller",
						Image:           imgURL,
						ImagePullPolicy: v1.PullPolicy("IfNotPresent"),
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceName(v1.ResourceCPU):    resource.MustParse("100m"),
								v1.ResourceName(v1.ResourceMemory): resource.MustParse("512Mi"),
							},
						},
						Ports:        portInfo,
						VolumeMounts: []v1.VolumeMount{{Name: "cert", MountPath: "/etc/kubeedge/certs"}, {Name: "conf", MountPath: "/etc/kubeedge/cloud/conf"}},
					},
				},
				Volumes: []v1.Volume{
					{Name: "cert", VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/etc/kubeedge/certs"}}},
					{Name: "conf", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{LocalObjectReference: v1.LocalObjectReference{Name: configmap}}}},
				},
			},
		},
	}
	return &deplObj
}

func newDeployment(cloudcore, edgecore bool, name, imgUrl, nodeselector, configmap string, replicas int) *apps.Deployment {
	var depObj *apps.DeploymentSpec
	var namespace string

	if edgecore == true {
		depObj = edgecoreDeploymentSpec(imgUrl, configmap, replicas)
		namespace = Namespace
	} else if cloudcore == true {
		depObj = cloudcoreDeploymentSpec(imgUrl, configmap, replicas)
		namespace = Namespace
	} else {
		depObj = nginxDeploymentSpec(imgUrl, nodeselector, replicas)
		namespace = Namespace
	}

	deployment := apps.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    map[string]string{"app": "kubeedge"},
			Namespace: namespace,
		},
		Spec: *depObj,
	}
	return &deployment
}

func newPodObj(podName, imgUrl, nodeselector string) *v1.Pod {
	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   podName,
			Labels: map[string]string{"app": "nginx"},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "nginx",
					Image: imgUrl,
					Ports: []v1.ContainerPort{{HostPort: 80, ContainerPort: 80}},
				},
			},
			NodeSelector: map[string]string{"disktype": nodeselector},
		},
	}
	return &pod
}

//GetDeployments to get the deployments list
func GetDeployments(list *apps.DeploymentList, getDeploymentApi string) error {

	err, resp := SendHttpRequest(http.MethodGet, getDeploymentApi)
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Failf("HTTP Response reading has failed: %v", err)
		return err
	}
	err = json.Unmarshal(contents, &list)
	if err != nil {
		Failf("Unmarshal HTTP Response has failed: %v", err)
		return err
	}
	return nil

}
func VerifyDeleteDeployment(getDeploymentApi string) int {
	err, resp := SendHttpRequest(http.MethodGet, getDeploymentApi)
	if err != nil {
		Failf("SendHttpRequest is failed: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

//HandlePod to handle app deployment/delete using pod spec.
func HandlePod(operation string, apiserver string, UID string, ImageUrl, nodeselector string) bool {
	var req *http.Request
	var err error
	var body io.Reader

	client := &http.Client{}
	switch operation {
	case "POST":
		body := newPodObj(UID, ImageUrl, nodeselector)
		respBytes, err := json.Marshal(body)
		if err != nil {
			Failf("Marshalling body failed: %v", err)
		}
		req, err = http.NewRequest(http.MethodPost, apiserver, bytes.NewBuffer(respBytes))
	case "DELETE":
		req, err = http.NewRequest(http.MethodDelete, apiserver+UID, body)
	}
	if err != nil {
		// handle error
		Failf("Frame HTTP request failed: %v", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	t := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		// handle error
		Failf("HTTP request is failed :%v", err)
		return false
	}
	InfoV6("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	return true
}

//HandleDeployment to handle app deployment/delete deployment.
func HandleDeployment(IsCloudCore, IsEdgeCore bool, operation, apiserver, UID, ImageUrl, nodeselector, configmapname string, replica int) bool {
	var req *http.Request
	var err error
	var body io.Reader

	defer ginkgo.GinkgoRecover()
	client := &http.Client{}
	switch operation {
	case "POST":
		depObj := newDeployment(IsCloudCore, IsEdgeCore, UID, ImageUrl, nodeselector, configmapname, replica)
		if err != nil {
			Failf("GenerateDeploymentBody marshalling failed: %v", err)
		}
		respBytes, err := json.Marshal(depObj)
		if err != nil {
			Failf("Marshalling body failed: %v", err)
		}
		req, err = http.NewRequest(http.MethodPost, apiserver, bytes.NewBuffer(respBytes))
	case "DELETE":
		req, err = http.NewRequest(http.MethodDelete, apiserver+UID, body)
	}
	if err != nil {
		// handle error
		Failf("Frame HTTP request failed: %v", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	t := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		// handle error
		Failf("HTTP request is failed :%v", err)
		return false
	}
	InfoV6("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	return true
}

//DeleteDeployment to delete deployment
func DeleteDeployment(DeploymentApi, deploymentname string) int {
	err, resp := SendHttpRequest(http.MethodDelete, DeploymentApi+"/"+deploymentname)
	if err != nil {
		// handle error
		Failf("HTTP request is failed :%v", err)
		return -1
	}

	defer resp.Body.Close()

	return resp.StatusCode
}

//PrintCombinedOutput to show the os command injuction in combined format
func PrintCombinedOutput(cmd *exec.Cmd) error {
	Info("===========> Executing: %s\n", strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		Failf("CombinedOutput failed", err)
		return err
	}
	if len(output) > 0 {
		Info("=====> Output: %s\n", string(output))
	}
	return nil
}

//ExposeCloudService function to expose the service for cloud deployment
func ExposeCloudService(name, serviceHandler string) error {
	ServiceObj := CreateServiceObject(name)
	respBytes, err := json.Marshal(ServiceObj)
	if err != nil {
		Failf("Marshalling body failed: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, serviceHandler, bytes.NewBuffer(respBytes))
	if err != nil {
		// handle error
		Failf("Frame HTTP request failed: %v", err)
		return err
	}
	client := &http.Client{}
	req.Header.Set("Content-Type", "application/json")
	t := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		// handle error
		Failf("HTTP request is failed :%v", err)
		return err
	}
	InfoV6("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	Expect(resp.StatusCode).Should(Equal(http.StatusCreated))
	return nil
}

//CreateServiceObject function to create a servcice object
func CreateServiceObject(name string) *v1.Service {
	var portInfo []v1.ServicePort

	portInfo = []v1.ServicePort{
		{
			Name: "websocket", Protocol: "TCP", Port: 10000, TargetPort: intstr.FromInt(10000),
		}, {
			Name: "quic", Protocol: "UDP", Port: 10001, TargetPort: intstr.FromInt(10001),
		},
	}

	Service := v1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{"app": "kubeedge"}},

		Spec: v1.ServiceSpec{
			Ports:    portInfo,
			Selector: map[string]string{"app": "edgecontroller"},
			Type:     "NodePort",
		},
	}
	return &Service
}

//GetServicePort function to get the service port created for deployment.
func GetServicePort(cloudName, serviceHandler string) (int32, int32) {
	var svc v1.ServiceList
	var wssport, quicport int32
	err, resp := SendHttpRequest(http.MethodGet, serviceHandler)
	if err != nil {
		// handle error
		Failf("HTTP request is failed :%v", err)
		return -1, -1
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Failf("HTTP Response reading has failed: %v", err)
		return -1, -1
	}

	err = json.Unmarshal(contents, &svc)
	if err != nil {
		Failf("Unmarshal HTTP Response has failed: %v", err)
		return -1, -1
	}
	defer resp.Body.Close()

	for _, svcs := range svc.Items {
		if svcs.Name == cloudName {
			for _, nodePort := range svcs.Spec.Ports {
				if nodePort.Name == api.ProtocolTypeQuic {
					quicport = nodePort.NodePort
				}
				if nodePort.Name == api.ProtocolTypeWS {
					wssport = nodePort.NodePort
				}
			}
		}
		break
	}
	return wssport, quicport
}

//DeleteSvc function to delete service
func DeleteSvc(svcname string) int {
	err, resp := SendHttpRequest(http.MethodDelete, svcname)
	if err != nil {
		// handle error
		Failf("HTTP request is failed :%v", err)
		return -1
	}

	defer resp.Body.Close()

	return resp.StatusCode
}

//HandleDeviceModel to handle app deployment/delete using pod spec.
func HandleDeviceModel(operation string, apiserver string, UID string, protocolType string) (bool, int) {
	var req *http.Request
	var err error
	var body io.Reader

	client := &http.Client{}
	switch operation {
	case "POST":
		body := newDeviceModelObject(protocolType, false)
		respBytes, err := json.Marshal(body)
		if err != nil {
			Failf("Marshalling body failed: %v", err)
		}
		req, err = http.NewRequest(http.MethodPost, apiserver, bytes.NewBuffer(respBytes))
		req.Header.Set("Content-Type", "application/json")
	case "PATCH":
		body := newDeviceModelObject(protocolType, true)
		respBytes, err := json.Marshal(body)
		if err != nil {
			Failf("Marshalling body failed: %v", err)
		}
		req, err = http.NewRequest(http.MethodPatch, apiserver+UID, bytes.NewBuffer(respBytes))
		req.Header.Set("Content-Type", "application/merge-patch+json")
	case "DELETE":
		req, err = http.NewRequest(http.MethodDelete, apiserver+UID, body)
		req.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		// handle error
		Failf("Frame HTTP request failed: %v", err)
		return false, 0
	}
	t := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		// handle error
		Failf("HTTP request is failed :%v", err)
		return false, 0
	}
	InfoV6("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	return true, resp.StatusCode
}

//HandleDeviceInstance to handle app deployment/delete using pod spec.
func HandleDeviceInstance(operation string, apiserver string, nodeSelector string, UID string, protocolType string) (bool, int) {
	var req *http.Request
	var err error
	var body io.Reader

	client := &http.Client{}
	switch operation {
	case "POST":
		body := newDeviceInstanceObject(nodeSelector, protocolType, false)
		respBytes, err := json.Marshal(body)
		if err != nil {
			Failf("Marshalling body failed: %v", err)
		}
		req, err = http.NewRequest(http.MethodPost, apiserver, bytes.NewBuffer(respBytes))
		req.Header.Set("Content-Type", "application/json")
	case "PATCH":
		body := newDeviceInstanceObject(nodeSelector, protocolType, true)
		respBytes, err := json.Marshal(body)
		if err != nil {
			Failf("Marshalling body failed: %v", err)
		}
		req, err = http.NewRequest(http.MethodPatch, apiserver+UID, bytes.NewBuffer(respBytes))
		req.Header.Set("Content-Type", "application/merge-patch+json")
	case "DELETE":
		req, err = http.NewRequest(http.MethodDelete, apiserver+UID, body)
		req.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		// handle error
		Failf("Frame HTTP request failed: %v", err)
		return false, 0
	}
	t := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		// handle error
		Failf("HTTP request is failed :%v", err)
		return false, 0
	}
	InfoV6("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	return true, resp.StatusCode
}

// newDeviceInstanceObject creates a new device instance object
func newDeviceInstanceObject(nodeSelector string, protocolType string, updated bool) *v1alpha1.Device {
	var deviceInstance v1alpha1.Device
	if updated == false {
		switch protocolType {
		case "bluetooth":
			deviceInstance = NewBluetoothDeviceInstance(nodeSelector)
		case "modbus":
			deviceInstance = NewModbusDeviceInstance(nodeSelector)
		case "led":
			deviceInstance = NewLedDeviceInstance(nodeSelector)
		case "incorrect-instance":
			deviceInstance = IncorrectDeviceInstance()
		}
	} else {
		switch protocolType {
		case "bluetooth":
			deviceInstance = UpdatedBluetoothDeviceInstance(nodeSelector)
		case "modbus":
			deviceInstance = UpdatedModbusDeviceInstance(nodeSelector)
		case "led":
			deviceInstance = UpdatedLedDeviceInstance(nodeSelector)
		case "incorrect-instance":
			deviceInstance = IncorrectDeviceInstance()
		}
	}
	return &deviceInstance
}

// newDeviceModelObject creates a new device model object
func newDeviceModelObject(protocolType string, updated bool) *v1alpha1.DeviceModel {
	var deviceModel v1alpha1.DeviceModel
	if updated == false {
		switch protocolType {
		case "bluetooth":
			deviceModel = NewBluetoothDeviceModel()
		case "modbus":
			deviceModel = NewModbusDeviceModel()
		case "led":
			deviceModel = NewLedDeviceModel()
		case "incorrect-model":
			deviceModel = IncorrectDeviceModel()
		}
	} else {
		switch protocolType {
		case "bluetooth":
			deviceModel = UpdatedBluetoothDeviceModel()
		case "modbus":
			deviceModel = UpdatedModbusDeviceModel()
		case "led":
			deviceModel = UpdatedLedDeviceModel()
		case "incorrect-model":
			deviceModel = IncorrectDeviceModel()
		}
	}
	return &deviceModel
}

//GetDeviceModel to get the deviceModel list and verify whether the contents of the device model matches with what is expected
func GetDeviceModel(list *v1alpha1.DeviceModelList, getDeviceModelApi string, expectedDeviceModel *v1alpha1.DeviceModel) ([]v1alpha1.DeviceModel, error) {
	err, resp := SendHttpRequest(http.MethodGet, getDeviceModelApi)
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Failf("HTTP Response reading has failed: %v", err)
		return nil, err
	}
	err = json.Unmarshal(contents, &list)
	if err != nil {
		Failf("Unmarshal HTTP Response has failed: %v", err)
		return nil, err
	}
	if expectedDeviceModel != nil {
		modelExists := false
		for _, deviceModel := range list.Items {
			if expectedDeviceModel.ObjectMeta.Name == deviceModel.ObjectMeta.Name {
				modelExists = true
				if reflect.DeepEqual(expectedDeviceModel.TypeMeta, deviceModel.TypeMeta) == false || expectedDeviceModel.ObjectMeta.Namespace != deviceModel.ObjectMeta.Namespace || reflect.DeepEqual(expectedDeviceModel.Spec, deviceModel.Spec) == false {
					return nil, errors.New("The device model is not matching with what was expected")
				}
			}
		}
		if !modelExists {
			return nil, errors.New("The requested device model is not found")
		}
	}
	return list.Items, nil
}

//GetDevice to get the device list
func GetDevice(list *v1alpha1.DeviceList, getDeviceApi string, expectedDevice *v1alpha1.Device) ([]v1alpha1.Device, error) {
	err, resp := SendHttpRequest(http.MethodGet, getDeviceApi)
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Failf("HTTP Response reading has failed: %v", err)
		return nil, err
	}
	err = json.Unmarshal(contents, &list)
	if err != nil {
		Failf("Unmarshal HTTP Response has failed: %v", err)
		return nil, err
	}
	if expectedDevice != nil {
		deviceExists := false
		for _, device := range list.Items {
			if expectedDevice.ObjectMeta.Name == device.ObjectMeta.Name {
				deviceExists = true
				if reflect.DeepEqual(expectedDevice.TypeMeta, device.TypeMeta) == false || expectedDevice.ObjectMeta.Namespace != device.ObjectMeta.Namespace || reflect.DeepEqual(expectedDevice.ObjectMeta.Labels, device.ObjectMeta.Labels) == false || reflect.DeepEqual(expectedDevice.Spec, device.Spec) == false {
					return nil, errors.New("The device is not matching with what was expected")
				}
				twinExists := false
				for _, expectedTwin := range expectedDevice.Status.Twins {
					for _, twin := range device.Status.Twins {
						if expectedTwin.PropertyName == twin.PropertyName {
							twinExists = true
							if reflect.DeepEqual(expectedTwin.Desired, twin.Desired) == false {
								return nil, errors.New("Status twin " + twin.PropertyName + " not as expected")
							}
						}
					}
				}
				if !twinExists {
					return nil, errors.New("Status twin(s) not found !!!!")
				}
			}
		}
		if !deviceExists {
			return nil, errors.New("The requested device is not found")
		}
	}
	return list.Items, nil
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

//MqttConnect function felicitates the MQTT connection
func MqttConnect() error {
	// Initiate the MQTT connection
	ClientOpts = MqttClientInit("tcp://127.0.0.1:1884", "eventbus", "", "")
	Client = MQTT.NewClient(ClientOpts)
	if TokenClient = Client.Connect(); TokenClient.Wait() && TokenClient.Error() != nil {
		return errors.New("client.Connect() Error is %s" + TokenClient.Error().Error())
	}
	return nil
}

//ChangeTwinValue sends the updated twin value to the edge through the MQTT broker
func ChangeTwinValue(updateMessage DeviceTwinUpdate, deviceID string) error {
	twinUpdateBody, err := json.Marshal(updateMessage)
	if err != nil {
		return errors.New("Error in marshalling: %s" + err.Error())
	}
	deviceTwinUpdate := DeviceETPrefix + deviceID + TwinETUpdateSuffix
	TokenClient = Client.Publish(deviceTwinUpdate, 0, false, twinUpdateBody)
	if TokenClient.Wait() && TokenClient.Error() != nil {
		return errors.New("client.publish() Error in device twin update is %s" + TokenClient.Error().Error())
	}
	return nil
}

//GetTwin function is used to get the device twin details from the edge
func GetTwin(updateMessage DeviceTwinUpdate, deviceID string) error {
	getTwin := DeviceETPrefix + deviceID + TwinETGetSuffix
	twinUpdateBody, err := json.Marshal(updateMessage)
	if err != nil {
		return errors.New("Error in marshalling: %s" + err.Error())
	}
	TokenClient = Client.Publish(getTwin, 0, false, twinUpdateBody)
	if TokenClient.Wait() && TokenClient.Error() != nil {
		return errors.New("client.publish() Error in device twin get  is: %s " + TokenClient.Error().Error())
	}
	return nil
}

//subscribe function subscribes  the device twin information through the MQTT broker
func TwinSubscribe(deviceID string) {
	getTwinResult := DeviceETPrefix + deviceID + TwinETGetResultSuffix
	TokenClient = Client.Subscribe(getTwinResult, 0, OnTwinMessageReceived)
	if TokenClient.Wait() && TokenClient.Error() != nil {
		Err("subscribe() Error in device twin result get  is " + TokenClient.Error().Error())
	}
	for {
		twin := DeviceTwinUpdate{}
		err := GetTwin(twin, deviceID)
		if err != nil {
			Err("Error in getting device twin: " + err.Error())
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
		Err("Error in unmarshalling: " + err.Error())
	}
}

// CompareConfigMaps is used to compare 2 config maps
func CompareConfigMaps(configMap, expectedConfigMap v1.ConfigMap) bool {
	if reflect.DeepEqual(expectedConfigMap.TypeMeta, configMap.TypeMeta) == false || expectedConfigMap.ObjectMeta.Namespace != configMap.ObjectMeta.Namespace || reflect.DeepEqual(expectedConfigMap.Data, configMap.Data) == false {
		return false

	}
	return true
}

// CompareTwin is used to compare 2 device Twins
func CompareTwin(deviceTwin map[string]*MsgTwin, expectedDeviceTwin map[string]*MsgTwin) bool {
	for key, _ := range expectedDeviceTwin {
		if deviceTwin[key].Metadata!=nil && deviceTwin[key].Expected.Value!=nil {
			if *deviceTwin[key].Metadata != *expectedDeviceTwin[key].Metadata || *deviceTwin[key].Expected.Value != *expectedDeviceTwin[key].Expected.Value {
				return false
			}
		}else {
			return false
		}
	}
	return true
}
