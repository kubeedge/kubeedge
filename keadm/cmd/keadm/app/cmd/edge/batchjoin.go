package edge

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

type Node struct {
	IP              string `yaml:"ip"`
	EdgeNodeName    string `yaml:"EdgeNodeName"`
	Architecture    string `yaml:"architecture"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	KubeedgeVersion string `yaml:"version"`
}

type KeadmParams struct {
	CloudCoreIPPort string `yaml:"cloudcore-ipport"`
	Token           string `yaml:"token"`
	KubeedgeVersion string `yaml:"kubeedge-version"`
}

type Config struct {
	KeadmParams KeadmParams `yaml:"keadm_params"`
	Nodes       []Node      `yaml:"nodes"`
}

var configFile string

// NewDeprecatedEdgeJoin returns KubeEdge edge join command.
func NewEdgeBatchjoin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batchjoin",
		Short: "Batch join nodes using a config file",
		Long:  `This command allows multiple nodes to join a cluster using a specified config file.`,
		Run: func(cmd *cobra.Command, args []string) {
			if configFile == "" {
				fmt.Println("Please provide a config file using -c")
				os.Exit(1)
			}
			fmt.Printf("Joining nodes using config file: %s\n", configFile)
			processBatchjoin(configFile)
		},
	}
	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to config file")
	return cmd
}
func processBatchjoin(cfgFile string) {
	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("Error reading config file: %v", err)
		return
	}
	var cfg Config
	err = yaml.Unmarshal(configData, &cfg)
	if err != nil {
		fmt.Printf("Error unmarshaling config data: %v", err)
		return
	}
	// Batch join edge nodes
	if err := batchJoinNodes(&cfg); err != nil {
		fmt.Printf("Failed to batch join nodes: %v", err)
	}
}
func batchJoinNodes(config *Config) error {
	ctx := context.Background()
	var wg sync.WaitGroup                        // Used to wait for all goroutines to complete
	errCh := make(chan error, len(config.Nodes)) // Used to collect errors

	for _, node := range config.Nodes {
		wg.Add(1) // Each node corresponds to a goroutine
		go func(n Node) {
			defer wg.Done() // Ensure Done is called when the goroutine finishes
			fmt.Printf("Processing node %s \n", n.IP)
			// Perform download and copy of keadm
			if err := downloadAndCopyKeadmToNode(ctx, &n); err != nil {
				errCh <- err // If an error occurs, send it to the error channel
				fmt.Printf("Processing node %s  downloadAndCopyKeadmToNode error \n", n.IP)
				return
			}
			// Execute the keadm join command
			if err := executeJoinCommand(ctx, &n, &config.KeadmParams); err != nil {
				fmt.Printf("Processing node %s  executeJoinCommand error \n", n.IP)
				errCh <- err // Send to the error channel when an error occurs
				return
			}
			fmt.Printf("Node %s joined successfully\n", n.IP)
		}(node)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		if err != nil {
			return err // 如果有任意一个错误，返回第一个错误
		}
	}

	return nil // All nodes processed successfully
}

func downloadAndCopyKeadmToNode(ctx context.Context, node *Node) error {
	// Download keadm from GitHub Release
	releaseURL := fmt.Sprintf("https://github.com/kubeedge/kubeedge/releases/download/%s/keadm-%s.tar.gz", node.KubeedgeVersion, node.Architecture)
	fmt.Println("releaseURL:", releaseURL)
	// SSH connect to the node
	client, err := createSSHClient(node.Username, node.IP, node.Password)
	if err != nil {
		return fmt.Errorf("failed to connect to node %s: %v", node.IP, err)
	}
	defer client.Close()

	// Execute remote command to download keadm
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create new SSH session: %v", err)
	}
	defer session.Close()
	fileName := fmt.Sprintf("keadm-%s.tar.gz", node.Architecture)
	cmd := fmt.Sprintf("curl -L %s -o /tmp/%s", releaseURL, fileName)
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("failed to download keadm on node %s: %v", node.IP, err)
	}
	return nil
}

func executeJoinCommand(ctx context.Context, node *Node, params *KeadmParams) error {
	// SSH connect to the node
	client, err := createSSHClient(node.Username, node.IP, node.Password)
	if err != nil {
		return fmt.Errorf("failed to connect to node %s: %v", node.IP, err)
	}
	defer client.Close()

	// Execute the keadm join command
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create new SSH session: %v", err)
	}
	defer session.Close()
	fileName := fmt.Sprintf("keadm-%s.tar.gz", node.Architecture)
	dirName := fmt.Sprintf("keadm-%s", node.Architecture)
	cmd := fmt.Sprintf("cd /tmp && tar xzf %s && cd %s/keadm && ./keadm join --cloudcore-ipport=%s --token=%s --kubeedge-version=%s --cgroupdriver systemd --runtimetype=docker",
		fileName, dirName, params.CloudCoreIPPort, params.Token, params.KubeedgeVersion)
	fmt.Println("cmd:", cmd)
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("failed to execute keadm join command on node %s: %v", node.IP, err)
	}
	return nil
}

func createSSHClient(username, host, password string) (*ssh.Client, error) {
	// Create SSH configuration
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to the SSH server
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), config)
	if err != nil {
		fmt.Printf("Failed to dial: %v \n", err)
		return nil, fmt.Errorf("failed to dial %s: %v", host, err)
	}
	return client, nil
}
