package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"sigs.k8s.io/yaml"

	edgecore "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

const (
	EdgeCoreConfigFile    = "/tmp/edgecore.yaml"
	CatEdgeCoreConfigFile = "cat /tmp/edgecore.yaml"
	RunEdgecore           = "sudo pkill edgecore; cd ${KUBEEDGE_ROOT}/_output/local/bin/; sudo nohup ./edgecore --config=" + EdgeCoreConfigFile + " > edgecore.log 2>&1 &"
	CheckEdgecore         = "sudo pgrep edgecore"
	CatEdgecoreLog        = "cat ${KUBEEDGE_ROOT}/_output/local/bin/edgecore.log"
	DBFile                = "/tmp/edgecore/edgecore.db"
)

func CreateEdgeCoreConfigFile(nodeName string) error {
	c := edgecore.NewDefaultEdgeCoreConfig()
	c.Modules.Edged.HostnameOverride = nodeName
	c.Modules.EdgeHub.TLSCAFile = "/tmp/edgecore/rootCA.crt"
	c.Modules.EdgeHub.TLSCertFile = "/tmp/edgecore/kubeedge.crt"
	c.Modules.EdgeHub.TLSPrivateKeyFile = "/tmp/edgecore/kubeedge.key"
	c.Modules.EventBus.Enable = true
	c.Modules.EventBus.MqttMode = edgecore.MqttModeInternal
	c.Modules.DBTest.Enable = true
	c.DataBase.DataSource = DBFile
	c.Modules.EdgeStream.Enable = false

	data, err := yaml.Marshal(c)
	if err != nil {
		fmt.Printf("Marshal edgecore config to yaml error %v\n", err)
		os.Exit(1)
	}
	if err := ioutil.WriteFile(EdgeCoreConfigFile, data, os.ModePerm); err != nil {
		fmt.Printf("Create edgecore config file %v error %v\n", EdgeCoreConfigFile, err)
		os.Exit(1)
	}
	return nil
}

func StartEdgeCore() error {
	//Run ./edgecore after node registration
	cmd := exec.Command("sh", "-c", RunEdgecore)
	if err := PrintCombinedOutput(cmd); err != nil {
		return err
	}
	//Expect(err).Should(BeNil())
	time.Sleep(5 * time.Second)

	catConfigcmd := exec.Command("sh", "-c", CatEdgeCoreConfigFile)
	fmt.Printf("===========> Executing: %s\n", strings.Join(catConfigcmd.Args, " "))
	cbytes, _ := catConfigcmd.CombinedOutput()
	fmt.Printf("config content:\n %v", string(cbytes))

	checkcmd := exec.Command("sh", "-c", CheckEdgecore)
	if err := PrintCombinedOutput(checkcmd); err != nil {
		catcmd := exec.Command("sh", "-c", CatEdgecoreLog)
		fmt.Printf("===========> Executing: %s\n", strings.Join(catcmd.Args, " "))
		bytes, _ := catcmd.CombinedOutput()
		fmt.Printf("edgecore log:\n %v", string(bytes))
		fmt.Printf("edgecore started error %v\n", err)
		os.Exit(1)
	}
	return nil
}

// PrintCombinedOutput to show the os command injuction in combined format
func PrintCombinedOutput(cmd *exec.Cmd) error {
	fmt.Printf("===========> Executing: %s\n", strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("CombinedOutput failed %v\n", err)
		return err
	}
	if len(output) > 0 {
		fmt.Printf("=====> Output: %s\n", string(output))
	}
	return nil
}
