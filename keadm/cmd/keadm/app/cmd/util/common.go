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
	"bytes"
	"fmt"
	"io"
	"k8s.io/klog"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/blang/semver"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

//Constants used by installers
const (
	UbuntuOSType   = "ubuntu"
	RaspbianOSType = "raspbian"
	CentOSType     = "centos"

	KubeEdgeDownloadURL          = "https://github.com/kubeedge/kubeedge/releases/download"
	EdgeCoreServiceFileURL       = "https://raw.githubusercontent.com/kubeedge/kubeedge/release-%s/build/tools/%s"
	KubeEdgePath                 = "/etc/kubeedge/"
	KubeEdgeUsrBinPath           = "/usr/local/bin"
	KubeEdgeConfPath             = KubeEdgePath + "kubeedge/edge/conf"
	KubeEdgeBinaryName           = "edgecore"
	KubeEdgeBinaryNamePre        = "edge_core"
	KubeEdgeCloudDefaultCertPath = KubeEdgePath + "certs/"
	KubeEdgeConfigEdgeYaml       = KubeEdgeConfPath + "/edge.yaml"
	KubeEdgeConfigModulesYaml    = KubeEdgeConfPath + "/modules.yaml"

	KubeEdgeCloudCertGenPath     = KubeEdgePath + "certgen.sh"
	KubeEdgeEdgeCertsTarFileName = "certs.tgz"
	KubeEdgeCloudConfPath        = KubeEdgePath + "kubeedge/cloud/conf"
	KubeEdgeCloudCoreYaml        = KubeEdgeCloudConfPath + "/controller.yaml"
	KubeEdgeCloudCoreModulesYaml = KubeEdgeCloudConfPath + "/modules.yaml"
	KubeCloudBinaryName          = "cloudcore"

	KubeEdgeNewConfigDir     = KubeEdgePath + "config/"
	KubeEdgeCloudCoreNewYaml = KubeEdgeNewConfigDir + "cloudcore.yaml"
	KubeEdgeEdgeCoreNewYaml  = KubeEdgeNewConfigDir + "edgecore.yaml"

	KubeEdgeLogPath = "/var/log/kubeedge/"
	KubeEdgeCrdPath = KubeEdgePath + "crds"

	KubeEdgeCRDDownloadURL = "https://raw.githubusercontent.com/kubeedge/kubeedge/master/build/crds"

	latestReleaseVersionURL = "https://kubeedge.io/latestversion"
	RetryTimes              = 5
)

//AddToolVals gets the value and default values of each flags and collects them in temporary cache
func AddToolVals(f *pflag.Flag, flagData map[string]types.FlagData) {
	flagData[f.Name] = types.FlagData{Val: f.Value.String(), DefVal: f.DefValue}
}

//CheckIfAvailable checks is val of a flag is empty then return the default value
func CheckIfAvailable(val, defval string) string {
	if val == "" {
		return defval
	}
	return val
}

//Common struct contains OS and Tool version properties and also embeds OS interface
type Common struct {
	types.OSTypeInstaller
	OSVersion   string
	ToolVersion semver.Version
	KubeConfig  string
	Master      string
}

//SetOSInterface defines a method to set the implemtation of the OS interface
func (co *Common) SetOSInterface(intf types.OSTypeInstaller) {
	co.OSTypeInstaller = intf
}

//Command defines commands to be executed and captures std out and std error
type Command struct {
	Cmd    *exec.Cmd
	StdOut []byte
	StdErr []byte
}

//ExecuteCommand executes the command and captures the output in stdOut
func (cm *Command) ExecuteCommand() {
	var err error
	cm.StdOut, err = cm.Cmd.Output()
	if err != nil {
		fmt.Println("Output failed: ", err)
		cm.StdErr = []byte(err.Error())
	}
}

//GetStdOutput gets StdOut field
func (cm Command) GetStdOutput() string {
	if len(cm.StdOut) != 0 {
		return strings.TrimRight(string(cm.StdOut), "\n")
	}
	return ""
}

//GetStdErr gets StdErr field
func (cm Command) GetStdErr() string {
	if len(cm.StdErr) != 0 {
		return strings.TrimRight(string(cm.StdErr), "\n")
	}
	return ""
}

//ExecuteCmdShowOutput captures both StdOut and StdErr after exec.cmd().
//It helps in the commands where it takes some time for execution.
func (cm Command) ExecuteCmdShowOutput() error {
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutIn, _ := cm.Cmd.StdoutPipe()
	stderrIn, _ := cm.Cmd.StderrPipe()

	var errStdout, errStderr error
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)
	err := cm.Cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start '%s' because of error : %s", strings.Join(cm.Cmd.Args, " "), err.Error())
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
		wg.Done()
	}()

	_, errStderr = io.Copy(stderr, stderrIn)
	wg.Wait()

	err = cm.Cmd.Wait()
	if err != nil {
		return fmt.Errorf("failed to run '%s' because of error : %s", strings.Join(cm.Cmd.Args, " "), err.Error())
	}
	if errStdout != nil || errStderr != nil {
		return fmt.Errorf("failed to capture stdout or stderr")
	}

	cm.StdOut, cm.StdErr = stdoutBuf.Bytes(), stderrBuf.Bytes()
	return nil
}

//GetOSVersion gets the OS name
func GetOSVersion() string {
	c := &Command{Cmd: exec.Command("sh", "-c", ". /etc/os-release && echo $ID")}
	c.ExecuteCommand()
	return c.GetStdOutput()
}

//GetOSInterface helps in returning OS specific object which implements OSTypeInstaller interface.
func GetOSInterface() types.OSTypeInstaller {
	switch GetOSVersion() {
	case UbuntuOSType, RaspbianOSType:
		return &UbuntuOS{}
	case CentOSType:
		return &CentOS{}
	default:
		fmt.Printf("This OS version is currently un-supported by keadm, %s", GetOSVersion())
		panic("This OS version is currently un-supported by keadm,")
	}
}

// IsCloudCore identifies if the node is having cloudcore already running.
// If so, then return true, else it can used as edge node and initialise it.
func IsCloudCore() (types.ModuleRunning, error) {
	osType := GetOSInterface()
	cloudCoreRunning, err := osType.IsKubeEdgeProcessRunning(KubeCloudBinaryName)
	if err != nil {
		return types.NoneRunning, err
	}

	if cloudCoreRunning {
		return types.KubeEdgeCloudRunning, nil
	}
	// check the process, and then check the service
	edgeCoreRunning, err := osType.IsKubeEdgeProcessRunning(KubeEdgeBinaryName)
	if err != nil {
		return types.NoneRunning, err
	}

	if edgeCoreRunning {
		return types.KubeEdgeEdgeRunning, nil
	}

	edgeCoreRunning, err = isEdgeCoreServiceRunning("edge")
	if err != nil {
		return types.NoneRunning, err
	}

	if edgeCoreRunning {
		return types.KubeEdgeEdgeRunning, nil
	}

	edgeCoreRunning, err = isEdgeCoreServiceRunning("edgecore")
	if err != nil {
		return types.NoneRunning, err
	}

	if edgeCoreRunning {
		return types.KubeEdgeEdgeRunning, nil
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

// runCommandWithShell executes the given command with "sh -c".
// It returns an error if the command outputs anything on the stderr.
func runCommandWithShell(command string) (string, error) {
	cmd := &Command{Cmd: exec.Command("sh", "-c", command)}
	err := cmd.ExecuteCmdShowOutput()
	if err != nil {
		return "", err
	}
	errout := cmd.GetStdErr()
	if errout != "" {
		return "", fmt.Errorf("failed to run command(%s), err:%s", command, errout)
	}
	return cmd.GetStdOutput(), nil
}

// runCommandWithStdout executes the given command with "sh -c".
// It returns the stdout and an error if the command outputs anything on the stderr.
func runCommandWithStdout(command string) (string, error) {
	cmd := &Command{Cmd: exec.Command("sh", "-c", command)}
	cmd.ExecuteCommand()

	if errout := cmd.GetStdErr(); errout != "" && errout != "exit status 3" {
		return "", fmt.Errorf("failed to run command(%s), err:%s", command, errout)
	}

	return cmd.GetStdOutput(), nil
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

//installKubeEdge downloads the provided version of KubeEdge.
//Untar's in the specified location /etc/kubeedge/ and then copies
//the binary to excecutables' path (eg: /usr/local/bin)
func installKubeEdge(componentType types.ComponentType, arch string, version semver.Version) error {
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
	filePath := fmt.Sprintf("%s%s", KubeEdgePath, filename)
	if _, err = os.Stat(filePath); err == nil {
		fmt.Printf("Expected or Default KubeEdge version %v is already downloaded and will checksum for it. \n", version)
		if success, _ := checkSum(filename, checksumFilename, version); !success {
			fmt.Printf("%v in your path checksum failed and do you want to delete this file and try to download again? \n", filename)
			for {
				confirm, err := askForconfirm()
				if err != nil {
					fmt.Println(err.Error())
					continue
				}
				if confirm {
					cmdStr := fmt.Sprintf("cd %s && rm -f %s", KubeEdgePath, filename)
					if _, err := runCommandWithStdout(cmdStr); err != nil {
						return err
					}
					klog.Infof("%v have been deleted and will try to download again", filename)
					if err := retryDownload(filename, checksumFilename, version); err != nil {
						return err
					}
				} else {
					klog.Warningf("failed to checksum and will continue to install.")
				}
				break
			}
		} else {
			klog.Infof("Expected or Default KubeEdge version %v is already downloaded and checksum successfully.", version)
		}
	} else if !os.IsNotExist(err) {
		return err
	} else {
		if err := retryDownload(filename, checksumFilename, version); err != nil {
			return err
		}
		return nil
	}

	/*
		When installing edgecore, if the version is >= 1.1,
		download the edgecore.service file from the KubeEdge/build/tools/ and place it in /etc/kubeedge/ acc.
	*/
	if componentType == types.EdgeCore {
		strippedVersion := fmt.Sprintf("%d.%d", version.Major, version.Minor)

		//	No need to download if the version is less than 1.1 (or 1.1.0)
		if version.GE(semver.MustParse("1.1.0")) {
			try := 0

			edgecoreServiceFileName := "edgecore.service"

			if version.EQ(semver.MustParse("1.1.0")) {
				edgecoreServiceFileName = "edge.service"
			}

			urlForServiceFile := fmt.Sprintf(EdgeCoreServiceFileURL, strippedVersion, edgecoreServiceFileName)
			for ; try < downloadRetryTimes; try++ {
				cmdStr := fmt.Sprintf("cd %s && sudo wget -k --no-check-certificate %s", KubeEdgePath, urlForServiceFile)
				_, err := runCommandWithStdout(cmdStr)
				if err != nil {
					return err
				}
				break
			}
			if try == downloadRetryTimes {
				return fmt.Errorf("failed to download %s", edgecoreServiceFileName)
			}
		}
	}

	// Compatible with 1.0.0
	var untarFileAndMoveCloudCore, untarFileAndMoveEdgeCore string
	if version.GE(semver.MustParse("1.1.0")) {
		if componentType == types.CloudCore {
			untarFileAndMoveCloudCore = fmt.Sprintf("cd %s && tar -C %s -xvzf %s && cp %s/%s/cloud/cloudcore/%s %s/",
				KubeEdgePath, KubeEdgePath, filename, KubeEdgePath, dirname, KubeCloudBinaryName, KubeEdgeUsrBinPath)
		}
		if componentType == types.EdgeCore {
			untarFileAndMoveEdgeCore = fmt.Sprintf("cd %s && tar -C %s -xvzf %s && cp %s%s/edge/%s %s/",
				KubeEdgePath, KubeEdgePath, filename, KubeEdgePath, dirname, KubeEdgeBinaryName, KubeEdgePath)
		}
	} else {
		untarFileAndMoveEdgeCore = fmt.Sprintf("cd %s && tar -C %s -xvzf %s && cp %skubeedge/edge/%s %s/.",
			KubeEdgePath, KubeEdgePath, filename, KubeEdgePath, KubeEdgeBinaryNamePre, KubeEdgePath)
		untarFileAndMoveCloudCore = fmt.Sprintf("cd %s && cp %skubeedge/cloud/%s %s/.",
			KubeEdgePath, KubeEdgePath, KubeCloudBinaryName, KubeEdgeUsrBinPath)
	}

	if componentType == types.CloudCore {
		stdout, err := runCommandWithStdout(untarFileAndMoveCloudCore)
		if err != nil {
			return err
		}
		fmt.Println(stdout)
	}
	if componentType == types.EdgeCore {
		stdout, err := runCommandWithStdout(untarFileAndMoveEdgeCore)
		if err != nil {
			return err
		}
		fmt.Println(stdout)
	}
	return nil
}

//runEdgeCore sets the environment variable GOARCHAIUS_CONFIG_PATH for the configuration path
//and the starts edgecore with logs being captured
func runEdgeCore(version semver.Version) error {
	// create the log dir for kubeedge
	err := os.MkdirAll(KubeEdgeLogPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgeLogPath)
	}

	var binaryName string

	if version.GE(semver.MustParse("1.1.0")) {
		binaryName = KubeEdgeBinaryName
	} else {
		binaryName = KubeEdgeBinaryNamePre
	}

	// add +x for edgecore
	command := fmt.Sprintf("chmod +x %s%s", KubeEdgePath, binaryName)
	if _, err := runCommandWithStdout(command); err != nil {
		return err
	}

	var binExec string

	systemdExist := hasSystemd()

	edgecoreServiceName := "edgecore"

	if version.GE(semver.MustParse("1.1.0")) && systemdExist {
		if version.EQ(semver.MustParse("1.1.0")) {
			edgecoreServiceName = "edge"
		}
		binExec = fmt.Sprintf("sudo ln /etc/kubeedge/%s.service /etc/systemd/system/%s.service && sudo systemctl daemon-reload && sudo systemctl enable %s && sudo systemctl start %s", edgecoreServiceName, edgecoreServiceName, edgecoreServiceName, edgecoreServiceName)
	} else {
		binExec = fmt.Sprintf("%s > %skubeedge/edge/%s.log 2>&1 &", KubeEdgeBinaryName, KubeEdgePath, binaryName)
	}

	cmd := &Command{Cmd: exec.Command("sh", "-c", binExec)}
	cmd.Cmd.Env = os.Environ()
	env := fmt.Sprintf("GOARCHAIUS_CONFIG_PATH=%skubeedge/edge", KubeEdgePath)
	cmd.Cmd.Env = append(cmd.Cmd.Env, env)
	err = cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	if version.GE(semver.MustParse("1.1.0")) {
		if systemdExist {
			fmt.Printf("KubeEdge edgecore is running, For logs visit: journalctl -u %s.service -b\n", edgecoreServiceName)
		} else {
			fmt.Println("KubeEdge edgecore is running, For logs visit: ", KubeEdgeLogPath+binaryName+".log")
		}
	} else {
		fmt.Println("KubeEdge edgecore is running, For logs visit", KubeEdgePath+"kubeedge/edge/"+binaryName+".log")
	}

	return nil
}

// killKubeEdgeBinary will search for KubeEdge process and forcefully kill it
func killKubeEdgeBinary(proc string) error {
	var binExec string
	if proc == "cloudcore" {
		binExec = fmt.Sprintf("kill -9 $(ps aux | grep '[%s]%s' | awk '{print $2}')", proc[0:1], proc[1:])
	} else {
		systemdExist := hasSystemd()

		var serviceName string
		if running, err := isEdgeCoreServiceRunning("edge"); err == nil && running {
			serviceName = "edge"
		}
		if running, err := isEdgeCoreServiceRunning("edgecore"); err == nil && running {
			serviceName = "edgecore"
		}

		if systemdExist {
			// remove the system service.
			binExec = fmt.Sprintf("sudo systemctl stop %s.service && sudo rm /etc/systemd/system/%s.service && sudo systemctl daemon-reload && systemctl reset-failed", serviceName, serviceName)
		} else {
			binExec = fmt.Sprintf("kill $(ps aux | grep '[%s]%s' | awk '{print $2}')", proc[0:1], proc[1:])
		}
	}
	if _, err := runCommandWithStdout(binExec); err != nil {
		return err
	}

	fmt.Println("KubeEdge", proc, "is stopped, For logs visit: ", KubeEdgeLogPath+proc+".log")
	return nil
}

//isKubeEdgeProcessRunning checks if the given process is running or not
func isKubeEdgeProcessRunning(proc string) (bool, error) {
	procRunning := fmt.Sprintf("ps aux | grep '[%s]%s' | awk '{print $2}'", proc[0:1], proc[1:])
	stdout, err := runCommandWithStdout(procRunning)
	if err != nil {
		return false, err
	}
	if stdout != "" {
		return true, nil
	}

	return false, nil
}

func isEdgeCoreServiceRunning(serviceName string) (bool, error) {
	serviceRunning := fmt.Sprintf("systemctl list-unit-files | grep enabled | grep %s ", serviceName)
	stdout, err := runCommandWithStdout(serviceRunning)

	if err != nil {
		return false, err
	}
	if stdout != "" {
		return true, nil
	}

	return false, nil
}

//	check if systemd exist
func hasSystemd() bool {
	cmd := "file /sbin/init"

	stdout, err := runCommandWithStdout(cmd)

	if err != nil {
		return false
	}

	if strings.Contains(stdout, "systemd") {
		return true
	}

	return false
}

func checkSum(filename, checksumFilename string, version semver.Version) (bool, error) {
	//Verify the tar with checksum
	fmt.Printf("%s checksum: \n", filename)
	cmdStr := fmt.Sprintf("cd %s && sha512sum %s | awk '{split($0,a,\"[ ]\"); print a[1]}'", KubeEdgePath, filename)
	actualChecksum, err := runCommandWithStdout(cmdStr)
	if err != nil {
		return false, err
	}

	fmt.Printf("%s content: \n", checksumFilename)
	cmdStr = fmt.Sprintf("wget -qO- %s/v%s/%s", KubeEdgeDownloadURL, version, checksumFilename)
	desiredChecksum, err := runCommandWithStdout(cmdStr)
	if err != nil {
		return false, err
	}

	if desiredChecksum != actualChecksum {
		fmt.Printf("Failed to verify the checksum of %s, try to download it again ... \n\n", filename)
		//Cleanup the downloaded files
		cmdStr = fmt.Sprintf("cd %s && rm -f %s", KubeEdgePath, filename)
		_, err = runCommandWithStdout(cmdStr)
		return false, err
	}
	return true, nil
}

func retryDownload(filename, checksumFilename string, version semver.Version) error {
	try := 0
	for ; try < downloadRetryTimes; try++ {
		//Download the tar from repo
		dwnldURL := fmt.Sprintf("cd %s && wget -k --no-check-certificate --progress=bar:force %s/v%s/%s",
			KubeEdgePath, KubeEdgeDownloadURL, version, filename)
		if _, err := runCommandWithShell(dwnldURL); err != nil {
			return err
		}

		//Verify the tar with checksum
		fmt.Printf("%s checksum: \n", filename)
		cmdStr := fmt.Sprintf("cd %s && sha512sum %s | awk '{split($0,a,\"[ ]\"); print a[1]}'", KubeEdgePath, filename)
		actualChecksum, err := runCommandWithStdout(cmdStr)
		if err != nil {
			return err
		}

		fmt.Printf("%s content: \n", checksumFilename)
		cmdStr = fmt.Sprintf("wget -qO- %s/v%s/%s", KubeEdgeDownloadURL, version, checksumFilename)
		desiredChecksum, err := runCommandWithStdout(cmdStr)
		if err != nil {
			return err
		}

		if desiredChecksum != actualChecksum {
			fmt.Printf("Failed to verify the checksum of %s, try to download it again ... \n\n", filename)
			//Cleanup the downloaded files
			cmdStr = fmt.Sprintf("cd %s && rm -f %s", KubeEdgePath, filename)
			if _, err := runCommandWithStdout(cmdStr); err != nil {
				return err
			}
		} else {
			break
		}
	}
	if try == downloadRetryTimes {
		return fmt.Errorf("failed to download %s", filename)
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
