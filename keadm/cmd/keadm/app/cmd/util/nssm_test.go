//go:build windows

package util

import (
	"fmt"
	"testing"
)

func TestInstallNSSM(t *testing.T) {
	err := InstallNSSM()
	if err != nil {
		t.Error(err)
	}
}

func TestIsServiceExist(t *testing.T) {
	t.Log(IsServiceExist("Power"))
}

func TestIsNSSMInstalled(t *testing.T) {
	t.Log(IsNSSMInstalled())
	fmt.Println("c")
}
