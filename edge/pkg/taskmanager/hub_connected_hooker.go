/*
Copyright 2025 The KubeEdge Authors.

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

package taskmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	daov2 "github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
	"github.com/kubeedge/kubeedge/edge/pkg/taskmanager/actions"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
	upgradeedge "github.com/kubeedge/kubeedge/pkg/upgrade/edge"
)

// ReportUpgradeStatus get the status of upgrade-related commands from upgrade_report.json
// and make corresponding processing according to the status.
func ReportUpgradeStatus(ctx context.Context) error {
	logger := klog.FromContext(ctx).WithName("report-upgrade-status")
	if !upgradeedge.JSONReporterInfoExists() {
		logger.V(1).Info("json reporter info not exists, skip report upgrade status")
		return nil
	}
	info, err := upgradeedge.ParseJSONReporterInfo()
	if err != nil {
		return fmt.Errorf("failed to parse json reporter info, err: %v", err)
	}

	upgrede := daov2.NewUpgrade()
	jobname, nodename, spec, err := upgrede.Get()
	if err != nil {
		return fmt.Errorf("failed to get upgrade record, err: %v", err)
	}
	if jobname == "" || nodename == "" {
		return errors.New("no upgrade record found or invalid info from meta data")
	}

	var action string
	switch info.EventType {
	case upgradeedge.EventTypeUpgrade:
		action = string(operationsv1alpha2.NodeUpgradeJobActionUpgrade)
	case upgradeedge.EventTypeRollback:
		action = string(operationsv1alpha2.NodeUpgradeJobActionRollBack)
	default:
		// To avoid the existence of other event type(backup) report files when EdgeCore restarts,
		// only a message is printed here and no error is returned.
		logger.Info("unsupported event type", "eventType", info.EventType)
		return nil
	}
	res := taskmsg.Resource{
		APIVersion:   operationsv1alpha2.SchemeGroupVersion.String(),
		ResourceType: operationsv1alpha2.ResourceNodeUpgradeJob,
		JobName:      jobname,
		NodeName:     nodename,
	}
	body := taskmsg.UpstreamMessage{
		Action:     action,
		Succ:       info.Success,
		Reason:     info.ErrorMessage,
		FinishTime: time.Now().UTC().Format(time.RFC3339),
		Extend:     taskmsg.FormatNodeUpgradeJobExtend(info.FromVersion, info.ToVersion),
	}
	message.ReportNodeTaskStatus(res, body)

	// After completing the upgrade result report, clean up the upgrade status record file
	if err := upgradeedge.RemoveJSONReporterInfo(); err != nil {
		logger.Error(err, "failed to remove json reporter info")
	}

	// Run rollback after upgrade failed. Rollback command will interrupt the edgecore process,
	// so put it at the end of the function.
	if info.EventType == upgradeedge.EventTypeUpgrade && !info.Success {
		specData, err := json.Marshal(spec)
		if err != nil {
			return fmt.Errorf("failed to marshal spec to json, err: %v", err)
		}
		actions.GetRunner(operationsv1alpha2.ResourceNodeUpgradeJob).
			RunAction(ctx, jobname, nodename, string(operationsv1alpha2.NodeUpgradeJobActionRollBack), specData)
	} else {
		// Remove the upgrade status record when the final action is completed.
		// - The upgrade command is successful;
		// - The rollback command is finished;
		upgradeDao := daov2.NewUpgrade()
		if err := upgradeDao.Delete(); err != nil {
			logger.Error(err, "failed to delete upgrade record")
		}
	}
	return nil
}
