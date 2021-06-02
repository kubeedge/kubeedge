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

package util

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	crdclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//K8SInstTool embedes Common struct and contains the default K8S version and
//a flag depicting if host is an edge or cloud node
//It implements ToolsInstaller interface
type K8SInstTool struct {
	Common
}

//InstallTools sets the OS interface, checks if K8S installation is required or not.
//If required then install the said version.
func (ks *K8SInstTool) InstallTools() error {
	ks.SetOSInterface(GetOSInterface())

	cloudCoreRunning, err := ks.IsKubeEdgeProcessRunning(KubeCloudBinaryName)
	if err != nil {
		return err
	}
	if cloudCoreRunning {
		return fmt.Errorf("CloudCore is already running on this node, please run reset to clean up first")
	}

	err = ks.IsK8SComponentInstalled(ks.KubeConfig, ks.Master)
	if err != nil {
		return err
	}

	fmt.Println("Kubernetes version verification passed, KubeEdge installation will start...")

	err = installCRDs(ks)
	if err != nil {
		return err
	}

	err = createKubeEdgeNs(ks.KubeConfig, ks.Master)
	if err != nil {
		return err
	}

	return nil
}

func createKubeEdgeNs(kubeConfig, master string) error {
	config, err := BuildConfig(kubeConfig, master)
	if err != nil {
		return fmt.Errorf("Failed to build config, err: %v", err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("Failed to create client, err: %v", err)
	}
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubeedge",
		},
	}

	_, err = client.CoreV1().Namespaces().Get(context.Background(), "kubeedge", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = client.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func installCRDs(ks *K8SInstTool) error {
	config, err := BuildConfig(ks.KubeConfig, ks.Master)
	if err != nil {
		return fmt.Errorf("Failed to build config, err: %v", err)
	}

	crdClient, err := crdclient.NewForConfig(config)
	if err != nil {
		return err
	}

	crds := map[string][]string{"devices": {"devices/devices_v1alpha2_device.yaml",
		"devices/devices_v1alpha2_devicemodel.yaml"}, "reliablesyncs": {"reliablesyncs/cluster_objectsync_v1alpha1.yaml",
		"reliablesyncs/objectsync_v1alpha1.yaml"},
		"router": {"router/router_v1_rule.yaml",
			"router/router_v1_ruleEndpoint.yaml"},
	}
	version := fmt.Sprintf("%d.%d", ks.ToolVersion.Major, ks.ToolVersion.Minor)
	CRDDownloadURL := fmt.Sprintf(KubeEdgeCRDDownloadURL, version)
	for dir := range crds {
		crdPath := KubeEdgeCrdPath + "/" + dir
		err = os.MkdirAll(crdPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("not able to create %s folder path", crdPath)
		}

		for _, crdFile := range crds[dir] {
			// check it first, do not download when it exists
			_, err := os.Lstat(KubeEdgeCrdPath + "/" + crdFile)
			if err != nil {
				if os.IsNotExist(err) {
					// Download the tar from repo
					downloadURL := fmt.Sprintf("cd %s && wget -k --no-check-certificate --progress=bar:force %s/%s", KubeEdgeCrdPath+"/"+dir, CRDDownloadURL, crdFile)
					cmd := NewCommand(downloadURL)
					if err := cmd.Exec(); err != nil {
						return err
					}
				} else {
					return err
				}
			}

			// not found err, create crd from crd file
			if dir == "router" {
				err = createKubeEdgeV1CRD(crdClient, KubeEdgeCrdPath+"/"+crdFile)
			} else {
				err = createKubeEdgeV1beta1CRD(crdClient, KubeEdgeCrdPath+"/"+crdFile)
			}
			if err != nil && !apierrors.IsAlreadyExists(err) {
				return err
			}
		}
	}

	return nil
}

func createKubeEdgeV1beta1CRD(clientset crdclient.Interface, crdFile string) error {
	content, err := ioutil.ReadFile(crdFile)
	if err != nil {
		return fmt.Errorf("read crd yaml error: %v", err)
	}

	kubeEdgeCRD := &apiextensionsv1beta1.CustomResourceDefinition{}
	err = yaml.Unmarshal(content, kubeEdgeCRD)
	if err != nil {
		return fmt.Errorf("unmarshal tfjobCRD error: %v", err)
	}

	_, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(context.Background(), kubeEdgeCRD, metav1.CreateOptions{})

	return err
}

func createKubeEdgeV1CRD(clientset crdclient.Interface, crdFile string) error {
	content, err := ioutil.ReadFile(crdFile)
	if err != nil {
		return fmt.Errorf("read crd yaml error: %v", err)
	}

	kubeEdgeCRD := &apiextensionsv1.CustomResourceDefinition{}
	err = yaml.Unmarshal(content, kubeEdgeCRD)
	if err != nil {
		return fmt.Errorf("unmarshal tfjobCRD error: %v", err)
	}

	_, err = clientset.ApiextensionsV1().CustomResourceDefinitions().Create(context.Background(), kubeEdgeCRD, metav1.CreateOptions{})

	return err
}

//TearDown shoud uninstall K8S, but it is not required either for cloud or edge node.
//It is defined so that K8SInstTool implements ToolsInstaller interface
func (ks *K8SInstTool) TearDown() error {
	return nil
}
