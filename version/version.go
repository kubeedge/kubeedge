package version

import "fmt"

var (
	// Package is filled at linking time
	Package = "github.com/kubeedge/kubeedge"

	// Version holds the complete version number. Filled in at linking time.
	Version = "0.0.1+unknown"

	// Revision is filled with the VCS (e.g. git) revision being used to build
	// the program at linking time.
	Revision = ""

	// GoVersion is golang Version
	GoVersion = ""

	// Branch is current git branch
	Branch = ""
)

func Print()  {
	fmt.Println("Package:", Package)
	fmt.Println("Version:", Version)
	fmt.Println("Revision:", Revision)
	fmt.Println("Goversion:", GoVersion)
	fmt.Println("Branch:", Branch)
}