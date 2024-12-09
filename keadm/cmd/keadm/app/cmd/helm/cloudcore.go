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
package helm

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/blang/semver"
	"helm.sh/helm/v3/pkg/action"
	helmcli "helm.sh/helm/v3/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/kubeedge/common/constants"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	kecharts "github.com/kubeedge/kubeedge/manifests"
)

const (
	cloudCoreHelmComponent = "cloudcore"

	dirCharts   = "charts"
	dirProfiles = "profiles"

	valuesFileName = "values.yaml"

	messageFormatInstallationSuccess = `%s started
=========CHART DETAILS=======
Name: %s
LAST DEPLOYED: %s
NAMESPACE: %s
STATUS: %s
REVISION: %d
`
	messageFormatFinalValues = "FINAL VALUES:\n%s"

	messageFormatUpgradationPrintConfig = `This is cloudcore configuration of the previous version.
If you want to revert configuration items, please manually modify the configmap 'cloudcore' 
and restart the cloudcore:
%s`
)

const (
	defaultHelmInstall  = true
	defaultHelmWait     = true
	defaultHelmCreateNs = true
)

var setsKeyImageTags = []string{"cloudCore.image.tag", "iptablesManager.image.tag", "controllerManager.image.tag"}
var setsKeyImageRepositories = map[string]string{"cloudCore.image.repository": "cloudcore", "iptablesManager.image.repository": "iptables-manager", "controllerManager.image.repository": "controller-manager"}

var helmSettings = helmcli.New()

// CloudCoreHelmTool a cloudcore helm chart operation wrapping tool
type CloudCoreHelmTool struct {
	util.Common
}

// NewCloudCoreHelmTool creates a new instance of CloudCoreHelmTool
func NewCloudCoreHelmTool(kubeConfig, kubeedgeVersion string) *CloudCoreHelmTool {
	common := util.Common{
		ToolVersion:     semver.MustParse(util.GetHelmVersion(kubeedgeVersion, util.RetryTimes)),
		KubeConfig:      kubeConfig,
		OSTypeInstaller: util.GetOSInterface(),
	}
	common.OSTypeInstaller.SetKubeEdgeVersion(common.ToolVersion)
	return &CloudCoreHelmTool{
		Common: common,
	}
}

// Install uses helm client to install cloudcore release
func (c *CloudCoreHelmTool) Install(opts *types.InitOptions) error {
	ver, err := util.GetCurrentVersion(opts.KubeEdgeVersion)
	if err != nil {
		return fmt.Errorf("failed to get version with err:%v", err)
	}
	opts.KubeEdgeVersion = ver

	// The flag --force would not care about whether the cloud components exist or not also.
	// If gives a external helm root, no need to check and verify, because it is always not a cloudcore.
	if !opts.Force && opts.ExternalHelmRoot == "" {
		if err := c.verifyCloudCoreProcessRunning(); err != nil {
			return err
		}
	}
	if err := c.Common.OSTypeInstaller.IsK8SComponentInstalled(c.Common.KubeConfig, c.Common.Master); err != nil {
		return fmt.Errorf("failed to verify k8s component installed, err: %v", err)
	}

	fmt.Printf("Kubernetes version verification passed, KubeEdge %s installation will start...\n", opts.KubeEdgeVersion)

	appendDefaultSets(opts.KubeEdgeVersion, opts.AdvertiseAddress, &opts.CloudInitUpdateBase)
	// Load profile values, and merges the sets flag
	var vals map[string]interface{}
	if opts.Profile != "" {
		// Load extern values, and merges the sets flag
		vals, err = MergeExternValues(opts.Profile, opts.GetValidSets())
		if err != nil {
			return err
		}
	} else {
		valueOpts := &Options{
			Values: opts.GetValidSets(),
		}
		vals, err = valueOpts.MergeValues()
		if err != nil {
			return err
		}
	}

	// TODO: think about how to support addons, and should we support addons?
	subDir := path.Join(dirCharts, cloudCoreHelmComponent)
	componentName := cloudCoreHelmComponent

	// Build a new renderer instance
	renderer := NewGenericRenderer(kecharts.BuiltinOrDir(opts.ExternalHelmRoot),
		subDir, componentName, constants.SystemNamespace, vals, opts.SkipCRDs)
	// Load the charts to this renderer
	if err := renderer.LoadChart(); err != nil {
		return fmt.Errorf("cannot load the given charts %s, error: %s", renderer.componentName, err.Error())
	}

	helper, err := NewHelper(opts.KubeConfig, constants.SystemNamespace)
	if err != nil {
		return err
	}
	// Determine whether the cloudcore release has been installed
	if rel, err := helper.GetRelease(renderer.componentName); err != nil {
		return err
	} else if rel != nil {
		return fmt.Errorf("the cloudcore release already exists, and you can upgrade the cloudcore using the `keadm upgrade`")
	}

	// Install the helm release cloudcore
	client := action.NewInstall(helper.GetConfig())
	client.DryRun = opts.DryRun
	client.CreateNamespace = defaultHelmCreateNs
	client.ReleaseName = renderer.componentName
	client.Namespace = constants.SystemNamespace
	// If the flag force is true, don't wait for the command result of helm install
	if !opts.Force {
		client.Wait = defaultHelmWait
		client.Timeout = DefaultHelmTimeout
	}
	rel, err := client.Run(renderer.chart, vals)
	if err != nil {
		return fmt.Errorf("failed to install release %s, err: %v",
			renderer.componentName, err)
	}

	// Print installation successful message
	var lastDeployed string
	if !rel.Info.LastDeployed.IsZero() {
		lastDeployed = rel.Info.LastDeployed.Format(time.ANSIC)
	}
	fmt.Printf(messageFormatInstallationSuccess,
		strings.ToTitle(renderer.componentName),
		rel.Name,
		lastDeployed,
		rel.Namespace,
		rel.Info.Status.String(),
		rel.Version)
	if opts.PrintFinalValues {
		cfgyml, err := yaml.Marshal(rel.Config)
		if err != nil {
			klog.Warningf("failed to marshal values, err: %v", err)
		}
		fmt.Printf(messageFormatFinalValues, string(cfgyml))
	}
	return nil
}

// Upgrade uses helm client to upgrade cloudcore release
func (c *CloudCoreHelmTool) Upgrade(opts *types.CloudUpgradeOptions) error {
	ver, err := util.GetCurrentVersion(opts.KubeEdgeVersion)
	if err != nil {
		return fmt.Errorf("failed to get version with err:%v", err)
	}
	opts.KubeEdgeVersion = ver

	if err := c.Common.OSTypeInstaller.IsK8SComponentInstalled(c.Common.KubeConfig, c.Common.Master); err != nil {
		return fmt.Errorf("failed to verify k8s component installed, err: %v", err)
	}

	fmt.Println("Kubernetes version verification passed, KubeEdge upgradation will start...")

	cloudcoreConfig, err := getCloudcoreHistoryConfig(opts.KubeConfig, constants.SystemNamespace)
	if err != nil {
		return fmt.Errorf("failed to get cloudcore history config, err: %v", err)
	}

	componentName := cloudCoreHelmComponent
	subDir := path.Join(dirCharts, cloudCoreHelmComponent)

	helper, err := NewHelper(opts.KubeConfig, constants.SystemNamespace)
	if err != nil {
		return err
	}
	// Determine whether the cloudcore release has been installed
	if rel, err := helper.GetRelease(componentName); err != nil {
		return err
	} else if rel == nil {
		return fmt.Errorf("the cloudcore release not found, and you can init the cloudcore using the `keadm init`")
	}

	appendDefaultSets(opts.KubeEdgeVersion, opts.AdvertiseAddress, &opts.CloudInitUpdateBase)

	var vals map[string]interface{}
	if len(opts.ValueFiles) == 0 && opts.Profile != "" {
		// Load profile values, and merges the sets flag
		vals, err = MergeExternValues(opts.Profile, opts.GetValidSets())
		if err != nil {
			return err
		}
	} else {
		valueOpts := &Options{
			ValueFiles: opts.ValueFiles,
			Values:     opts.GetValidSets(),
		}
		vals, err = valueOpts.MergeValues()
		if err != nil {
			return err
		}
	}

	// Build a new renderer instance
	renderer := NewGenericRenderer(kecharts.BuiltinOrDir(""),
		subDir, componentName, constants.SystemNamespace, vals, false)
	// Load the charts to this renderer
	if err := renderer.LoadChart(); err != nil {
		return fmt.Errorf("cannot load the given charts %s, err: %s", renderer.componentName, err.Error())
	}

	// Upgrade the helm release cloudcore
	client := action.NewUpgrade(helper.GetConfig())
	client.ReuseValues = opts.ReuseValues
	client.DryRun = opts.DryRun
	// If the flag force is true, don't wait for the command result of helm upgrade
	if !opts.Force {
		client.Wait = defaultHelmWait
		client.Timeout = DefaultHelmTimeout
	}
	rel, err := client.Run(renderer.componentName, renderer.chart, vals)
	if err != nil {
		return fmt.Errorf("failed to upgrade release %s, err: %v", renderer.componentName, err)
	}

	// Print upgradation successful message
	var lastDeployed string
	if !rel.Info.LastDeployed.IsZero() {
		lastDeployed = rel.Info.LastDeployed.Format(time.ANSIC)
	}
	fmt.Printf(messageFormatInstallationSuccess,
		strings.ToTitle(renderer.componentName),
		rel.Name,
		lastDeployed,
		rel.Namespace,
		rel.Info.Status.String(),
		rel.Version)

	if opts.PrintFinalValues {
		cfgyml, err := yaml.Marshal(rel.Config)
		if err != nil {
			klog.Warningf("failed to marshal values, err: %v", err)
		}
		fmt.Printf(messageFormatFinalValues, string(cfgyml))
	}

	fmt.Printf(messageFormatUpgradationPrintConfig, cloudcoreConfig)
	return nil
}

// Uninstall uses helm client to uninstall cloudcore release
func (c *CloudCoreHelmTool) Uninstall(opts *types.ResetOptions) error {
	// clean kubeedge namespace
	if err := c.Common.CleanNameSpace(constants.SystemNamespace, opts.Kubeconfig); err != nil {
		return fmt.Errorf("failed to clean kubeedge namespace, err: %v", err)
	}
	return nil
}

func (c *CloudCoreHelmTool) verifyCloudCoreProcessRunning() error {
	cloudcoreRunning, err := c.Common.OSTypeInstaller.IsKubeEdgeProcessRunning(util.KubeCloudBinaryName)
	if err != nil {
		return fmt.Errorf("failed to verify the cloudcore binnary already running, err: %v", err)
	}
	if cloudcoreRunning {
		return fmt.Errorf("the cloudcore is already running on this node, please run reset to clean up first")
	}
	return nil
}

// appendDefaultSets sets some default values configuration to via --sets
func appendDefaultSets(version, advertiseAddress string, opts *types.CloudInitUpdateBase) {
	if version != "" {
		for _, k := range setsKeyImageTags {
			if !opts.HasSets(k) {
				opts.Sets = append(opts.Sets, fmt.Sprintf("%s=%s", k, version))
			}
		}
	}
	if opts.ImageRepository != "" {
		opts.ImageRepository = strings.TrimSuffix(opts.ImageRepository, "/")
		for k, v := range setsKeyImageRepositories {
			if !opts.HasSets(k) {
				opts.Sets = append(opts.Sets, fmt.Sprintf("%s=%s", k, opts.ImageRepository+"/"+v))
			}
		}
	}
	if advertiseAddress != "" {
		for index, addr := range strings.Split(advertiseAddress, ",") {
			opts.Sets = append(opts.Sets, fmt.Sprintf("%s[%d]=%s",
				"cloudCore.modules.cloudHub.advertiseAddress", index, addr))
		}
	}
}

// getCloudcoreHistoryConfig ...
func getCloudcoreHistoryConfig(kubeconfig, namespace string) (string, error) {
	kcli, err := util.KubeClient(kubeconfig)
	if err != nil {
		return "", err
	}
	cm, err := kcli.CoreV1().ConfigMaps(namespace).
		Get(context.TODO(), "cloudcore", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get configmap cloudcore, err: %v", err)
	}
	return cm.Data["cloudcore.yaml"], nil
}

// getValuesFile ...
func getValuesFile(profileKey string) string {
	pf := profileKey
	if !strings.HasSuffix(profileKey, ".yaml") {
		pf += ".yaml"
	}
	return path.Join(dirProfiles, pf)
}
