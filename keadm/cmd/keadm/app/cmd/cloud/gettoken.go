package cloud

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	gettokenLongDescription = `
"keadm gettoken" command prints the token to use for establishing bidirectional trust between edge nodes and cloudcore.
A token can be used when a edge node is about to join the cluster. With this token the cloudcore then approve the
certificate request.
`
	gettokenExample = `
keadm gettoken --kube-config /root/.kube/config
- kube-config is the absolute path of kubeconfig which used to build secure connectivity between keadm and kube-apiserver
to get the token.
`
)

// NewGettoken gets the token for edge nodes to join the cluster
func NewGettoken() *cobra.Command {
	init := newGettokenOptions()

	cmd := &cobra.Command{
		Use:     "gettoken",
		Short:   "To get the token for edge nodes to join the cluster",
		Long:    gettokenLongDescription,
		Example: gettokenExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := queryToken(constants.SystemNamespace, common.TokenSecretName, init.Kubeconfig)
			if err != nil {
				fmt.Printf("failed to get token, err is %s\n", err)
				return err
			}
			return showToken(token)
		},
	}
	addGettokenFlags(cmd, init)
	return cmd
}

func addGettokenFlags(cmd *cobra.Command, gettokenOptions *common.GettokenOptions) {
	cmd.Flags().StringVar(&gettokenOptions.Kubeconfig, common.KubeConfig, gettokenOptions.Kubeconfig,
		"Use this key to set kube-config path, eg: $HOME/.kube/config")
}

// newGettokenOptions return common options
func newGettokenOptions() *common.GettokenOptions {
	opts := &common.GettokenOptions{}
	opts.Kubeconfig = common.DefaultKubeConfig
	return opts
}

// queryToken gets token from k8s
func queryToken(namespace string, name string, kubeConfigPath string) ([]byte, error) {
	client, err := util.KubeClient(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	secret, err := client.CoreV1().Secrets(namespace).Get(context.Background(), name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret.Data[common.TokenDataName], nil
}

// showToken prints the token
func showToken(data []byte) error {
	_, err := fmt.Println(string(data))
	if err != nil {
		return err
	}
	return nil
}
