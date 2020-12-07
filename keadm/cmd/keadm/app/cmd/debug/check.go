package debug

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
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

type CheckObject common.CheckObject

// NewEdgecheck returns KubeEdge edge check command.
func NewCheck(out io.Writer, collectOptions *common.CheckOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "check",
		Short:   edgeCheckShortDescription,
		Long:    edgeCheckLongDescription,
		Example: edgeCheckExample,
	}
	for _, v := range common.CheckObjectMap {
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
		Run: func(cmd *cobra.Command, args []string) {
			object.ExecuteCheck(object.Use, co)
		},
	}
	switch object.Use {
	case common.ArgCheckAll:
		cmd.Flags().StringVarP(&co.Domain, "domain", "d", co.Domain, "specify test domain")
		cmd.Flags().StringVarP(&co.IP, "ip", "i", co.IP, "specify test ip")
		cmd.Flags().StringVarP(&co.CloudHubServer, "cloud-hub-server", "s", co.CloudHubServer, "specify cloudhub server")
		cmd.Flags().StringVarP(&co.Runtime, "runtime", "r", co.Runtime, "specify test runtime")
		cmd.Flags().StringVarP(&co.DNSIP, "dns-ip", "D", co.DNSIP, "specify test dns ip")
		cmd.Flags().StringVarP(&co.Config, common.EdgecoreConfig, "c", co.Config,
			fmt.Sprintf("Specify configuration file, defalut is %s", common.EdgecoreConfigPath))
	case common.ArgCheckDNS:
		cmd.Flags().StringVarP(&co.Domain, "domain", "d", co.Domain, "specify test domain")
		cmd.Flags().StringVarP(&co.DNSIP, "dns-ip", "D", co.DNSIP, "specify test dns ip")
	case common.ArgCheckNetwork:
		cmd.Flags().StringVarP(&co.IP, "ip", "i", co.IP, "specify test ip")
		cmd.Flags().StringVarP(&co.CloudHubServer, "cloud-hub-server", "s", co.CloudHubServer, "specify cloudhub server")
		cmd.Flags().StringVarP(&co.Config, common.EdgecoreConfig, "c", co.Config,
			fmt.Sprintf("Specify configuration file, defalut is %s", common.EdgecoreConfigPath))
	case common.ArgCheckRuntime:
		cmd.Flags().StringVarP(&co.Runtime, "runtime", "r", co.Runtime, "specify test runtime")
	}

	return cmd
}

// Add flags
func NewCheckOptins() *common.CheckOptions {
	co := &common.CheckOptions{}
	co.Runtime = common.DefaultRuntime
	co.Domain = "www.github.com"
	co.Timeout = 1
	return co
}

// Start to check data
func (co *CheckObject) ExecuteCheck(use string, ob *common.CheckOptions) {
	err := fmt.Errorf("")

	if ob.Config == "" {
		ob.Config = common.EdgecoreConfigPath
	}

	switch use {
	case common.ArgCheckAll:
		err = CheckAll(ob)
	case common.ArgCheckCPU:
		err = CheckCPU()
	case common.ArgCheckMemory:
		err = CheckMemory()
	case common.ArgCheckDisk:
		err = CheckDisk()
	case common.ArgCheckDNS:
		err = CheckDNSSpecify(ob.Domain, ob.DNSIP)
	case common.ArgCheckNetwork:
		err = CheckNetWork(ob.IP, ob.Timeout, ob.CloudHubServer, ob.EdgecoreServer, ob.Config)
	case common.ArgCheckRuntime:
		err = CheckRuntime(ob.Runtime)
	case common.ArgCheckPID:
		err = CheckPid()
	}

	if err != nil {
		fmt.Println(err)
		util.PrintFail(use, common.StrCheck)
	} else {
		util.PrintSuccedd(use, common.StrCheck)
	}
}

func CheckAll(ob *common.CheckOptions) error {
	err := CheckCPU()
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

	err = CheckDNSSpecify(ob.Domain, ob.DNSIP)
	if err != nil {
		return err
	}

	err = CheckNetWork(ob.IP, ob.Timeout, ob.CloudHubServer, ob.EdgecoreServer, ob.Config)
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

func CheckCPU() error {
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return err
	}

	cpuNum, err := cpu.Counts(true)
	if err != nil {
		return err
	}

	fmt.Printf("CPU total: %v core, Allowed > %v core\n", cpuNum, common.AllowedValueCPU)
	fmt.Printf("CPU usage rate: %.2f, Allowed rate < %v\n", percent[0]/100, common.AllowedCurrentValueCPURate)

	if cpuNum < common.AllowedValueCPU || percent[0]/100 > common.AllowedCurrentValueCPURate {
		return errors.New("cpu check failed")
	}
	return nil
}

func CheckMemory() error {
	mem, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	fmt.Printf("Memory total: %.2f MB, Allowed > %v MB\n", float32(mem.Total)/common.MB, common.AllowedValueMemory/common.MB)
	fmt.Printf("Memory Free total: %.2f MB, Allowed > %v MB\n", float32(mem.Free)/common.MB, common.AllowedCurrentValueMem/common.MB)
	fmt.Printf("Memory usage rate: %.2f, Allowed rate < %v\n", mem.UsedPercent/100,
		common.AllowedCurrentValueMemRate)

	if mem.Total < common.AllowedValueMemory ||
		mem.Free < common.AllowedCurrentValueMem ||
		mem.UsedPercent/100 > common.AllowedCurrentValueMemRate {
		return errors.New("memory check failed")
	}

	return nil
}

func CheckDisk() error {
	parts, err := disk.Partitions(false)
	if err != nil {
		return err
	}

	diskInfo, err := disk.Usage(parts[0].Mountpoint)
	if err != nil {
		return err
	}

	fmt.Printf("Disk total: %.2f MB, Allowed > %v MB\n", float32(diskInfo.Total)/common.MB, common.AllowedValueDisk/common.MB)
	fmt.Printf("Disk Free total: %.2f MB, Allowed > %vMB\n", float32(diskInfo.Free)/common.MB, common.AllowedCurrentValueDisk/common.MB)
	fmt.Printf("Disk usage rate: %.2f, Allowed rate < %v\n", diskInfo.UsedPercent/100, common.AllowedCurrentValueDiskRate)

	if diskInfo.Total < common.AllowedValueDisk ||
		diskInfo.Free < common.AllowedCurrentValueDisk ||
		diskInfo.UsedPercent/100 > common.AllowedCurrentValueDiskRate {
		return errors.New("disk check failed")
	}

	return nil
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

func CheckDNSSpecify(domain string, dns string) error {
	if dns != "" {
		net.DefaultResolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Millisecond * time.Duration(4000),
				}
				return d.DialContext(ctx, "udp", fmt.Sprintf("%s:53", dns))
			},
		}
	}
	return CheckDNS(domain)
}

func CheckNetWork(IP string, timeout int, cloudhubServer string, edgecoreServer string, config string) error {
	if edgecoreServer == "" {
		edgecoreServer = "127.0.0.1:10350"
	}

	if config != "" {
		edgeconfig, err := util.ParseEdgecoreConfig(config)
		if err != nil {
			err = fmt.Errorf("parse Edgecore config failed")
		} else {
			if cloudhubServer == "" {
				cloudhubServer = edgeconfig.Modules.EdgeHub.WebSocket.Server
			}
		}
	}

	if IP == "" {
		result, err := util.ExecShellFilter(common.CmdGetDNSIP)
		if err != nil {
			return err
		}
		IP = result
	}
	if IP != "" {
		result, err := util.ExecShellFilter(fmt.Sprintf(common.CmdPing, IP, timeout))

		if err != nil {
			return err
		}
		if result != "0%" {
			return fmt.Errorf("ping %s timeout", IP)
		}
		fmt.Printf("ping %s success\n", IP)
	}

	if cloudhubServer != "" {
		err := CheckHTTP("https://" + cloudhubServer)
		if err != nil {
			return fmt.Errorf("check cloudhubServer %s failed, %v", cloudhubServer, err)
		}
		fmt.Printf("check cloudhubServer %s success\n", cloudhubServer)
	}

	if edgecoreServer != "" {
		err := CheckHTTP("http://" + edgecoreServer)
		if err != nil {
			return fmt.Errorf("check edgecoreServer %s failed, %v", edgecoreServer, edgecoreServer)
		}
		fmt.Printf("check edgecoreServer %s success\n", edgecoreServer)
	}

	return nil
}

func CheckHTTP(url string) error {
	cfg := &tls.Config{InsecureSkipVerify: false}
	httpTransport := &http.Transport{TLSClientConfig: cfg}
	// setup a http client
	httpClient := &http.Client{Transport: httpTransport, Timeout: time.Second * 3}
	response, err := httpClient.Get(url)
	if err != nil {
		if !strings.Contains(err.Error(), "x509") {
			return fmt.Errorf(" connect fail: %s", err.Error())
		}
	} else {
		defer response.Body.Close()
	}
	return nil
}

func CheckRuntime(runtime string) error {
	if runtime == common.DefaultRuntime {
		result, err := util.ExecShellFilter(common.CmdGetStatusDocker)
		if err != nil {
			return err
		}
		if result != "active" {
			return fmt.Errorf("docker is not running: %s", result)
		}
		fmt.Printf("docker is running\n")
		return nil
	}
	return fmt.Errorf("now only support docker: %s", runtime)
	// TODO
}

func CheckPid() error {
	rMax, err := util.ExecShellFilter(common.CmdGetMaxProcessNum)
	if err != nil {
		return err
	}
	r, err := util.ExecShellFilter(common.CmdGetProcessNum)
	if err != nil {
		return err
	}
	vMax, err := strconv.ParseFloat(rMax, 32)
	v, err := strconv.ParseFloat(r, 32)
	rate := (1 - v/vMax)
	if rate > common.AllowedValuePIDRate {
		fmt.Printf("Maximum PIDs: %s; Running processes: %s\n", rMax, r)
		return nil
	}
	return fmt.Errorf("Maximum PIDs: %s; Running processes: %s", rMax, r)
}
