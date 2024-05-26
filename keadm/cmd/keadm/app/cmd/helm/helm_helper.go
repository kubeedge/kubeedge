/*
Copyright 2024 The KubeEdge Authors.

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
package helm

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"helm.sh/helm/v3/pkg/strvals"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	kecharts "github.com/kubeedge/kubeedge/manifests"
)

// MergeProfileValues merges values from specified profile and directly via --set.
func MergeProfileValues(profile string, sets []string,
) (map[string]interface{}, error) {
	bff, err := fs.ReadFile(kecharts.FS, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to read build in profile '%s', err: %v", profile, err)
	}
	vals := make(map[string]interface{})
	if err := yaml.Unmarshal(bff, &vals); err != nil {
		return nil, fmt.Errorf("failed to unmarshal values: %v", err)
	}
	klog.V(4).Infof("combine values: \n\tvalues:%v\n\tsets:%v", vals, sets)
	for _, kv := range sets {
		if err := strvals.ParseInto(kv, vals); err != nil {
			return nil, fmt.Errorf("failed to parse --set data: %s, err: %v", kv, err)
		}
	}
	return vals, nil
}

// Helper a helm client wrapping tool.
type Helper struct {
	cfg *action.Configuration
}

// NewHelper creates a new instance of Helper, and initialize the configuration for reuse.
func NewHelper(kubeconfig, namespace string) (*Helper, error) {
	cf := genericclioptions.NewConfigFlags(true)
	cf.KubeConfig = &kubeconfig
	cf.Namespace = &namespace
	cfg := &action.Configuration{}
	// Make log message print only when you want to debug
	logFunc := func(format string, v ...interface{}) {
		klog.V(4).Infof(format, v...)
	}
	if err := cfg.Init(cf, namespace, "", logFunc); err != nil {
		return nil, fmt.Errorf("failed to init helm action configuration, err: %v", err)
	}
	return &Helper{cfg: cfg}, nil
}

func (h *Helper) GetConfig() *action.Configuration {
	return h.cfg
}

// GetRelease gets a helm release by release name, returns a nil value if not found.
func (h *Helper) GetRelease(releaseName string) (*release.Release, error) {
	rel, err := h.cfg.Releases.Last(releaseName)
	if err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			klog.V(3).Infof("not found release %s", releaseName)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get release %s, err: %v", releaseName, err)
	}
	return rel, nil
}

// GetValues returns a helm release installed values.
func (h *Helper) GetValues(releaseName string) (map[string]interface{}, error) {
	gv := action.NewGetValues(h.cfg)
	vals, err := gv.Run(releaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to get release %s values, err: %v", releaseName, err)
	}
	return vals, nil
}

// MergeExternValues merges values from specified extern value.yaml and directly via --set.
func MergeExternValues(profile string, sets []string,
) (map[string]interface{}, error) {
	bff, err := os.ReadFile(profile)
	if err != nil {
		return nil, fmt.Errorf("failed to read build in profile '%s', err: %v", profile, err)
	}
	vals := make(map[string]interface{})
	if err := yaml.Unmarshal(bff, &vals); err != nil {
		return nil, fmt.Errorf("failed to unmarshal values: %v", err)
	}
	klog.V(4).Infof("combine values: \n\tvalues:%v\n\tsets:%v", vals, sets)
	for _, kv := range sets {
		if err := strvals.ParseInto(kv, vals); err != nil {
			return nil, fmt.Errorf("failed to parse --set data: %s, err: %v", kv, err)
		}
	}
	return vals, nil
}
