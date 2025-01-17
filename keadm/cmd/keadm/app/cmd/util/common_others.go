//go:build !windows

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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/blang/semver"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/constants"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

// Constants used by installers
const (
	KubeEdgePath        = "/etc/kubeedge/"
	KubeEdgeBackupPath  = "/etc/kubeedge/backup/"
	KubeEdgeUpgradePath = "/etc/kubeedge/upgrade/"
	KubeEdgeUsrBinPath  = "/usr/local/bin"

	KubeEdgeLogPath = "/var/log/kubeedge/"

	KubeEdgeSocketPath = "/var/lib/kubeedge/"

	EdgeRootDir = "/var/lib/edged"

	EdgeKubeletDir = "/var/lib/kubelet"

	SystemdBootPath = "/run/systemd/system"
)

// IsKubeEdgeProcessRunning checks if the given process is running or not
func IsKubeEdgeProcessRunning(proc string) (bool, error) {
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

// HasSystemd checks if systemd exist.
// if command run failed, then check it by sd_booted.
func HasSystemd() bool {
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

// RunningModuleV2 identifies cloudcore/edgecore running or not.
// only used for cloudcore container install and edgecore binary install
func RunningModuleV2(opt *types.ResetOptions) types.ModuleRunning {
	result := CloudCoreRunningModuleV2(opt)
	if result == types.KubeEdgeCloudRunning {
		return result
	}
	return EdgeCoreRunningModuleV2(opt)
}

func CloudCoreRunningModuleV2(opt *types.ResetOptions) types.ModuleRunning {
	cloudCoreRunning, err := IsCloudcoreContainerRunning(constants.SystemNamespace, opt.Kubeconfig)
	if err != nil {
		// just log the error, maybe we do not care
		klog.Warningf("failed to check cloudcore is running: %v", err)
	}
	if cloudCoreRunning {
		return types.KubeEdgeCloudRunning
	}
	return types.NoneRunning
}

func EdgeCoreRunningModuleV2(*types.ResetOptions) types.ModuleRunning {
	osType := GetOSInterface()
	edgeCoreRunning, err := osType.IsKubeEdgeProcessRunning(KubeEdgeBinaryName)
	if err != nil {
		// just log the error, maybe we do not care
		klog.Warningf("failed to check edgecore is running: %v", err)
	}
	if edgeCoreRunning {
		return types.KubeEdgeEdgeRunning
	}

	return types.NoneRunning
}

// Compress compresses folders or files

// KillKubeEdgeBinary will search for KubeEdge process and forcefully kill it
func KillKubeEdgeBinary(proc string) error {
	var binExec string
	if proc == "cloudcore" {
		binExec = fmt.Sprintf("pkill %s", proc)
	} else {
		systemdExist := HasSystemd()

		var serviceName string
		if running, err := isEdgeCoreServiceRunning("edge"); err == nil && running {
			serviceName = "edge"
		}
		if running, err := isEdgeCoreServiceRunning(KubeEdgeBinaryName); err == nil && running {
			serviceName = KubeEdgeBinaryName
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

func checkSum(filename, checksumFilename string, version semver.Version, tarballPath string) (bool, error) {
	//Verify the tar with checksum
	fmt.Printf("%s checksum: \n", filename)

	filepath := fmt.Sprintf("%s/%s", tarballPath, filename)
	actualChecksum, err := computeSHA512Checksum(filepath)
	if err != nil {
		return false, fmt.Errorf("failed to compute checksum for %s: %v", filename, err)
	}

	fmt.Printf("%s content: \n", checksumFilename)
	checksumFilepath := fmt.Sprintf("%s/%s", tarballPath, checksumFilename)

	if _, err := os.Stat(checksumFilepath); err == nil {
		fmt.Printf("Expected or Default checksum file %s is already downloaded. \n", checksumFilename)
		content, err := os.ReadFile(checksumFilepath)
		if err != nil {
			return false, err
		}
		checksum := strings.Replace(string(content), "\n", "", -1)
		if checksum != actualChecksum {
			fmt.Printf("Failed to verify the checksum of %s ... \n\n", filename)
			return false, nil
		}
	} else {
		getDesiredCheckSum := NewCommand(fmt.Sprintf("wget -qO- %s/v%s/%s", KubeEdgeDownloadURL, version, checksumFilename))
		if err := getDesiredCheckSum.Exec(); err != nil {
			return false, err
		}

		if getDesiredCheckSum.GetStdOut() != actualChecksum {
			fmt.Printf("Failed to verify the checksum of %s ... \n\n", filename)
			return false, nil
		}
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

// runEdgeCore starts edgecore with logs being captured
func runEdgeCore() error {
	// create the log dir for kubeedge
	err := os.MkdirAll(KubeEdgeLogPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgeLogPath)
	}

	systemdExist := HasSystemd()

	var binExec string
	if systemdExist {
		binExec = fmt.Sprintf("sudo ln /etc/kubeedge/%s.service /etc/systemd/system/%s.service && sudo systemctl daemon-reload && sudo systemctl enable %s && sudo systemctl start %s",
			types.EdgeCore, types.EdgeCore, types.EdgeCore, types.EdgeCore)
	} else {
		binExec = fmt.Sprintf("%s/%s > %skubeedge/edge/%s.log 2>&1 &", KubeEdgeUsrBinPath, KubeEdgeBinaryName, KubeEdgePath, KubeEdgeBinaryName)
	}

	cmd := NewCommand(binExec)
	if err := cmd.Exec(); err != nil {
		return err
	}
	fmt.Println(cmd.GetStdOut())

	if systemdExist {
		fmt.Printf("KubeEdge edgecore is running, For logs visit: journalctl -u %s.service -xe\n", types.EdgeCore)
	} else {
		fmt.Println("KubeEdge edgecore is running, For logs visit: ", KubeEdgeLogPath+KubeEdgeBinaryName+".log")
	}
	return nil
}

// installKubeEdge downloads the provided version of KubeEdge.
// Untar's in the specified location /etc/kubeedge/ and then copies
// the binary to excecutables' path (eg: /usr/local/bin)
func installKubeEdge(options types.InstallOptions, version semver.Version) error {
	// program's architecture: amd64, arm64, arm
	arch := runtime.GOARCH

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

	//Check if the same version exists, then skip the download and just checksum for it
	//and if checksum failed, there will be an option to choose to continue to untar or quit.
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
					fmt.Printf("%v have been deleted and will try to download again\n", filename)
					if err := retryDownload(filename, checksumFilename, version, options.TarballPath); err != nil {
						return err
					}
				} else {
					fmt.Println("failed to checksum and will continue to install.")
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
