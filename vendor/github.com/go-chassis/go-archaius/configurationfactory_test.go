package archaius_test

import (
	"testing"

	"fmt"
	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-archaius/core"
	"github.com/go-chassis/go-archaius/sources/commandline-source"
	"github.com/go-chassis/go-archaius/sources/enviromentvariable-source"
	"github.com/go-chassis/go-archaius/sources/file-source"
	"github.com/go-chassis/go-archaius/sources/memory-source"
	"github.com/go-chassis/go-archaius/sources/test-source"
	"github.com/go-mesh/openlogging"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"path/filepath"
	"time"
)

type EventListener struct{}

func (e EventListener) Keys() []string {
	return []string{"age"}
}
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func populateCmdConfig() {
	os.Args = append(os.Args, "--testcmdkey1=cmdkey1")
	os.Args = append(os.Args, "--testcmdkey2=cmdkey2")
	os.Args = append(os.Args, "--commonkey=cmdsource1")
}

func TestConfigFactory(t *testing.T) {
	t.Log("write files")
	d, err := os.Getwd()
	assert.NoError(t, err)
	t.Log("Test configurationfactory.go")
	f1content := "APPLICATION_ID: CSE\n  \ncse:\n  service:\n    registry:\n      type: servicecenter\n  protocols:\n       highway:\n         listenAddress: 127.0.0.1:8080\n  \nssl:\n  test.consumer.certFile: test.cer\n  test.consumer.keyFile: test.key\n"

	confdir := filepath.Join(d, "conf")
	filename1 := filepath.Join(d, "conf", "chassis.yaml")

	os.Remove(filename1)
	os.Remove(confdir)
	err = os.Mkdir(confdir, 0777)
	check(err)
	f1, err1 := os.Create(filename1)
	check(err1)
	defer os.Remove(confdir)
	defer f1.Close()
	defer os.Remove(filename1)

	_, err1 = io.WriteString(f1, f1content)
	populateCmdConfig()
	t.Log("init factory")
	factory, err := archaius.NewConfigFactory()
	assert.NoError(t, err)
	factory.Init()
	// build-in config sources
	ms := memoryconfigsource.NewMemoryConfigurationSource()
	err = factory.AddSource(ms)
	assert.NoError(t, err)
	cmdSource := commandlinesource.NewCommandlineConfigSource()
	err = factory.AddSource(cmdSource)
	assert.NoError(t, err)
	envSource := envconfigsource.NewEnvConfigurationSource()
	err = factory.AddSource(envSource)
	assert.NoError(t, err)

	t.Log("verifying methods before config factory initialization")
	factory.DeInit()
	t.Log(factory.GetValue("testkey"))

	assert.Equal(t, nil, factory.GetValue("testkey"))
	assert.Error(t, factory.AddSource(nil))
	assert.Equal(t, map[string]interface{}(map[string]interface{}(nil)), factory.GetConfigurations())
	assert.Equal(t, false, factory.IsKeyExist("testkey"))
	assert.Equal(t, nil, factory.Unmarshal("testkey"))
	assert.Equal(t, nil, factory.GetConfigurationByKey("testkey"))
	assert.Equal(t, nil, factory.GetConfigurationByKeyAndDimensionInfo("data@default#0.1", "hello"))
	t.Log("DeInit")
	factory.DeInit()
	t.Log("Init")
	factory.Init()
	defer factory.DeInit()

	//note: lowest value has highest priority
	//testSource priority 	=	0
	//memSourcePriority 	= 	1
	//commandlinePriority 	= 	2
	//envSourcePriority 	= 	3
	//fileSourcePriority    = 	4

	time.Sleep(10 * time.Millisecond)
	fmt.Println("wake up")
	eventHandler := EventListener{}
	t.Log("Register Listener")
	err = factory.RegisterListener(eventHandler, "a*")
	assert.NoError(t, err)
	defer factory.UnRegisterListener(eventHandler, "a*")
	defer t.Log("UnRegister Listener")

	fmt.Println("verifying existing configuration keyvalue pair")
	configvalue := factory.GetConfigurationByKey("commonkey")
	if configvalue != "cmdsource1" {
		t.Error("Failed to get the existing keyvalue pair")
	}

	fmt.Println("Adding filesource to the configfactroy")
	fsource := filesource.NewFileSource()
	fsource.AddFile(filename1, 0, nil)
	err = factory.AddSource(fsource)
	assert.NoError(t, err)

	fmt.Println("Generating event through testsource(priority 1)")
	fmt.Println("Generating event through testsource(priority 1)")
	ms.AddKeyValue("commonkey", "memsource1")

	fmt.Println("verifying the key of lower priority source")
	time.Sleep(10 * time.Millisecond)
	configvalue = factory.GetConfigurationByKey("commonkey")
	if configvalue != "memsource1" {
		t.Error("Failed to get the existing keyvalue pair")
	}

	fmt.Println("Adding testsource to the configfactory")
	testConfig := map[string]interface{}{"aaa": "111", "bbb": "222", "commonkey": "testsource1"}
	testSource := testsource.NewTestSource(testConfig)
	err = factory.AddSource(testSource)
	assert.NoError(t, err)
	defer testsource.CleanupTestSource()

	fmt.Println("verifying common configuration keyvalue pair ")
	time.Sleep(10 * time.Millisecond)
	configvalue = factory.GetConfigurationByKey("commonkey")
	if configvalue != "testsource1" {
		t.Error("Failed to get the key highest priority sorce")
	}

	fmt.Println("verifying filesource configurations ")
	configurations := factory.GetConfigurations()
	if configurations["testcmdkey2"] != "cmdkey2" || configurations["APPLICATION_ID"] != "CSE" {
		t.Error("Failed to get the configurations")
	}

	confByDI := factory.GetConfigurationsByDimensionInfo("darklaunch@default#0.0.1")
	assert.NotEqual(t, confByDI, nil)

	addDI, _ := factory.AddByDimensionInfo("darklaunch@default#0.0.1")
	assert.NotEqual(t, addDI, nil)

	if factory.IsKeyExist("commonkey") != true || factory.IsKeyExist("notexistingkey") != false {
		t.Error("Failed to get the exist status of the keys")
	}

	fmt.Println("verifying memsource configurations and accessing in different data type formats")
	ms.AddKeyValue("stringkey", "true")
	time.Sleep(10 * time.Millisecond)
	configvalue2, err := factory.GetValue("stringkey").ToBool()
	if err != nil || configvalue2 != true {
		t.Error("failed to get the value in bool")
	}

	ms.AddKeyValue("boolkey", "hello")
	time.Sleep(10 * time.Millisecond)
	configvalue3, err := factory.GetValue("boolkey").ToBool()
	if err != nil || configvalue3 != false {
		t.Error("Failed to get the value for string in convertion to bool")
	}

	configvalue4, err := factory.GetValue("nokey").ToBool()
	if err == nil || configvalue4 != false {
		t.Error("Error for nil key and value")
	}

	data, err := factory.GetValueByDI("darklaunch@default#0.0.1", "hi").ToString()
	assert.Equal(t, data, "")
	assert.Error(t, err)

	configmap := make(map[string]interface{}, 0)
	err = factory.Unmarshal(&configmap)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, "cmdkey1", configmap["testcmdkey1"])
	assert.Equal(t, "111", configmap["aaa"])

	//supplying nil listener.
	var listener core.EventListener
	err = factory.RegisterListener(listener, "key")
	if err == nil {
		t.Error("Failed to detect the nil listener while registering")
	}

	//supplying nil listener
	err = factory.UnRegisterListener(listener, "key")
	if err == nil {
		t.Error("Failed to detect the nil listener while unregistering")
	}
}

func (e EventListener) Event(event *core.Event) {
	openlogging.GetLogger().Infof("config value after change ", event.Key, " | ", event.Value)
}
