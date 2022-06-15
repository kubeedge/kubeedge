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
	"os"
	"strings"

	"github.com/blang/semver"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/kubeedge/common/constants"
)

//K8SInstTool embeds Common struct and contains the default K8S version and
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
		return fmt.Errorf("failed to build config, err: %v", err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create client, err: %v", err)
	}
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.SystemNamespace,
		},
	}

	_, err = client.CoreV1().Namespaces().Get(context.Background(), ns.Name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		if _, err = client.CoreV1().Namespaces().Create(
			context.Background(), ns, metav1.CreateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func installCRDs(ks *K8SInstTool) error {
	config, err := BuildConfig(ks.KubeConfig, ks.Master)
	if err != nil {
		return fmt.Errorf("failed to build config, err: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create a new dynamic client: %v", err)
	}

	crds := map[string][]string{
		"devices": {
			"devices_v1alpha2_device.yaml",
			"devices_v1alpha2_devicemodel.yaml",
		},
		"reliablesyncs": {
			"cluster_objectsync_v1alpha1.yaml",
			"objectsync_v1alpha1.yaml",
		},
		"router": {
			"router_v1_rule.yaml",
			"router_v1_ruleEndpoint.yaml",
		},
	}
	version := fmt.Sprintf("%d.%d", ks.ToolVersion.Major, ks.ToolVersion.Minor)

	// if the specified the version is greater than the latest version
	// this means we haven't released the version, this may only occur in keadm e2e test
	// in this case, we will install the latest version CRDs
	if latestVersion, err := GetLatestVersion(); err == nil {
		if v, err := semver.Parse(strings.TrimPrefix(latestVersion, "v")); err == nil {
			if ks.ToolVersion.GT(v) {
				version = fmt.Sprintf("%d.%d", v.Major, v.Minor)
			}
		}
	}
	fmt.Printf("keadm will install %s CRDs\n", version)

	CRDDownloadURL := fmt.Sprintf(KubeEdgeCRDDownloadURL, version)
	for dir := range crds {
		crdPath := KubeEdgeCrdPath + "/" + dir
		err = os.MkdirAll(crdPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("not able to create %s folder path", crdPath)
		}

		for _, crdFile := range crds[dir] {
			// check it first, do not download when it exists
			_, err := os.Lstat(crdPath + "/" + crdFile)
			if err != nil {
				if os.IsNotExist(err) {
					// Download the tar from repo
					downloadURL := fmt.Sprintf("cd %s && wget -k --no-check-certificate --progress=bar:force %s/%s", KubeEdgeCrdPath+"/"+dir, CRDDownloadURL, dir+"/"+crdFile)
					cmd := NewCommand(downloadURL)
					if err := cmd.Exec(); err != nil {
						return err
					}
				} else {
					return err
				}
			}

			// not found err, create crd from crd file
			err = createKubeEdgeV1CRD(dynamicClient, crdPath+"/"+crdFile)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				return err
			}
		}
	}

	return nil
}

func createKubeEdgeV1CRD(dynamicClient dynamic.Interface, crdFile string) error {
	content, err := os.ReadFile(crdFile)
	if err != nil {
		return fmt.Errorf("read crd yaml error: %v", err)
	}

	kubeEdgeCRD := &unstructured.Unstructured{}
	err = yaml.Unmarshal(content, kubeEdgeCRD)
	if err != nil {
		return fmt.Errorf("unmarshal tfjobCRD error: %v", err)
	}

	gvk := kubeEdgeCRD.GetObjectKind().GroupVersionKind()
	gvr, _ := meta.UnsafeGuessKindToResource(gvk)

	_, err = dynamicClient.Resource(gvr).Create(context.Background(), kubeEdgeCRD, metav1.CreateOptions{})

	return err
}

//TearDown should uninstall K8S, but it is not required either for cloud or edge node.
//It is defined so that K8SInstTool implements ToolsInstaller interface
func (ks *K8SInstTool) TearDown() error {
	return nil
}
