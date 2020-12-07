package debug

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

var (
	edgecollectLongDescription = `Collect all the data of the current node, and then Operations Engineer can use them to debug.
`
	edgecollectExample = `
# Collect all items and specified the output directory path
keadm debug collect --output-path .
`
)

var pringDeatilFlag = false

// NewCollect returns KubeEdge collect command.
func NewCollect(out io.Writer, collectOptions *common.CollectOptions) *cobra.Command {
	if collectOptions == nil {
		collectOptions = newCollectOptions()
	}

	cmd := &cobra.Command{
		Use:     "collect",
		Short:   "Obtain all the data of the current node",
		Long:    edgecollectLongDescription,
		Example: edgecollectExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := ExecuteCollect(collectOptions)
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	cmd.AddCommand()
	addCollectOtherFlags(cmd, collectOptions)
	return cmd
}

// dd flags
func addCollectOtherFlags(cmd *cobra.Command, collectOptions *common.CollectOptions) {
	cmd.Flags().StringVarP(&collectOptions.Config, common.EdgecoreConfig, "c", collectOptions.Config,
		fmt.Sprintf("Specify configuration file, defalut is %s", common.EdgecoreConfigPath))
	cmd.Flags().BoolVarP(&collectOptions.Detail, "detail", "d", false,
		"Whether to print internal log output")
	//cmd.Flags().StringVar(&collectOptions.OutputPath, "output-path", collectOptions.OutputPath,
	//	"Cache data and store data compression packages in a directory that default to the current directory")
	cmd.Flags().StringVarP(&collectOptions.OutputPath, "output-path", "o", collectOptions.OutputPath,
		"Cache data and store data compression packages in a directory that default to the current directory")
	cmd.Flags().StringVarP(&collectOptions.LogPath, "log-path", "l", util.KubeEdgeLogPath,
		"Specify log file")
}

// newCollectOptions returns a struct ready for being used for creating cmd collect flags.
func newCollectOptions() *common.CollectOptions {
	opts := &common.CollectOptions{}

	opts.Config = common.EdgecoreConfigPath
	opts.OutputPath = "."
	opts.Detail = false
	return opts
}

//Start to collect data
func ExecuteCollect(collectOptions *common.CollectOptions) error {
	//verification parameters
	err := VerificationParameters(collectOptions)
	if err != nil {
		return err
	}

	fmt.Println("Start collecting data")
	// create tmp direction
	tmpName, timenow, err := makeDirTmp()
	if err != nil {
		return err
	}
	printDetail(fmt.Sprintf("create tmp file: %s", tmpName))

	err = collectSystemData(fmt.Sprintf("%s/system", tmpName))
	if err != nil {
		fmt.Printf("collect System data failed")
	}
	printDetail("collect systemd data finish")

	edgeconfig, err := util.ParseEdgecoreConfig(collectOptions.Config)

	if err != nil {
		fmt.Printf("fail to load edgecore config: %s", err.Error())
	}
	err = collectEdgecoreData(fmt.Sprintf("%s/edgecore", tmpName), edgeconfig, collectOptions)
	if err != nil {
		fmt.Printf("collect edgecore data failed")
	}
	printDetail("collect edgecore data finish")

	if edgeconfig.Modules.Edged.RuntimeType == "docker" ||
		edgeconfig.Modules.Edged.RuntimeType == "" {
		err = collectRuntimeData(fmt.Sprintf("%s/runtime", tmpName))
		if err != nil {
			fmt.Printf("collect runtime data failed")
			return err
		}
		printDetail("collect runtime data finish")
	} else {
		fmt.Printf("now runtime only support: docker")
		// TODO
		// other runtime
	}

	OutputPath := collectOptions.OutputPath
	zipName := fmt.Sprintf("%s/edge_%s.tar.gz", OutputPath, timenow)
	err = util.Compress(zipName, []string{tmpName})
	if err != nil {
		return err
	}
	printDetail("Data compressed successfully")

	// delete tmp direction
	os.RemoveAll(tmpName)
	printDetail("Remove tmp data finish")

	fmt.Printf("Data collected successfully, path: %s\n", zipName)
	return nil
}

// verification parameters for debug collect
func VerificationParameters(collectOptions *common.CollectOptions) error {
	if !util.FileExists(collectOptions.Config) {
		return fmt.Errorf("edgecore config %s does not exist", collectOptions.Config)
	}

	path, err := filepath.Abs(collectOptions.OutputPath)
	if err != nil {
		return err
	}
	if !util.FileExists(path) {
		return fmt.Errorf("output-path %s does not exist", path)
	}
	collectOptions.OutputPath = path

	if collectOptions.Detail {
		pringDeatilFlag = true
	}

	return nil
}

func makeDirTmp() (string, string, error) {
	timenow := time.Now().Format("2006_0102_150405")
	tmpName := fmt.Sprintf("/tmp/edge_%s", timenow)
	return tmpName, timenow, os.Mkdir(tmpName, os.ModePerm)
}

// collect system data
func collectSystemData(tmpPath string) error {
	printDetail(fmt.Sprintf("create tmp file: %s", tmpPath))
	err := os.Mkdir(tmpPath, os.ModePerm)
	if err != nil {
		return err
	}

	// arch info
	ExecuteShell(common.CmdArchInfo, tmpPath)
	// cpu info
	CopyFile(common.PathCpuinfo, tmpPath)
	// memory info
	CopyFile(common.PathMemory, tmpPath)
	// diskinfo info
	ExecuteShell(common.CmdDiskInfo, tmpPath)
	// hosts info
	CopyFile(common.PathHosts, tmpPath)
	// resolv info
	CopyFile(common.PathDNSResolv, tmpPath)
	// process info
	ExecuteShell(common.CmdProcessInfo, tmpPath)
	// date info
	ExecuteShell(common.CmdDateInfo, tmpPath)
	// uptime info
	ExecuteShell(common.CmdUptimeInfo, tmpPath)
	// history info
	ExecuteShell(common.CmdHistorynfo, tmpPath)
	// network info
	ExecuteShell(common.CmdNetworkInfo, tmpPath)

	return nil
}

// collect edgecore data
func collectEdgecoreData(tmpPath string, config *v1alpha1.EdgeCoreConfig, ops *common.CollectOptions) error {
	printDetail(fmt.Sprintf("create tmp file: %s", tmpPath))
	err := os.Mkdir(tmpPath, os.ModePerm)
	if err != nil {
		return err
	}

	if config.DataBase.DataSource != "" {
		CopyFile(config.DataBase.DataSource, tmpPath)
	} else {
		CopyFile(v1alpha1.DataBaseDataSource, tmpPath)
	}
	if ops.LogPath != "" {
		CopyFile(ops.LogPath, tmpPath)
	} else {
		CopyFile(util.KubeEdgeLogPath, fmt.Sprintf("%s/log", tmpPath))
	}

	CopyFile(common.PathEdgecoreService, tmpPath)
	CopyFile(constants.DefaultConfigDir, tmpPath)

	if config.Modules.EdgeHub.TLSCertFile != "" && config.Modules.EdgeHub.TLSPrivateKeyFile != "" {
		CopyFile(config.Modules.EdgeHub.TLSCertFile, tmpPath)
		CopyFile(config.Modules.EdgeHub.TLSPrivateKeyFile, tmpPath)
	} else {
		printDetail(fmt.Sprintf("not found cert config, use default path: %s", tmpPath))
		CopyFile(common.DefaultCertPath+"/", tmpPath)
	}

	if config.Modules.EdgeHub.TLSCAFile != "" {
		CopyFile(config.Modules.EdgeHub.TLSCAFile, tmpPath)
	} else {
		printDetail(fmt.Sprintf("not found ca config, use default path: %s", tmpPath))
		CopyFile(constants.DefaultCAKeyFile, tmpPath)
	}

	ExecuteShell(common.CmdEdgecoreVersion, tmpPath)
	return nil
}

// collect runtime/docker data
func collectRuntimeData(tmpPath string) error {
	printDetail(fmt.Sprintf("create tmp file: %s", tmpPath))
	err := os.Mkdir(tmpPath, os.ModePerm)
	if err != nil {
		return err
	}

	CopyFile(common.PathDockerService, tmpPath)
	ExecuteShell(common.CmdDockerVersion, tmpPath)
	ExecuteShell(common.CmdDockerInfo, tmpPath)
	ExecuteShell(common.CmdDockerImageInfo, tmpPath)
	ExecuteShell(common.CmdContainerInfo, tmpPath)
	ExecuteShell(common.CmdContainerLogInfo, tmpPath)
	return nil
}

func CopyFile(pathSrc, tmpPath string) {
	c := fmt.Sprintf(common.CmdCopyFile, pathSrc, tmpPath)
	printDetail(fmt.Sprintf("Copy File: %s", c))
	cmd := exec.Command("sh", "-c", c)
	_, err := cmd.Output()
	if err != nil {
		fmt.Printf("fail to copy file:  %s", c)
		fmt.Printf("Output: %s\n", err.Error())
	}
}

func ExecuteShell(cmdStr string, tmpPath string) {
	c := fmt.Sprintf(cmdStr, tmpPath)
	printDetail(fmt.Sprintf("Execute Shell: %s", c))
	cmd := exec.Command("sh", "-c", c)
	_, err := cmd.Output()
	if err != nil {
		fmt.Printf("fail to execute Shell: %s\n", c)
		fmt.Printf("Output: %s\n", err.Error())
	}
}

func printDetail(msg string) {
	if pringDeatilFlag {
		fmt.Println(msg)
	}
}
