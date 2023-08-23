package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInstallNSSM(t *testing.T) {
	err := InstallNSSM()
	assert.NoError(t, err)
}
