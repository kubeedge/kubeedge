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
	"crypto/sha512"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/blang/semver"
	"github.com/spf13/pflag"
	versionutil "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/constants"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
	pkgversion "github.com/kubeedge/kubeedge/pkg/version"
)

var (
	kubeReleaseRegex = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)([-0-9a-zA-Z_\.+]*)?$`)
)

// Constants used by installers
const (
	KubeEdgeDownloadURL  = "https://github.com/kubeedge/kubeedge/releases/download"
	EdgeServiceFile      = "edgecore.service"
	CloudServiceFile     = "cloudcore.service"
	ServiceFileURLFormat = "https://raw.githubusercontent.com/kubeedge/kubeedge/release-%s/build/tools/%s"
	KubeEdgePath         = "/etc/kubeedge/"
	KubeEdgeBackupPath   = "/etc/kubeedge/backup/"
	KubeEdgeUpgradePath  = "/etc/kubeedge/upgrade/"
	KubeEdgeUsrBinPath   = "/usr/local/bin"
	KubeEdgeBinaryName   = "edgecore"
	KeadmBinaryName      = "keadm"

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

// RunningModuleV2 identifies cloudcore/edgecore running or not.
// only used for cloudcore container install and edgecore binary install
func RunningModuleV2(opt *types.ResetOptions) types.ModuleRunning {
	osType := GetOSInterface()
	cloudCoreRunning, err := IsCloudcoreContainerRunning(constants.SystemNamespace, opt.Kubeconfig)
	if err != nil {
		// just log the error, maybe we do not care
		klog.Warningf("failed to check cloudcore is running: %v", err)
	}
	if cloudCoreRunning {
		return types.KubeEdgeCloudRunning
	}

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
	// curl https://kubeedge.io/latestversion
	resp, err := http.Get(latestReleaseVersionURL)
	if err != nil {
		return "", fmt.Errorf("failed to get latest version from %s: %v", latestReleaseVersionURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get latest version from %s, expected %d, got status code: %d", latestReleaseVersionURL, http.StatusOK, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, constants.MaxRespBodyLength))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func GetCurrentVersion(version string) (string, error) {
	if kubeReleaseRegex.MatchString(version) {
		if strings.HasPrefix(version, "v") {
			return version, nil
		}
		return "v" + version, nil
	}

	// By default, the static version number set at build time is used.
	clientVersion, clientVersionErr := keadmVersion(pkgversion.Get().String())
	remoteVersion, err := GetLatestVersion()
	if err != nil {
		if clientVersionErr == nil {
			// Handle air-gapped environments by falling back to the client version.
			klog.Warningf("could not fetch a KubeEdge version from the internet: %v", err)
			klog.Warningf("falling back to the local client version: %s", clientVersion)
			return GetCurrentVersion(clientVersion)
		}
	}
	if clientVersionErr != nil {
		if err != nil {
			klog.Warningf("could not obtain neither client nor remote version; fall back to: %s", types.DefaultKubeEdgeVersion)
			return GetCurrentVersion(types.DefaultKubeEdgeVersion)
		}

		remoteVersion, err = keadmVersion(remoteVersion)
		if err != nil {
			return "", err
		}
		klog.Warningf("could not obtain client version; using remote version: %s", remoteVersion)
		return GetCurrentVersion(remoteVersion)
	}

	// both the client and the remote version are obtained; validate them and pick a stable version
	remoteVersion, err = validateStableVersion(remoteVersion, clientVersion)
	if err != nil {
		return "", err
	}
	return GetCurrentVersion(remoteVersion)
}

// keadmVersion returns the version of the client without metadata.
func keadmVersion(info string) (string, error) {
	v, err := versionutil.ParseSemantic(info)
	if err != nil {
		return "", fmt.Errorf("keadm version error: %v", err)
	}
	// There is no utility in versionutil to get the version without the metadata,
	// so this needs some manual formatting.
	// Discard offsets after a release label and keep the labels down to e.g. `alpha.0` instead of
	// including the offset e.g. `alpha.0.206`. This is done to comply with GCR image tags.
	pre := v.PreRelease()
	patch := v.Patch()
	if len(pre) > 0 {
		if patch > 0 {
			// If the patch version is more than zero, decrement it and remove the label.
			// this is done to comply with the latest stable patch release.
			patch = patch - 1
			pre = ""
		} else {
			split := strings.Split(pre, ".")
			if len(split) > 2 {
				pre = split[0] + "." + split[1] // Exclude the third element
			} else if len(split) < 2 {
				pre = split[0] + ".0" // Append .0 to a partial label
			}
			pre = "-" + pre
		}
	}
	vStr := fmt.Sprintf("v%d.%d.%d%s", v.Major(), v.Minor(), patch, pre)
	return vStr, nil
}

// Validate if the remote version is one Minor release newer than the client version.
// This is done to conform with "stable-X" and only allow remote versions from
// the same Patch level release.
func validateStableVersion(remoteVersion, clientVersion string) (string, error) {
	verRemote, err := versionutil.ParseGeneric(remoteVersion)
	if err != nil {
		return "", fmt.Errorf("remote version error: %v", err)
	}
	verClient, err := versionutil.ParseGeneric(clientVersion)
	if err != nil {
		return "", fmt.Errorf("client version error: %v", err)
	}
	// If the remote Major version is bigger or if the Major versions are the same,
	// but the remote Minor is bigger use the client version release. This handles Major bumps too.
	if verClient.Major() < verRemote.Major() ||
		(verClient.Major() == verRemote.Major()) && verClient.Minor() < verRemote.Minor() {
		klog.Infof("remote version is much newer: %s; falling back to: %s", remoteVersion, clientVersion)
		return clientVersion, nil
	}
	return remoteVersion, nil
}

// BuildConfig builds config from flags
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
		return fmt.Errorf("failed to build config, err: %v", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to init discovery client, err: %v", err)
	}

	discoveryClient.RESTClient().Post()
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to get the version of K8s master, please check whether K8s was successfully installed, err: %v", err)
	}

	return checkKubernetesVersion(serverVersion)
}

func checkKubernetesVersion(serverVersion *version.Info) error {
	reg := regexp.MustCompile(`[[:digit:]]*`)
	minorVersion := reg.FindString(serverVersion.Minor)

	k8sMinorVersion, err := strconv.Atoi(minorVersion)
	if err != nil {
		return fmt.Errorf("could not parse the minor version of K8s, error: %s", err)
	}
	if k8sMinorVersion >= types.DefaultK8SMinimumVersion {
		return nil
	}

	return fmt.Errorf("your minor version of K8s is lower than %d, please reinstall newer version", types.DefaultK8SMinimumVersion)
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

// computeSHA512Checksum returns the SHA512 checksum of the given file
func computeSHA512Checksum(filepath string) (string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return "", err
	}

	defer func() {
		err := f.Close()
		if err != nil {
			fmt.Printf("failed to close file, path: %v, error: %v \n", filepath, err)
		}
	}()

	h := sha512.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
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

// Compress compresses folders or files
func Compress(tarName string, paths []string) error {
	tarFile, err := os.Create(tarName)
	if err != nil {
		return err
	}
	defer func() {
		err := tarFile.Close()
		if err != nil {
			fmt.Printf("failed to close tar file, path: %v, error: %v \n", tarName, err)
		}
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

			defer func() {
				err := srcFile.Close()
				if err != nil {
					fmt.Printf("failed to close file, path: %v, error: %v \n", file, err)
				}
			}()

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
		return false, fmt.Errorf("invalid Input")
	}
}

// ExecShellFilter executes shell script and filter
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

func ParseEdgecoreConfig(edgecorePath string) (*v1alpha2.EdgeCoreConfig, error) {
	edgeCoreConfig := v1alpha2.NewDefaultEdgeCoreConfig()
	if err := edgeCoreConfig.Parse(edgecorePath); err != nil {
		return nil, err
	}
	return edgeCoreConfig, nil
}

// PrintFail prints fail
func PrintFail(cmd string, s string) {
	v := fmt.Sprintf("|%s %s failed|", s, cmd)
	printResult(v)
}

// PrintSucceed prints success
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
	if HasSystemd() {
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

		// if the specified the version is greater than the latest version
		// this means we haven't released the version, this may only occur in keadm e2e test
		// in this case, we will download the latest version service file
		if latestVersion, err := GetLatestVersion(); err == nil {
			if v, err := semver.Parse(strings.TrimPrefix(latestVersion, "v")); err == nil {
				if version.GT(v) {
					strippedVersion = fmt.Sprintf("%d.%d", v.Major, v.Minor)
				}
			}
		}
		fmt.Printf("keadm will download version %s service file\n", strippedVersion)

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
