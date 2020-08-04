package debug

import (
	"fmt"

	"io"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	constant "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/spf13/cobra"
)

var (
	edgeCheckLongDescription = `Obtain all the data of the current node, and then provide it to the operation
and maintenance personnel to locate the problem`

	edgeCheckShortDescription = `Check specific information.`
	edgeCheckExample          = `
        # Check all items .
        keadm debug check all

        # Check whether the node arch is supported .
        keadm debug check arch

        # Check whether the node CPU meets  requirements.
        keadm debug check cpu

        # Check whether the node memory meets  requirements.
        keadm debug check mem

        # check whether the node disk meets  requirements.
        keadm debug check disk

        # Check whether the node DNS can resolve a specific domain name.
        keadm debug check dns -d www.github.com

        # Check whether the node network meets requirements.
        keadm debug check network

        # Check whether the number of free processes on the node meets requirements.
        keadm debug check pid

        # Check whether runtime(Docker) is installed on the node.
        keadm debug check runtime
`
)

type CheckObject types.CheckObject

// NewEdgecheck returns KubeEdge edge check command.
func NewEdgeCheck(out io.Writer, collectOptions *types.CheckOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "check",
		Short:   edgeCheckShortDescription,
		Long:    edgeCheckLongDescription,
		Example: edgeCheckExample,
	}
	for _, v := range constant.CheckObjectMap {
		cmd.AddCommand(NewSubEdgeCheck(out, CheckObject(v)))
	}
	return cmd
}

// NewEdgecheck returns KubeEdge edge check subcommand.
func NewSubEdgeCheck(out io.Writer, object CheckObject) *cobra.Command {
	co := NewCheckOptins()
	cmd := &cobra.Command{
		Short: object.Desc,
		Use:   object.Use,
		RunE: func(cmd *cobra.Command, args []string) error {
			return object.ExecuteCheck(object.Use, co)
		},
	}
	switch object.Use {
	case constant.ArgCheckAll:
		cmd.Flags().StringVarP(&co.Domain, "domain", "d", co.Domain, "specify test domain")
		cmd.Flags().StringVarP(&co.IP, "ip", "i", co.IP, "specify test ip")
		cmd.Flags().StringVarP(&co.EdgeHubURL, "edge-hub-url", "e", co.EdgeHubURL, "specify edgehub url,")
		cmd.Flags().StringVarP(&co.Runtime, "runtime", "r", co.Runtime, "specify test runtime")
	case constant.ArgCheckDNS:
		cmd.Flags().StringVarP(&co.Domain, "domain", "d", co.Domain, "specify test domain")
	case constant.ArgCheckNetwork:
		cmd.Flags().StringVarP(&co.IP, "ip", "i", co.IP, "specify test ip")
		cmd.Flags().StringVarP(&co.EdgeHubURL, "edge-hub-url", "e", co.EdgeHubURL, "specify edgehub url,")
	case constant.ArgCheckRuntime:
		cmd.Flags().StringVarP(&co.Runtime, "runtime", "r", co.Runtime, "specify test runtime")
	}

	return cmd
}

// add flags
func NewCheckOptins() *types.CheckOptions {
	co := &types.CheckOptions{}
	co.Runtime = "docker"
	co.Domain = "www.github.com"
	co.Timeout = 1
	return co
}

//Start to check data
func (co *CheckObject) ExecuteCheck(use string, ob *types.CheckOptions) error {
	err := fmt.Errorf("")

	switch use {
	case constant.ArgCheckAll:
		err = CheckAll(ob)
	case constant.ArgCheckArch:
		err = CheckArch()
	case constant.ArgCheckCPU:
		err = CheckCPU()
	case constant.ArgCheckMemory:
		err = CheckMemory()
	case constant.ArgCheckDisk:
		err = CheckDisk()
	case constant.ArgCheckDNS:
		err = CheckDNS(ob.Domain)
	case constant.ArgCheckNetwork:
		err = CheckNetWork(ob.IP, ob.Timeout, ob.EdgeHubURL)
	case constant.ArgCheckRuntime:
		err = CheckRuntime(ob.Runtime)
	case constant.ArgCheckPID:
		err = CheckPid()
	}

	if err != nil {
		checkFail(use)
	} else {
		checkSuccedd(use)
	}

	return err
}

func CheckAll(ob *types.CheckOptions) error {
	err := CheckArch()
	if err != nil {
		return err
	}

	err = CheckCPU()
	if err != nil {
		return err
	}

	err = CheckMemory()
	if err != nil {
		return err
	}

	err = CheckDisk()
	if err != nil {
		return err
	}

	err = CheckDNS(ob.Domain)
	if err != nil {
		return err
	}

	err = CheckNetWork(ob.IP, ob.Timeout, ob.EdgeHubURL)
	if err != nil {
		return err
	}

	err = CheckPid()
	if err != nil {
		return err
	}

	err = CheckRuntime(ob.Runtime)
	if err != nil {
		return err
	}
	return nil
}

func CheckArch() error {
	o, err := execShellFilter(constant.CmdGetArch)
	if !IsContain(constant.AllowedValueArch, string(o)) {
		return fmt.Errorf("arch not support: %s", string(o))
	}
	fmt.Printf("arch is : %s\n", string(o))
	return err
}

func CheckCPU() error {
	return ComparisonSize(constant.CmdGetCPUNum, constant.AllowedValueCPU, constant.ArgCheckCPU, constant.UnitCore)
}

func CheckMemory() error {
	return ComparisonSize(constant.CmdGetMenorySize, constant.AllowedValueMemory, constant.ArgCheckMemory, constant.UnitMB)
}

func CheckDisk() error {
	return ComparisonSize(constant.CmdGetDiskSize, constant.AllowedValueDisk, constant.ArgCheckDisk, constant.UnitGB)
}

func CheckDNS(domain string) error {
	r, err := net.LookupHost(domain)
	if err != nil {
		return fmt.Errorf("dns resolution failed, domain: %s err: %s", domain, err)
	}
	if len(r) > 0 {
		fmt.Printf("dns resolution success, domain: %s ip: %s\n", domain, r[0])
	} else {
		fmt.Printf("dns resolution success, domain: %s ip: null\n", domain)
	}
	return err
}

func CheckNetWork(IP string, timeout int, edgeHubURL string) error {
	if IP == "" && edgeHubURL == "" {
		result, err := execShellFilter(constant.CmdGetDNSIP)
		if err != nil {
			return err
		}
		IP = result
	}
	if edgeHubURL != "" {
		err := CheckHTTP(edgeHubURL)
		if err != nil {
			return err
		}
	}
	if IP != "" {
		result, err := execShellFilter(fmt.Sprintf(constant.CmdPing, IP, timeout))

		if err != nil {
			return err
		}
		if result != "1" {
			return fmt.Errorf("ping %s timeout", IP)
		}
		fmt.Printf("ping %s success\n", IP)
	}
	return nil
}

func CheckHTTP(url string) error {
	// setup a http client
	httpTransport := &http.Transport{}
	httpClient := &http.Client{Transport: httpTransport, Timeout: time.Second * 3}
	response, err := httpClient.Get(url)
	if err != nil {
		if !strings.Contains(err.Error(), "x509") {
			return fmt.Errorf("edgehub url connect fail: %s", err.Error())
		}
	} else {
		fmt.Printf("edgehub url connect success: %s\n", url)
		defer response.Body.Close()
	}
	return nil
}

func CheckRuntime(runtime string) error {
	if runtime == "docker" {
		result, err := execShellFilter(constant.CmdGetStatusDocker)
		if err != nil {
			return err
		}
		if result != "active" {
			return fmt.Errorf("docker is not running: %s", result)
		}
		fmt.Printf("docker is running\n")
	} else {
		return fmt.Errorf("now only support docker: %s", runtime)
		// TODO
	}
	return nil
}

func CheckPid() error {
	rMax, err := execShellFilter(constant.CmdGetMaxProcessNum)
	if err != nil {
		return err
	}
	r, err := execShellFilter(constant.CmdGetProcessNum)
	if err != nil {
		return err
	}
	vMax, err := strconv.ParseFloat(rMax, 32)
	v, err := strconv.ParseFloat(r, 32)
	rate := (1 - v/vMax)
	if rate > constant.AllowedValuePIDRate {
		fmt.Printf("Maximum PIDs: %s; Running processes: %s\n", rMax, r)
		return nil
	}
	return fmt.Errorf("Maximum PIDs: %s; Running processes: %s", rMax, r)
}

/**
Execute command and compare size
c:       cmd
require: Minimum resource requirement
name：   the name  of check item
unit:    resourceUnit, e.g. MB，GB
*/
func ComparisonSize(c string, require string, name string, unit string) error {
	result, err := execShellFilter(c)
	if err != nil {
		return fmt.Errorf("exec \"%s\" fail: %s", c, err.Error())
	}
	if len(result) == 0 {
		return fmt.Errorf("exec \"%s\" fail", c)
	}
	resultInt, err := ConverData(result)
	if err != nil {
		return fmt.Errorf("conver %s fail: %s", result, err.Error())
	}
	requireInt, err := ConverData(require)
	if err != nil {
		return fmt.Errorf("conver %s fail: %s", require, err)
	}

	if resultInt < requireInt {
		return fmt.Errorf("%s requirements: %s, current value: %s", name, require, result)
	}
	fmt.Printf("%s requirements: %s, current value: %s\n", name, require, result)

	return nil
}

// Execute shell script and filter
func execShellFilter(c string) (string, error) {
	cmd := exec.Command("sh", "-c", c)
	o, err := cmd.Output()
	str := strings.Replace(string(o), " ", "", -1)
	str = strings.Replace(str, "\n", "", -1)
	if err != nil {
		return str, fmt.Errorf("exec fail: %s, %s", c, err)
	}
	return str, nil
}

// Determine if it is in the array
func IsContain(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

/**
Convert data string to int type
input: data, for example: 1GB、1G、1MB、1024k、1024
*/
func ConverData(input string) (int, error) {
	// If it is a number, just return
	v, err := strconv.Atoi(input)
	if err == nil {
		return v, nil
	}

	re, err := regexp.Compile(`([0-9]+)([a-zA-z]+)`)
	if err != nil {
		return 0, err
	}
	result := re.FindStringSubmatch(input)
	if len(result) != 3 {
		return 0, fmt.Errorf("regexp err")
	}
	v, err = strconv.Atoi(result[1])
	if err != nil {
		return 0, err
	}
	unit := strings.ToUpper(result[2])
	unit = unit[:1]

	switch unit {
	case "G":
		v = v * constant.GB
	case "M":
		v = v * constant.MB
	case "K":
		v = v * constant.KB
	default:
		return 0, fmt.Errorf("unit err")
	}
	return v, nil
}

//print fail
func checkFail(s string) {
	s = s + " check failed."
	fmt.Println("\n+-------------------+")
	fmt.Printf("|%s|\n", s)
	fmt.Println("+-------------------+")
}

//print success
func checkSuccedd(s string) {
	s = s + " check succeed."
	fmt.Println("\n+-------------------+")
	fmt.Printf("|%s|\n", s)
	fmt.Println("+-------------------+")
}
