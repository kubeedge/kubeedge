// +build linux

/*
Copyright 2014 The Kubernetes Authors.

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

package mount

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
	"k8s.io/klog"
	utilexec "k8s.io/utils/exec"
	utilio "k8s.io/utils/io"
	utilpath "k8s.io/utils/path"
)

const (
	// How many times to retry for a consistent read of /proc/mounts.
	maxListTries = 3
	// Number of fields per line in /proc/mounts as per the fstab man page.
	expectedNumFieldsPerLine = 6
	// At least number of fields per line in /proc/<pid>/mountinfo.
	expectedAtLeastNumFieldsPerMountInfo = 10
	// Location of the mount file to use
	procMountsPath = "/proc/mounts"
	// Location of the mountinfo file
	procMountInfoPath = "/proc/self/mountinfo"
	// 'fsck' found errors and corrected them
	fsckErrorsCorrected = 1
	// 'fsck' found errors but exited without correcting them
	fsckErrorsUncorrected = 4
)

// Mounter provides the default implementation of mount.Interface
// for the linux platform.  This implementation assumes that the
// kubelet is running in the host's root mount namespace.
type Mounter struct {
	mounterPath string
	withSystemd bool
}

// New returns a mount.Interface for the current system.
// It provides options to override the default mounter behavior.
// mounterPath allows using an alternative to `/bin/mount` for mounting.
func New(mounterPath string) Interface {
	return &Mounter{
		mounterPath: mounterPath,
		withSystemd: detectSystemd(),
	}
}

// Mount mounts source to target as fstype with given options. 'source' and 'fstype' must
// be an empty string in case it's not required, e.g. for remount, or for auto filesystem
// type, where kernel handles fstype for you. The mount 'options' is a list of options,
// currently come from mount(8), e.g. "ro", "remount", "bind", etc. If no more option is
// required, call Mount with an empty string list or nil.
func (mounter *Mounter) Mount(source string, target string, fstype string, options []string) error {
	// Path to mounter binary if containerized mounter is needed. Otherwise, it is set to empty.
	// All Linux distros are expected to be shipped with a mount utility that a support bind mounts.
	mounterPath := ""
	bind, bindOpts, bindRemountOpts := IsBind(options)
	if bind {
		err := mounter.doMount(mounterPath, defaultMountCommand, source, target, fstype, bindOpts)
		if err != nil {
			return err
		}
		return mounter.doMount(mounterPath, defaultMountCommand, source, target, fstype, bindRemountOpts)
	}
	// The list of filesystems that require containerized mounter on GCI image cluster
	fsTypesNeedMounter := map[string]struct{}{
		"nfs":       {},
		"glusterfs": {},
		"ceph":      {},
		"cifs":      {},
	}
	if _, ok := fsTypesNeedMounter[fstype]; ok {
		mounterPath = mounter.mounterPath
	}
	return mounter.doMount(mounterPath, defaultMountCommand, source, target, fstype, options)
}

// doMount runs the mount command. mounterPath is the path to mounter binary if containerized mounter is used.
func (m *Mounter) doMount(mounterPath string, mountCmd string, source string, target string, fstype string, options []string) error {
	mountArgs := MakeMountArgs(source, target, fstype, options)
	if len(mounterPath) > 0 {
		mountArgs = append([]string{mountCmd}, mountArgs...)
		mountCmd = mounterPath
	}

	if m.withSystemd {
		// Try to run mount via systemd-run --scope. This will escape the
		// service where kubelet runs and any fuse daemons will be started in a
		// specific scope. kubelet service than can be restarted without killing
		// these fuse daemons.
		//
		// Complete command line (when mounterPath is not used):
		// systemd-run --description=... --scope -- mount -t <type> <what> <where>
		//
		// Expected flow:
		// * systemd-run creates a transient scope (=~ cgroup) and executes its
		//   argument (/bin/mount) there.
		// * mount does its job, forks a fuse daemon if necessary and finishes.
		//   (systemd-run --scope finishes at this point, returning mount's exit
		//   code and stdout/stderr - thats one of --scope benefits).
		// * systemd keeps the fuse daemon running in the scope (i.e. in its own
		//   cgroup) until the fuse daemon dies (another --scope benefit).
		//   Kubelet service can be restarted and the fuse daemon survives.
		// * When the fuse daemon dies (e.g. during unmount) systemd removes the
		//   scope automatically.
		//
		// systemd-mount is not used because it's too new for older distros
		// (CentOS 7, Debian Jessie).
		mountCmd, mountArgs = AddSystemdScope("systemd-run", target, mountCmd, mountArgs)
	} else {
		// No systemd-run on the host (or we failed to check it), assume kubelet
		// does not run as a systemd service.
		// No code here, mountCmd and mountArgs are already populated.
	}

	klog.V(4).Infof("Mounting cmd (%s) with arguments (%s)", mountCmd, mountArgs)
	command := exec.Command(mountCmd, mountArgs...)
	output, err := command.CombinedOutput()
	if err != nil {
		args := strings.Join(mountArgs, " ")
		klog.Errorf("Mount failed: %v\nMounting command: %s\nMounting arguments: %s\nOutput: %s\n", err, mountCmd, args, string(output))
		return fmt.Errorf("mount failed: %v\nMounting command: %s\nMounting arguments: %s\nOutput: %s\n",
			err, mountCmd, args, string(output))
	}
	return err
}

// detectSystemd returns true if OS runs with systemd as init. When not sure
// (permission errors, ...), it returns false.
// There may be different ways how to detect systemd, this one makes sure that
// systemd-runs (needed by Mount()) works.
func detectSystemd() bool {
	if _, err := exec.LookPath("systemd-run"); err != nil {
		klog.V(2).Infof("Detected OS without systemd")
		return false
	}
	// Try to run systemd-run --scope /bin/true, that should be enough
	// to make sure that systemd is really running and not just installed,
	// which happens when running in a container with a systemd-based image
	// but with different pid 1.
	cmd := exec.Command("systemd-run", "--description=Kubernetes systemd probe", "--scope", "true")
	output, err := cmd.CombinedOutput()
	if err != nil {
		klog.V(2).Infof("Cannot run systemd-run, assuming non-systemd OS")
		klog.V(4).Infof("systemd-run failed with: %v", err)
		klog.V(4).Infof("systemd-run output: %s", string(output))
		return false
	}
	klog.V(2).Infof("Detected OS with systemd")
	return true
}

// MakeMountArgs makes the arguments to the mount(8) command.
// Implementation is shared with NsEnterMounter
func MakeMountArgs(source, target, fstype string, options []string) []string {
	// Build mount command as follows:
	//   mount [-t $fstype] [-o $options] [$source] $target
	mountArgs := []string{}
	if len(fstype) > 0 {
		mountArgs = append(mountArgs, "-t", fstype)
	}
	if len(options) > 0 {
		mountArgs = append(mountArgs, "-o", strings.Join(options, ","))
	}
	if len(source) > 0 {
		mountArgs = append(mountArgs, source)
	}
	mountArgs = append(mountArgs, target)

	return mountArgs
}

// AddSystemdScope adds "system-run --scope" to given command line
// implementation is shared with NsEnterMounter
func AddSystemdScope(systemdRunPath, mountName, command string, args []string) (string, []string) {
	descriptionArg := fmt.Sprintf("--description=Kubernetes transient mount for %s", mountName)
	systemdRunArgs := []string{descriptionArg, "--scope", "--", command}
	return systemdRunPath, append(systemdRunArgs, args...)
}

// Unmount unmounts the target.
func (mounter *Mounter) Unmount(target string) error {
	klog.V(4).Infof("Unmounting %s", target)
	command := exec.Command("umount", target)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Unmount failed: %v\nUnmounting arguments: %s\nOutput: %s\n", err, target, string(output))
	}
	return nil
}

// List returns a list of all mounted filesystems.
func (*Mounter) List() ([]MountPoint, error) {
	return ListProcMounts(procMountsPath)
}

func (mounter *Mounter) IsMountPointMatch(mp MountPoint, dir string) bool {
	deletedDir := fmt.Sprintf("%s\\040(deleted)", dir)
	return ((mp.Path == dir) || (mp.Path == deletedDir))
}

// IsLikelyNotMountPoint determines if a directory is not a mountpoint.
// It is fast but not necessarily ALWAYS correct. If the path is in fact
// a bind mount from one part of a mount to another it will not be detected.
// It also can not distinguish between mountpoints and symbolic links.
// mkdir /tmp/a /tmp/b; mount --bind /tmp/a /tmp/b; IsLikelyNotMountPoint("/tmp/b")
// will return true. When in fact /tmp/b is a mount point. If this situation
// if of interest to you, don't use this function...
func (mounter *Mounter) IsLikelyNotMountPoint(file string) (bool, error) {
	stat, err := os.Stat(file)
	if err != nil {
		return true, err
	}
	rootStat, err := os.Stat(filepath.Dir(strings.TrimSuffix(file, "/")))
	if err != nil {
		return true, err
	}
	// If the directory has a different device as parent, then it is a mountpoint.
	if stat.Sys().(*syscall.Stat_t).Dev != rootStat.Sys().(*syscall.Stat_t).Dev {
		return false, nil
	}

	return true, nil
}

// DeviceOpened checks if block device in use by calling Open with O_EXCL flag.
// If pathname is not a device, log and return false with nil error.
// If open returns errno EBUSY, return true with nil error.
// If open returns nil, return false with nil error.
// Otherwise, return false with error
func (mounter *Mounter) DeviceOpened(pathname string) (bool, error) {
	return ExclusiveOpenFailsOnDevice(pathname)
}

// PathIsDevice uses FileInfo returned from os.Stat to check if path refers
// to a device.
func (mounter *Mounter) PathIsDevice(pathname string) (bool, error) {
	pathType, err := mounter.GetFileType(pathname)
	isDevice := pathType == FileTypeCharDev || pathType == FileTypeBlockDev
	return isDevice, err
}

// ExclusiveOpenFailsOnDevice is shared with NsEnterMounter
func ExclusiveOpenFailsOnDevice(pathname string) (bool, error) {
	var isDevice bool
	finfo, err := os.Stat(pathname)
	if os.IsNotExist(err) {
		isDevice = false
	}
	// err in call to os.Stat
	if err != nil {
		return false, fmt.Errorf(
			"PathIsDevice failed for path %q: %v",
			pathname,
			err)
	}
	// path refers to a device
	if finfo.Mode()&os.ModeDevice != 0 {
		isDevice = true
	}

	if !isDevice {
		klog.Errorf("Path %q is not referring to a device.", pathname)
		return false, nil
	}
	fd, errno := unix.Open(pathname, unix.O_RDONLY|unix.O_EXCL, 0)
	// If the device is in use, open will return an invalid fd.
	// When this happens, it is expected that Close will fail and throw an error.
	defer unix.Close(fd)
	if errno == nil {
		// device not in use
		return false, nil
	} else if errno == unix.EBUSY {
		// device is in use
		return true, nil
	}
	// error during call to Open
	return false, errno
}

//GetDeviceNameFromMount: given a mount point, find the device name from its global mount point
func (mounter *Mounter) GetDeviceNameFromMount(mountPath, pluginMountDir string) (string, error) {
	return GetDeviceNameFromMountLinux(mounter, mountPath, pluginMountDir)
}

func getDeviceNameFromMount(mounter Interface, mountPath, pluginMountDir string) (string, error) {
	return GetDeviceNameFromMountLinux(mounter, mountPath, pluginMountDir)
}

// GetDeviceNameFromMountLinux find the device name from /proc/mounts in which
// the mount path reference should match the given plugin mount directory. In case no mount path reference
// matches, returns the volume name taken from its given mountPath
// This implementation is shared with NsEnterMounter
func GetDeviceNameFromMountLinux(mounter Interface, mountPath, pluginMountDir string) (string, error) {
	refs, err := mounter.GetMountRefs(mountPath)
	if err != nil {
		klog.V(4).Infof("GetMountRefs failed for mount path %q: %v", mountPath, err)
		return "", err
	}
	if len(refs) == 0 {
		klog.V(4).Infof("Directory %s is not mounted", mountPath)
		return "", fmt.Errorf("directory %s is not mounted", mountPath)
	}
	for _, ref := range refs {
		if strings.HasPrefix(ref, pluginMountDir) {
			volumeID, err := filepath.Rel(pluginMountDir, ref)
			if err != nil {
				klog.Errorf("Failed to get volume id from mount %s - %v", mountPath, err)
				return "", err
			}
			return volumeID, nil
		}
	}

	return path.Base(mountPath), nil
}

// ListProcMounts is shared with NsEnterMounter
func ListProcMounts(mountFilePath string) ([]MountPoint, error) {
	content, err := utilio.ConsistentRead(mountFilePath, maxListTries)
	if err != nil {
		return nil, err
	}
	return parseProcMounts(content)
}

func parseProcMounts(content []byte) ([]MountPoint, error) {
	out := []MountPoint{}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if line == "" {
			// the last split() item is empty string following the last \n
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != expectedNumFieldsPerLine {
			return nil, fmt.Errorf("wrong number of fields (expected %d, got %d): %s", expectedNumFieldsPerLine, len(fields), line)
		}

		mp := MountPoint{
			Device: fields[0],
			Path:   fields[1],
			Type:   fields[2],
			Opts:   strings.Split(fields[3], ","),
		}

		freq, err := strconv.Atoi(fields[4])
		if err != nil {
			return nil, err
		}
		mp.Freq = freq

		pass, err := strconv.Atoi(fields[5])
		if err != nil {
			return nil, err
		}
		mp.Pass = pass

		out = append(out, mp)
	}
	return out, nil
}

func (mounter *Mounter) MakeRShared(path string) error {
	return DoMakeRShared(path, procMountInfoPath)
}

func (mounter *Mounter) GetFileType(pathname string) (FileType, error) {
	return getFileType(pathname)
}

func (mounter *Mounter) MakeDir(pathname string) error {
	err := os.MkdirAll(pathname, os.FileMode(0755))
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func (mounter *Mounter) MakeFile(pathname string) error {
	f, err := os.OpenFile(pathname, os.O_CREATE, os.FileMode(0644))
	defer f.Close()
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func (mounter *Mounter) ExistsPath(pathname string) (bool, error) {
	return utilpath.Exists(utilpath.CheckFollowSymlink, pathname)
}

func (mounter *Mounter) EvalHostSymlinks(pathname string) (string, error) {
	return filepath.EvalSymlinks(pathname)
}

// formatAndMount uses unix utils to format and mount the given disk
func (mounter *SafeFormatAndMount) formatAndMount(source string, target string, fstype string, options []string) error {
	readOnly := false
	for _, option := range options {
		if option == "ro" {
			readOnly = true
			break
		}
	}

	options = append(options, "defaults")

	if !readOnly {
		// Run fsck on the disk to fix repairable issues, only do this for volumes requested as rw.
		klog.V(4).Infof("Checking for issues with fsck on disk: %s", source)
		args := []string{"-a", source}
		out, err := mounter.Exec.Run("fsck", args...)
		if err != nil {
			ee, isExitError := err.(utilexec.ExitError)
			switch {
			case err == utilexec.ErrExecutableNotFound:
				klog.Warningf("'fsck' not found on system; continuing mount without running 'fsck'.")
			case isExitError && ee.ExitStatus() == fsckErrorsCorrected:
				klog.Infof("Device %s has errors which were corrected by fsck.", source)
			case isExitError && ee.ExitStatus() == fsckErrorsUncorrected:
				return fmt.Errorf("'fsck' found errors on device %s but could not correct them: %s.", source, string(out))
			case isExitError && ee.ExitStatus() > fsckErrorsUncorrected:
				klog.Infof("`fsck` error %s", string(out))
			}
		}
	}

	// Try to mount the disk
	klog.V(4).Infof("Attempting to mount disk: %s %s %s", fstype, source, target)
	mountErr := mounter.Interface.Mount(source, target, fstype, options)
	if mountErr != nil {
		// Mount failed. This indicates either that the disk is unformatted or
		// it contains an unexpected filesystem.
		existingFormat, err := mounter.GetDiskFormat(source)
		if err != nil {
			return err
		}
		if existingFormat == "" {
			if readOnly {
				// Don't attempt to format if mounting as readonly, return an error to reflect this.
				return errors.New("failed to mount unformatted volume as read only")
			}

			// Disk is unformatted so format it.
			args := []string{source}
			// Use 'ext4' as the default
			if len(fstype) == 0 {
				fstype = "ext4"
			}

			if fstype == "ext4" || fstype == "ext3" {
				args = []string{
					"-F",  // Force flag
					"-m0", // Zero blocks reserved for super-user
					source,
				}
			}
			klog.Infof("Disk %q appears to be unformatted, attempting to format as type: %q with options: %v", source, fstype, args)
			_, err := mounter.Exec.Run("mkfs."+fstype, args...)
			if err == nil {
				// the disk has been formatted successfully try to mount it again.
				klog.Infof("Disk successfully formatted (mkfs): %s - %s %s", fstype, source, target)
				return mounter.Interface.Mount(source, target, fstype, options)
			}
			klog.Errorf("format of disk %q failed: type:(%q) target:(%q) options:(%q)error:(%v)", source, fstype, target, options, err)
			return err
		} else {
			// Disk is already formatted and failed to mount
			if len(fstype) == 0 || fstype == existingFormat {
				// This is mount error
				return mountErr
			} else {
				// Block device is formatted with unexpected filesystem, let the user know
				return fmt.Errorf("failed to mount the volume as %q, it already contains %s. Mount error: %v", fstype, existingFormat, mountErr)
			}
		}
	}
	return mountErr
}

// GetDiskFormat uses 'blkid' to see if the given disk is unformatted
func (mounter *SafeFormatAndMount) GetDiskFormat(disk string) (string, error) {
	args := []string{"-p", "-s", "TYPE", "-s", "PTTYPE", "-o", "export", disk}
	klog.V(4).Infof("Attempting to determine if disk %q is formatted using blkid with args: (%v)", disk, args)
	dataOut, err := mounter.Exec.Run("blkid", args...)
	output := string(dataOut)
	klog.V(4).Infof("Output: %q, err: %v", output, err)

	if err != nil {
		if exit, ok := err.(utilexec.ExitError); ok {
			if exit.ExitStatus() == 2 {
				// Disk device is unformatted.
				// For `blkid`, if the specified token (TYPE/PTTYPE, etc) was
				// not found, or no (specified) devices could be identified, an
				// exit code of 2 is returned.
				return "", nil
			}
		}
		klog.Errorf("Could not determine if disk %q is formatted (%v)", disk, err)
		return "", err
	}

	var fstype, pttype string

	lines := strings.Split(output, "\n")
	for _, l := range lines {
		if len(l) <= 0 {
			// Ignore empty line.
			continue
		}
		cs := strings.Split(l, "=")
		if len(cs) != 2 {
			return "", fmt.Errorf("blkid returns invalid output: %s", output)
		}
		// TYPE is filesystem type, and PTTYPE is partition table type, according
		// to https://www.kernel.org/pub/linux/utils/util-linux/v2.21/libblkid-docs/.
		if cs[0] == "TYPE" {
			fstype = cs[1]
		} else if cs[0] == "PTTYPE" {
			pttype = cs[1]
		}
	}

	if len(pttype) > 0 {
		klog.V(4).Infof("Disk %s detected partition table type: %s", disk, pttype)
		// Returns a special non-empty string as filesystem type, then kubelet
		// will not format it.
		return "unknown data, probably partitions", nil
	}

	return fstype, nil
}

// isShared returns true, if given path is on a mount point that has shared
// mount propagation.
func isShared(mount string, mountInfoPath string) (bool, error) {
	info, err := findMountInfo(mount, mountInfoPath)
	if err != nil {
		return false, err
	}

	// parse optional parameters
	for _, opt := range info.optionalFields {
		if strings.HasPrefix(opt, "shared:") {
			return true, nil
		}
	}
	return false, nil
}

// This represents a single line in /proc/<pid>/mountinfo.
type mountInfo struct {
	// Unique ID for the mount (maybe reused after umount).
	id int
	// The ID of the parent mount (or of self for the root of this mount namespace's mount tree).
	parentID int
	// The value of `st_dev` for files on this filesystem.
	majorMinor string
	// The pathname of the directory in the filesystem which forms the root of this mount.
	root string
	// Mount source, filesystem-specific information. e.g. device, tmpfs name.
	source string
	// Mount point, the pathname of the mount point.
	mountPoint string
	// Optional fieds, zero or more fields of the form "tag[:value]".
	optionalFields []string
	// The filesystem type in the form "type[.subtype]".
	fsType string
	// Per-mount options.
	mountOptions []string
	// Per-superblock options.
	superOptions []string
}

// parseMountInfo parses /proc/xxx/mountinfo.
func parseMountInfo(filename string) ([]mountInfo, error) {
	content, err := utilio.ConsistentRead(filename, maxListTries)
	if err != nil {
		return []mountInfo{}, err
	}
	contentStr := string(content)
	infos := []mountInfo{}

	for _, line := range strings.Split(contentStr, "\n") {
		if line == "" {
			// the last split() item is empty string following the last \n
			continue
		}
		// See `man proc` for authoritative description of format of the file.
		fields := strings.Fields(line)
		if len(fields) < expectedAtLeastNumFieldsPerMountInfo {
			return nil, fmt.Errorf("wrong number of fields in (expected at least %d, got %d): %s", expectedAtLeastNumFieldsPerMountInfo, len(fields), line)
		}
		id, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, err
		}
		parentID, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, err
		}
		info := mountInfo{
			id:           id,
			parentID:     parentID,
			majorMinor:   fields[2],
			root:         fields[3],
			mountPoint:   fields[4],
			mountOptions: strings.Split(fields[5], ","),
		}
		// All fields until "-" are "optional fields".
		i := 6
		for ; i < len(fields) && fields[i] != "-"; i++ {
			info.optionalFields = append(info.optionalFields, fields[i])
		}
		// Parse the rest 3 fields.
		i += 1
		if len(fields)-i < 3 {
			return nil, fmt.Errorf("expect 3 fields in %s, got %d", line, len(fields)-i)
		}
		info.fsType = fields[i]
		info.source = fields[i+1]
		info.superOptions = strings.Split(fields[i+2], ",")
		infos = append(infos, info)
	}
	return infos, nil
}

func findMountInfo(path, mountInfoPath string) (mountInfo, error) {
	infos, err := parseMountInfo(mountInfoPath)
	if err != nil {
		return mountInfo{}, err
	}

	// process /proc/xxx/mountinfo in backward order and find the first mount
	// point that is prefix of 'path' - that's the mount where path resides
	var info *mountInfo
	for i := len(infos) - 1; i >= 0; i-- {
		if PathWithinBase(path, infos[i].mountPoint) {
			info = &infos[i]
			break
		}
	}
	if info == nil {
		return mountInfo{}, fmt.Errorf("cannot find mount point for %q", path)
	}
	return *info, nil
}

// DoMakeRShared is common implementation of MakeRShared on Linux. It checks if
// path is shared and bind-mounts it as rshared if needed. mountCmd and
// mountArgs are expected to contain mount-like command, DoMakeRShared will add
// '--bind <path> <path>' and '--make-rshared <path>' to mountArgs.
func DoMakeRShared(path string, mountInfoFilename string) error {
	shared, err := isShared(path, mountInfoFilename)
	if err != nil {
		return err
	}
	if shared {
		klog.V(4).Infof("Directory %s is already on a shared mount", path)
		return nil
	}

	klog.V(2).Infof("Bind-mounting %q with shared mount propagation", path)
	// mount --bind /var/lib/kubelet /var/lib/kubelet
	if err := syscall.Mount(path, path, "" /*fstype*/, syscall.MS_BIND, "" /*data*/); err != nil {
		return fmt.Errorf("failed to bind-mount %s: %v", path, err)
	}

	// mount --make-rshared /var/lib/kubelet
	if err := syscall.Mount(path, path, "" /*fstype*/, syscall.MS_SHARED|syscall.MS_REC, "" /*data*/); err != nil {
		return fmt.Errorf("failed to make %s rshared: %v", path, err)
	}

	return nil
}

// GetSELinux is common implementation of GetSELinuxSupport on Linux.
func GetSELinux(path string, mountInfoFilename string) (bool, error) {
	info, err := findMountInfo(path, mountInfoFilename)
	if err != nil {
		return false, err
	}

	// "seclabel" can be both in mount options and super options.
	for _, opt := range info.superOptions {
		if opt == "seclabel" {
			return true, nil
		}
	}
	for _, opt := range info.mountOptions {
		if opt == "seclabel" {
			return true, nil
		}
	}
	return false, nil
}

func (mounter *Mounter) GetMountRefs(pathname string) ([]string, error) {
	pathExists, pathErr := PathExists(pathname)
	if !pathExists {
		return []string{}, nil
	} else if IsCorruptedMnt(pathErr) {
		klog.Warningf("GetMountRefs found corrupted mount at %s, treating as unmounted path", pathname)
		return []string{}, nil
	} else if pathErr != nil {
		return nil, fmt.Errorf("error checking path %s: %v", pathname, pathErr)
	}
	realpath, err := filepath.EvalSymlinks(pathname)
	if err != nil {
		return nil, err
	}
	return SearchMountPoints(realpath, procMountInfoPath)
}

func (mounter *Mounter) GetSELinuxSupport(pathname string) (bool, error) {
	return GetSELinux(pathname, procMountInfoPath)
}

func (mounter *Mounter) GetFSGroup(pathname string) (int64, error) {
	realpath, err := filepath.EvalSymlinks(pathname)
	if err != nil {
		return 0, err
	}
	return GetFSGroupLinux(realpath)
}

func (mounter *Mounter) GetMode(pathname string) (os.FileMode, error) {
	return GetModeLinux(pathname)
}

// GetFSGroupLinux is shared between Linux and NsEnterMounter
// pathname must already be evaluated for symlinks
func GetFSGroupLinux(pathname string) (int64, error) {
	info, err := os.Stat(pathname)
	if err != nil {
		return 0, err
	}
	return int64(info.Sys().(*syscall.Stat_t).Gid), nil
}

// GetModeLinux is shared between Linux and NsEnterMounter
func GetModeLinux(pathname string) (os.FileMode, error) {
	info, err := os.Stat(pathname)
	if err != nil {
		return 0, err
	}
	return info.Mode(), nil
}

// SearchMountPoints finds all mount references to the source, returns a list of
// mountpoints.
// This function assumes source cannot be device.
// Some filesystems may share a source name, e.g. tmpfs. And for bind mounting,
// it's possible to mount a non-root path of a filesystem, so we need to use
// root path and major:minor to represent mount source uniquely.
// This implementation is shared between Linux and NsEnterMounter
func SearchMountPoints(hostSource, mountInfoPath string) ([]string, error) {
	mis, err := parseMountInfo(mountInfoPath)
	if err != nil {
		return nil, err
	}

	mountID := 0
	rootPath := ""
	majorMinor := ""

	// Finding the underlying root path and major:minor if possible.
	// We need search in backward order because it's possible for later mounts
	// to overlap earlier mounts.
	for i := len(mis) - 1; i >= 0; i-- {
		if hostSource == mis[i].mountPoint || PathWithinBase(hostSource, mis[i].mountPoint) {
			// If it's a mount point or path under a mount point.
			mountID = mis[i].id
			rootPath = filepath.Join(mis[i].root, strings.TrimPrefix(hostSource, mis[i].mountPoint))
			majorMinor = mis[i].majorMinor
			break
		}
	}

	if rootPath == "" || majorMinor == "" {
		return nil, fmt.Errorf("failed to get root path and major:minor for %s", hostSource)
	}

	var refs []string
	for i := range mis {
		if mis[i].id == mountID {
			// Ignore mount entry for mount source itself.
			continue
		}
		if mis[i].root == rootPath && mis[i].majorMinor == majorMinor {
			refs = append(refs, mis[i].mountPoint)
		}
	}

	return refs, nil
}
