/*
Copyright 2022 The KubeEdge Authors.

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

package cmd

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/kubeedge/common/constants"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/apis/upgrade/v1alpha2"
)

// NewEdgeUpgrade returns KubeEdge edge upgrade command.
func NewEdgeUpgrade() *cobra.Command {
	upgradeOptions := newUpgradeOptions()

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade edge component. Upgrade the edge node to the right version.",
		Long:  "Upgrade edge component. Upgrade the edge node to the right version.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// upgrade edgecore
			return upgradeOptions.upgrade()
		},
	}

	addUpgradeFlags(cmd, upgradeOptions)
	return cmd
}

// newJoinOptions returns a struct ready for being used for creating cmd join flags.
func newUpgradeOptions() *UpgradeOptions {
	opts := &UpgradeOptions{}
	opts.ToVersion = "v" + common.DefaultKubeEdgeVersion
	opts.Config = constants.DefaultConfigDir + "edgecore.yaml"

	return opts
}

func (up *UpgradeOptions) upgrade() error {
	// get EdgeCore configuration from edgecore.yaml config file
	data, err := os.ReadFile(up.Config)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %v", up.Config, err)
	}

	configure := &v1alpha1.EdgeCoreConfig{}
	err = yaml.Unmarshal(data, configure)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file %s: %v", up.Config, err)
	}

	upgrade := Upgrade{
		UpgradeID:      up.UpgradeID,
		FromVersion:    up.FromVersion,
		ToVersion:      up.ToVersion,
		EdgeCoreConfig: configure,
		Status:         string(v1alpha2.UpgradeUpgrading),
	}

	defer func() {
		// report upgrade result to cloudhub
		if err := upgrade.reportUpgradeResult(); err != nil {
			klog.Errorf("failed to report upgrade result to cloud: %v", err)
		}
	}()

	// run script to do upgrade operation
	err = upgrade.PreProcess()
	if err != nil {
		upgrade.UpdateStatus(string(v1alpha2.UpgradeFailedRollbackSuccess))
		upgrade.UpdateFailureReason(fmt.Sprintf("upgrade error: %v", err))
		return fmt.Errorf("upgrade pre process failed: %v", err)
	}

	err = upgrade.Process()
	if err != nil {
		rbErr := upgrade.Rollback()
		if rbErr != nil {
			upgrade.UpdateStatus(string(v1alpha2.UpgradeFailedRollbackFailed))
			upgrade.UpdateFailureReason(fmt.Sprintf("upgrade error: %v, rollback error: %v", err, rbErr))
		} else {
			upgrade.UpdateStatus(string(v1alpha2.UpgradeFailedRollbackSuccess))
			upgrade.UpdateFailureReason(fmt.Sprintf("upgrade error: %v", err))
		}
		return fmt.Errorf("upgrade process failed: %v", err)
	}

	upgrade.UpdateStatus(string(v1alpha2.UpgradeSuccess))

	return nil
}

func (up *Upgrade) WriteHelperScripts() error {
	scripts := []string{"log_util.sh", "common.sh"}
	for _, script := range scripts {
		content, err := BuiltinFile(script)
		if err != nil {
			return fmt.Errorf("failed to read %v: %v", script, err)
		}

		err = os.WriteFile(filepath.Join(util.KubeEdgeUpgradePath, up.ToVersion, script), content, 0750)
		if err != nil {
			return fmt.Errorf("failed to write data to %v: %v", script, err)
		}
	}

	return nil
}

func (up *Upgrade) PreProcess() error {
	klog.Infof("upgrade preprocess start")

	if err := os.MkdirAll(filepath.Join(util.KubeEdgeUpgradePath, up.ToVersion), 0755); err != nil {
		return fmt.Errorf("mkdirall failed: %v", err)
	}

	if err := up.WriteHelperScripts(); err != nil {
		return err
	}

	content, err := BuiltinFile("preprocess.sh")
	if err != nil {
		return fmt.Errorf("failed to read preprocess.sh: %v", err)
	}

	err = os.WriteFile(filepath.Join(util.KubeEdgeUpgradePath, up.ToVersion, "preprocess.sh"), content, 0750)
	if err != nil {
		return fmt.Errorf("failed to write data to preprocess.sh: %v", err)
	}

	preCmd := fmt.Sprintf("export FROM_VERSION=%s && export TO_VERSION=%s && %s",
		up.FromVersion, up.ToVersion, filepath.Join(util.KubeEdgeUpgradePath, up.ToVersion, "preprocess.sh"))
	cmd := util.NewCommand(preCmd)
	err = cmd.Exec()
	if err != nil {
		return fmt.Errorf("failed to exec preprocess script: %v", err)
	}

	return nil
}
func (up *Upgrade) Process() error {
	klog.Infof("upgrade process start")

	content, err := BuiltinFile("upgrade.sh")
	if err != nil {
		return fmt.Errorf("failed to read upgrade.sh: %v", err)
	}

	err = os.WriteFile(filepath.Join(util.KubeEdgeUpgradePath, up.ToVersion, "upgrade.sh"), content, 0750)
	if err != nil {
		return fmt.Errorf("failed to write data to upgrade.sh: %v", err)
	}

	upgradeCmd := fmt.Sprintf("export FROM_VERSION=%s && export TO_VERSION=%s && %s",
		up.FromVersion, up.ToVersion, filepath.Join(util.KubeEdgeUpgradePath, up.ToVersion, "upgrade.sh"))
	cmd := util.NewCommand(upgradeCmd)
	err = cmd.Exec()
	if err != nil {
		return fmt.Errorf("failed to exec upgrade script: %v", err)
	}

	return nil
}

func (up *Upgrade) Rollback() error {
	klog.Infof("upgrade rollback process start")

	content, err := BuiltinFile("rollback.sh")
	if err != nil {
		return fmt.Errorf("failed to read rollback.sh: %v", err)
	}

	err = os.WriteFile(filepath.Join(util.KubeEdgeUpgradePath, up.ToVersion, "rollback.sh"), content, 0750)
	if err != nil {
		return fmt.Errorf("failed to write data to rollback.sh: %v", err)
	}

	rollbackCmd := fmt.Sprintf("export FROM_VERSION=%s && export TO_VERSION=%s && %s",
		up.FromVersion, up.ToVersion, filepath.Join(util.KubeEdgeUpgradePath, up.ToVersion, "rollback.sh"))
	cmd := util.NewCommand(rollbackCmd)
	err = cmd.Exec()
	if err != nil {
		return fmt.Errorf("failed to exec rollback script: %v", err)
	}

	return nil
}

func (up *Upgrade) UpdateStatus(status string) {
	up.Status = status
}

func (up *Upgrade) UpdateFailureReason(reason string) {
	up.Reason = reason
}

func (up *Upgrade) reportUpgradeResult() error {
	resp := &commontypes.UpgradeResponse{
		UpgradeID:   up.UpgradeID,
		NodeName:    up.EdgeCoreConfig.Modules.Edged.HostnameOverride,
		FromVersion: up.FromVersion,
		ToVersion:   up.ToVersion,
		Status:      up.Status,
		Reason:      up.Reason,
	}

	var caCrt []byte
	caCertPath := up.EdgeCoreConfig.Modules.EdgeHub.TLSCAFile
	caCrt, err := os.ReadFile(caCertPath)
	if err != nil {
		return fmt.Errorf("failed to read ca: %v", err)
	}

	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(caCrt)

	certFile := up.EdgeCoreConfig.Modules.EdgeHub.TLSCertFile
	keyFile := up.EdgeCoreConfig.Modules.EdgeHub.TLSPrivateKeyFile
	cliCrt, err := tls.LoadX509KeyPair(certFile, keyFile)

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// use TLS configuration
		TLSClientConfig: &tls.Config{
			RootCAs:            rootCAs,
			InsecureSkipVerify: false,
			Certificates:       []tls.Certificate{cliCrt},
		},
	}

	client := &http.Client{Transport: transport, Timeout: 30 * time.Second}

	respData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal failed: %v", err)
	}
	url := up.EdgeCoreConfig.Modules.EdgeHub.HTTPServer + constants.DefaultUpgradeURL
	result, err := client.Post(url, "application/json", bytes.NewReader(respData))
	if err != nil {
		return fmt.Errorf("post http request failed: %v", err)
	}
	defer result.Body.Close()

	return nil
}

type UpgradeOptions struct {
	UpgradeID   string
	FromVersion string
	ToVersion   string
	Config      string
}

type Upgrade struct {
	UpgradeID      string
	FromVersion    string
	ToVersion      string
	EdgeCoreConfig *v1alpha1.EdgeCoreConfig

	Status string
	Reason string
}

func addUpgradeFlags(cmd *cobra.Command, upgradeOptions *UpgradeOptions) {
	cmd.Flags().StringVar(&upgradeOptions.UpgradeID, "upgradeID", upgradeOptions.UpgradeID,
		"Use this key to specify Upgrade CR ID")

	cmd.Flags().StringVar(&upgradeOptions.FromVersion, "fromVersion", upgradeOptions.FromVersion,
		"Use this key to specify the origin version before upgrade")

	cmd.Flags().StringVar(&upgradeOptions.ToVersion, "toVersion", upgradeOptions.ToVersion,
		"Use this key to upgrade the required KubeEdge version")

	cmd.Flags().StringVar(&upgradeOptions.Config, "config", upgradeOptions.Config,
		"Use this key to specify the path to the edgecore configuration file.")
}
