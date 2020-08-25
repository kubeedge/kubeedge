// +build go1.10,debug

// Add -tags debug into go build arguments before debugging,
// if your go version is go1.10 onward.
// example: go build -tags debug -o server -gcflags "all=-N -l" server.go
// Chassis customized debug tag to resolve dlv debug issue:
// https://github.com/golang/go/issues/23733
// https://github.com/derekparker/delve/issues/865

package goplugin

import (
	"errors"
)

var errGoPluginDisabled = errors.New("plugin is disabled by build tag: debug")

func lookUp(plugName, symName string) (interface{}, error) {
	return nil, errGoPluginDisabled
}
