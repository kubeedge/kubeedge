package debug

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

var (
	edgecollectLongDescription = `Obtain all the data of the current node, and then provide it to the operation
and maintenance personnel to locate the problem
`
	edgecollectExample = `
# Check all items and specified as the current directory
keadm debug collect --output-path .
`
)

// NewEdgeJoin returns KubeEdge edge join command.
func NewEdgeCollect(out io.Writer, collectOptions *types.ColletcOptions) *cobra.Command {
	if collectOptions == nil {
		collectOptions = newCollectOptions()
	}

	cmd := &cobra.Command{
		Use:     "collect",
		Short:   "Obtain all the data of the current node",
		Long:    edgecollectLongDescription,
		Example: edgecollectExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteCollect(collectOptions)
		},
	}
	cmd.AddCommand()
	addCollectOtherFlags(cmd, collectOptions)
	return cmd
}

// add flags
func addCollectOtherFlags(cmd *cobra.Command, collectOptions *types.ColletcOptions) {
	cmd.Flags().StringVarP(&collectOptions.Config, types.EdgecoreConfig, "c", collectOptions.Config,
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
func newCollectOptions() *types.ColletcOptions {
	opts := &types.ColletcOptions{}

	opts.Config = types.EdgecoreConfigPath
	opts.OutputPath = "."
	opts.Detail = false
	return opts
}

//Start to collect data
func ExecuteCollect(collectOptions *types.ColletcOptions) error {
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
	klog.Infof("create tmp file: %s", tmpName)

	collectSystemData(fmt.Sprintf("%s/system", tmpName))
	klog.Infof("collect systemd data finish")

	edgeconfig, err := util.ParseEdgecoreConfig(collectOptions.Config)

	if err != nil {
		klog.Warningf("fail to load edgecore config: %s", err.Error())
	}
	collectEdgecoreData(fmt.Sprintf("%s/edgecore", tmpName), edgeconfig, collectOptions)
	klog.Infof("collect edgecore data finish")

	if edgeconfig.Modules.Edged.RuntimeType == "docker" ||
		edgeconfig.Modules.Edged.RuntimeType == "" {
		collectRuntimeData(fmt.Sprintf("%s/runtime", tmpName))
		klog.Infof("collect runtime data finish")
	} else {
		klog.Warningf("now runtime only support: docker")
		// TODO
		// other runtime
	}

	OutputPath := collectOptions.OutputPath
	zipName := fmt.Sprintf("%s/edge_%s.tar.gz", OutputPath, timenow)
	util.Compress(zipName, []string{tmpName})
	klog.Infof("compress data finish")

	// delete tmp direction
	os.RemoveAll(tmpName)
	klog.V(2).Infof("Remove tmp data finish")

	fmt.Printf("collecting data finish, path: %s\n", zipName)
	return nil
}

// verification parameters for debug collect
func VerificationParameters(collectOptions *types.ColletcOptions) error {
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

	var klogFlags flag.FlagSet
	klog.InitFlags(&klogFlags)
	klogFlags.Set("logtostderr", "false")

	if collectOptions.Detail {
		klogFlags.Set("stderrthreshold", "INFO")
	} else {
		klogFlags.Set("stderrthreshold", "WARNING")
	}

	return nil
}

func makeDirTmp() (string, string, error) {
	timenow := time.Now().Format("2006_0102_150405")
	tmpName := fmt.Sprintf("/tmp/edge_%s", timenow)
	return tmpName, timenow, os.Mkdir(tmpName, os.ModePerm)
}

// collect system data
func collectSystemData(tmpPath string) {
	klog.Infof("create tmp file: %s", tmpPath)
	os.Mkdir(tmpPath, os.ModePerm)

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
}

// collect edgecore data
func collectEdgecoreData(tmpPath string, config *v1alpha1.EdgeCoreConfig, ops *types.ColletcOptions) {
	klog.Infof("create tmp file: %s", tmpPath)
	os.Mkdir(tmpPath, os.ModePerm)

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
		klog.Infof("not found cert config, use default path: %s", tmpPath)
		CopyFile(common.DefaultCertPath+"/", tmpPath)
	}

	if config.Modules.EdgeHub.TLSCAFile != "" {
		CopyFile(config.Modules.EdgeHub.TLSCAFile, tmpPath)
	} else {
		klog.Infof("not found ca config, use default path: %s", tmpPath)
		CopyFile(constants.DefaultCAKeyFile, tmpPath)
	}

	ExecuteShell(common.CmdEdgecoreVersion, tmpPath)
}

// collect runtime/docker data
func collectRuntimeData(tmpPath string) {
	klog.Infof("create tmp file: %s", tmpPath)
	os.Mkdir(tmpPath, os.ModePerm)

	CopyFile(common.PathDockerService, tmpPath)
	ExecuteShell(common.CmdDockerVersion, tmpPath)
	ExecuteShell(common.CmdDockerInfo, tmpPath)
	ExecuteShell(common.CmdDockerImageInfo, tmpPath)
	ExecuteShell(common.CmdContainerInfo, tmpPath)
	ExecuteShell(common.CmdContainerLogInfo, tmpPath)
}

func CopyFile(pathSrc, tmpPath string) {
	c := fmt.Sprintf(common.CmdCopyFile, pathSrc, tmpPath)
	klog.Infof("Copy File: %s", c)
	cmd := exec.Command("sh", "-c", c)
	_, err := cmd.Output()
	if err != nil {
		klog.Warningf("fail to copy file:  %s", c)
		klog.Warningf("Output: %s", err.Error())
	}
}

func ExecuteShell(cmdStr string, tmpPath string) {
	c := fmt.Sprintf(cmdStr, tmpPath)
	klog.Infof("Execute Shell: %s", c)
	cmd := exec.Command("sh", "-c", c)
	_, err := cmd.Output()
	if err != nil {
		klog.Warningf("fail to execute Shell: %s", c)
		klog.Warningf("Output: %s", err.Error())
	}
}
