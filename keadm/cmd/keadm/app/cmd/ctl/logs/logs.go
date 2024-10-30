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

package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
)

var edgePodLogsShortDescription = `Get pod logs in edge node`

// PodLogsOptions defines the options for getting logs of edge pod
type PodLogsOptions struct {
	Namespace string
	Container string
	// Follow is true if the logs should be streamed
	Follow bool
	// Previous is true if print the logs for the previous instance of the container in a pod(if it exists).
	Previous bool
	// SinceSecond is the duration in seconds for which to return logs newer than a relative duration like 5s, 2m, or 3h. Only one of sinceSeconds or sinceTime may be specified.
	SinceSecond string
	// SinceTime is the RFC3339 date for which to return logs newer than this date. Only one of sinceSeconds or sinceTime may be specified.
	SinceTime string
	// Timestamps is true if include timestamps on each line in the log output
	Timestamps bool
	// TailLines is the lines of recent log file to display.
	TailLines string
	// LimitBytes is the maximum bytes of logs to return.
	LimitBytes string
	// InsecureSkipTLSVerifyBackend is true if the server's certificate will not be checked for validity.
	InsecureSkipTLSVerifyBackend bool
}

func NewEdgePodLogs() *cobra.Command {
	logsOpts := NewLogsPodOpts()
	cmd := &cobra.Command{
		Use:   "logs",
		Short: edgePodLogsShortDescription,
		Long:  edgePodLogsShortDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("no pod specified for logs")
			}
			cmdutil.CheckErr(logsOpts.getPodLogs(args))
			return nil
		},
	}
	AddLogsPodFlags(cmd, logsOpts)
	return cmd
}

func NewLogsPodOpts() *PodLogsOptions {
	podLogsOptions := &PodLogsOptions{}
	podLogsOptions.Namespace = "default"
	return podLogsOptions
}

func AddLogsPodFlags(cmd *cobra.Command, podLogsOptions *PodLogsOptions) {
	cmd.Flags().StringVarP(&podLogsOptions.Namespace, common.FlagNameNamespace, "n", podLogsOptions.Namespace, "If present, the namespace scope for this CLI request")
	cmd.Flags().StringVarP(&podLogsOptions.Container, common.FlagNameContainer, "c", "", "Print the logs of this container")
	cmd.Flags().BoolVarP(&podLogsOptions.Follow, common.FlagNameFollow, "f", false, "Specify if the logs should be streamed")
	cmd.Flags().BoolVarP(&podLogsOptions.Previous, common.FlagNamePrevious, "p", false, "If true, print the logs for the previous instance of the container in a pod if it exists")
	cmd.Flags().StringVar(&podLogsOptions.SinceSecond, common.FlagNameSinceSecond, "0", "Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs.")
	cmd.Flags().StringVar(&podLogsOptions.SinceTime, common.FlagNameSinceTime, "", "Only return logs after a specific date (RFC3339). Defaults to all logs.")
	cmd.Flags().BoolVar(&podLogsOptions.Timestamps, common.FlagNameTimestamps, false, "Include timestamps on each line in the log output")
	cmd.Flags().StringVar(&podLogsOptions.TailLines, common.FlagNameTailLines, "-1", "Lines of recent log file to display. Defaults to -1 with no selector, showing all log lines otherwise 10, if a selector is provided.")
	cmd.Flags().StringVar(&podLogsOptions.LimitBytes, common.FlagNameLimitBytes, "-1", "Maximum bytes of logs to return. Defaults to no limit.")
	cmd.Flags().BoolVar(&podLogsOptions.InsecureSkipTLSVerifyBackend, common.FlagNameInsecureSkipTLSVerifyBackend, false, "If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure.")
}

func (o *PodLogsOptions) getPodLogs(args []string) error {
	kubeClient, err := client.KubeClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	logsResponse, err := logsPod(ctx, kubeClient, args[0], o)
	if err != nil {
		return err
	}

	for _, logMsg := range logsResponse.LogMessages {
		klog.Info(logMsg)
	}

	for _, errMsg := range logsResponse.ErrMessages {
		klog.Info(errMsg)
	}

	return nil
}

func logsPod(ctx context.Context, clientSet *kubernetes.Clientset, pod string, o *PodLogsOptions) (*types.LogsResponse, error) {
	logRequest := clientSet.CoreV1().RESTClient().
		Get().
		Namespace(o.Namespace).
		Resource("pods").
		Name(pod).
		SubResource("log")

	req := addQueryParams(logRequest, o)

	if o.Follow {
		// Stream logs
		logStream, err := req.Stream(context.TODO())
		if err != nil {
			return nil, err
		}
		defer logStream.Close()

		if _, err := io.Copy(os.Stdout, logStream); err != nil {
			return nil, err
		}
	}
	result := req.Do(ctx)

	if result.Error() != nil {
		return nil, result.Error()
	}

	statusCode := -1
	result.StatusCode(&statusCode)
	if statusCode != 200 {
		return nil, fmt.Errorf("failed to get logs for pod %s, status code: %d", pod, statusCode)
	}

	body, err := result.Raw()
	if err != nil {
		return nil, err
	}

	var logsResponse types.LogsResponse
	err = json.Unmarshal(body, &logsResponse)
	if err != nil {
		return nil, err
	}

	return &logsResponse, nil
}

func addQueryParams(req *rest.Request, o *PodLogsOptions) *rest.Request {
	if o.Container != "" {
		req.Param("container", o.Container)
	}
	if o.Follow {
		req.Param("follow", fmt.Sprintf("%v", o.Follow))
	}
	if o.Previous {
		req.Param("previous", fmt.Sprintf("%v", o.Previous))
	}
	if o.SinceSecond != "0" {
		req.Param("sinceSeconds", fmt.Sprintf("%v", o.SinceSecond))
	}
	if o.SinceTime != "" {
		req.Param("sinceTime", fmt.Sprintf("%v", o.SinceTime))
	}
	if o.Timestamps {
		req.Param("timestamps", fmt.Sprintf("%v", o.Timestamps))
	}
	if o.TailLines != "-1" {
		req.Param("tailLines", fmt.Sprintf("%v", o.TailLines))
	}
	if o.LimitBytes != "-1" {
		req.Param("limitBytes", fmt.Sprintf("%v", o.LimitBytes))
	}
	if o.InsecureSkipTLSVerifyBackend {
		req.Param("insecureSkipTLSVerifyBackend", fmt.Sprintf("%v", o.InsecureSkipTLSVerifyBackend))
	}

	return req
}
