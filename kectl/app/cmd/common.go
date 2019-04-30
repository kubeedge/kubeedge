/*
Copyright 2019 The Kubeedge Authors.

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
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
)

//Constants used by installers
const (
	KubeCloudCertificatePath   = "/etc/kubeedge/kubeedge/certificates/"
	KubeControllerConfig       = "/etc/kubeedge/kubeedge/cloud/conf/controller.yaml"
	KubeCloudBinaryName        = "edgecontroller"
	KubeCloudApiserverYamlPath = "/etc/kubernetes/manifests/kube-apiserver.yaml"
	KubeCloudReplaceIndex      = 25
	KubeCloudReplaceString     = "    - --insecure-bind-address=0.0.0.0\n"
)

type ToolsInstallerCloud interface {
	InstallTools() error
}
