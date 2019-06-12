package testsource

import (
	"errors"
	"github.com/go-chassis/go-archaius/core"
	"sync"
)

//TestingSource is a struct
type TestingSource struct {
	Configuration  map[string]interface{}
	changeCallback core.DynamicConfigCallback
	sync.Mutex
	priority int
}

var _ core.ConfigSource = &TestingSource{}

var testSource *TestingSource

//NewTestSource is a function for creating new test source
func NewTestSource(initConf map[string]interface{}) core.ConfigSource {
	if testSource == nil {
		testSource = new(TestingSource)
		testSource.Configuration = make(map[string]interface{})
		for key, value := range initConf {
			testSource.Configuration[key] = value
		}
	}

	return testSource
}

//GetTestSource returns a test source object
func GetTestSource() *TestingSource {
	return testSource
}

//AddConfig adds a new configuration
func AddConfig(key string, value interface{}) {

	event := new(core.Event)
	event.EventSource = testSource.GetSourceName()
	event.Key = key
	event.Value = value

	testSource.Lock()
	if _, ok := testSource.Configuration[key]; !ok {
		event.EventType = core.Create
	} else {
		event.EventType = core.Update
	}
	testSource.Configuration[key] = value
	testSource.Unlock()

	if testSource.changeCallback != nil {
		testSource.changeCallback.OnEvent(event)
	}
}

//RemoveConfig removes a configuration
func RemoveConfig(key string, value interface{}) {
	testSource.Lock()
	defer testSource.Unlock()
	event := new(core.Event)
	event.EventSource = testSource.GetSourceName()
	event.Key = key
	event.Value = value
	event.EventType = core.Delete
	if testSource.changeCallback != nil {
		delete(testSource.Configuration, key)
	}
}

//CleanupTestSource cleans up every configuration
func CleanupTestSource() {
	testSource.Cleanup()
	testSource = nil
}

//GetConfigurations gets all test configurations
func (test *TestingSource) GetConfigurations() (map[string]interface{}, error) {
	config := make(map[string]interface{})
	testSource.Lock()
	defer testSource.Unlock()

	for key, value := range test.Configuration {
		config[key] = value
	}

	return config, nil
}

//GetPriority returns priority of the test configuration
func (test *TestingSource) GetPriority() int {
	return 0
}

//SetPriority custom priority
func (test *TestingSource) SetPriority(priority int) {
	//No need to implement
}

//Cleanup cleans a particular test configuration up
func (test *TestingSource) Cleanup() error {
	testSource.Lock()
	defer testSource.Unlock()
	test.Configuration = make(map[string]interface{})
	test.changeCallback = nil
	return nil
}

//GetSourceName returns name of test configuration
func (*TestingSource) GetSourceName() string {
	return "TestingSource"
}

//DynamicConfigHandler dynamically handles a test configuration
func (test *TestingSource) DynamicConfigHandler(callback core.DynamicConfigCallback) error {
	testSource.Lock()
	defer testSource.Unlock()
	test.changeCallback = callback
	return nil
}

//GetConfigurationByKey gets required test configuration for a particular key
func (test *TestingSource) GetConfigurationByKey(key string) (interface{}, error) {
	testSource.Lock()
	defer testSource.Unlock()
	configValue, ok := test.Configuration[key]
	if !ok {
		return nil, errors.New("invalid key")
	}
	return configValue, nil
}

//GetConfigurationByKeyAndDimensionInfo gets a required test configuration for particular key and dimension info pair
func (*TestingSource) GetConfigurationByKeyAndDimensionInfo(key, di string) (interface{}, error) {
	return nil, nil
}

//AddDimensionInfo adds dimension info for a test configuration
func (*TestingSource) AddDimensionInfo(dimensionInfo string) (map[string]string, error) {
	return nil, nil
}

//GetConfigurationsByDI gets required test configuration for a particular dimension info
func (TestingSource) GetConfigurationsByDI(dimensionInfo string) (map[string]interface{}, error) {
	return nil, nil
}
