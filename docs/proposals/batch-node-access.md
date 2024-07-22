## Motivation

This proposal aims to solve the node access efficiency problem on the current KubeEdge edge computing platform.

- The current KubeEdge only supports manual single-node access, which is cumbersome and time-consuming for large-scale deployment of edge nodes, and batch node access needs to be realized to improve efficiency.



## Goals

Batch access to edge nodes in the KubeEdge system:

- Simplifies node access operations, improves deployment efficiency, and ensures system stability and reliability.



## Proposal

- **Configuration file support:** Supports configuration files in YAML format, containing information such as the CloudCore address, authentication token, KubeEdge version, etc. required for each node connection.
- **Development automation tools:** Read edge node information from the configuration file and connect each node automatically to dynamically build the `keadm batchjoin` command.
- **Logging:** Records the status and execution results of each node registration, generating log files for tracking and review.



## Use Cases

- **New Node Deployment:** When a large number of new edge nodes are needed to access the KubeEdge system, the registration and configuration of nodes can be completed quickly, saving a lot of labor and time costs.

- **System Expansion:** When the system needs to be expanded, new edge nodes can be quickly integrated into the existing KubeEdge cluster to ensure system scalability and extensibility.



## Design overview

![](../../../../../边缘计算-批量节点接入项目/草稿4.0.png)



## Implementation Details

#### Preparation

- On the control node, verify the IP addresses that can reach each edge node by using the ping command.

#### Keadm tool acquisition and preparation

- Confirm the existence and configuration of the keadm tools on the control node, and if there is no tar.gz package that conforms to the naming convention, download the required keadm tools from the GitHub release.

#### Configuration file preparation

- Create a YAML-formatted configuration file `config.yaml` to store all edge node information that needs to be accessed in batches.

- **nodes**: Contains detailed information about each edge node.
  - **ip**: The IP address of the edge node.
  - **nodeName**: The name of each edge node, used to distinguish and identify each node in the cluster.
  - **architecture**: The hardware architecture type of the edge node, such as `amd64`, `arm`, or `arm64`.
  - **username**: The username required for SSH login to the edge node.
  - **password**: The password required for SSH login to the edge node.

- **keadm_params**: Contains detailed information for configuring edge node access.
  - **cloudcore-ipport**: Specifies the address and port of CloudCore for communication between edge nodes and CloudCore.
  - **token**: The authentication token used to ensure that edge nodes can securely access CloudCore.
  - **kubeedge_version**: Specifies the version of KubeEdge to be installed.

#### Source code modification and new command addition

- Download the keadm source code for the KubeEdge project and modify it to add the new command `keadm batchjoin`.

- Find the command definition file in the source code and add the new command processing logic.

```go
// batchjoinCmd represents the batchjoin command
var batchjoinCmd = &cobra.Command{
    Use:   "batchjoin",
    Short: "Batch join multiple edge nodes to the KubeEdge cluster",
    Long:  `batchjoin allows you to batch join multiple edge nodes to the KubeEdge cluster by providing a configuration file.`,
    Run: func(cmd *cobra.Command, args []string) {
        configFilePath, _ := cmd.Flags().GetString("config")
        if err := batchJoinNodes(configFilePath); err != nil {
            log.Fatalf("Failed to batch join nodes: %v", err)
        }
    },
}

func init() {
    rootCmd.AddCommand(batchjoinCmd)
    batchjoinCmd.Flags().StringP("config", "c", "", "Path to the configuration file")
    batchjoinCmd.MarkFlagRequired("config")
}
```

- **`batchjoinCmd`**: Defines the structure of the `batchjoin` command, including the name of the command, a short description and a detailed description.
- **`Run`**: Defines the execution logic of the command, which reads the configuration file path and calls the `batchJoinNodes` function to perform the batch join operation.
- **`init`**: Adds the `batchjoin` command to the root command in the `init` function and defines the command line arguments.

#### Logic for implementing batch access

Implement the `batchJoinNodes` function, which is responsible for loading the configuration file, batch connecting to each edge node using SSH libraries (e.g. `github.com/melbahja/goph`), and executing the `keadm join` command.

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/melbahja/goph"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type NodeConfig struct { 
    IP            string `yaml:"ip"`
    NodeName      string `yaml:"nodeName"`
    Architecture  string `yaml:"architecture"`
    Username      string `yaml:"username"`
    Password      string `yaml:"password"`
}

type KeadmParams struct {
    CloudCoreIPPort string `yaml:"cloudcore-ipport"`
    Token           string `yaml:"token"`
    KubeEdgeVersion string `yaml:"kubeedge_version"`
}

type Config struct {
	KeadmParams KeadmParams `yaml:"keadm_params"`
	Nodes       []NodeConfig `yaml:"nodes"`
}

func batchJoinNodes(configFile string) {
	// Open log file for writing
	...
    
	// Set log output to both stdout and log file
	log.SetOutput(logFile)

	// Load config file
	config := loadConfig(configFile)

	// Connect to each node and execute keadm join
	for _, node := range config.Nodes {
		client, err := connectToNode(node)
		if err != nil {
			log.Printf("Failed to connect to node %s: %v", node.IP, err)
			continue
		}

		// Build keadm join command
		command := buildKeadmJoinCommand(config.KeadmParams)

		// Execute keadm join command
		output, err := executeCommand(client, command)
		if err != nil {
			log.Printf("Failed to execute keadm join on node %s: %v", node.IP, err)
			continue
		}

		log.Printf("Successfully joined %s to KubeEdge: %s", node.IP, output)
	}
}

func loadConfig(filename string) Config {
	var config Config
	...
	return config
}

func connectToNode(node NodeConfig) (*goph.Client, error) {
	...
	return client, nil
}

func executeCommand(client *goph.Client, command string) (string, error) {
	...
	return string(out), nil
}

func buildKeadmJoinCommand(params KeadmParams) string {
	return fmt.Sprintf("keadm join --cloudcore-ipport=%s --token=%s --kubeedge-version=%s", params.CloudCoreIPPort, params.Token, params.KubeEdgeVersion)
}

func main() {
	configFile := "config.yaml"
	batchJoinNodes(configFile)
}
```

-  **Logging**: Use the `log` package to create a log file and set it to output to both standard output and the log file.
- **Configuration file loading**: Load node configuration and command parameters from the specified YAML configuration file.
- **Remote command execution**: Use the `github.com/melbahja/goph` package to establish an SSH connection to each node and execute the `keadm join` command to join the node to the KubeEdge cluster.

#### Logging

- Implements detailed logging to record the registration status, execution results, and exceptions of each node.



## Road map

#### System Design (7.1-7.15)

- Design of detailed implementation scenarios.

#### Development and Testing (7.16-8.31)

- Write code to implement project functionality.
- Execute tests to verify system functionality.

#### Deployment and Validation (9.1-9.15)

- Deploy the system to the test environment and perform validation.
- Gather feedback and perform problem fixing and optimization.
- Prepare the system for formal deployment.

#### Code tuning and organization (9.16-9.30)

- Code tuning and organization.
- Write comprehensive documentation.