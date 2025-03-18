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

package edge

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

const (
	baseDir                = "/tmp/kubeedge/keadm"
	packageDir             = baseDir + "/package"
	binDir                 = baseDir + "/bin"
	defautKeadmDownloadURL = "https://github.com/kubeedge/kubeedge/releases/download"
	defautSSHPort          = 22
	defaultMaxRunNum       = 5
)

func NewEdgeBatchProcess() *cobra.Command {
	bacthProcessOpts := &common.BatchProcessOptions{}
	var cfg *common.Config
	step := common.NewStep()
	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Batch process nodes using a config file",
		Long:  `This command allows batch process multiple nodes using a specified config file.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			step.Printf("Checking config file.")
			if bacthProcessOpts.ConfigFile == "" {
				klog.Error("Please provide a config file using -c")
				return errors.New("config file not provided")
			}
			klog.Infof("bacth process nodes using config file: %s\n", bacthProcessOpts.ConfigFile)

			step.Printf("Parsing the configuration file.")
			configData, err := os.ReadFile(bacthProcessOpts.ConfigFile)
			if err != nil {
				return errors.Errorf("error reading config file: %v", err)
			}
			err = yaml.Unmarshal(configData, &cfg)
			if err != nil {
				return errors.Errorf("error unmarshaling config data: %v", err)
			}

			step.Printf("Checking whether the node names are duplicated.")
			nodeNameMap := make(map[string]struct{})
			for i, node := range cfg.Nodes {
				if _, ok := nodeNameMap[node.NodeName]; ok {
					return errors.Errorf("node name %s is duplicated", node.NodeName)
				}
				nodeNameMap[node.NodeName] = struct{}{}

				replacedCmd, err := replacePlaceholders(node.KeadmCmd, cfg.Keadm.CmdTplArgs)
				if err != nil {
					return errors.Errorf("error replacing placeholders in keadm command: %v", err)
				}
				cfg.Nodes[i].KeadmCmd = replacedCmd
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return processBatchProcess(cfg, step)
		},
	}
	// Adding the gen-config subcommand
	cmd.AddCommand(NewBatchProcessGenConfig())
	addBacthProcessOtherFlags(cmd, bacthProcessOpts)
	return cmd
}

func processBatchProcess(cfg *common.Config, step *common.Step) error {
	// Create log file to store results
	logFile, err := os.Create("batch_process_log.txt")
	if err != nil {
		return errors.Errorf("failed to create log file: %v", err)
	}
	defer logFile.Close()
	logWriter := bufio.NewWriter(logFile)

	// Ensure all log entries are written to file
	defer logWriter.Flush()

	// Get keadm packages
	step.Printf("Preparing keadm packages.")
	if err = prepareKeadmPackages(cfg); err != nil {
		return errors.Errorf("failed to prepare keadm packages, %v", err)
	}

	step.Printf("Batch process nodes.")
	// Batch process edge nodes
	if err = batchProcessNodes(cfg, logWriter); err != nil {
		return errors.Errorf("failed to batch process nodes, %v", err)
	}

	return nil
}

// Obtain the keadm installation package according to the configuration.
// If "enable" is set to true, then download it; otherwise, obtain it from the offlinePackageDir and extract it.
func prepareKeadmPackages(cfg *common.Config) error {
	if cfg.Keadm.Download.Enable == nil || *cfg.Keadm.Download.Enable {
		return downloadKeadmPackages(cfg)
	}
	return useOfflinePackages(cfg)
}

// Obtain and extract the installation package from the offlinePackageDir provided by the user
func useOfflinePackages(cfg *common.Config) error {
	for _, arch := range cfg.Keadm.ArchGroup {
		if cfg.Keadm.OfflinePackageDir == nil {
			return errors.New("offlinePackageDir is not provided")
		}
		packagePath := filepath.Join(*cfg.Keadm.OfflinePackageDir, arch, fmt.Sprintf("keadm-%s-linux-%s.tar.gz", cfg.Keadm.KeadmVersion, arch))
		if _, err := os.Stat(packagePath); os.IsNotExist(err) {
			return errors.Errorf("package for keadm-%s-linux-%s.tar.gz not found in %s", cfg.Keadm.KeadmVersion, arch, *cfg.Keadm.OfflinePackageDir)
		}

		binOutputDir := filepath.Join(binDir, arch)
		if err := os.MkdirAll(binOutputDir, os.ModePerm); err != nil {
			return errors.Errorf("failed to create directory %s: %v", binOutputDir, err)
		}

		if err := extractTarGz(packagePath, binOutputDir); err != nil {
			return err
		}
		klog.Infof("Extracted keadm package for keadm-%s-linux-%s.tar.gz to %s", cfg.Keadm.KeadmVersion, arch, binOutputDir)
	}
	return nil
}

// download keadm package
func downloadKeadmPackages(cfg *common.Config) error {
	for _, arch := range cfg.Keadm.ArchGroup {
		var url string
		if cfg.Keadm.Download.URL == nil {
			url = fmt.Sprintf("%s/%s/keadm-%s-linux-%s.tar.gz", defautKeadmDownloadURL, cfg.Keadm.KeadmVersion, cfg.Keadm.KeadmVersion, arch)
		} else {
			url = fmt.Sprintf("%s/keadm-%s-linux-%s.tar.gz", *cfg.Keadm.Download.URL, cfg.Keadm.KeadmVersion, arch)
		}
		outputDir := filepath.Join(packageDir, arch)
		if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
			return errors.Errorf("failed to create directory %s: %v", outputDir, err)
		}
		outputPath := filepath.Join(outputDir, fmt.Sprintf("keadm-%s-linux-%s.tar.gz", cfg.Keadm.KeadmVersion, arch))

		// decompressed into target directory
		binOutputDir := filepath.Join(binDir, arch)
		if err := os.MkdirAll(binOutputDir, os.ModePerm); err != nil {
			return errors.Errorf("failed to create directory %s: %v", binOutputDir, err)
		}

		// attempt extract
		klog.Infof("Attempting to extract keadm for %s to %s", arch, binOutputDir)
		if err := extractTarGz(outputPath, binOutputDir); err != nil {
			klog.Warningf("Failed to extract file %s, will attempt to download: %v", outputPath, err)

			// download keadm
			klog.Infof("Downloading keadm for %s from %s to %s", arch, url, outputPath)
			if err = downloadFile(url, outputPath); err != nil {
				return err
			}

			// attempt extract again
			klog.Infof("Re-attempting to extract keadm for %s to %s", arch, binOutputDir)
			if err = extractTarGz(outputPath, binOutputDir); err != nil {
				return errors.Errorf("failed to extract file after download: %v", err)
			}
		}

		klog.Infof("Downloaded and extracted keadm for %s to %s", arch, binOutputDir)
	}
	return nil
}

func downloadFile(url, outputPath string) error {
	cmd := exec.Command("curl", "-L", "-o", outputPath, url)
	if err := cmd.Run(); err != nil {
		return errors.Errorf("failed to download file from %s: %v", url, err)
	}
	return nil
}

func extractTarGz(tarFile, destDir string) error {
	cmd := exec.Command("tar", "-xzvf", tarFile, "-C", destDir)
	if err := cmd.Run(); err != nil {
		return errors.Errorf("failed to extract tar.gz file: %v", err)
	}
	return nil
}

// batch process nodes
func batchProcessNodes(cfg *common.Config, logWriter *bufio.Writer) error {
	// Set default value for MaxRunNum
	if cfg.MaxRunNum == 0 {
		cfg.MaxRunNum = defaultMaxRunNum
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.MaxRunNum)
	for _, node := range cfg.Nodes {
		wg.Add(1)
		go func(node common.Node) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var result string
			if err := processNode(&node, cfg); err != nil {
				result = fmt.Sprintf("Failed to process node %s: %v", node.NodeName, err)
				klog.Error(result)
			} else {
				result = fmt.Sprintf("Successfully processed node %s", node.NodeName)
				klog.Info(result)
			}

			// Log result to file
			_, err := logWriter.WriteString(result + "\n")
			if err != nil {
				klog.Errorf("Failed to write log entry for node %s: %v", node.NodeName, err)
			}
		}(node)
	}
	wg.Wait()
	return nil
}

// Process single node
func processNode(node *common.Node, cfg *common.Config) error {
	klog.Infof("Processing node %s", node.NodeName)
	client, err := connectSSH(node.SSH)
	if err != nil {
		return errors.Errorf("failed to connect to %s: %v", node.NodeName, err)
	}
	defer client.Close()

	if err = createRemoteDir(client, baseDir); err != nil {
		return err
	}

	if node.CopyFrom != nil {
		if err = uploadFiles(client, node.NodeName, *node.CopyFrom, baseDir); err != nil {
			return err
		}
	}

	// get node Arch
	arch, err := getNodeArch(client)
	if err != nil {
		return err
	}
	keadmPath := filepath.Join(binDir, arch, fmt.Sprintf("keadm-%s-linux-%s/keadm/keadm", cfg.Keadm.KeadmVersion, arch))
	if err = uploadFile(client, node.NodeName, keadmPath, filepath.Join(baseDir, "keadm")); err != nil {
		return err
	}

	if err = executeKeadmCommand(client, node.NodeName, node.KeadmCmd); err != nil {
		return err
	}

	klog.Infof("Node %s processing completed", node.NodeName)
	return nil
}

// SSH connection
func connectSSH(sshConfig common.SSH) (*ssh.Client, error) {
	var auth ssh.AuthMethod

	switch sshConfig.Auth.Type {
	case "password":
		if sshConfig.Auth.PasswordAuth != nil {
			auth = ssh.Password(sshConfig.Auth.PasswordAuth.Password)
		} else {
			return nil, errors.New("passwordAuth field is empty")
		}
	case "privateKey":
		if sshConfig.Auth.PrivateKeyAuth != nil {
			key, err := os.ReadFile(sshConfig.Auth.PrivateKeyAuth.PrivateKeyPath)
			if err != nil {
				return nil, errors.Errorf("failed to read private key: %v", err)
			}
			signer, err := ssh.ParsePrivateKey(key)
			if err != nil {
				return nil, errors.Errorf("failed to parse private key: %v", err)
			}
			auth = ssh.PublicKeys(signer)
		} else {
			return nil, errors.New("privateKeyAuth field is empty")
		}
	default:
		return nil, errors.Errorf("unsupported authentication type: %s", sshConfig.Auth.Type)
	}
	config := &ssh.ClientConfig{
		User: sshConfig.Username,
		Auth: []ssh.AuthMethod{auth},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 5 * time.Second,
	}

	var sshPort int
	if sshConfig.Port == nil {
		sshPort = defautSSHPort
	} else {
		sshPort = *sshConfig.Port
	}

	addr := fmt.Sprintf("%s:%d", sshConfig.IP, sshPort)
	return ssh.Dial("tcp", addr, config)
}

// create remote directory
func createRemoteDir(client *ssh.Client, dir string) error {
	session, err := client.NewSession()
	if err != nil {
		return errors.Errorf("failed to create new SSH session: %v", err)
	}
	defer session.Close()
	cmd := fmt.Sprintf("mkdir -p %s", dir)
	if err = session.Run(cmd); err != nil {
		return errors.Errorf("failed to create directory %s: %v", dir, err)
	}
	return nil
}

// upload file
func uploadFile(client *ssh.Client, nodeName, srcPath, destPath string) error {
	// use ssh.Client to create an SFTP client
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return errors.Errorf("failed to create SFTP client: %v", err)
	}
	defer sftpClient.Close()

	// open local file
	localFile, err := os.Open(srcPath)
	if err != nil {
		return errors.Errorf("failed to open local file %s: %v", srcPath, err)
	}
	defer localFile.Close()

	// create remote file
	remoteFile, err := sftpClient.Create(destPath)
	if err != nil {
		return errors.Errorf("failed to create remote file %s: %v", destPath, err)
	}
	defer remoteFile.Close()

	// copy from local file to remote file
	klog.Infof("Uploading file %s to %s:%s\n", srcPath, nodeName, destPath)
	_, err = remoteFile.ReadFrom(localFile)
	if err != nil {
		return errors.Errorf("failed to upload file: %v", err)
	}

	// set remote file permissions
	if err = sftpClient.Chmod(destPath, os.FileMode(0755)); err != nil {
		return errors.Errorf("failed to set remote file permissions: %v", err)
	}

	klog.Infof("Successfully Uploaded file %s to %s:%s\n", srcPath, nodeName, destPath)
	return nil
}

// batch upload files
func uploadFiles(client *ssh.Client, nodeName, srcDir, destDir string) error {
	files, err := os.ReadDir(srcDir)
	if err != nil {
		return errors.Errorf("failed to read directory %s: %v", srcDir, err)
	}
	for _, file := range files {
		srcPath := filepath.Join(srcDir, file.Name())
		destPath := filepath.Join(destDir, file.Name())
		if err = uploadFile(client, nodeName, srcPath, destPath); err != nil {
			return errors.Errorf("failed to upload file %s: %v", srcPath, err)
		}
	}
	return nil
}

// execute keadm command
func executeKeadmCommand(client *ssh.Client, nodeName, cmd string) error {
	session, err := client.NewSession()
	if err != nil {
		return errors.Errorf("failed to create new SSH session: %v", err)
	}
	defer session.Close()
	var execCmd string
	parts := strings.Fields(cmd)
	if len(parts) >= 1 && parts[0] == "reset" {
		execCmd = fmt.Sprintf("cd %s && yes |./keadm %s", baseDir, cmd)
	} else {
		execCmd = fmt.Sprintf("cd %s && ./keadm %s", baseDir, cmd)
	}
	klog.Infof("%s: Executing command %s", nodeName, execCmd)

	// Redirect output to console
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	if err = session.Run(execCmd); err != nil {
		return errors.Errorf("failed to execute keadm command %s: %v", execCmd, err)
	}
	return nil
}

func getNodeArch(client *ssh.Client) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", errors.Errorf("failed to create new SSH session: %v", err)
	}
	defer session.Close()
	cmd := "uname -m"
	var stdout bytes.Buffer
	session.Stdout = &stdout
	if err = session.Run(cmd); err != nil {
		return "", errors.Errorf("failed to execute command %s: %v", cmd, err)
	}
	return determineArchitecture(strings.TrimSpace(stdout.String())), nil
}

func replacePlaceholders(templateStr string, args map[string]string) (string, error) {
	tmpl, err := template.New("cmd").Parse(templateStr)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	if err = tmpl.Execute(&buffer, args); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// Determine architecture type based on uname -m output
func determineArchitecture(output string) string {
	switch output {
	case "x86_64", "amd64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	case "armv7l", "armv6l", "arm32", "arm":
		return "arm"
	default:
		return ""
	}
}

func addBacthProcessOtherFlags(cmd *cobra.Command, batchProcessOpts *common.BatchProcessOptions) {
	cmd.Flags().StringVarP(&batchProcessOpts.ConfigFile, "config", "c", "", "Path to config file")
}
