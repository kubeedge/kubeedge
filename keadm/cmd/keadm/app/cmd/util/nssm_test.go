package util

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInstallNSSM(t *testing.T) {
	err := InstallNSSM()
	assert.NoError(t, err)
}

func TestIsServiceExist(t *testing.T) {
	t.Log(IsServiceExist("Power"))
}

func TestIsNSSMInstalled(t *testing.T) {
	t.Log(IsNSSMInstalled())
	fmt.Println("c")
}
