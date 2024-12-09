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

package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/moby/term"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
)

// PodExecOptions defines the options for executing command in edge pod
type PodExecOptions struct {
	Namespace string
	Container string

	// Stdin is true if stdin is to be passed to the container
	Stdin bool
	// Stdout is true if output is passed to stdout
	Stdout bool
	// Stderr is true if output is passed to stderr
	Stderr bool

	// TTY is true if stdin is a TTY
	TTY bool
}

var edgePodExecShortDescription = `Execute command in edge pod`

// NewEdgePodExec returns KubeEdge exec edge pod command.
func NewEdgePodExec() *cobra.Command {
	execOpts := NewEdgePodExecOpts()
	cmd := &cobra.Command{
		Use:   "exec",
		Short: edgePodExecShortDescription,
		Long:  edgePodExecShortDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("no pod specified for exec")
			}
			cmdutil.CheckErr(execOpts.execPod(args))
			return nil
		},
	}
	AddPodExecFlags(cmd, execOpts)
	return cmd
}

func NewEdgePodExecOpts() *PodExecOptions {
	podExecOptions := &PodExecOptions{}
	return podExecOptions
}

func AddPodExecFlags(cmd *cobra.Command, execOpts *PodExecOptions) {
	cmd.Flags().StringVarP(&execOpts.Namespace, common.FlagNameNamespace, "n", "default", "Namespace of the pod")
	cmd.Flags().StringVarP(&execOpts.Container, common.FlagNameContainer, "c", "", "Container name")
	cmd.Flags().BoolVarP(&execOpts.Stdin, common.FlagNameStdin, "i", false, "Pass stdin to the container")
	cmd.Flags().BoolVarP(&execOpts.TTY, common.FlagNameTTY, "t", false, "Allocate a TTY")
}

func (o *PodExecOptions) execPod(args []string) error {
	kubeClient, err := client.KubeClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	pod := args[0]
	commands := args[1:]

	execResponse, err := podExec(ctx, kubeClient, pod, commands, o)
	if err != nil {
		return err
	}
	if execResponse == nil {
		return nil
	}

	for _, runOutMsg := range execResponse.RunOutMessages {
		klog.Info(runOutMsg)
	}
	for _, runErrMsg := range execResponse.RunErrMessages {
		klog.Info(runErrMsg)
	}
	for _, errMsg := range execResponse.ErrMessages {
		klog.Info(errMsg)
	}
	return nil
}

func podExec(ctx context.Context, clientSet *kubernetes.Clientset, pod string, commands []string, o *PodExecOptions) (*types.ExecResponse, error) {
	exexRequest := clientSet.CoreV1().RESTClient().
		Post().
		Namespace(o.Namespace).
		Resource("pods").
		Name(pod).
		SubResource("exec")
	req := addQueryParams(exexRequest, o)

	for _, cmd := range commands {
		req.Param("command", cmd)
	}

	config, err := client.GetKubeConfig()
	if err != nil {
		return nil, err
	}

	if o.TTY {
		stdFd := os.Stdin.Fd()

		restoreFunc, err := disableEcho(stdFd)
		if err != nil {
			return nil, fmt.Errorf("failed to disable echo: %v", err)
		}
		defer restoreFunc()

		exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
		if err != nil {
			return nil, fmt.Errorf("failed to create executor: %v", err)
		}

		err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			Tty:    true,
		})
		if err != nil {
			return nil, fmt.Errorf("error in Stream: %v", err)
		}

		return nil, nil
	}

	result := req.Do(ctx)
	if result.Error() != nil {
		return nil, result.Error()
	}

	statusCode := -1
	result.StatusCode(&statusCode)
	if statusCode != 200 {
		return nil, fmt.Errorf("failed to exec command in pod %s, status code: %d", pod, statusCode)
	}

	body, err := result.Raw()
	if err != nil {
		return nil, err
	}

	var execResponse types.ExecResponse
	err = json.Unmarshal(body, &execResponse)
	if err != nil {
		return nil, err
	}

	return &execResponse, nil
}

func addQueryParams(req *rest.Request, o *PodExecOptions) *rest.Request {
	if o.Container != "" {
		req.Param("container", o.Container)
	}
	if o.Stdin {
		req.Param("stdin", "true")
	}
	if o.Stdout {
		req.Param("stdout", "true")
	}
	if o.Stderr {
		req.Param("stderr", "true")
	}
	if o.TTY {
		req.Param("stdin", "true")
		req.Param("stdout", "true")
		req.Param("tty", "true")
	}
	return req
}

// disableEcho disables echo for the terminal in order to avoid echoing the input characters.
func disableEcho(fd uintptr) (func(), error) {
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}

	return func() {
		_ = term.RestoreTerminal(fd, state)
	}, nil
}
