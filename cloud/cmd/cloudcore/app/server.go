/*
Copyright 2019 The KubeEdge Authors.

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

package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1/validation"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/cmd/cloudcore/app/options"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudstream"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudstream/iptables"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/monitor"
	"github.com/kubeedge/kubeedge/cloud/pkg/csrapprovercontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/policycontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/router"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/features"
	"github.com/kubeedge/kubeedge/pkg/util"
	"github.com/kubeedge/kubeedge/pkg/util/flag"
	"github.com/kubeedge/kubeedge/pkg/version"
)

// For testing
var (
	// Functions that can be stubbed for testing
	getHostnameFunc             = util.GetHostname
	getLocalIPFunc              = util.GetLocalIP
	createNamespaceIfNeededFunc = client.CreateNamespaceIfNeeded
)

func NewCloudCoreCommand() *cobra.Command {
	opts := options.NewCloudCoreOptions()
	cmd := &cobra.Command{
		Use: "cloudcore",
		Long: `CloudCore is the core cloud part of KubeEdge, which contains three modules: cloudhub,
edgecontroller, and devicecontroller. Cloudhub is a web server responsible for watching changes at the cloud side,
caching and sending messages to EdgeHub. EdgeController is an extended kubernetes controller which manages
edge nodes and pods metadata so that the data can be targeted to a specific edge node. DeviceController is an extended
kubernetes controller which manages devices so that the device metadata/status date can be synced between edge and cloud.`,
		Run: func(cmd *cobra.Command, args []string) {
			flag.PrintMinConfigAndExitIfRequested(v1alpha1.NewMinCloudCoreConfig())
			flag.PrintDefaultConfigAndExitIfRequested(v1alpha1.NewDefaultCloudCoreConfig())
			flag.PrintFlags(cmd.Flags())

			if errs := opts.Validate(); len(errs) > 0 {
				klog.Exit(util.SpliceErrors(errs))
			}

			config, err := opts.Config()
			if err != nil {
				klog.Exit(err)
			}
			if errs := validation.ValidateCloudCoreConfiguration(config); len(errs) > 0 {
				klog.Exit(util.SpliceErrors(errs.ToAggregate().Errors()))
			}

			if err := features.DefaultMutableFeatureGate.SetFromMap(config.FeatureGates); err != nil {
				klog.Exit(err)
			}

			// start monitor server
			go monitor.ServeMonitor(config.CommonConfig.MonitorServer)

			// To help debugging, immediately log version
			klog.Infof("Version: %+v", version.Get())
			enableImpersonation := config.Modules.CloudHub.Authorization != nil &&
				config.Modules.CloudHub.Authorization.Enable &&
				!config.Modules.CloudHub.Authorization.Debug
			client.InitKubeEdgeClient(config.KubeAPIConfig, enableImpersonation)

			// Negotiate TunnelPort for multi cloudcore instances
			waitTime := rand.Int31n(10)
			time.Sleep(time.Duration(waitTime) * time.Second)
			tunnelport, err := NegotiateTunnelPort()
			if err != nil {
				panic(err)
			}

			config.CommonConfig.TunnelPort = *tunnelport

			if changed := v1alpha1.AdjustCloudCoreConfig(config); changed {
				updateCloudCoreConfigMap(config)
			}

			ctx := beehiveContext.GetContext()
			if features.DefaultFeatureGate.Enabled(features.RequireAuthorization) {
				go csrapprovercontroller.NewCSRApprover(client.GetKubeClient(), informers.GetInformersManager().GetKubeInformerFactory().Certificates().V1().CertificateSigningRequests()).
					Run(5, ctx.Done())
			}

			gis := informers.GetInformersManager()

			registerModules(config)

			if config.Modules.IptablesManager == nil || config.Modules.IptablesManager.Enable && config.Modules.IptablesManager.Mode == v1alpha1.InternalMode {
				// By default, IptablesManager manages tunnel port related iptables rules
				// The internal mode will share the host network, forward to the stream port.
				streamPort := int(config.Modules.CloudStream.StreamPort)
				go iptables.NewIptablesManager(config.KubeAPIConfig, streamPort).Run(ctx)
			}

			// Start all modules
			core.StartModules()
			gis.Start(ctx.Done())
			core.GracefulShutdown()
		},
	}
	fs := cmd.Flags()
	namedFs := opts.Flags()
	flag.AddFlags(namedFs.FlagSet("global"))
	globalflag.AddGlobalFlags(namedFs.FlagSet("global"), cmd.Name())
	for _, f := range namedFs.FlagSets {
		fs.AddFlagSet(f)
	}

	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), namedFs, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFs, cols)
	})

	return cmd
}

// RegisterModules registers all the modules started in cloudcore
// Exported for testing
func RegisterModules(c *v1alpha1.CloudCoreConfig) {
	registerModules(c)
}

// registerModules register all the modules started in cloudcore
func registerModules(c *v1alpha1.CloudCoreConfig) {
	enableAuthorization := c.Modules.CloudHub.Authorization != nil &&
		c.Modules.CloudHub.Authorization.Enable &&
		!c.Modules.CloudHub.Authorization.Debug

	cloudhub.Register(c.Modules.CloudHub)
	edgecontroller.Register(c.Modules.EdgeController)
	devicecontroller.Register(c.Modules.DeviceController)
	taskmanager.Register(c.Modules.TaskManager)
	synccontroller.Register(c.Modules.SyncController)
	cloudstream.Register(c.Modules.CloudStream, c.CommonConfig)
	router.Register(c.Modules.Router)
	dynamiccontroller.Register(c.Modules.DynamicController, enableAuthorization)
	policycontroller.Register(client.CrdConfig)
}

// For testing - allows dependency injection
func NegotiateTunnelPortWithClient(kubeClient kubernetes.Interface) (*int, error) {
	ctx := context.Background()
	err := createNamespaceIfNeededFunc(ctx, constants.SystemNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create system namespace: %v", err)
	}

	tunnelPort, err := kubeClient.CoreV1().ConfigMaps(constants.SystemNamespace).
		Get(ctx, modules.TunnelPort, metav1.GetOptions{})

	if err != nil && !apierror.IsNotFound(err) {
		return nil, err
	}

	hostnameOverride := getHostnameFunc()
	localIP, _ := getLocalIPFunc(hostnameOverride)

	var record iptables.TunnelPortRecord
	if err == nil {
		recordStr, found := tunnelPort.Annotations[modules.TunnelPortRecordAnnotationKey]
		recordBytes := []byte(recordStr)
		if !found {
			return nil, errors.New("failed to get tunnel port record")
		}

		if err := json.Unmarshal(recordBytes, &record); err != nil {
			return nil, err
		}

		port, found := record.IPTunnelPort[localIP]
		if found {
			return &port, nil
		}

		port = NegotiatePortFunc(record.Port)

		record.IPTunnelPort[localIP] = port
		record.Port[port] = true

		recordBytes, err := json.Marshal(record)
		if err != nil {
			return nil, err
		}

		tunnelPort.Annotations[modules.TunnelPortRecordAnnotationKey] = string(recordBytes)

		if _, err := kubeClient.CoreV1().ConfigMaps(constants.SystemNamespace).
			Update(ctx, tunnelPort, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}

		return &port, nil
	}

	if apierror.IsNotFound(err) {
		port := NegotiatePortFunc(record.Port)
		record := iptables.TunnelPortRecord{
			IPTunnelPort: map[string]int{
				localIP: port,
			},
			Port: map[int]bool{
				port: true,
			},
		}
		recordBytes, err := json.Marshal(record)
		if err != nil {
			return nil, err
		}

		_, err = kubeClient.CoreV1().ConfigMaps(constants.SystemNamespace).Create(ctx, &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      modules.TunnelPort,
				Namespace: constants.SystemNamespace,
				Annotations: map[string]string{
					modules.TunnelPortRecordAnnotationKey: string(recordBytes),
				},
			},
		}, metav1.CreateOptions{})

		if err != nil {
			return nil, err
		}

		return &port, nil
	}

	return nil, errors.New("failed to negotiate the tunnel port")
}

var kubeClientGetter = client.GetKubeClient

func NegotiateTunnelPort() (*int, error) {
	return NegotiateTunnelPortWithClient(kubeClientGetter())
}

// UpdateCloudCoreConfigMapWithClient updates the cloudcore configmap with the given client
// Exported for testing
func UpdateCloudCoreConfigMapWithClient(c *v1alpha1.CloudCoreConfig, kubeClient kubernetes.Interface) error {
	cloudCoreCM, err := kubeClient.CoreV1().ConfigMaps(constants.SystemNamespace).Get(context.TODO(), constants.CloudConfigMapName, metav1.GetOptions{})
	if err != nil {
		klog.Warningf("failed to get CloudCore configMap %s/%s: %v", constants.SystemNamespace, constants.CloudConfigMapName, err)
		return err
	}

	configBytes, err := yaml.Marshal(c)
	if err != nil {
		klog.Errorf("Failed to marshal cloudcore config: %v", err)
		return err
	}

	cloudCoreCM.Data["cloudcore.yaml"] = string(configBytes)

	_, err = kubeClient.CoreV1().ConfigMaps(constants.SystemNamespace).Update(context.TODO(), cloudCoreCM, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Failed to update cloudcore config: %v", err)
		return err
	}
	return nil
}

var updateCloudCoreConfigMapWarningf = klog.Warningf

func updateCloudCoreConfigMap(c *v1alpha1.CloudCoreConfig) {
	err := UpdateCloudCoreConfigMapWithClient(c, kubeClientGetter())
	if err != nil {
		updateCloudCoreConfigMapWarningf("Failed to update cloudcore config: %v", err)
	}
}

// NegotiatePortFunc finds the next available port starting from ServerPort
// Exported for testing
func NegotiatePortFunc(portRecord map[int]bool) int {
	for port := constants.ServerPort; ; {
		port++
		if _, found := portRecord[port]; !found {
			return port
		}
	}
}
