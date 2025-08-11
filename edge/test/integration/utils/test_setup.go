package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	edgecore "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/edge"
	"github.com/kubeedge/kubeedge/pkg/version"
)

const (
	RunEdgecoreCmdFormat = "sudo pkill edgecore; cd ${KUBEEDGE_ROOT}/_output/local/bin/; nohup sudo ./edgecore --config=%s > edgecore.log 2>&1 &"
	CheckEdgecoreCmd     = "sudo pgrep edgecore"
	CatEdgecoreLogCmd    = "cat ${KUBEEDGE_ROOT}/_output/local/bin/edgecore.log"
)

func CreateEdgeCoreConfigFile(nodeName string) error {
	c := edgecore.NewDefaultEdgeCoreConfig()
	c.EdgeCoreVersion = version.Get().String()
	c.Modules.Edged.HostnameOverride = nodeName
	c.Modules.EdgeHub.TLSCAFile = "/tmp/edgecore/rootCA.crt"
	c.Modules.EdgeHub.TLSCertFile = "/tmp/edgecore/kubeedge.crt"
	c.Modules.EdgeHub.TLSPrivateKeyFile = "/tmp/edgecore/kubeedge.key"
	c.Modules.DeviceTwin.DMISockPath = "/etc/kubeedge/dmi.sock"
	c.Modules.EventBus.Enable = true
	c.Modules.EventBus.MqttMode = edgecore.MqttModeInternal
	c.Modules.DBTest.Enable = true
	c.DataBase.DataSource = edge.DBFile
	c.Modules.EdgeStream.Enable = false

	if err := c.WriteTo(edge.ConfigFile); err != nil {
		fmt.Printf("Create edgecore config file %v error %v\n", edge.ConfigFile, err)
		os.Exit(1)
	}
	return nil
}

func StartEdgeCore() error {
	//Run ./edgecore after node registration
	cmd := exec.Command("sh", "-c", fmt.Sprintf(RunEdgecoreCmdFormat, edge.ConfigFile))
	if err := PrintCombinedOutput(cmd); err != nil {
		return err
	}
	//Expect(err).Should(BeNil())
	time.Sleep(5 * time.Second)

	catConfigcmd := exec.Command("sh", "-c", "cat "+edge.ConfigFile)
	fmt.Printf("===========> Executing: %s\n", strings.Join(catConfigcmd.Args, " "))
	cbytes, _ := catConfigcmd.CombinedOutput()
	fmt.Printf("config content:\n %v", string(cbytes))

	checkcmd := exec.Command("sh", "-c", CheckEdgecoreCmd)
	if err := PrintCombinedOutput(checkcmd); err != nil {
		catcmd := exec.Command("sh", "-c", CatEdgecoreLogCmd)
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
