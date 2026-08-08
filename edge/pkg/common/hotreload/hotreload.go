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

// Package hotreload lets edgecore pick up a small set of configuration
// fields from disk while it is running, instead of requiring a full
// restart. Only fields that every module already reads fresh on each use
// (e.g. from a package level Configure value) are eligible, so applying
// them cannot leave a module running with half of an old configuration and
// half of a new one.
package hotreload

import (
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	edgehubconfig "github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	metamanagerconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/config"
)

// Watch starts a goroutine that reloads the fields listed above from
// configFile whenever edgecore receives SIGHUP. keadm's `config-update`
// command sends this signal after writing a new config file, provided every
// field being changed is safe to hot reload; otherwise it restarts edgecore
// as before.
func Watch(configFile string) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)
	go func() {
		for range sigChan {
			reload(configFile)
		}
	}()
}

// reload re-reads configFile and applies the subset of fields that support
// hot reload. Any other field change in the file is ignored here: it only
// takes effect on the next full edgecore start, since keadm already
// restarts edgecore whenever a non-hot-reloadable field is updated.
func reload(configFile string) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		klog.Errorf("hot reload: failed to read config file %s: %v", configFile, err)
		return
	}

	cfg := &v1alpha2.EdgeCoreConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		klog.Errorf("hot reload: failed to parse config file %s: %v", configFile, err)
		return
	}

	if cfg.Modules == nil {
		klog.Errorf("hot reload: config file %s has no modules section", configFile)
		return
	}

	if eh := cfg.Modules.EdgeHub; eh != nil {
		if eh.Heartbeat != edgehubconfig.GetHeartbeat() {
			edgehubconfig.SetHeartbeat(eh.Heartbeat)
			klog.Infof("hot reload: edgehub heartbeat updated to %ds", eh.Heartbeat)
		}
	}

	if mm := cfg.Modules.MetaManager; mm != nil {
		if mm.RemoteQueryTimeout != metamanagerconfig.GetRemoteQueryTimeout() {
			metamanagerconfig.SetRemoteQueryTimeout(mm.RemoteQueryTimeout)
			klog.Infof("hot reload: metamanager remoteQueryTimeout updated to %ds", mm.RemoteQueryTimeout)
		}
	}
}
