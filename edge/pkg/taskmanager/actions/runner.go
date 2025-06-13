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

package actions

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
)

// runners is a global map variables,
// used to cache the implementation of the job action runner.
var runners = map[string]*ActionRunner{}

// Init registers the node job action runner.
func Init() {
	RegisterRunner(operationsv1alpha2.ResourceImagePrePullJob, newImagePrePullJobRunner())
	RegisterRunner(operationsv1alpha2.ResourceConfigUpdateJob, newConfigUpdateJobRunner())
	RegisterRunner(operationsv1alpha2.ResourceConfigUpdateJob, newNodeUpgradeJobRunner())
}

// registerRunner registers the implementation of the job action runner.
func RegisterRunner(name string, runner *ActionRunner) {
	runners[name] = runner
}

// GetRunner returns the implementation of the job action runner.
func GetRunner(name string) *ActionRunner {
	return runners[name]
}

type ActionResponse interface {
	// Error returns an error if the task run fails, otherwise return nil
	Error() error
	// NeedInterrupt returns whether the action flow needs to be interrupted.
	// If true, the action flow will be interrupted.
	NeedInterrupt() bool
}

type baseActionResponse struct {
	err       error
	interrupt bool
}

// Check whether baseActionResponse implements ActionResponse interface.
var _ ActionResponse = (*baseActionResponse)(nil)

func (resp baseActionResponse) Error() error {
	return resp.err
}

func (resp baseActionResponse) NeedInterrupt() bool {
	return resp.interrupt
}

// ActionFun defines the function type of the job action handler.
type ActionFun = func(
	ctx context.Context,
	jobname, nodename string,
	specser SpecSerializer,
) ActionResponse

// RunnHookFunc defines the function type of the job runner hook.
type RunnHookFunc func(
	ctx context.Context,
	jobname, nodename, action string,
	specser SpecSerializer,
) error

// baseActionRunner defines the abstruct of the job action runner.
// The implementation of ActionRunner must compose this structure.
type ActionRunner struct {
	// actions defines the function implementation of each action.
	Actions map[string]ActionFun
	// flow defines the action flow of node job.
	Flow *actionflow.Flow
	// PreRun defines the pre-run hook of the job runner.
	PreRun RunnHookFunc
	// PostRun defines the post-run hook of the job runner.
	PostRun RunnHookFunc
	// ReportActionStatus uses to report status of node action. If the err is not nil,
	// the failure status needs to be reported.
	ReportActionStatus func(jobname, nodename, action string, resp ActionResponse)
	// GetSpecSerializer returns serializer for parse the spec data.
	GetSpecSerializer func(specData []byte) (SpecSerializer, error)
	// Logger define a logger in the specified format to print information.
	Logger logr.Logger
}

// Add job action runner to runners.
func (r *ActionRunner) addAction(action string, handler ActionFun) {
	if r.Actions == nil {
		r.Actions = make(map[string]ActionFun)
	}
	r.Actions[action] = handler
}

// Get job action runner from runners, returns error when not found.
func (r *ActionRunner) mustGetAction(action string) (ActionFun, error) {
	actionFn, ok := r.Actions[action]
	if !ok {
		return nil, fmt.Errorf("invalid job action %s", action)
	}
	return actionFn, nil
}

// RunAction runs the job action.
func (r *ActionRunner) RunAction(ctx context.Context, jobname, nodename, action string, specData []byte) {
	logger := r.Logger.WithValues("job", jobname)
	ser, err := r.GetSpecSerializer(specData)
	if err != nil {
		logger.Error(err, "failed to get spec serializer, report to cloud")
		r.ReportActionStatus(jobname, nodename, action, &baseActionResponse{err: err})
		return
	}

	act := r.Flow.Find(action)
	if r.PreRun != nil {
		if err := r.PreRun(ctx, jobname, nodename, action, ser); err != nil {
			logger.Error(err, "failed to run pre-run hook, report to cloud")
			r.ReportActionStatus(jobname, nodename, action, &baseActionResponse{err: err})
			return
		}
	}
	for {
		logger.V(1).Info("run action", "action", act.Name)
		actionFn, err := r.mustGetAction(act.Name)
		if err != nil {
			logger.Error(err, "failed to get action handler, report to cloud")
			r.ReportActionStatus(act.Name, jobname, nodename, &baseActionResponse{err: err})
			return
		}
		resp := actionFn(ctx, jobname, nodename, ser)
		r.ReportActionStatus(jobname, nodename, act.Name, resp)
		if err := resp.Error(); err != nil {
			logger.Error(err, "run action failed", "action", act.Name)
			if next := act.NextFailure; next != nil && !resp.NeedInterrupt() {
				logger.V(1).Info("run next failure action")
				act = next
				continue
			}
			break
		}
		if resp.NeedInterrupt() {
			logger.V(1).Info("action needs to be interrupted", "action", act.Name)
			break
		}
		next := act.NextSuccessful
		if next == nil {
			logger.V(1).Info("action is the last item of the successful line, stop running", "action", act.Name)
			break
		}
		act = next
	}
	if act.NextSuccessful == nil && r.PostRun != nil {
		if err := r.PostRun(ctx, jobname, nodename, action, ser); err != nil {
			logger.Error(err, "failed to run pre-run hook, report to cloud")
		}
	}
}
