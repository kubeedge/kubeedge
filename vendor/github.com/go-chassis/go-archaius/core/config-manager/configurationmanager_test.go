package configmanager_test

import (
	"github.com/go-chassis/go-archaius/core"
	"github.com/go-chassis/go-archaius/core/config-manager"
	"github.com/go-chassis/go-archaius/core/event-system"
	"github.com/go-chassis/go-archaius/sources/commandline-source"
	"github.com/go-chassis/go-archaius/sources/file-source"
	"github.com/go-chassis/go-archaius/sources/memory-source"
	"github.com/go-chassis/go-archaius/sources/test-source"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

//GlobalCfg chassis.yaml 配置项
type GlobalCfg struct {
	AppID      string            `yaml:"APPLICATION_ID"` //Deprecated
	Panel      ControlPanel      `yaml:"control"`
	Ssl        map[string]string `yaml:"ssl"`
	Tracing    TracingStruct     `yaml:"tracing"`
	DataCenter *DataCenterInfo   `yaml:"region"`
}

// DataCenterInfo gives data center information
type DataCenterInfo struct {
	Name          string `yaml:"name"`
	Region        string `yaml:"region"`
	AvailableZone string `yaml:"availableZone"`
}

// TracingStruct tracing structure
type TracingStruct struct {
	Tracer   string            `yaml:"tracer"`
	Settings map[string]string `yaml:"settings"`
}

//ControlPanel define control panel config
type ControlPanel struct {
	Infra    string            `yaml:"infra"`
	Settings map[string]string `yaml:"settings"`
}

// LBWrapper loadbalancing structure
type LBWrapper struct {
	Prefix *LoadBalancingConfig `yaml:"cse"`
}

// LoadBalancingConfig loadbalancing structure
type LoadBalancingConfig struct {
	LBConfig *LoadBalancing `yaml:"loadbalance"`
}

// LoadBalancing loadbalancing structure
type LoadBalancing struct {
	Strategy              map[string]string            `yaml:"strategy"`
	RetryEnabled          bool                         `yaml:"retryEnabled"`
	RetryOnNext           int                          `yaml:"retryOnNext"`
	RetryOnSame           int                          `yaml:"retryOnSame"`
	Filters               string                       `yaml:"serverListFilters"`
	Backoff               BackoffStrategy              `yaml:"backoff"`
	SessionStickinessRule SessionStickinessRule        `yaml:"SessionStickinessRule"`
	AnyService            map[string]LoadBalancingSpec `yaml:",inline"`
}

// LoadBalancingSpec loadbalancing structure
type LoadBalancingSpec struct {
	Strategy              map[string]string     `yaml:"strategy"`
	SessionStickinessRule SessionStickinessRule `yaml:"SessionStickinessRule"`
	RetryEnabled          bool                  `yaml:"retryEnabled"`
	RetryOnNext           int                   `yaml:"retryOnNext"`
	RetryOnSame           int                   `yaml:"retryOnSame"`
	Backoff               BackoffStrategy       `yaml:"backoff"`
}

// SessionStickinessRule loadbalancing structure
type SessionStickinessRule struct {
	SessionTimeoutInSeconds int `yaml:"sessionTimeoutInSeconds"`
	SuccessiveFailedTimes   int `yaml:"successiveFailedTimes"`
}

// BackoffStrategy back off strategy
type BackoffStrategy struct {
	Kind  string `yaml:"kind"`
	MinMs int    `yaml:"minMs"`
	MaxMs int    `yaml:"maxMs"`
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func populateCmdConfig() {
	os.Args = append(os.Args, "--testcmdkey1=cmdkey1")
	os.Args = append(os.Args, "--testcmdkey2=cmdkey2")
	os.Args = append(os.Args, "--aaa=cmdkey3")
}

func TestConfigurationManager(t *testing.T) {

	testConfig := map[string]interface{}{"aaa": "111", "bbb": "222"}
	testSource := testsource.NewTestSource(testConfig)

	dispatcher := eventsystem.NewDispatcher()
	confmanager := configmanager.NewConfigurationManager(dispatcher)
	t.Log("Test configurationmanager.go")

	//note: lowest value has highest priority
	//testSource priority 	=	0
	//memorySourcePriority 	= 	1
	//commandlinePriority 	= 	2
	//envSourcePriority 	= 	3
	//fileSourcePriority    = 	4

	t.Log("Adding testSource to the configuration manager")
	err := confmanager.AddSource(testSource, testSource.GetPriority())
	if err != nil {
		t.Error("Error in adding testSource to the  configuration manager", err)
	}

	//supplying duplicate source
	err = confmanager.AddSource(testSource, testSource.GetPriority())
	if err == nil {
		t.Error("Failed to identify the duplicate config source")
	}

	t.Log("verifying the key testsource key existence from configmanager")
	configvalue := confmanager.GetConfigurationsByKey("aaa")
	if configvalue != "111" {
		t.Error("Failed to get the existing keyvalue pair from configmanager")
	}

	//getting command line configurations
	populateCmdConfig()
	cmdlinesource := commandlinesource.NewCommandlineConfigSource()

	t.Log("Adding cmdlinesource to the configuration manager")
	err = confmanager.AddSource(cmdlinesource, cmdlinesource.GetPriority())
	if err != nil {
		t.Error("Error in adding cmdlinesource to the  configuration manager", err)
	}

	t.Log("Verifying the key of lowest priority(cmdline) source")
	configvalue = confmanager.GetConfigurationsByKey("aaa")
	if configvalue == "cmdkey3" {
		t.Error("Failed to get the keyvalue pair of the highest priority source from configmanager")
	}

	//Accessing not existing key in configmanager
	configvalue = confmanager.GetConfigurationsByKey("notExistingKey")
	if configvalue != nil {
		t.Error("configmanager having invalidkeys")
	}

	t.Log("accessing all configurations")
	configurations := confmanager.GetConfigurations()
	if configurations["testcmdkey1"] != "cmdkey1" && configurations["bbb"] != "222" {
		t.Error("Failed to get configurations")
	}

	confByDI, _ := confmanager.GetConfigurationsByDimensionInfo("darklaunch@default#0.0.1")
	assert.NotEqual(t, confByDI, nil)

	addDI, _ := confmanager.AddDimensionInfo("testdi@default")
	assert.NotEqual(t, addDI, nil)

	t.Log("create event through testsource")
	time.Sleep(10 * time.Millisecond)
	testsource.AddConfig("zzz", "333")
	time.Sleep(10 * time.Millisecond)
	t.Log("Accessing keyvalue pair of created event")
	configvalue = confmanager.GetConfigurationsByKey("zzz")
	if configvalue != "333" {
		t.Error("Failed to get the keyvalue pair for created event")
	}

	t.Log("Refresh the testSource configurations")
	testsource.AddConfig("ccc", "444")
	err = confmanager.Refresh(testSource.GetSourceName())
	if err != nil {
		t.Error(err)
	}

	t.Log("verifying the configurations after updation")
	configurations = confmanager.GetConfigurations()
	if configurations["ccc"] != "444" {
		t.Error("Failed to refresh the configurations")
	}

	//Verifying with the invalidsource refreshing
	if err = confmanager.Refresh("InvalidSource"); err == nil {
		t.Error(err)
	}

	//Supplying nil event
	ConfManager2 := &configmanager.ConfigurationManager{}
	var event *core.Event = nil
	ConfManager2.OnEvent(event)

	t.Log("Adding MEMORY source to generate the events based on priority of the key")
	extsource := memoryconfigsource.NewMemoryConfigurationSource()
	confmanager.AddSource(extsource, extsource.GetPriority())

	t.Log("Create event through extsource")
	extsource.AddKeyValue("Commonkey", "extsource")
	time.Sleep(10 * time.Millisecond)
	if configvalue = confmanager.GetConfigurationsByKey("Commonkey"); configvalue != "extsource" {
		t.Error("Failed to get the create event of extsource from configmanager")
	}

	t.Log("update event through testsource(highest priority)")
	testsource.AddConfig("Commonkey", "testsource")
	time.Sleep(10 * time.Millisecond)
	configvalue = confmanager.GetConfigurationsByKey("Commonkey")
	if configvalue != "testsource" {
		t.Error("Failed to get the update event of highest priority source")
	}

	t.Log("update event through extsource(lowest priority)")
	extsource.AddKeyValue("Commonkey", "extsource2")
	time.Sleep(10 * time.Millisecond)
	configvalue = confmanager.GetConfigurationsByKey("Commonkey")
	if configvalue == "extsource2" {
		t.Error("key is updaing from lowest priority source")
	}

	t.Log("update event through testsource(highest priority)")
	testsource.AddConfig("Commonkey", "testsource2")
	time.Sleep(10 * time.Millisecond)
	configvalue = confmanager.GetConfigurationsByKey("Commonkey")
	if configvalue != "testsource2" {
		t.Error("Failed to get the update event of highest priority source")
	}

	t.Log("checking the functionality of IsKeyExist")
	configkey3 := confmanager.IsKeyExist("Commonkey")
	configkey4 := confmanager.IsKeyExist("notexistingkey")
	if configkey3 != true && configkey4 != false {
		t.Error("Failed to identify the status of the keys")
	}

	configmap := make(map[string]interface{}, 0)
	err = confmanager.Unmarshal(&configmap)
	if err != nil {
		t.Error("Failed to unmarshal map: ", err)
	}

	if configmap["Commonkey"] != "testsource2" || configmap["testcmdkey2"] != "cmdkey2" {
		t.Error("Failed to get the keyvalue pairs through unmarshall")
	}

	var testobj string
	err = confmanager.Unmarshal(testobj)
	if err == nil {
		t.Error("Failed to detect invalid object while unmarshalling")
	}

	testsource.CleanupTestSource()
	confmanager.Cleanup()

}

//GetWorkDir is a function used to get the working directory
func GetWorkDir() (string, error) {
	wd, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}
	return wd, nil
}
func TestConfigurationManager_AddSource(t *testing.T) {

	file := []byte(`
region:
  name: us-east
  availableZone: us-east-1
APPLICATION_ID: CSE
register_type: servicecenter
cse:
  service:
    registry:
      type: servicecenter
      scope: full
      autodiscovery: false
      address: 10.19.169.119:30100
      #register: manual
      refeshInterval : 30s
      watch: true
  protocols:
    highway:
      listenAddress: 127.0.0.1:8082
      advertiseAddress: 127.0.0.1:8082
      transport: tcp
    rest:
      listenAddress: 127.0.0.1:8083
      advertiseAddress: 127.0.0.1:8083
      transport: tcp
  handler:
    chain:
      provider:
        default: bizkeeper-provider
ssl:
  cipherPlugin: default
  verifyPeer: false
  cipherSuits: TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
  protocol: TLSv1.2
  caFile:
  certFile:
  keyFile:
  certPwdFile:

commonkey3 : filesource
`)

	loadbalanceConf := []byte(`
--- 
cse: 
  loadbalance: 
    ShoppingCart: 
      backoff: 
        maxMs: 400
        minMs: 200
        kind: constant
      retryEnabled: true
      retryOnNext: 2
      retryOnSame: 3
      serverListFilters: zoneaware
      strategy: 
        name: WeightedResponse
    shoppingCart: 
      backoff: 
        maxMs: 300
        minMs: 200
        kind: constant
      retryEnabled: true
      retryOnNext: 2
      retryOnSame: 3
      serverListFilters: zoneaware
      strategy: 
        name: WeightedResponse
    backoff: 
      maxMs: 400
      minMs: 200
      kind: constant
    retryEnabled: false
    retryOnNext: 2
    retryOnSame: 3
    serverListFilters: zoneaware
    strategy: 
      name: WeightedResponse

`)

	root, _ := GetWorkDir()
	os.Setenv("CHASSIS_HOME", root)
	t.Log(os.Getenv("CHASSIS_HOME"))

	tmpdir := filepath.Join(root, "tmp")
	file1 := filepath.Join(root, "tmp", "chassis.yaml")
	lbFileName := filepath.Join(root, "tmp", "load_balancing.yaml")

	os.Remove(file1)
	os.Remove(lbFileName)
	os.Remove(tmpdir)
	err := os.Mkdir(tmpdir, 0777)
	check(err)
	defer os.Remove(tmpdir)

	f1, err := os.Create(file1)
	check(err)
	defer f1.Close()
	defer os.Remove(file1)
	_, err = io.WriteString(f1, string(file))
	f2, err := os.Create(lbFileName)
	check(err)
	defer f2.Close()
	defer os.Remove(lbFileName)
	_, err = io.WriteString(f2, string(loadbalanceConf))

	dispatcher := eventsystem.NewDispatcher()
	confmanager := configmanager.NewConfigurationManager(dispatcher)

	fsource := filesource.NewFileSource()
	fsource.AddFile(file1, 0, nil)

	confmanager.AddSource(fsource, fsource.GetPriority())
	time.Sleep(2 * time.Second)

	t.Log("verifying Unmarshalling")
	globalDef := GlobalCfg{}
	err = confmanager.Unmarshal(&globalDef)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "CSE", globalDef.AppID)
	assert.Equal(t, "default", globalDef.Ssl["cipherPlugin"])
	assert.Equal(t, "us-east", globalDef.DataCenter.Name)

	confmanager = configmanager.NewConfigurationManager(dispatcher)

	fsource.AddFile(lbFileName, 0, nil)

	confmanager.AddSource(fsource, fsource.GetPriority())
	time.Sleep(2 * time.Second)

	lbConfig := LBWrapper{}
	err = confmanager.Unmarshal(&lbConfig)
	if err != nil {
		t.Error(err)
	}
	t.Log(lbConfig.Prefix.LBConfig)

	t.Log(lbConfig.Prefix.LBConfig.AnyService)
	assert.Equal(t, "WeightedResponse", lbConfig.Prefix.LBConfig.AnyService["ShoppingCart"].Strategy["name"])
	assert.Equal(t, true, lbConfig.Prefix.LBConfig.AnyService["ShoppingCart"].RetryEnabled)
	assert.Equal(t, false, lbConfig.Prefix.LBConfig.RetryEnabled)
	assert.NotEqual(t, "WeightedResponse", lbConfig.Prefix.LBConfig.AnyService["TargetService"].Strategy["name"])
	assert.Equal(t, 300, int(lbConfig.Prefix.LBConfig.AnyService["shoppingCart"].Backoff.MaxMs))
	assert.Equal(t, 400, int(lbConfig.Prefix.LBConfig.AnyService["ShoppingCart"].Backoff.MaxMs))
	assert.Equal(t, 400, int(lbConfig.Prefix.LBConfig.Backoff.MaxMs))

	err = confmanager.Unmarshal("invalidobject")
	if err == nil {
		t.Error("Failed tp detect the invalid object while unmarshalling")
	}

	Namestring := "String"
	err = confmanager.Unmarshal(&Namestring)
	if err != nil {
		t.Error("Unmarshalling is fail on string object")
	}

	t.Log("verifying the commonkey across the sources ")
	assert.Equal(t, "filesource", confmanager.GetConfigurationsByKey("commonkey3"))

	extsource := memoryconfigsource.NewMemoryConfigurationSource()
	confmanager.AddSource(extsource, extsource.GetPriority())

	//update the event through extsource
	extsource.AddKeyValue("commonkey3", "extsource")
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, "extsource", confmanager.GetConfigurationsByKey("commonkey3"))

	assert.NotEqual(t, "extsource", confmanager.GetConfigurationsByKeyAndDimensionInfo("data@default#0.1", "commonkey3"))

	// deleting the common key in filesource
	_, err = exec.Command("sed", "-i", "/commonkey3/d", file1).Output()
	assert.Equal(t, nil, err)
	time.Sleep(10 * time.Millisecond)
	assert.NotEqual(t, "filesource", confmanager.GetConfigurationsByKey("commonkey3"))
	assert.Equal(t, "extsource", confmanager.GetConfigurationsByKey("commonkey3"))

	//update the event through extsource
	extsource.AddKeyValue("commonkey3", "extsource2")
	time.Sleep(10 * time.Millisecond)
	assert.NotEqual(t, "filesource", confmanager.GetConfigurationsByKey("commonkey3"))
	assert.Equal(t, "extsource2", confmanager.GetConfigurationsByKey("commonkey3"))

	testConfig := map[string]interface{}{"aaa": "111", "bbb": "222"}
	testSource := testsource.NewTestSource(testConfig)
	err = confmanager.AddSource(testSource, testSource.GetPriority())
	assert.Equal(t, nil, err)
	time.Sleep(10 * time.Millisecond)

	//updating the common key from high priority source(testsource)
	testsource.AddConfig("commonkey3", "testsource")
	time.Sleep(10 * time.Millisecond)
	assert.NotEqual(t, "filesource", confmanager.GetConfigurationsByKey("commonkey3"))
	assert.NotEqual(t, "extsource2", confmanager.GetConfigurationsByKey("commonkey3"))
	assert.Equal(t, "testsource", confmanager.GetConfigurationsByKey("commonkey3"))

	confmanager.Cleanup()

}
