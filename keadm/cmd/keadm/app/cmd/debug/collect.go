package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
)

var (
	edgecollectLongDescription = `Collect all the data of the current node, and then Operations Engineer can use them to debug.
`
	edgecollectExample = `
# Collect all items and specified the output directory path
keadm debug collect --output-path .
`
)

var printDeatilFlag = false

// NewCollect returns KubeEdge collect command.
func NewCollect() *cobra.Command {
	collectOptions := newCollectOptions()

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
		fmt.Sprintf("Specify configuration file, default is %s", common.EdgecoreConfigPath))
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

// ExecuteCollect starts to collect data
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

	if edgeconfig.Modules.Edged.ContainerRuntime == constants.DefaultRuntimeType ||
		edgeconfig.Modules.Edged.ContainerRuntime == "" {
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
	if err = os.RemoveAll(tmpName); err != nil {
		return err
	}

	printDetail("Remove tmp data finish")

	fmt.Printf("Data collected successfully, path: %s\n", zipName)
	return nil
}

// VerificationParameters verifies parameters for debug collect
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
		printDeatilFlag = true
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
	if err = ExecuteShell(common.CmdArchInfo, tmpPath); err != nil {
		return err
	}
	// cpu info
	if err = CopyFile(common.PathCpuinfo, tmpPath); err != nil {
		return err
	}
	// memory info
	if err = CopyFile(common.PathMemory, tmpPath); err != nil {
		return err
	}
	// diskinfo info
	if err = ExecuteShell(common.CmdDiskInfo, tmpPath); err != nil {
		return err
	}
	// hosts info
	if err = CopyFile(common.PathHosts, tmpPath); err != nil {
		return err
	}
	// resolv info
	if err = CopyFile(common.PathDNSResolv, tmpPath); err != nil {
		return err
	}
	// process info
	if err = ExecuteShell(common.CmdProcessInfo, tmpPath); err != nil {
		return err
	}
	// date info
	if err = ExecuteShell(common.CmdDateInfo, tmpPath); err != nil {
		return err
	}
	// uptime info
	if err = ExecuteShell(common.CmdUptimeInfo, tmpPath); err != nil {
		return err
	}
	// history info
	if err = ExecuteShell(common.CmdHistorynfo, tmpPath); err != nil {
		return err
	}
	// network info
	return ExecuteShell(common.CmdNetworkInfo, tmpPath)
}

// collect edgecore data
func collectEdgecoreData(tmpPath string, config *v1alpha2.EdgeCoreConfig, ops *common.CollectOptions) error {
	printDetail(fmt.Sprintf("create tmp file: %s", tmpPath))
	err := os.Mkdir(tmpPath, os.ModePerm)
	if err != nil {
		return err
	}

	if config.DataBase.DataSource != "" {
		if err = CopyFile(config.DataBase.DataSource, tmpPath); err != nil {
			return err
		}
	} else {
		if err = CopyFile(v1alpha2.DataBaseDataSource, tmpPath); err != nil {
			return err
		}
	}
	if ops.LogPath != "" {
		if err = CopyFile(ops.LogPath, tmpPath); err != nil {
			return err
		}
	} else {
		if err = CopyFile(util.KubeEdgeLogPath, fmt.Sprintf("%s/log", tmpPath)); err != nil {
			return err
		}
	}

	if err = CopyFile(common.PathEdgecoreService, tmpPath); err != nil {
		return err
	}
	if err = CopyFile(constants.DefaultConfigDir, tmpPath); err != nil {
		return err
	}

	if config.Modules.EdgeHub.TLSCertFile != "" && config.Modules.EdgeHub.TLSPrivateKeyFile != "" {
		if err = CopyFile(config.Modules.EdgeHub.TLSCertFile, tmpPath); err != nil {
			return err
		}
		if err = CopyFile(config.Modules.EdgeHub.TLSPrivateKeyFile, tmpPath); err != nil {
			return err
		}
	} else {
		printDetail(fmt.Sprintf("not found cert config, use default path: %s", tmpPath))
		if err = CopyFile(common.DefaultCertPath+"/", tmpPath); err != nil {
			return err
		}
	}

	if config.Modules.EdgeHub.TLSCAFile != "" {
		if err = CopyFile(config.Modules.EdgeHub.TLSCAFile, tmpPath); err != nil {
			return err
		}
	} else {
		printDetail(fmt.Sprintf("not found ca config, use default path: %s", tmpPath))
		if err = CopyFile(constants.DefaultCAKeyFile, tmpPath); err != nil {
			return err
		}
	}

	return ExecuteShell(common.CmdEdgecoreVersion, tmpPath)
}

// collect runtime/docker data
func collectRuntimeData(tmpPath string) error {
	printDetail(fmt.Sprintf("create tmp file: %s", tmpPath))
	err := os.Mkdir(tmpPath, os.ModePerm)
	if err != nil {
		return err
	}

	if err = CopyFile(common.PathDockerService, tmpPath); err != nil {
		return err
	}

	cmdStrings := []string{common.CmdDockerVersion, common.CmdDockerInfo, common.CmdDockerImageInfo, common.CmdContainerInfo, common.CmdContainerLogInfo}
	for _, cmd := range cmdStrings {
		if err = ExecuteShell(cmd, tmpPath); err != nil {
			return err
		}
	}

	return nil
}

func CopyFile(pathSrc, tmpPath string) error {
	cmd := util.NewCommand(fmt.Sprintf(common.CmdCopyFile, pathSrc, tmpPath))
	return cmd.Exec()
}

func ExecuteShell(cmdStr string, tmpPath string) error {
	cmd := util.NewCommand(fmt.Sprintf(cmdStr, tmpPath))
	return cmd.Exec()
}

func printDetail(msg string) {
	if printDeatilFlag {
		fmt.Println(msg)
	}
}
