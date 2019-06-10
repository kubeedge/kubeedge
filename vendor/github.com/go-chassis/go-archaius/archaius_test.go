package archaius_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-archaius/core"
	"github.com/go-mesh/openlogging"
	"github.com/stretchr/testify/assert"
)

type EListener struct{}

func (e EListener) Event(event *core.Event) {
	openlogging.GetLogger().Infof("config value after change ", event.Key, " | ", event.Value)
}

var filename2 string

func TestNew(t *testing.T) {
	f1Bytes := []byte(`
age: 14
name: peter
`)
	f2Bytes := []byte(`
addr: 14
number: 1
`)
	d, _ := os.Getwd()
	filename1 := filepath.Join(d, "f1.yaml")
	filename2 = filepath.Join(d, "f2.yaml")
	os.Remove(filename1)
	os.Remove(filename2)
	f1, err := os.Create(filename1)
	assert.NoError(t, err)
	defer f1.Close()
	f2, err := os.Create(filename2)
	assert.NoError(t, err)
	defer f2.Close()
	_, err = io.WriteString(f1, string(f1Bytes))
	t.Log(string(f1Bytes))
	assert.NoError(t, err)
	_, err = io.WriteString(f2, string(f2Bytes))
	assert.NoError(t, err)

	err = archaius.Init(
		archaius.WithRequiredFiles([]string{filename1}),
		archaius.WithOptionalFiles([]string{filename2}))
	assert.NoError(t, err)

}
func TestAddFile(t *testing.T) {
	s := archaius.Get("number")
	s2 := archaius.Get("age")
	assert.Equal(t, 14, s2)
	assert.Equal(t, 1, s)

}
func TestConfig_Get(t *testing.T) {
	s := archaius.Get("age")
	n := archaius.Get("name")
	assert.Equal(t, 14, s)
	assert.Equal(t, "peter", n)
}
func TestConfig_GetInt(t *testing.T) {
	s := archaius.Get("age")
	assert.Equal(t, 14, s)
}
func TestConfig_RegisterListener(t *testing.T) {
	eventHandler := EListener{}
	err := archaius.RegisterListener(eventHandler, "a*")
	assert.NoError(t, err)
	defer archaius.UnRegisterListener(eventHandler, "a*")

}
func TestInitConfigCenter(t *testing.T) {
	err := archaius.EnableConfigCenterSource(archaius.ConfigCenterInfo{}, nil)
	assert.Error(t, err)
	err = archaius.EnableConfigCenterSource(archaius.ConfigCenterInfo{
		ClientType: "fake",
	}, nil)
	assert.Error(t, err)
}
func TestClean(t *testing.T) {
	err := archaius.Clean()
	assert.NoError(t, err)
	s := archaius.Get("age")
	assert.Equal(t, nil, s)

	err = archaius.Init(
		archaius.WithOptionalFiles([]string{filename2}))
	s = archaius.Get("addr")
	assert.Equal(t, 14, s)
}
