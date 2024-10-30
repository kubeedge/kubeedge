// Copyright 2024 WeiXingEsther
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package options

package edge

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

type Node struct {
	IP           string `yaml:"ip"`
	EdgeNodeName string `yaml:"EdgeNodeName"`
	Username        string `yaml:"username"`
	KubeedgeVersion string `yaml:"version"`
	PrivateKey      string `yaml:"privateKey"`
	SSHPort         int    `yaml:"sshPort"`
	Runtimetype     string `yaml:"runtimetype"`
}

type KeadmParams struct {
	CloudCoreIPPort string `yaml:"cloudcore-ipport"`
	Token           string `yaml:"token"`
	KubeedgeVersion string `yaml:"kubeedge-version"`
}

type Config struct {
	KeadmParams KeadmParams `yaml:"keadm_params"`
	Nodes       []Node      `yaml:"nodes"`
	MaxRunNum   int         `yaml:"maxRunNum"`
}

var configFile string

// NewDeprecatedEdgeJoin returns KubeEdge batch edge join command.
func NewEdgeBatchJoin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batchjoin",
		Short: "Batch join nodes using a config file",
		Long:  `This command allows multiple nodes to join a cluster using a specified config file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if configFile == "" {
				klog.Errorf("Please provide a config file using -c")
				os.Exit(1)
			}
			klog.Infof("Joining nodes using config file: %s\n", configFile)
			return processBatchjoin(configFile)
		},
	}
	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to config file")
	return cmd
}
func processBatchjoin(cfgFile string) error {
	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		klog.Errorf("Error reading config file: %v", err)
		return err
	}
	var cfg Config
	err = yaml.Unmarshal(configData, &cfg)
	if err != nil {
		klog.Errorf("Error unmarshaling config data: %v", err)
		return err
	}
	// Batch join edge nodes
	if err := batchJoinNodes(&cfg); err != nil {
		klog.Errorf("Failed to batch join nodes: %v", err)
		return err
	}
	return nil
}

func batchJoinNodes(config *Config) error {
	err := DownloadTask(config)
	if err != nil {
		klog.Errorf("failed to DownloadTask: %v", err)
		return err
	}
	// Perform related operations
	ctx := context.Background()
	var wg sync.WaitGroup
	concurrencyLimit := make(chan struct{}, config.MaxRunNum) // Limit to a maximum of maxRunNum Goroutines

	for _, node := range config.Nodes {
		wg.Add(1)                      // Each node corresponds to a goroutine
		concurrencyLimit <- struct{}{} // Occupy one Goroutine slot
		go func(n Node) {
			defer wg.Done()                       // Ensure Done is called when the goroutine finishes
			defer func() { <-concurrencyLimit }() // Release a Goroutine slot
			klog.Errorf("Processing node %s", n.IP)
			arch, err := checkSystem(&n)
			if err != nil {
				klog.Errorf("failed to checkSystem: %v", err)
				return
			}
			if arch == "" {
				klog.Errorf("Unknow arch")
				return
			}
			// Copy to the remote host
			if err := CopyKeadmToNode(ctx, &n, arch); err != nil {
				klog.Errorf("Processing node %s  CopyKeadmToNode error", n.IP)
				return
			}
			// Execute the keadm join command
			if err := executeJoinCommand(ctx, &n, &config.KeadmParams, arch); err != nil {
				klog.Errorf("Processing node %s  executeJoinCommand error", n.IP)
				return
			}
			klog.Infof("Node %s joined successfully\n", n.IP)
		}(node)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	return nil // All nodes processed successfully
}
func DownloadTask(config *Config) error {
	for _, node := range config.Nodes {
		// Determine the system architecture and generate the required filename
		arch, err := checkSystem(&node)
		if err != nil {
			klog.Errorf("failed to checkSystem: %v", err)
			return err
		}
		if arch == "" {
			klog.Errorf("Unknow arch")
			return err
		}
		// Check if the required file already exists
		fileName := fmt.Sprintf("keadm-%s-linux-%s.tar.gz", node.KubeedgeVersion, arch)
		exist, err := CheckFileExist(fileName)
		if err != nil {
			klog.Errorf("failed CheckFileExist: %v", err)
			return err
		}

		if exist {
			klog.Infof("fileName:%s is exist, so do not downland opr.", fileName)
			continue
		}
		// If not found, perform the download operation
		ctx := context.Background()
		if err := downloadKeadmToHost(ctx, &node, arch); err != nil {
			klog.Errorf("Processing node %s  downloadAndCopyKeadmToNode error", node.IP)
			return err
		}
	}
	return nil
}
func CheckFileExist(fileName string) (bool, error) {
	// Concatenate the path ./data/ and the filename
	filePath := filepath.Join("./data", fileName)

	// Use os.Stat to check the file status
	_, err := os.Stat(filePath)

	// If there is no error, the file exists
	if err == nil {
		return true, nil
	}

	// If the error is due to the file not existing, return false
	if os.IsNotExist(err) {
		return false, nil
	}

	// Return for other errors
	return false, err
}

// CopyKeadmToNode copies a local file to the remote node
func CopyKeadmToNode(ctx context.Context, node *Node, arch string) error {
	// Construct local file path and remote file path
	fileName := fmt.Sprintf("keadm-%s-linux-%s.tar.gz", node.KubeedgeVersion, arch)
	localPath := filepath.Join("./data", fileName)

	remotePath := fmt.Sprintf("/tmp/%s", fileName)

	// Check if the local file exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		klog.Errorf("Local file does not exist:%s", localPath)
		return err
	}

	// Create SSH client
	client, err := createSSHClient(node.Username, node.IP, node.PrivateKey, node.SSHPort)
	if err != nil {
		klog.Errorf("Unable to connect to the node %s: %v", node.IP, err)
		return err
	}
	defer client.Close()

	// Create a new SSH session
	session, err := client.NewSession()
	if err != nil {
		klog.Errorf("Unable to create SSH session: %v", err)
		return err
	}
	defer session.Close()

	// Read local file content
	fileContent, err := ioutil.ReadFile(localPath)
	if err != nil {
		klog.Errorf("Unable to read local file %s: %v", localPath, err)
		return err
	}

	// Use SCP to transfer the file to the remote node
	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()

		// Send SCP file header information
		fmt.Fprintf(w, "C0644 %d %s\n", len(fileContent), fileName)
		w.Write(fileContent)
		fmt.Fprint(w, "\x00")
	}()

	// Execute SCP command to create the file on the remote node
	cmd := fmt.Sprintf("scp -t %s", filepath.Dir(remotePath))
	if err := session.Run(cmd); err != nil {
		klog.Errorf("Copy file to node %s Failed: %v", node.IP, err)
		return err
	}

	klog.Infof("File %s successfully copied to node %s at %s", fileName, node.IP, remotePath)
	return nil
}

func downloadKeadmToHost(ctx context.Context, node *Node, arch string) error {
	// Construct download link and filename
	releaseURL := fmt.Sprintf(
		"https://github.com/kubeedge/kubeedge/releases/download/%s/keadm-%s-linux-%s.tar.gz",
		node.KubeedgeVersion, node.KubeedgeVersion, arch,
	)
	fileName := fmt.Sprintf("keadm-%s-linux-%s.tar.gz", node.KubeedgeVersion, arch)
	downloadPath := filepath.Join("./data", fileName)

	// Print download link for debugging
	klog.Infof("releaseURL:%s", releaseURL)

	// Use curl to download the file to the local /tmp directory
	cmd := exec.CommandContext(ctx, "curl", "-L", releaseURL, "-o", downloadPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	klog.Infof("Downloading %s to %s", releaseURL, downloadPath)
	if err := cmd.Run(); err != nil {
		klog.Errorf("Failed to download keadm: %v", err)
		return err
	}
	return nil
}
func executeJoinCommand(ctx context.Context, node *Node, params *KeadmParams, arch string) error {
	// SSH connect to the node
	client, err := createSSHClient(node.Username, node.IP, node.PrivateKey, node.SSHPort)
	if err != nil {
		klog.Errorf("failed to connect to node %s: %v", node.IP, err)
		return err
	}
	defer client.Close()

	// Execute the keadm join command
	session, err := client.NewSession()
	if err != nil {
		klog.Errorf("failed to create new SSH session: %v", err)
		return err
	}
	defer session.Close()

	fileName := fmt.Sprintf("keadm-%s-linux-%s.tar.gz", node.KubeedgeVersion, arch)
	dirName := fmt.Sprintf("keadm-%s-linux-%s", node.KubeedgeVersion, arch)
	cmd := fmt.Sprintf("cd /tmp && tar xzf %s && cd %s/keadm && ./keadm join --cloudcore-ipport=%s --token=%s --kubeedge-version=%s --cgroupdriver systemd --runtimetype=%s",
		fileName, dirName, params.CloudCoreIPPort, params.Token, params.KubeedgeVersion, node.Runtimetype)
	klog.Infof("cmd:%v", cmd)
	//if err := session.Run(cmd); err != nil {
	//	klog.Errorf("failed to execute keadm join command on node %s: %v", node.IP, err)
	//		return err
	//	}
	return nil
}
func checkSystem(node *Node) (string, error) {
	client, err := createSSHClient(node.Username, node.IP, node.PrivateKey, node.SSHPort)
	if err != nil {
		klog.Errorf("failed to connect to node %s: %v", node.IP, err)
		return "", err
	}
	defer client.Close()
	// Execute uname -m to determine system architecture
	output, err := executeCommand(client, "uname -m")
	if err != nil {
		return "", err
	}
	arch := determineArchitecture(output)
	return arch, nil
}

// Function to execute command
func executeCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		klog.Errorf("failed to create new SSH session: %v", err)
		return "", err
	}
	defer session.Close()

	var outputBuf bytes.Buffer
	session.Stdout = &outputBuf

	if err := session.Run(command); err != nil {
		klog.Errorf("execute cmd:%s  err:%v", command, err)
		return "", err
	}

	return strings.TrimSpace(outputBuf.String()), nil
}

// Load private key
func loadPrivateKey(file string) (ssh.Signer, error) {
	key, err := ioutil.ReadFile(file)
	if err != nil {
		klog.Errorf("unable to read private key: %v", err)
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		klog.Errorf("unable to parse private key: %v", err)
		return nil, err
	}
	return signer, nil
}
func createSSHClient(username, host, privateKey string, port int) (*ssh.Client, error) {
	// Load private key
	signer, err := loadPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	// Create SSH configuration
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to the SSH server
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		klog.Errorf("Failed to dial: %v \n", err)
		return nil, err
	}
	return client, nil
}

// Determine architecture type based on uname -m output
func determineArchitecture(output string) string {
	switch output {
	case "x86_64":
		return "amd64"
	case "aarch64":
		return "arm64"
	case "armv7l", "armv6l":
		return "arm"
	default:
		return ""
	}
}
