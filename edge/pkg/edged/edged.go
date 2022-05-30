/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

@CHANGELOG
KubeEdge Authors: To create mini-kubelet for edge deployment scenario,
This file is derived from K8S Kubelet code with reduced set of methods
Changes done are
1. Package edged got some functions from "k8s.io/kubernetes/pkg/kubelet/kubelet.go"
and made some variant
*/

package edged

import (
	"context"
	"fmt"
	"os"

	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/component-base/featuregate"
	"k8s.io/klog/v2"
	kubeletserver "k8s.io/kubernetes/cmd/kubelet/app"
	kubeletoptions "k8s.io/kubernetes/cmd/kubelet/app/options"
	"k8s.io/kubernetes/pkg/kubelet"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	edgedconfig "github.com/kubeedge/kubeedge/edge/pkg/edged/config"
	kubebridge "github.com/kubeedge/kubeedge/edge/pkg/edged/kubeclientbridge"
	metaclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

// edged is the main edged implementation.
type edged struct {
	enable      bool
	context     context.Context
	KuberServer *kubeletoptions.KubeletServer
	KubeletDeps *kubelet.Dependencies
	FeatureGate featuregate.FeatureGate
}

var _ core.Module = (*edged)(nil)

// Register register edged
func Register(e *v1alpha1.Edged) {
	edgedconfig.InitConfigure(e)
	edged, err := newEdged(e.KubeletServer.EnableServer)
	if err != nil {
		klog.Errorf("init new edged error, %v", err)
		os.Exit(1)
	}
	core.Register(edged)
}

func (e *edged) Name() string {
	return modules.EdgedModuleName
}

func (e *edged) Group() string {
	return modules.EdgedGroup
}

//Enable indicates whether this module is enabled
func (e *edged) Enable() bool {
	return e.KuberServer.EnableServer
}

func (e *edged) Start() {
	klog.Info("Starting edged...")
	err := kubeletserver.Run(e.context, e.KuberServer, e.KubeletDeps, e.FeatureGate)
	if err != nil {
		klog.Errorf("Start edged failed, err: %v", err)
		os.Exit(1)
	}
}

// newEdged creates new edged object and initialises it
func newEdged(enable bool) (*edged, error) {
	var ed *edged
	var err error
	if !enable {
		return &edged{
			enable: enable,
		}, nil
	}

	kubeletServer := edgedconfig.Config.KubeletServer
	// use kubeletServer to construct the default KubeletDeps
	kubeletDeps, err := kubeletserver.UnsecuredDependencies(&kubeletServer, utilfeature.DefaultFeatureGate)
	if err != nil {
		klog.ErrorS(err, "Failed to construct kubelet dependencies")
		return nil, fmt.Errorf("failed to construct kubelet dependencies")
	}
	MakeKubeClientBridge(kubeletDeps)

	ed = &edged{
		context:     context.Background(),
		KuberServer: &kubeletServer,
		KubeletDeps: kubeletDeps,
		FeatureGate: utilfeature.DefaultFeatureGate,
	}

	return ed, nil
}

// MakeKubeClientBridge make kubeclient bridge to replace kubeclient with metaclient
func MakeKubeClientBridge(kubeletDeps *kubelet.Dependencies) {
	client := kubebridge.NewSimpleClientset(metaclient.New())

	kubeletDeps.KubeClient = client
	kubeletDeps.EventClient = client.CoreV1()
	kubeletDeps.HeartbeatClient = client
}
