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

package edge

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
	upgradev1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
)

var (
	// idempotencyRecord is a file that is used to avoid upgrading node twice once a time.
	// If the file exist, we don't allow upgrade node again
	// we only allow upgrade nodes when the file NOT exist
	idempotencyRecord = filepath.Join(util.KubeEdgePath, "idempotency_record")
)

// NewEdgeUpgrade returns KubeEdge edge upgrade command.
func NewEdgeUpgrade() *cobra.Command {
	upgradeOptions := newUpgradeOptions()

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade edge component. Upgrade the edge node to the desired version.",
		Long:  "Upgrade edge component. Upgrade the edge node to the desired version.",
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

	configure := &v1alpha2.EdgeCoreConfig{}
	err = yaml.Unmarshal(data, configure)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file %s: %v", up.Config, err)
	}

	upgrade := Upgrade{
		UpgradeID:      up.UpgradeID,
		HistoryID:      up.HistoryID,
		FromVersion:    up.FromVersion,
		ToVersion:      up.ToVersion,
		Image:          up.Image,
		ConfigFilePath: up.Config,
		EdgeCoreConfig: configure,
	}

	defer func() {
		// report upgrade result to cloudhub
		if err := upgrade.reportUpgradeResult(); err != nil {
			klog.Errorf("failed to report upgrade result to cloud: %v", err)
		}
		// cleanup idempotency record
		if err := os.Remove(idempotencyRecord); err != nil {
			klog.Errorf("failed to remove idempotency_record file(%s): %v", idempotencyRecord, err)
		}
	}()

	// only allow upgrade when last upgrade finished
	if util.FileExists(idempotencyRecord) {
		upgrade.UpdateStatus(string(upgradev1alpha1.UpgradeFailedRollbackSuccess))
		upgrade.UpdateFailureReason("last upgrade not finished, not allowed upgrade again")
		return fmt.Errorf("last upgrade not finished, not allowed upgrade again")
	}

	// create idempotency_record file
	if err := os.MkdirAll(filepath.Dir(idempotencyRecord), 0750); err != nil {
		upgrade.UpdateStatus(string(upgradev1alpha1.UpgradeFailedRollbackSuccess))
		reason := fmt.Sprintf("failed to create idempotency_record dir: %v", err)
		upgrade.UpdateFailureReason(reason)
		return fmt.Errorf(reason)
	}
	if _, err := os.Create(idempotencyRecord); err != nil {
		upgrade.UpdateStatus(string(upgradev1alpha1.UpgradeFailedRollbackSuccess))
		reason := fmt.Sprintf("failed to create idempotency_record file: %v", err)
		upgrade.UpdateFailureReason(reason)
		return fmt.Errorf(reason)
	}

	// run script to do upgrade operation
	err = upgrade.PreProcess()
	if err != nil {
		upgrade.UpdateStatus(string(upgradev1alpha1.UpgradeFailedRollbackSuccess))
		upgrade.UpdateFailureReason(fmt.Sprintf("upgrade error: %v", err))
		return fmt.Errorf("upgrade pre process failed: %v", err)
	}

	err = upgrade.Process()
	if err != nil {
		rbErr := upgrade.Rollback()
		if rbErr != nil {
			upgrade.UpdateStatus(string(upgradev1alpha1.UpgradeFailedRollbackFailed))
			upgrade.UpdateFailureReason(fmt.Sprintf("upgrade error: %v, rollback error: %v", err, rbErr))
		} else {
			upgrade.UpdateStatus(string(upgradev1alpha1.UpgradeFailedRollbackSuccess))
			upgrade.UpdateFailureReason(fmt.Sprintf("upgrade error: %v", err))
		}
		return fmt.Errorf("upgrade process failed: %v", err)
	}

	upgrade.UpdateStatus(string(upgradev1alpha1.UpgradeSuccess))

	return nil
}

func (up *Upgrade) PreProcess() error {
	klog.Infof("upgrade preprocess start")
	backupPath := filepath.Join(util.KubeEdgeBackupPath, up.FromVersion)
	if err := os.MkdirAll(backupPath, 0750); err != nil {
		return fmt.Errorf("mkdirall failed: %v", err)
	}

	// backup edgecore.db: copy from origin path to backup path
	if err := copy(up.EdgeCoreConfig.DataBase.DataSource, filepath.Join(backupPath, "edgecore.db")); err != nil {
		return fmt.Errorf("failed to backup db: %v", err)
	}
	// backup edgecore.yaml: copy from origin path to backup path
	if err := copy(up.ConfigFilePath, filepath.Join(backupPath, "edgecore.yaml")); err != nil {
		return fmt.Errorf("failed to back config: %v", err)
	}
	// backup edgecore: copy from origin path to backup path
	if err := copy(filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName), filepath.Join(backupPath, util.KubeEdgeBinaryName)); err != nil {
		return fmt.Errorf("failed to backup edgecore: %v", err)
	}

	// download the request version edgecore
	klog.Infof("Begin to download version %s edgecore", up.ToVersion)
	upgradePath := filepath.Join(util.KubeEdgeUpgradePath, up.ToVersion)
	container, err := util.NewContainerRuntime(up.EdgeCoreConfig.Modules.Edged.ContainerRuntime, up.EdgeCoreConfig.Modules.Edged.RemoteRuntimeEndpoint)
	if err != nil {
		return fmt.Errorf("failed to new container runtime: %v", err)
	}

	image := up.Image

	err = container.PullImages([]string{image})
	if err != nil {
		return fmt.Errorf("pull image failed: %v", err)
	}
	files := map[string]string{
		filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName): filepath.Join(upgradePath, util.KubeEdgeBinaryName),
	}
	err = container.CopyResources(image, files)
	if err != nil {
		return fmt.Errorf("failed to cp file from image to host: %v", err)
	}

	return nil
}

func copy(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	// copy file using src file mode
	destination, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceFileStat.Mode())
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}

func (up *Upgrade) Process() error {
	klog.Infof("upgrade process start")

	// stop origin edgecore
	err := util.KillKubeEdgeBinary(util.KubeEdgeBinaryName)
	if err != nil {
		return fmt.Errorf("failed to stop edgecore: %v", err)
	}

	// copy new edgecore from upgradePath to /usr/local/bin
	upgradePath := filepath.Join(util.KubeEdgeUpgradePath, up.ToVersion)
	err = copy(filepath.Join(upgradePath, util.KubeEdgeBinaryName), filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName))
	if err != nil {
		return fmt.Errorf("failed to cp file: %v", err)
	}

	// generate edgecore.service
	if util.HasSystemd() {
		err = common.GenerateServiceFile(util.KubeEdgeBinaryName, fmt.Sprintf("%s --config %s", filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName), up.ConfigFilePath))
		if err != nil {
			return fmt.Errorf("failed to create edgecore.service file: %v", err)
		}
	}

	// start new edgecore service
	err = runEdgeCore()
	if err != nil {
		return fmt.Errorf("failed to start edgecore: %v", err)
	}

	return nil
}

func (up *Upgrade) Rollback() error {
	klog.Infof("upgrade rollback process start")

	// stop edgecore
	err := util.KillKubeEdgeBinary(util.KubeEdgeBinaryName)
	if err != nil {
		return fmt.Errorf("failed to stop edgecore: %v", err)
	}

	// rollback origin config/db/binary

	// backup edgecore.db: copy from backup path to origin path
	backupPath := filepath.Join(util.KubeEdgeBackupPath, up.FromVersion)
	if err := copy(filepath.Join(backupPath, "edgecore.db"), up.EdgeCoreConfig.DataBase.DataSource); err != nil {
		return fmt.Errorf("failed to rollback db: %v", err)
	}
	// backup edgecore.yaml: copy from backup path to origin path
	if err := copy(filepath.Join(backupPath, "edgecore.yaml"), up.ConfigFilePath); err != nil {
		return fmt.Errorf("failed to back config: %v", err)
	}
	// backup edgecore: copy from backup path to origin path
	if err := copy(filepath.Join(backupPath, util.KubeEdgeBinaryName), filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName)); err != nil {
		return fmt.Errorf("failed to backup edgecore: %v", err)
	}

	// generate edgecore.service
	if util.HasSystemd() {
		err = common.GenerateServiceFile(util.KubeEdgeBinaryName, fmt.Sprintf("%s --config %s", filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName), up.ConfigFilePath))
		if err != nil {
			return fmt.Errorf("failed to create edgecore.service file: %v", err)
		}
	}

	// start edgecore
	err = runEdgeCore()
	if err != nil {
		return fmt.Errorf("failed to start origin edgecore: %v", err)
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
	resp := &commontypes.NodeUpgradeJobResponse{
		UpgradeID:   up.UpgradeID,
		HistoryID:   up.HistoryID,
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
	url := up.EdgeCoreConfig.Modules.EdgeHub.HTTPServer + constants.DefaultNodeUpgradeURL
	result, err := client.Post(url, "application/json", bytes.NewReader(respData))
	if err != nil {
		return fmt.Errorf("post http request failed: %v", err)
	}
	defer result.Body.Close()

	return nil
}

type UpgradeOptions struct {
	UpgradeID   string
	HistoryID   string
	FromVersion string
	ToVersion   string
	Config      string
	Image       string
}

type Upgrade struct {
	UpgradeID      string
	HistoryID      string
	FromVersion    string
	ToVersion      string
	Image          string
	ConfigFilePath string
	EdgeCoreConfig *v1alpha2.EdgeCoreConfig

	Status string
	Reason string
}

func addUpgradeFlags(cmd *cobra.Command, upgradeOptions *UpgradeOptions) {
	cmd.Flags().StringVar(&upgradeOptions.UpgradeID, "upgradeID", upgradeOptions.UpgradeID,
		"Use this key to specify Upgrade CR ID")

	cmd.Flags().StringVar(&upgradeOptions.HistoryID, "historyID", upgradeOptions.HistoryID,
		"Use this key to specify Upgrade CR status history ID.")

	cmd.Flags().StringVar(&upgradeOptions.FromVersion, "fromVersion", upgradeOptions.FromVersion,
		"Use this key to specify the origin version before upgrade")

	cmd.Flags().StringVar(&upgradeOptions.ToVersion, "toVersion", upgradeOptions.ToVersion,
		"Use this key to upgrade the required KubeEdge version")

	cmd.Flags().StringVar(&upgradeOptions.Config, "config", upgradeOptions.Config,
		"Use this key to specify the path to the edgecore configuration file.")

	cmd.Flags().StringVar(&upgradeOptions.Image, "image", upgradeOptions.Image,
		"Use this key to specify installation image to download.")
}
