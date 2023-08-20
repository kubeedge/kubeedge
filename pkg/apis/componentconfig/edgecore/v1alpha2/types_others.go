//go:build !windows

package v1alpha2

const (
	CGroupDriverCGroupFS = "cgroupfs"
	CGroupDriverSystemd  = "systemd"

	// DataBaseDataSource is edge.db
	DataBaseDataSource = "/var/lib/kubeedge/edgecore.db"
)
