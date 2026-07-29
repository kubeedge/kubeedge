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

package edge

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const edgeCoreService = "edgecore.service"

// hotReloadHealthWindow is how long we watch edgecore after a hot reload
// signal before declaring the new configuration healthy, and
// hotReloadHealthPoll is how often we check during that window. Both are
// vars, rather than consts, so tests can shrink them.
var (
	hotReloadHealthWindow = 10 * time.Second
	hotReloadHealthPoll   = time.Second
)

// signalEdgeCoreReload and edgeCoreServiceState are indirections over the
// systemctl calls hotReload makes, so tests can exercise its retry and
// rollback logic without spawning real processes.
var (
	signalEdgeCoreReload = func() error {
		return exec.Command("sudo", "systemctl", "kill", "-s", "HUP", edgeCoreService).Run()
	}
	edgeCoreServiceState = func() (string, error) {
		out, err := exec.Command("systemctl", "is-active", edgeCoreService).Output()
		return strings.TrimSpace(string(out)), err
	}
)

// hotReloadableFields lists the dotted config keys, matched case
// insensitively, that edgecore already reads fresh from memory on every use.
// Applying only fields from this set never requires restarting edgecore, so
// keadm can signal a reload instead of restarting the process.
//
// A field only belongs on this list once the corresponding module has been
// verified to re-read it on every use rather than caching it once at
// startup; anything else keeps going through the regular restart path.
var hotReloadableFields = map[string]bool{
	"modules.edgehub.heartbeat":              true,
	"modules.metamanager.remotequerytimeout": true,
}

// isHotReloadable reports whether every "key=value" pair in the comma
// separated sets string is safe to apply without restarting edgecore.
func isHotReloadable(sets string) bool {
	sets = strings.TrimSpace(sets)
	if sets == "" {
		return false
	}

	fields := strings.Split(sets, ",")
	for _, field := range fields {
		key, _, ok := strings.Cut(strings.TrimSpace(field), "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return false
		}
		if !hotReloadableFields[strings.ToLower(key)] {
			return false
		}
	}
	return true
}

// hotReload asks the running edgecore process to reload its configuration
// in place and then watches it over hotReloadHealthWindow to confirm it
// stayed healthy. The caller is expected to roll back and fall back to a
// full restart when hotReload returns an error.
func hotReload() error {
	if err := signalEdgeCoreReload(); err != nil {
		return fmt.Errorf("failed to signal edgecore to reload configuration: %v", err)
	}

	deadline := time.Now().Add(hotReloadHealthWindow)
	for time.Now().Before(deadline) {
		time.Sleep(hotReloadHealthPoll)
		if !edgeCoreActive() {
			return fmt.Errorf("edgecore.service is not active after the configuration reload")
		}
	}
	return nil
}

// edgeCoreActive reports whether edgecore.service is currently active.
func edgeCoreActive() bool {
	state, err := edgeCoreServiceState()
	if err != nil {
		return false
	}
	return state == "active"
}
