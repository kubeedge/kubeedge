/*
 * Copyright 2017 Huawei Technologies Co., Ltd
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
* Created by on 2017/6/22.
 */
package filesource

import (
	"github.com/go-chassis/go-archaius/core"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type EListener struct {
	Name      string
	EventName string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type TestDynamicConfigHandler struct {
	EventName  string
	EventKey   string
	EventValue interface{}
}

func (t *TestDynamicConfigHandler) OnEvent(e *core.Event) {
	t.EventKey = e.Key
	t.EventName = e.EventType
	t.EventValue = e.Value
}

//GetWorkDir is a function used to get the working directory
func GetWorkDir() (string, error) {
	wd, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}
	return wd, nil
}
func TestNewYamlConfigurationSource1(t *testing.T) {

	root, _ := GetWorkDir()
	os.Setenv("CHASSIS_HOME", root)

	t.Log("Test yamlconfigurationsource.go")

	confdir := filepath.Join(root, "conf")
	file1 := filepath.Join(root, "conf", "test1.yaml")
	file2 := filepath.Join(root, "conf", "test2.yaml")

	f1content := []byte(`
testfilekey1: filekey1
testfilekey2: filekey2
testfilekey3:
  subtestkey1: filekey31
  subtestkey2: filekey32
`)
	f2content := "NAME21: test21\n \nNAME22: test22"

	os.Remove(file1)
	os.Remove(file2)
	os.Remove(confdir)
	err := os.Mkdir(confdir, 0777)
	check(err)
	defer os.Remove(confdir)

	f1, err := os.Create(file1)
	check(err)
	f2, err := os.Create(file2)
	check(err)
	defer f1.Close()
	defer f2.Close()
	defer os.Remove(file1)
	defer os.Remove(file2)

	_, err = io.WriteString(f1, string(f1content))
	check(err)
	_, err = io.WriteString(f2, f2content)
	check(err)

	fSource := NewFileSource()

	//Configuration file1 is adding to the filesource
	err = fSource.AddFile(file1, 0, nil)
	if err != nil {
		t.Error(err)
	}

	//Duplicate file(file1) is adding to the filesource
	err = fSource.AddFile(file1, 0, nil)
	if err != nil {
		t.Error(err)
	}

	//Not existing path file adding to the filesource
	err = fSource.AddFile(confdir+"/notexistingdir/notexisting.yaml", 0, nil)
	if err == nil {
		t.Error("filesource working on not existing path")
	}

	//Not existing file adding to the filesource
	err = fSource.AddFile(confdir+"/notexisting.yaml", 0, nil)
	if err == nil {
		t.Error("filesource working on not existing file")
	}

	//Adding directory to the filesource
	err = fSource.AddFile(confdir, 0, nil)
	if err != nil {
		t.Error("Failed to add directory to the filesource")
	}

	t.Log("verifying filesource configurations by GetConfigurations method")
	_, err = fSource.GetConfigurations()
	if err != nil {
		t.Error("Failed to get the configurations from filesource")
	}

	t.Log("verifying filesource configurations by GetConfigurationByKey method")
	configkey, _ := fSource.GetConfigurationByKey("testfilekey3.subtestkey1")
	if configkey != "filekey31" {
		t.Error("Failed to the filesource keyvalue pair")
	}

	configkey, _ = fSource.GetConfigurationByKey("NAME21")
	if configkey != "test21" {
		t.Error("Failed to the filesource keyvalue pair")
	}

	t.Log("Verifying the filesource priority")
	extsorcepriority := fSource.GetPriority()
	if extsorcepriority != 4 {
		t.Error("filesource priority is mismatched")
	}

	t.Log("Verifying the filesource name")
	filesourcename := fSource.GetSourceName()
	if filesourcename != "FileSource" {
		t.Error("filesource name is mismatched")
	}

	t.Log("Filesource cleanup")
	filesourcecleanup := fSource.Cleanup()
	if filesourcecleanup != nil {
		t.Error("filesource cleanup is Failed")
	}

}

func TestDynamicConfigurations(t *testing.T) {

	root, _ := GetWorkDir()
	os.Setenv("CHASSIS_HOME", root)

	tmpdir := filepath.Join(root, "tmp")
	filename1 := filepath.Join(root, "tmp", "test1.yaml")
	filename2 := filepath.Join(root, "tmp", "test2.yaml")
	filename3 := filepath.Join(root, "tmp", "test3.yaml")
	filename4 := filepath.Join(root, "tmp", "test4.yaml")
	filename5 := filepath.Join(root, "tmp", "test5.yaml")

	yamlContent1 := "yamlkeytest11: test11\n \nyamlkeytest12: test12\n \nyamlkeytest123: test1231"
	yamlContent2 := "yamlkeytest21: test21\n \nyamlkeytest22: test22\n \nyamlkeytest123: test1232"
	yamlContent3 := "yamlkeytest31: test31\n \nyamlkeytest32: test32\n \nyamlkeytest123: test1233"
	yamlContent4 := "yamlkeytest41: test41\n \nyamlkeytest42: test32\n \nyamlkeytest45: test454"
	yamlContent5 := "yamlkeytest51: test51\n \nyamlkeytest52: test52\n \nyamlkeytest123: test1233"

	os.Remove(filename1)
	os.Remove(filename2)
	os.Remove(filename3)
	os.Remove(filename4)
	os.Remove(filename5)
	os.Remove(tmpdir)
	err := os.Mkdir(tmpdir, 0777)
	check(err)
	defer os.Remove(tmpdir)

	f1, err := os.Create(filename1)
	check(err)
	defer f1.Close()
	defer os.Remove(filename1)
	f2, err := os.Create(filename2)
	check(err)
	defer f2.Close()
	defer os.Remove(filename2)
	f3, err := os.Create(filename3)
	check(err)
	defer f3.Close()
	defer os.Remove(filename3)
	f4, err := os.Create(filename4)
	check(err)
	defer f4.Close()
	defer os.Remove(filename4)
	f5, err := os.Create(filename5)
	check(err)
	defer f5.Close()
	defer os.Remove(filename5)

	_, err = io.WriteString(f1, yamlContent1)
	check(err)
	_, err = io.WriteString(f2, yamlContent2)
	check(err)
	_, err = io.WriteString(f3, yamlContent3)
	check(err)
	_, err = io.WriteString(f4, yamlContent4)
	check(err)
	_, err = io.WriteString(f5, yamlContent5)
	check(err)

	fSource := NewFileSource()
	fSource.AddFile(filename1, 0, nil)
	fSource.AddFile(filename2, 1, nil)
	fSource.AddFile(filename3, 2, nil)

	dynHandler := new(TestDynamicConfigHandler)
	fSource.DynamicConfigHandler(dynHandler)
	time.Sleep(1 * time.Second)

	t.Log("generate event by inserting some value into file")
	yamlContent1 = "\nyamlkeytest13: test13\n"
	_, err = io.WriteString(f1, yamlContent1)
	check(err)
	time.Sleep(10 * time.Millisecond)

	t.Log("Verifying the key of highest priority file(filename1)")
	configkey, err := fSource.GetConfigurationByKey("yamlkeytest13")
	if configkey != "test13" {
		t.Error("Failed to get the latest event key value pair")
	}

	//Accessing key of file2 priority is 1
	configkey, _ = fSource.GetConfigurationByKey("yamlkeytest21")
	if configkey != "test21" {
		t.Error("Failed to get the latest event key value pair")
	}

	//verifying the key of highest priority file(filename1)
	configkey, _ = fSource.GetConfigurationByKey("yamlkeytest123")
	if configkey != "test1231" {
		t.Error("Failed to get the latest event key value pair")
	}

	//generating the key from highest priority file(filename1)
	yamlContent1 = "\nyamlkeytest123: test12311\n"
	_, err = io.WriteString(f1, yamlContent1)
	check(err)
	time.Sleep(10 * time.Millisecond)

	//Verifying the of highest priority file(filename1)
	configkey, err = fSource.GetConfigurationByKey("yamlkeytest123")
	if configkey != "test12311" {
		t.Error("filesource updating the key from lowest priority file")
	}

	t.Log("generating the key from lowest priority file(filename3)")
	yamlContent3 = "\nyamlkeytest123: test12333\n"
	_, err = io.WriteString(f3, yamlContent3)
	check(err)
	time.Sleep(10 * time.Millisecond)

	t.Log("verifying the key of lowest priority file(filename3)")
	configkey, err = fSource.GetConfigurationByKey("yamlkeytest123")
	if configkey == "test12333" {
		t.Error("filesource updating the key from lowest priority file")
	}

	t.Log("adding new files after dynhandler is inited")
	fSource.AddFile(filename4, 3, nil)
	fSource.AddFile(filename5, 4, nil)
	time.Sleep(10 * time.Millisecond)

	t.Log("verifying the configurations of newely added files")
	configkey, err = fSource.GetConfigurationByKey("yamlkeytest41")
	assert.Equal(t, "test41", configkey)
	configkey, _ = fSource.GetConfigurationByKey("yamlkeytest51")
	assert.Equal(t, "test51", configkey)

	t.Log("creating the event from newely added file(filename4)")
	yamlContent4 = "\nyamlkeytest45: test454\n"
	_, err = io.WriteString(f4, yamlContent4)
	check(err)
	time.Sleep(10 * time.Millisecond)
	configkey, _ = fSource.GetConfigurationByKey("yamlkeytest45")
	assert.Equal(t, "test454", configkey)

	t.Log("update event from lowest priority file(filename5)")
	yamlContent5 = "\nyamlkeytest45: test455\n"
	_, err = io.WriteString(f5, yamlContent5)
	check(err)
	time.Sleep(10 * time.Millisecond)
	configkey, _ = fSource.GetConfigurationByKey("yamlkeytest45")
	t.Log("verifying the event from lowest priority file(filename5)")
	assert.NotEqual(t, "test455", configkey)
	assert.Equal(t, "test454", configkey)

	data, err := fSource.GetConfigurationByKeyAndDimensionInfo("data@default#0.1", "hello")
	if data != nil || err != nil {
		t.Error("Failed to get configuration by dimension info and key")
	}

	t.Log("filesource cleanup")
	filesourcecleanup := fSource.Cleanup()
	if filesourcecleanup != nil {
		t.Error("filesource cleanup is Failed")
	}
}

func TestNewYamlConfigurationSource2(t *testing.T) {

	root, _ := GetWorkDir()
	os.Setenv("CHASSIS_HOME", root)

	tmpdir := filepath.Join(root, "tmp")
	file1 := filepath.Join(root, "tmp", "test1.invalid")
	file1content := "NAME1==== test11\n \nNAME12: test12"

	os.Remove(file1)
	os.Remove(tmpdir)
	err := os.Mkdir(tmpdir, 0777)
	check(err)
	defer os.Remove(tmpdir)

	f1, err := os.Create(file1)
	check(err)
	defer f1.Close()
	defer os.Remove(file1)
	_, err = io.WriteString(f1, file1content)
	check(err)

	fSource := NewFileSource()

	t.Log("improper configuration file adding to the filesource")
	err = fSource.AddFile(file1, 0, nil)
	if err == nil {
		t.Error(err)
	}

	t.Log("supplying nil dynHandler")
	var dynHandler core.DynamicConfigCallback
	err = fSource.DynamicConfigHandler(dynHandler)
	if err == nil {
		t.Error("file source working on nil callback")
	}

	t.Log("filoesource cleanup")
	filesourcecleanup := fSource.Cleanup()
	if filesourcecleanup != nil {
		t.Error("filesource cleanup is Failed")
	}
}
