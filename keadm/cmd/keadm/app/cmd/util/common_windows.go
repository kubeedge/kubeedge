//go:build windows

/*
Copyright 2023 The KubeEdge Authors.

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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/blang/semver"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

// Constants used by installers
const (
	KubeEdgePath        = "C:\\etc\\kubeedge\\"
	KubeEdgeBackupPath  = "C:\\etc\\kubeedge\\backup\\"
	KubeEdgeUpgradePath = "C:\\etc\\kubeedge\\upgrade\\"
	KubeEdgeUsrBinPath  = "C:\\usr\\local\\bin"

	KubeEdgeLogPath = "C:\\var\\log\\kubeedge\\"

	KubeEdgeSocketPath = "C:\\var\\lib\\kubeedge\\"

	EdgeRootDir = "C:\\var\\lib\\edged"

	EdgeKubeletDir = "C:\\var\\lib\\kubelet"

	downloadFileScript = `
function DownloadFile($destination, $source) {
    Write-Host("Downloading $source to $destination")
    curl.exe --silent --fail -Lo $destination $source

    if (!$?) {
        Write-Error "Download $source failed"
        exit 1
    }
}
DownloadFile %s %s
`
)

func IsServiceExist(service string) bool {
	cmd := NewCommand(fmt.Sprintf("Get-Service '%s'", service))
	_err := cmd.Exec()
	return _err == nil
}

// IsKubeEdgeProcessRunning checks if the given process is running or not
func IsKubeEdgeProcessRunning(proc string) (bool, error) {
	// dont use nssm, maybe haven't installed yet
	if IsServiceExist(proc) {
		return true, nil
	}
	return false, nil
}

func HasSystemd() bool {
	return false
}

func CopyFile(src, dst string) error {
	return NewCommand(fmt.Sprintf("Copy-Item -Path %s -Destination %s -Force", src, dst)).Exec()
}

// RunningModuleV2 identifies cloudcore/edgecore running or not.
// only used for cloudcore container install and edgecore binary install
func RunningModuleV2(opt *types.ResetOptions) types.ModuleRunning {
	if IsNSSMServiceExist(KubeEdgeBinaryName) {
		return types.KubeEdgeEdgeRunning
	}
	return types.NoneRunning
}

func DownloadEdgecoreBin(options types.InstallOptions, version semver.Version) error {
	return installKubeEdge(options, version)
}

// installKubeEdge downloads the provided version of KubeEdge Edgecore For windows.
// Untar's in the specified location c:/etc/kubeedge/ and then copies
// the binary to excecutables' path (eg: c:/usr/local/bin)
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
	dirname := fmt.Sprintf("kubeedge-v%s-windows-%s", version, arch)
	filename := fmt.Sprintf("kubeedge-v%s-windows-%s.tar.gz", version, arch)
	checksumFilename := fmt.Sprintf("checksum_kubeedge-v%s-windows-%s.tar.gz.txt", version, arch)
	filePath := filepath.Join(options.TarballPath, filename)
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

	// decompress the release pkg
	if err = DecompressTarGz(filePath, options.TarballPath); err != nil {
		return err
	}
	// check if the edgecore.exe exists
	if !FileExists(filepath.Join(options.TarballPath, dirname, "edge", "edgecore.exe")) {
		return fmt.Errorf("cannot find edgecore binary at %s", filepath.Join(options.TarballPath, dirname, "edge", "edgecore.exe"))
	}
	os.MkdirAll(KubeEdgeUsrBinPath, os.ModePerm)
	// copy the binary to the executable path
	if err = CopyFile(filepath.Join(options.TarballPath, dirname, "edge", "edgecore.exe"), filepath.Join(KubeEdgeUsrBinPath, KubeEdgeBinaryName+".exe")); err != nil {
		return err
	}

	return nil
}

func checkSum(filename, checksumFilename string, version semver.Version, tarballPath string) (bool, error) {
	//Verify the tar with checksum
	fmt.Printf("%s checksum: \n", filename)

	actualChecksum, err := computeSHA512Checksum(filepath.Join(tarballPath, filename))
	if err != nil {
		return false, fmt.Errorf("failed to compute checksum for %s: %v", filename, err)
	}

	fmt.Printf("%s content: \n", checksumFilename)
	checksumFilepath := filepath.Join(tarballPath, checksumFilename)

	if _, err := os.Stat(checksumFilepath); err != nil {
		// download checksum file
		dwnldURL := fmt.Sprintf(downloadFileScript,
			checksumFilepath, fmt.Sprintf("%s/v%s/%s", KubeEdgeDownloadURL, version, checksumFilename))
		if err := NewCommand(dwnldURL).Exec(); err != nil {
			return false, err
		}
	}

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

	return true, nil
}

func retryDownload(filename, checksumFilename string, version semver.Version, tarballPath string) error {
	filePath := filepath.Join(tarballPath, filename)
	for try := 0; try < downloadRetryTimes; try++ {
		//Download the tar from repo
		dwnldURL := fmt.Sprintf(downloadFileScript,
			filePath, fmt.Sprintf("%s/v%s/%s", KubeEdgeDownloadURL, version, filename))
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
		if err = NewCommand(fmt.Sprintf("Remove-Item -Force %s", filePath)).Exec(); err != nil {
			return err
		}
	}
	return fmt.Errorf("failed to download %s", filename)
}

func runEdgeCore() error {
	return errors.New("not implemented")
}

func KillKubeEdgeBinary(proc string) error {
	return errors.New("not implemented")
}

func EdgeCoreRunningModuleV2(opt *types.ResetOptions) types.ModuleRunning {
	return types.NoneRunning
}

func CloudCoreRunningModuleV2(opt *types.ResetOptions) types.ModuleRunning {
	return types.NoneRunning
}
