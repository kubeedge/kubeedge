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

package common

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
)

// CloudInitUpdateBase defines common flags for init and upgrade in the cloud.
type CloudInitUpdateBase struct {
	KubeConfig       string
	KubeEdgeVersion  string
	AdvertiseAddress string
	Profile          string
	ExternalHelmRoot string
	Sets             []string
	ValueFiles       []string
	Force            bool
	DryRun           bool
	PrintFinalValues bool
	ImageRepository  string
}

const requiredSetSplitLen = 2

// GetValidSets returns a valid sets, if the item is an invalid key-value,
// it is removed from the sets and print the error message.
func (b CloudInitUpdateBase) GetValidSets() []string {
	if b.Sets == nil {
		return nil
	}
	res := make([]string, 0, len(b.Sets))
	for _, s := range b.Sets {
		p := strings.SplitN(s, "=", requiredSetSplitLen)
		if len(p) != requiredSetSplitLen {
			fmt.Println("Unsupported sets flag: ", s)
			continue
		}
		res = append(res, s)
	}
	return res
}

// HasSets returns the key is in the sets
func (b CloudInitUpdateBase) HasSets(key string) bool {
	for _, kv := range b.Sets {
		p := strings.SplitN(kv, "=", requiredSetSplitLen)
		if len(p) == requiredSetSplitLen && p[0] == key {
			return true
		}
	}
	return false
}

// InitOptions defines cloud init flags
type InitOptions struct {
	Manifests string
	SkipCRDs  bool
	CloudInitUpdateBase
}

// CloudUpgradeOptions defines cloud upgrade flags
type CloudUpgradeOptions struct {
	ReuseValues bool
	CloudInitUpdateBase
}

// JoinOptions defines edge join flags
type JoinOptions struct {
	KubeEdgeVersion       string
	CertPath              string
	CloudCoreIPPort       string
	EdgeNodeName          string
	RemoteRuntimeEndpoint string
	Token                 string
	CertPort              string
	CGroupDriver          string
	Labels                []string
	Sets                  []string
	PreRun                string
	PostRun               string

	// WithMQTT ...
	// Deprecated: the mqtt broker is alreay managed by the DaemonSet in the cloud
	WithMQTT bool

	ImageRepository string
	HubProtocol     string
	TarballPath     string
}

type CheckOptions struct {
	Domain         string
	DNSIP          string
	IP             string
	Timeout        int
	CloudHubServer string
	EdgecoreServer string
	Config         string
}

type CheckObject struct {
	Use  string
	Desc string
	Cmd  string
}

// CollectOptions has the kubeedge debug collect information filled by CLI
type CollectOptions struct {
	Config     string
	OutputPath string
	Detail     bool
	LogPath    string
}

type ResetOptions struct {
	Kubeconfig string
	Force      bool
	Endpoint   string
	PreRun     string
	PostRun    string
}

type GettokenOptions struct {
	Kubeconfig string
}

type DiagnoseOptions struct {
	Pod          string
	Namespace    string
	Config       string
	CheckOptions *CheckOptions
	DBPath       string
}

type DiagnoseObject struct {
	Desc string
	Use  string
}

// BatchProcessOptions has the kubeedge batch process information filled by CLI
type BatchProcessOptions struct {
	ConfigFile string
}

// Config defines the batch-process config file format
type Config struct {
	Keadm     Keadm  `yaml:"keadm"`
	Nodes     []Node `yaml:"nodes"`
	MaxRunNum int    `yaml:"maxRunNum"`
}

// Keadm defines the keadm config file format
type Keadm struct {
	Download          Download          `yaml:"download"`
	KeadmVersion      string            `yaml:"keadmVersion"`
	ArchGroup         []string          `yaml:"archGroup"`
	OfflinePackageDir *string           `yaml:"offlinePackageDir,omitempty"`
	CmdTplArgs        map[string]string `yaml:"cmdTplArgs,omitempty"`
}

// Download defines the url and enable flag
type Download struct {
	URL    *string `yaml:"url,omitempty"`
	Enable *bool   `yaml:"enable"`
}

// Node defines the node information used in batch-process config file
type Node struct {
	NodeName string  `yaml:"nodeName"`
	KeadmCmd string  `yaml:"keadmCmd"`
	CopyFrom *string `yaml:"copyFrom,omitempty"`
	SSH      SSH     `yaml:"ssh"`
}

// SSH defines the ssh information used in batch-process config file
type SSH struct {
	IP       string     `yaml:"ip"`
	Username string     `yaml:"username"`
	Port     *int       `yaml:"port,omitempty"`
	Auth     AuthConfig `yaml:"auth"`
}

// AuthConfig defines the auth information used in batch-process config file
type AuthConfig struct {
	Type           string          `yaml:"type"`
	PasswordAuth   *PasswordAuth   `yaml:"passwordAuth,omitempty"`
	PrivateKeyAuth *PrivateKeyAuth `yaml:"privateKeyAuth,omitempty"`
}

// PasswordAuth is defined to know the password auth information used in batch-process config file
type PasswordAuth struct {
	Password string `yaml:"password"`
}

// PrivateKeyAuth is defined to know the private key auth information used in batch-process config file
type PrivateKeyAuth struct {
	PrivateKeyPath string `yaml:"privateKeyPath"`
}

// ModuleRunning is defined to know the running status of KubeEdge components
type ModuleRunning uint8

// Different possible values for ModuleRunning type
const (
	NoneRunning ModuleRunning = iota
	KubeEdgeCloudRunning
	KubeEdgeEdgeRunning
)

// ComponentType is the type of KubeEdge components, cloudcore or edgecore
type ComponentType string

// All Component type
const (
	CloudCore ComponentType = "cloudcore"
	EdgeCore  ComponentType = "edgecore"
)

// InstallOptions is defined to know the options for installing kubeedge
type InstallOptions struct {
	ComponentType ComponentType
	TarballPath   string
}

// ToolsInstaller interface for tools with install and teardown methods.
type ToolsInstaller interface {
	InstallTools() error
	TearDown() error
}

// OSTypeInstaller interface for methods to be executed over a specified OS distribution type
type OSTypeInstaller interface {
	InstallMQTT() error
	IsK8SComponentInstalled(string, string) error
	SetKubeEdgeVersion(version semver.Version)
	InstallKubeEdge(InstallOptions) error
	RunEdgeCore() error
	KillKubeEdgeBinary(string) error
	IsKubeEdgeProcessRunning(string) (bool, error)
}

// FlagData stores value and default value of the flags used in this command
type FlagData struct {
	Val    interface{}
	DefVal interface{}
}
