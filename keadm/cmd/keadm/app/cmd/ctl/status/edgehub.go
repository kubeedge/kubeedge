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

package status

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

type EdgeHubStatusOptions struct {
	NodeName       string
	edgeCoreStatus string
	edgeHubStatus  string
}

var (
	edgeHubStatusShortDescription = `Check EdgeHub status on edge node`
	edgeHubStatusLongDescription  = `Check EdgeHub connection status and health on specified edge node.
This command verifies if EdgeHub is running and properly connected to the cloud.`
	edgeHubStatusExample = `
# Check EdgeHub status for a specific node
keadm ctl status edgehub --node node1

# Check EdgeHub status with detailed output
keadm ctl status edgehub --node edge-node-01
`
)

// NewEdgeHubStatus returns KubeEdge EdgeHub status command.
func NewEdgeHubStatus() *cobra.Command {
	opts := &EdgeHubStatusOptions{}

	cmd := &cobra.Command{
		Use:     "edgehub",
		Short:   edgeHubStatusShortDescription,
		Long:    edgeHubStatusLongDescription,
		Example: edgeHubStatusExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run()
		},
	}

	cmd.Flags().StringVar(&opts.NodeName, "node", "", "Edge node name")
	_ = cmd.MarkFlagRequired("node")

	return cmd
}

// Run executes the EdgeHub status check
func (opts *EdgeHubStatusOptions) Run() error {
	fmt.Printf("Checking EdgeHub status for node: %s\n\n", opts.NodeName)

	// Initialize Kubernetes client once
	kubeClient, err := metaclient.KubeClient()
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client: %v", err)
	}

	// Check if edgecore is running
	if err := opts.checkEdgeCoreStatus(kubeClient); err != nil {
		return fmt.Errorf("failed to check edgecore status: %v", err)
	}

	// Check EdgeHub connection status
	if err := opts.checkEdgeHubConnection(kubeClient); err != nil {
		return fmt.Errorf("failed to check EdgeHub connection: %v", err)
	}

	// Display overall status
	opts.displayOverallStatus()
	return nil
}

// checkEdgeCoreStatus checks if edgecore process is running
func (opts *EdgeHubStatusOptions) checkEdgeCoreStatus(kubeClient kubernetes.Interface) error {
	fmt.Println("Checking EdgeCore process status...")

	ctx := context.Background()

	// Check node status to verify edgecore is running
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, opts.NodeName, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("Failed to get node status: %v\n", err)
		return err
	}

	// Check if node is ready
	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady && condition.Status == v1.ConditionTrue {
			fmt.Println("EdgeCore is running and node is ready")
			opts.edgeCoreStatus = "Running"
			return nil
		}
	}

	fmt.Println("EdgeCore node is not ready")
	opts.edgeCoreStatus = "Not Ready"
	return fmt.Errorf("node not ready")
}

// checkEdgeHubConnection checks EdgeHub connection to cloud
func (opts *EdgeHubStatusOptions) checkEdgeHubConnection(kubeClient kubernetes.Interface) error {
	fmt.Println("Checking EdgeHub connection status...")

	ctx := context.Background()

	// Check pods on the edge node to verify EdgeHub connectivity
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", opts.NodeName),
	})
	if err != nil {
		fmt.Printf("Failed to list pods on node: %v\n", err)
		return err
	}

	runningPods := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == v1.PodRunning {
			runningPods++
		}
	}

	if runningPods > 0 {
		fmt.Printf("EdgeHub connection appears healthy (%d running pods)\n", runningPods)
		opts.edgeHubStatus = "Connected"
		return nil
	}

	fmt.Println("No running pods found - EdgeHub may have connectivity issues")
	opts.edgeHubStatus = "Not Connected"
	return fmt.Errorf("no running pods found on node %s", opts.NodeName)
}

// displayOverallStatus displays the overall status summary
func (opts *EdgeHubStatusOptions) displayOverallStatus() {
	fmt.Println("\nEdgeHub Status Summary:")
	fmt.Printf("Node: %s\n", opts.NodeName)
	fmt.Printf("EdgeCore: %s\n", opts.edgeCoreStatus)
	fmt.Printf("EdgeHub: %s\n", opts.edgeHubStatus)

	// Determine overall status
	if opts.edgeCoreStatus == "Running" && opts.edgeHubStatus == "Connected" {
		fmt.Println("Overall Status: Healthy")
	} else {
		fmt.Println("Overall Status: Unhealthy")
	}
}
