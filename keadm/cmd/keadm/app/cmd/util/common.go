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
	"bytes"
	"compress/gzip"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/spf13/pflag"
	versionutil "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/common/constants"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
	pkgversion "github.com/kubeedge/kubeedge/pkg/version"
)

var (
	kubeReleaseRegex = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)([-\w.+]*)?$`)
)

// Constants used by installers
const (
	KubeEdgeDownloadURL  = "https://github.com/kubeedge/kubeedge/releases/download"
	EdgeServiceFile      = "edgecore.service"
	CloudServiceFile     = "cloudcore.service"
	ServiceFileURLFormat = "https://raw.githubusercontent.com/kubeedge/kubeedge/release-%s/build/tools/%s"
	KubeEdgeBinaryName   = "edgecore"
	KeadmBinaryName      = "keadm"

	KubeCloudBinaryName = "cloudcore"

	KubeEdgeConfigDir        = KubeEdgePath + "config/"
	KubeEdgeCloudCoreNewYaml = KubeEdgeConfigDir + "cloudcore.yaml"
	KubeEdgeEdgeCoreNewYaml  = KubeEdgeConfigDir + "edgecore.yaml"

	KubeEdgeCrdPath = KubeEdgePath + "crds"

	KubeEdgeCRDDownloadURL = "https://raw.githubusercontent.com/kubeedge/kubeedge/release-%s/build/crds"

	latestReleaseVersionURL = "https://kubeedge.io/latestversion"
	RetryTimes              = 5

	APT    string = "apt"
	YUM    string = "yum"
	PACMAN string = "pacman"

	EdgeCoreSELinuxLabel = "system_u:object_r:bin_t:s0"
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

func DecompressTarGz(gzFilePath, dest string) error {
	reader, err := os.Open(gzFilePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	err = os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		return err
	}

	archive, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer archive.Close()

	tr := tar.NewReader(archive)

	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			writer, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(writer, tr); err != nil {
				writer.Close() // Close the file explicitly here in case of an error
				return err
			}
			writer.Close() // Close the file explicitly after successful write
		}
	}
}

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
			// if path is a dir, don't continue
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

// GetHelmVersion returns the verified version of Helm. If the verification fails,
// obtain the remote version first and then use the default value
func GetHelmVersion(version string, retryTimes int) string {
	if kubeReleaseRegex.MatchString(version) {
		return strings.TrimPrefix(version, "v")
	}

	for i := 0; i < retryTimes; i++ {
		v, err := GetLatestVersion()
		if err != nil {
			fmt.Println("Failed to get the latest KubeEdge release version, error: ", err)
			continue
		}
		if v != "" {
			// do not obtain remote version again
			return GetHelmVersion(v, 0)
		}
	}
	// returns default version
	fmt.Println("Failed to get the latest KubeEdge release version, will use default version: ", types.DefaultKubeEdgeVersion)
	return types.DefaultKubeEdgeVersion
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

func ReportTaskResult(config *v1alpha2.EdgeCoreConfig, taskType, taskID string, event fsm.Event) error {
	resp := &commontypes.NodeTaskResponse{
		NodeName: config.Modules.Edged.HostnameOverride,
		Event:    event.Type,
		Action:   event.Action,
		Time:     time.Now().UTC().Format(time.RFC3339),
		Reason:   event.Msg,
	}
	edgeHub := config.Modules.EdgeHub
	var caCrt []byte
	caCertPath := edgeHub.TLSCAFile
	caCrt, err := os.ReadFile(caCertPath)
	if err != nil {
		return fmt.Errorf("failed to read ca: %v", err)
	}

	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(caCrt)

	certFile := edgeHub.TLSCertFile
	keyFile := edgeHub.TLSPrivateKeyFile
	cliCrt, err := tls.LoadX509KeyPair(certFile, keyFile)

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// use TLS configuration
		TLSClientConfig: &tls.Config{
			RootCAs:            rootCAs,
			InsecureSkipVerify: false,
			Certificates:       []tls.Certificate{cliCrt},
		},
	}

	client := &http.Client{Transport: transport, Timeout: 30 * time.Second}

	respData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal failed: %v", err)
	}
	url := edgeHub.HTTPServer + fmt.Sprintf("/task/%s/name/%s/node/%s/status", taskType, taskID, config.Modules.Edged.HostnameOverride)
	result, err := client.Post(url, "application/json", bytes.NewReader(respData))

	if err != nil {
		return fmt.Errorf("post http request failed: %v", err)
	}
	klog.Error("report result ", result)
	defer result.Body.Close()

	return nil
}
