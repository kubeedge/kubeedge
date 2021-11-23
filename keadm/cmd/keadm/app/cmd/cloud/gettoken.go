package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	gettokenLongDescription = `Outputs the cloudcore token for edge nodes to join the cluster with "keadm join".

This command prints the token to use for establishing bidirectional trust between the edge nodes and cloudcore.
With this token, cloudcore will approve the certificate request from the edge nodes.`
	gettokenExample = `# Obtaining the token
keadm gettoken --kube-config /root/.kube/config

# From the edge node
keadm join <token>`
)

// NewGettoken gets the token for edge nodes to join the cluster
func NewGettoken() *cobra.Command {
	init := &common.GettokenOptions{
		Kubeconfig: common.DefaultKubeConfig,
	}

	cmd := &cobra.Command{
		Use:     "gettoken",
		Args:    cobra.NoArgs,
		Short:   "Outputs the cloudcore token.",
		Long:    gettokenLongDescription,
		Example: gettokenExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := util.KubeClient(init.Kubeconfig)
			if err != nil {
				fmt.Fprint(os.Stderr, err)
				os.Exit(1)
			}
			token, err := queryToken(client, constants.SystemNamespace, common.TokenSecretName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to get the cloudcore token: %v", err)
				os.Exit(1)
			}
			_, err = fmt.Printf("%s", string(token))
			if err != nil {
				fmt.Fprint(os.Stderr, err)
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&init.Kubeconfig, common.KubeConfig, init.Kubeconfig,
		"The absolute path to the kubeconfig file (default /root/.kube/config)")

	return cmd
}

// queryToken gets token from k8s
func queryToken(kubeClient kubernetes.Interface, namespace string, name string) ([]byte, error) {
	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(context.Background(), name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret.Data[common.TokenDataName], nil
}
