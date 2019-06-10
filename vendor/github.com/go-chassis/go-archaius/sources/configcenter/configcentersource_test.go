package configcenter_test

import (
	_ "github.com/go-chassis/go-chassis-config/configcenter"

	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-archaius/core"
	"github.com/go-chassis/go-archaius/sources/configcenter"
	"github.com/stretchr/testify/assert"

	"encoding/json"
	"errors"
	"github.com/go-chassis/go-chassis-config"
	"math/rand"
	"testing"
	"time"
)

type Testingsource struct {
	configuration  map[string]interface{}
	changeCallback core.DynamicConfigCallback
}

type TestDynamicConfigHandler struct {
	EventName  string
	EventKey   string
	EventValue interface{}
}

func (ccenter *TestDynamicConfigHandler) OnEvent(event *core.Event) {

	ccenter.EventName = event.EventType
	ccenter.EventKey = event.Key
	ccenter.EventValue = event.Value
}

func (*Testingsource) GetDimensionInfo() string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz"
	result := make([]byte, 5)

	for i := 0; i < 5; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}

	dimensioninfo := string(result)
	return dimensioninfo
}

func (*Testingsource) GetConfigServer() []string {
	configserver := []string{`http://10.18.206.218:30103`}

	return configserver
}

func (*Testingsource) GetInvalidConfigServer() []string {
	return nil
}

func TestGetConfigurationsForInvalidCCIP(t *testing.T) {
	testSource := &Testingsource{}

	t.Log("Test configcenter.go")
	opts := config.Options{
		DimensionInfo: testSource.GetDimensionInfo(),
		TenantName:    "default",
	}
	cc, err := config.NewClient("config_center", opts)
	assert.NoError(t, err)
	ccs := configcenter.NewConfigCenterSource(cc, testSource.GetDimensionInfo(), 1,
		1)

	_, er := ccs.GetConfigurations()
	if er != nil {
		assert.Error(t, er)
	}

	time.Sleep(2 * time.Second)
	configcenter.ConfigCenterConfig.Cleanup()
}

func TestGetConfigurationsWithCCIP(t *testing.T) {
	testSource := &Testingsource{}
	opts := config.Options{
		DimensionInfo: testSource.GetDimensionInfo(),
		TenantName:    "default",
	}
	cc, err := config.NewClient("config_center", opts)
	assert.NoError(t, err)
	ccs := configcenter.NewConfigCenterSource(cc, testSource.GetDimensionInfo(), 1, 1)

	t.Log("Accessing concenter source configurations")
	time.Sleep(2 * time.Second)
	_, er := ccs.GetConfigurations()
	if er != nil {
		assert.Error(t, er)
	}
	archaius.Init()
	t.Log("concenter source adding to the archaiuscleanup")
	e := archaius.AddSource(ccs)
	if e != nil {
		assert.Error(t, e)
	}

	t.Log("verifying configcenter configurations by GetConfigurations method")
	_, err = ccs.GetConfigurationByKey("refreshInterval")
	if err != nil {
		assert.Error(t, err)
	}

	_, err = ccs.GetConfigurationByKey("test")
	if err != nil {
		assert.Error(t, err)
	}

	_, err = ccs.GetConfigurationByKeyAndDimensionInfo("data@default#0.1", "test")
	if err != nil {
		assert.Error(t, err)
	}

	t.Log("verifying configcenter name")
	sourceName := configcenter.ConfigCenterConfig.GetSourceName()
	if sourceName != "ConfigCenterSource" {
		t.Error("config-center name is mismatched")
	}

	t.Log("verifying configcenter priority")
	priority := configcenter.ConfigCenterConfig.GetPriority()
	if priority != 0 {
		t.Error("config-center priority is mismatched")
	}

	t.Log("concenter source cleanup")
	configcenter.ConfigCenterConfig.Cleanup()

}

func Test_DynamicConfigHandler(t *testing.T) {
	testsource := &Testingsource{}
	opts := config.Options{
		DimensionInfo: testsource.GetDimensionInfo(),
		TenantName:    "default",
	}
	cc, err := config.NewClient("config_center", opts)
	assert.NoError(t, err)
	ccs := configcenter.NewConfigCenterSource(cc, testsource.GetDimensionInfo(), 1, 1)

	dynamicconfig := new(TestDynamicConfigHandler)

	ccs.DynamicConfigHandler(dynamicconfig)

	//post the new key, or update the already existing key, or delete the existing key to get the events
	time.Sleep(4 * time.Second)

	if dynamicconfig.EventName == "" {
		err := errors.New("failed to get the event if key is created or updated or deleted")
		assert.Error(t, err)
	}

}

func Test_OnReceive(t *testing.T) {
	testSource := &Testingsource{}
	opts := config.Options{
		DimensionInfo: testSource.GetDimensionInfo(),
		TenantName:    "default",
	}
	cc, err := config.NewClient("config_center", opts)
	assert.NoError(t, err)
	ccs := configcenter.NewConfigCenterSource(cc, testSource.GetDimensionInfo(), 1, 1)

	_, er := ccs.GetConfigurations()
	if er != nil {
		assert.Error(t, er)
	}

	dynamicconfig := new(TestDynamicConfigHandler)

	configCenterEvent := new(configcenter.ConfigCenterEvent)
	configCenterEvent.Action = "test"
	check := make(map[string]interface{})
	check["refreshMode"] = 7
	data, _ := json.Marshal(&check)
	configCenterEvent.Value = string(data)

	configCenterEventHandler := new(configcenter.ConfigCenterEventHandler)
	configCenterEventHandler.ConfigSource = configcenter.ConfigCenterConfig
	configCenterEventHandler.Callback = dynamicconfig

	configCenterEventHandler.OnReceive(check)
}
