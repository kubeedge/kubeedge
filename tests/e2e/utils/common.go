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
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	Namespace = "default"
)

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
							Limits: v1.ResourceList{
								v1.ResourceName(v1.ResourceCPU):    resource.MustParse("200m"),
								v1.ResourceName(v1.ResourceMemory): resource.MustParse("1Gi"),
							},
						},
						Ports:        []v1.ContainerPort{{ContainerPort: 10000, Protocol: "TCP", Name: "cloudhub"}},
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
	return nil
}

//CreateServiceObject function to create a servcice object
func CreateServiceObject(name string) *v1.Service {
	Service := v1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{"app": "kubeedge"}},

		Spec: v1.ServiceSpec{
			Ports:    []v1.ServicePort{{Protocol: "TCP", Port: 10000, TargetPort: intstr.FromInt(10000)}},
			Selector: map[string]string{"app": "edgecontroller"},
			Type:     "NodePort",
		},
	}
	return &Service
}

//GetServicePort function to get the service port created for deployment.
func GetServicePort(cloudName, serviceHandler string) int32 {
	var svc v1.ServiceList
	var nodePort int32
	err, resp := SendHttpRequest(http.MethodGet, serviceHandler)
	if err != nil {
		// handle error
		Failf("HTTP request is failed :%v", err)
		return -1
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Failf("HTTP Response reading has failed: %v", err)
		return -1
	}

	err = json.Unmarshal(contents, &svc)
	if err != nil {
		Failf("Unmarshal HTTP Response has failed: %v", err)
		return -1
	}
	defer resp.Body.Close()

	for _, svcs := range svc.Items {
		if svcs.Name == cloudName {
			nodePort = svcs.Spec.Ports[0].NodePort
		}
		break
	}

	return nodePort
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
