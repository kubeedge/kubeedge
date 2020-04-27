package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/common/constants"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	getokenLongDescription = `
"keadm getoken" command prints the token to use for establishing bidirectional trust between edge nodes and cloudcore. 
A token can be used when a edge node is about to join the cluster. With this token the cloudcore then approve the 
certificate request.
`
	getokenExample = `
keadm getoken --kube-config = /root/.kube/config
- kube-config is the absolute path of kubeconfig which used to build secure connectivity between keadm and kube-apiserver
to get the token. 
`
)

func NewGetoken(out io.Writer, init *types.GetokenOptions) *cobra.Command {
	if init == nil {
		init = newGetokenOptions()
	}
	cmd := &cobra.Command{
		Use:        "getoken",
		Short:      "To get the token for edge nodes to join the cluster",
		Long:       getokenLongDescription,
		Example:	getokenLongDescription,
		RunE:		func (cmd *cobra.Command, args []string) error {
						token, err := queryToken(util.NameSpaceCloudCore, util.TokenSecretName, init.Kubeconfig)
						if err != nil {
							fmt.Println("failed to get token")
							return err
						}
						return showToken(token, out)
					},
	}
	addGetokenFlags(cmd, init)
	return cmd
}


func addGetokenFlags(cmd *cobra.Command, getokenOptions *types.GetokenOptions) {
	cmd.Flags().StringVar(&getokenOptions.Kubeconfig, types.KubeConfig, getokenOptions.Kubeconfig,
		"Use this key to set kube-config path, eg: $HOME/.kube/config")
}
//
func newGetokenOptions() *types.GetokenOptions {
	opts := &types.GetokenOptions{}
	opts.Kubeconfig = constants.DefaultKubeConfig
	return opts
}

// queryToken gets token from k8s
func queryToken(namespace string, name string, kubeConfigPath string) ([]byte, error) {
	client, err := kubeClient(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	secret, err := client.CoreV1().Secrets(namespace).Get(name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret.Data[util.TokenDataName], nil
}

// showToken prints the token
func showToken(data []byte, out io.Writer) error{
	_, err := out.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func kubeConfig(kubeconfigPath string) (conf *rest.Config, err error){
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	kubeConfig.QPS = float32(constants.DefaultKubeQPS)
	kubeConfig.Burst = int(constants.DefaultKubeBurst)
	kubeConfig.ContentType = constants.DefaultKubeContentType

	return kubeConfig, nil
}

// KubeClient from config
func kubeClient(kubeConfigPath string) (*kubernetes.Clientset, error) {
	kubeConfig, err := kubeConfig(kubeConfigPath)
	if err != nil {
		klog.Warningf("get kube config failed with error: %s", err)
		return nil, err
	}
	return kubernetes.NewForConfig(kubeConfig)
}