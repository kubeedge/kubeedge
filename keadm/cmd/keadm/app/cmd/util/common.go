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

package util

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/blang/semver"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

// Constants used by installers
const (
	KubeEdgeDownloadURL  = "https://github.com/kubeedge/kubeedge/releases/download"
	EdgeServiceFile      = "edgecore.service"
	CloudServiceFile     = "cloudcore.service"
	ServiceFileURLFormat = "https://raw.githubusercontent.com/kubeedge/kubeedge/release-%s/build/tools/%s"
	KubeEdgePath         = "/etc/kubeedge/"
	KubeEdgeUsrBinPath   = "/usr/local/bin"
	KubeEdgeBinaryName   = "edgecore"

	KubeCloudBinaryName = "cloudcore"

	KubeEdgeConfigDir        = KubeEdgePath + "config/"
	KubeEdgeCloudCoreNewYaml = KubeEdgeConfigDir + "cloudcore.yaml"
	KubeEdgeEdgeCoreNewYaml  = KubeEdgeConfigDir + "edgecore.yaml"

	KubeEdgeLogPath = "/var/log/kubeedge/"
	KubeEdgeCrdPath = KubeEdgePath + "crds"

	KubeEdgeSocketPath = "/var/lib/kubeedge/"

	EdgeRootDir = "/var/lib/edged"

	SystemdBootPath = "/run/systemd/system"

	KubeEdgeCRDDownloadURL = "https://raw.githubusercontent.com/kubeedge/kubeedge/release-%s/build/crds"

	latestReleaseVersionURL = "https://kubeedge.io/latestversion"
	RetryTimes              = 5

	OSArchAMD64 string = "amd64"
	OSArchARM64 string = "arm64"
	OSArchARM32 string = "arm"

	APT    string = "apt"
	YUM    string = "yum"
	PACMAN string = "pacman"
)

// AddToolVals gets the value and default values of each flags and collects them in temporary cache
func AddToolVals(f *pflag.Flag, flagData map[string]types.FlagData) {
	flagData[f.Name] = types.FlagData{Val: f.Value.String(), DefVal: f.DefValue}
}

// CheckIfAvailable checks is val of a flag is empty then return the default value
func CheckIfAvailable(val, defval string) string {
	if val == "" {
		return defval
	}
	return val
}

// Common struct contains OS and Tool version properties and also embeds OS interface
type Common struct {
	types.OSTypeInstaller
	OSVersion   string
	ToolVersion semver.Version
	KubeConfig  string
	Master      string
}

// SetOSInterface defines a method to set the implemtation of the OS interface
func (co *Common) SetOSInterface(intf types.OSTypeInstaller) {
	co.OSTypeInstaller = intf
}

// GetPackageManager get package manager of OS
func GetPackageManager() string {
	cmd := NewCommand("command -v apt || command -v yum || command -v pacman")
	err := cmd.Exec()
	if err != nil {
		fmt.Println(err)
		return ""
	}

	if strings.HasSuffix(cmd.GetStdOut(), APT) {
		return APT
	} else if strings.HasSuffix(cmd.GetStdOut(), YUM) {
		return YUM
	} else if strings.HasSuffix(cmd.GetStdOut(), PACMAN) {
		return PACMAN
	} else {
		return ""
	}
}

// GetOSInterface helps in returning OS specific object which implements OSTypeInstaller interface.
func GetOSInterface() types.OSTypeInstaller {
	switch GetPackageManager() {
	case APT:
		return &DebOS{}
	case YUM:
		return &RpmOS{}
	case PACMAN:
		return &PacmanOS{}
	default:
		fmt.Println("Failed to detect supported package manager command(apt, yum, pacman), exit")
		panic("Failed to detect supported package manager command(apt, yum, pacman), exit")
	}
}

// RunningModule identifies cloudcore/edgecore running or not.
func RunningModule() (types.ModuleRunning, error) {
	osType := GetOSInterface()
	cloudCoreRunning, err := osType.IsKubeEdgeProcessRunning(KubeCloudBinaryName)

	if cloudCoreRunning {
		return types.KubeEdgeCloudRunning, nil
	} else if err != nil {
		return types.NoneRunning, err
	}

	edgeCoreRunning, err := osType.IsKubeEdgeProcessRunning(KubeEdgeBinaryName)

	if edgeCoreRunning {
		return types.KubeEdgeEdgeRunning, nil
	} else if err != nil {
		return types.NoneRunning, err
	}

	return types.NoneRunning, nil
}

// GetLatestVersion return the latest non-prerelease, non-draft version of kubeedge in releases
func GetLatestVersion() (string, error) {
	//Download the tar from repo
	versionURL := "curl -k " + latestReleaseVersionURL
	cmd := exec.Command("sh", "-c", versionURL)
	latestReleaseData, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(latestReleaseData), nil
}

// build Config from flags
func BuildConfig(kubeConfig, master string) (conf *rest.Config, err error) {
	config, err := clientcmd.BuildConfigFromFlags(master, kubeConfig)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// isK8SComponentInstalled checks if said K8S version is already installed in the host
func isK8SComponentInstalled(kubeConfig, master string) error {
	config, err := BuildConfig(kubeConfig, master)
	if err != nil {
		return fmt.Errorf("Failed to build config, err: %v", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("Failed to init discovery client, err: %v", err)
	}

	discoveryClient.RESTClient().Post()
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return fmt.Errorf("Failed to get the version of K8s master, please check whether K8s was successfully installed, err: %v", err)
	}

	return checkKubernetesVersion(serverVersion)
}

func checkKubernetesVersion(serverVersion *version.Info) error {
	reg := regexp.MustCompile(`[[:digit:]]*`)
	minorVersion := reg.FindString(serverVersion.Minor)

	k8sMinorVersion, err := strconv.Atoi(minorVersion)
	if err != nil {
		return fmt.Errorf("Could not parse the minor version of K8s, error: %s", err)
	}
	if k8sMinorVersion >= types.DefaultK8SMinimumVersion {
		return nil
	}

	return fmt.Errorf("Your minor version of K8s is lower than %d, please reinstall newer version", types.DefaultK8SMinimumVersion)
}

// installKubeEdge downloads the provided version of KubeEdge.
// Untar's in the specified location /etc/kubeedge/ and then copies
// the binary to excecutables' path (eg: /usr/local/bin)
func installKubeEdge(options types.InstallOptions, arch string, version semver.Version) error {
	// create the storage path of the kubeedge installation packages
	if options.TarballPath == "" {
		options.TarballPath = KubeEdgePath
	} else {
		err := os.MkdirAll(options.TarballPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("not able to create %s folder path", options.TarballPath)
		}
	}

	err := os.MkdirAll(KubeEdgePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgePath)
	}

	if arch == "armhf" {
		arch = "arm"
	}

	//Check if the same version exists, then skip the download and just checksum for it
	//and if checksum failed, there will be a option to choose to continue to untar or quit.
	//checksum available at download URL. So that both can be compared to see if
	//proper download has happened and then only proceed further.
	//Currently it is missing and once checksum is in place, checksum check required
	//to be added here.
	dirname := fmt.Sprintf("kubeedge-v%s-linux-%s", version, arch)
	filename := fmt.Sprintf("kubeedge-v%s-linux-%s.tar.gz", version, arch)
	checksumFilename := fmt.Sprintf("checksum_kubeedge-v%s-linux-%s.tar.gz.txt", version, arch)
	filePath := fmt.Sprintf("%s/%s", options.TarballPath, filename)
	if _, err = os.Stat(filePath); err == nil {
		fmt.Printf("Expected or Default KubeEdge version %v is already downloaded and will checksum for it. \n", version)
		if success, _ := checkSum(filename, checksumFilename, version, options.TarballPath); !success {
			fmt.Printf("%v in your path checksum failed and do you want to delete this file and try to download again? \n", filename)
			for {
				confirm, err := askForconfirm()
				if err != nil {
					fmt.Println(err.Error())
					continue
				}
				if confirm {
					cmdStr := fmt.Sprintf("cd %s && rm -f %s", options.TarballPath, filename)
					if err := NewCommand(cmdStr).Exec(); err != nil {
						return err
					}
					klog.Infof("%v have been deleted and will try to download again", filename)
					if err := retryDownload(filename, checksumFilename, version, options.TarballPath); err != nil {
						return err
					}
				} else {
					klog.Warningf("failed to checksum and will continue to install.")
				}
				break
			}
		} else {
			fmt.Println("Expected or Default KubeEdge version", version, "is already downloaded")
		}
	} else if !os.IsNotExist(err) {
		return err
	} else {
		if err := retryDownload(filename, checksumFilename, version, options.TarballPath); err != nil {
			return err
		}
	}

	if err := downloadServiceFile(options.ComponentType, version, KubeEdgePath); err != nil {
		return fmt.Errorf("fail to download service file,error:{%s}", err.Error())
	}

	var untarFileAndMoveCloudCore, untarFileAndMoveEdgeCore string

	if options.ComponentType == types.CloudCore {
		untarFileAndMoveCloudCore = fmt.Sprintf("cd %s && tar -C %s -xvzf %s && cp %s/%s/cloud/cloudcore/%s %s/",
			options.TarballPath, options.TarballPath, filename, options.TarballPath, dirname, KubeCloudBinaryName, KubeEdgeUsrBinPath)

		cmd := NewCommand(untarFileAndMoveCloudCore)
		if err := cmd.Exec(); err != nil {
			return err
		}
		fmt.Println(cmd.GetStdOut())
	} else if options.ComponentType == types.EdgeCore {
		untarFileAndMoveEdgeCore = fmt.Sprintf("cd %s && tar -C %s -xvzf %s && cp %s/%s/edge/%s %s/",
			options.TarballPath, options.TarballPath, filename, options.TarballPath, dirname, KubeEdgeBinaryName, KubeEdgeUsrBinPath)
		cmd := NewCommand(untarFileAndMoveEdgeCore)
		if err := cmd.Exec(); err != nil {
			return err
		}
		fmt.Println(cmd.GetStdOut())
	}

	return nil
}

// runEdgeCore starts edgecore with logs being captured
func runEdgeCore(version semver.Version) error {
	// create the log dir for kubeedge
	err := os.MkdirAll(KubeEdgeLogPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgeLogPath)
	}

	var binExec string

	systemdExist := hasSystemd()

	edgecoreServiceName := "edgecore"

	if systemdExist {
		binExec = fmt.Sprintf("sudo ln /etc/kubeedge/%s.service /etc/systemd/system/%s.service && sudo systemctl daemon-reload && sudo systemctl enable %s && sudo systemctl start %s", edgecoreServiceName, edgecoreServiceName, edgecoreServiceName, edgecoreServiceName)
	} else {
		binExec = fmt.Sprintf("%s/%s > %skubeedge/edge/%s.log 2>&1 &", KubeEdgeUsrBinPath, KubeEdgeBinaryName, KubeEdgePath, KubeEdgeBinaryName)
	}

	cmd := NewCommand(binExec)
	if err := cmd.Exec(); err != nil {
		return err
	}

	fmt.Println(cmd.GetStdOut())

	if systemdExist {
		fmt.Printf("KubeEdge edgecore is running, For logs visit: journalctl -u %s.service -b\n", edgecoreServiceName)
	} else {
		fmt.Println("KubeEdge edgecore is running, For logs visit: ", KubeEdgeLogPath+KubeEdgeBinaryName+".log")
	}

	return nil
}

// killKubeEdgeBinary will search for KubeEdge process and forcefully kill it
func killKubeEdgeBinary(proc string) error {
	var binExec string
	if proc == "cloudcore" {
		binExec = fmt.Sprintf("pkill %s", proc)
	} else {
		systemdExist := hasSystemd()

		var serviceName string
		if running, err := isEdgeCoreServiceRunning("edge"); err == nil && running {
			serviceName = "edge"
		}
		if running, err := isEdgeCoreServiceRunning("edgecore"); err == nil && running {
			serviceName = "edgecore"
		}

		if systemdExist && serviceName != "" {
			// remove the system service.
			serviceFilePath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
			serviceFileRemoveExec := fmt.Sprintf("&& sudo rm %s", serviceFilePath)
			if _, err := os.Stat(serviceFilePath); err != nil && os.IsNotExist(err) {
				serviceFileRemoveExec = ""
			}
			binExec = fmt.Sprintf("sudo systemctl stop %s.service && sudo systemctl disable %s.service %s && sudo systemctl daemon-reload", serviceName, serviceName, serviceFileRemoveExec)
		} else {
			binExec = fmt.Sprintf("pkill %s", proc)
		}
	}
	cmd := NewCommand(binExec)
	if err := cmd.Exec(); err != nil {
		return err
	}

	fmt.Println(proc, "is stopped")
	return nil
}

// isKubeEdgeProcessRunning checks if the given process is running or not
func isKubeEdgeProcessRunning(proc string) (bool, error) {
	procRunning := fmt.Sprintf("pidof %s 2>&1", proc)
	cmd := NewCommand(procRunning)

	err := cmd.Exec()

	if cmd.ExitCode == 0 {
		return true, nil
	} else if cmd.ExitCode == 1 {
		return false, nil
	}

	return false, err
}

func isEdgeCoreServiceRunning(serviceName string) (bool, error) {
	serviceRunning := fmt.Sprintf("systemctl list-unit-files | grep enabled | grep %s ", serviceName)
	cmd := NewCommand(serviceRunning)
	err := cmd.Exec()

	if cmd.ExitCode == 0 {
		return true, nil
	} else if cmd.ExitCode == 1 {
		return false, nil
	}

	return false, err
}

// check if systemd exist
// if command run failed, then check it by sd_booted
func hasSystemd() bool {
	cmd := "file /sbin/init"

	if err := NewCommand(cmd).Exec(); err == nil {
		return true
	}
	// checks whether `SystemdBootPath` exists and is a directory
	// reference http://www.freedesktop.org/software/systemd/man/sd_booted.html
	fi, err := os.Lstat(SystemdBootPath)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

func checkSum(filename, checksumFilename string, version semver.Version, tarballPath string) (bool, error) {
	//Verify the tar with checksum
	fmt.Printf("%s checksum: \n", filename)
	getActualCheckSum := NewCommand(fmt.Sprintf("cd %s && sha512sum %s | awk '{split($0,a,\"[ ]\"); print a[1]}'", tarballPath, filename))
	if err := getActualCheckSum.Exec(); err != nil {
		return false, err
	}

	fmt.Printf("%s content: \n", checksumFilename)
	getDesiredCheckSum := NewCommand(fmt.Sprintf("wget -qO- %s/v%s/%s", KubeEdgeDownloadURL, version, checksumFilename))
	if err := getDesiredCheckSum.Exec(); err != nil {
		return false, err
	}

	if getDesiredCheckSum.GetStdOut() != getActualCheckSum.GetStdOut() {
		fmt.Printf("Failed to verify the checksum of %s ... \n\n", filename)
		return false, nil
	}
	return true, nil
}

func retryDownload(filename, checksumFilename string, version semver.Version, tarballPath string) error {
	filePath := filepath.Join(tarballPath, filename)
	for try := 0; try < downloadRetryTimes; try++ {
		//Download the tar from repo
		dwnldURL := fmt.Sprintf("cd %s && wget -k --no-check-certificate --progress=bar:force %s/v%s/%s",
			tarballPath, KubeEdgeDownloadURL, version, filename)
		if err := NewCommand(dwnldURL).Exec(); err != nil {
			return err
		}

		//Verify the tar with checksum
		success, err := checkSum(filename, checksumFilename, version, tarballPath)
		if err != nil {
			return err
		}
		if success {
			return nil
		}
		fmt.Printf("Failed to verify the checksum of %s, try to download it again ... \n\n", filename)
		//Cleanup the downloaded files
		if err = NewCommand(fmt.Sprintf("rm -f %s", filePath)).Exec(); err != nil {
			return err
		}
	}
	return fmt.Errorf("failed to download %s", filename)
}

// Compressed folders or files
func Compress(tarName string, paths []string) (err error) {
	tarFile, err := os.Create(tarName)
	if err != nil {
		return err
	}
	defer func() {
		err = tarFile.Close()
	}()

	absTar, err := filepath.Abs(tarName)
	if err != nil {
		return err
	}

	// enable compression if file ends in .gz
	tw := tar.NewWriter(tarFile)
	if strings.HasSuffix(tarName, ".gz") || strings.HasSuffix(tarName, ".gzip") {
		gz := gzip.NewWriter(tarFile)
		defer gz.Close()
		tw = tar.NewWriter(gz)
	}
	defer tw.Close()

	// walk each specified path and add encountered file to tar
	for _, path := range paths {
		// validate path
		path = filepath.Clean(path)
		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if absPath == absTar {
			fmt.Printf("tar file %s cannot be the source\n", tarName)
			continue
		}
		if absPath == filepath.Dir(absTar) {
			fmt.Printf("tar file %s cannot be in source %s\n", tarName, absPath)
			continue
		}

		walker := func(file string, finfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// fill in header info using func FileInfoHeader
			hdr, err := tar.FileInfoHeader(finfo, finfo.Name())
			if err != nil {
				return err
			}

			relFilePath := file
			if filepath.IsAbs(path) {
				relFilePath, err = filepath.Rel(path, file)
				if err != nil {
					return err
				}
			}
			// ensure header has relative file path
			hdr.Name = relFilePath

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			// if path is a dir, dont continue
			if finfo.Mode().IsDir() {
				return nil
			}

			// add file to tar
			srcFile, err := os.Open(file)
			if err != nil {
				return err
			}
			defer srcFile.Close()
			_, err = io.Copy(tw, srcFile)
			if err != nil {
				return err
			}
			return nil
		}

		// build tar
		if err := filepath.Walk(path, walker); err != nil {
			fmt.Printf("failed to add %s to tar: %s\n", path, err)
		}
	}
	return nil
}

func askForconfirm() (bool, error) {
	var s string

	fmt.Println("[y/N]: ")
	if _, err := fmt.Scan(&s); err != nil {
		return false, err
	}

	s = strings.ToLower(strings.TrimSpace(s))

	if s == "y" {
		return true, nil
	} else if s == "n" {
		return false, nil
	} else {
		return false, fmt.Errorf("Invalid Input")
	}
}

// Execute shell script and filter
func ExecShellFilter(c string) (string, error) {
	cmd := NewCommand(c)
	if err := cmd.Exec(); err != nil {
		return "", err
	}

	return cmd.GetStdOut(), nil
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

func ParseEdgecoreConfig(edgecorePath string) (*v1alpha1.EdgeCoreConfig, error) {
	edgeCoreConfig := v1alpha1.NewDefaultEdgeCoreConfig()
	if err := edgeCoreConfig.Parse(edgecorePath); err != nil {
		return nil, err
	}
	return edgeCoreConfig, nil
}

// Determine if it is in the array
func IsContain(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

// print fail
func PrintFail(cmd string, s string) {
	v := fmt.Sprintf("|%s %s failed|", s, cmd)
	printResult(v)
}

//print success
func PrintSucceed(cmd string, s string) {
	v := fmt.Sprintf("|%s %s succeed|", s, cmd)
	printResult(v)
}

func printResult(s string) {
	line := "|"
	if len(s) > 2 {
		for i := 0; i < len(s)-2; i++ {
			line = line + "-"
		}
		line = line + "|"
	}

	fmt.Println("")
	fmt.Println(line)
	fmt.Println(s)
	fmt.Println(line)
}

func downloadServiceFile(componentType types.ComponentType, version semver.Version, storeDir string) error {
	// No need to download if
	// 1. the systemd not exists
	// 2. the service file already exists
	if hasSystemd() {
		var ServiceFileName string
		switch componentType {
		case types.CloudCore:
			ServiceFileName = CloudServiceFile
		case types.EdgeCore:
			ServiceFileName = EdgeServiceFile
		default:
			return fmt.Errorf("component type %s not support", componentType)
		}
		ServiceFilePath := storeDir + "/" + ServiceFileName
		strippedVersion := fmt.Sprintf("%d.%d", version.Major, version.Minor)
		ServiceFileURL := fmt.Sprintf(ServiceFileURLFormat, strippedVersion, ServiceFileName)
		if _, err := os.Stat(ServiceFilePath); err != nil {
			if os.IsNotExist(err) {
				cmdStr := fmt.Sprintf("cd %s && sudo -E wget -t %d -k --no-check-certificate %s", storeDir, RetryTimes, ServiceFileURL)
				fmt.Printf("[Run as service] start to download service file for %s\n", componentType)
				if err := NewCommand(cmdStr).Exec(); err != nil {
					return err
				}
				fmt.Printf("[Run as service] success to download service file for %s\n", componentType)
			} else {
				return err
			}
		} else {
			fmt.Printf("[Run as service] service file already exisits in %s, skip download\n", ServiceFilePath)
		}
	}
	return nil
}
