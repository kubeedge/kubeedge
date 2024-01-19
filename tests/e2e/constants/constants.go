package constants

import "time"

const (
	Interval = 5 * time.Second
	Timeout  = 10 * time.Minute

	E2ELabelKey   = "kubeedge"
	E2ELabelValue = "e2e-test"

	NodeName = "edge-node"

	MakeModbusMapperProject   = "cd /home/runner/work/kubeedge/kubeedge/staging/src/github.com/kubeedge/mapper-framework;make generate modbus;"
	GetModbusExampleCode      = "cd /home/runner/work/kubeedge;git clone https://github.com/wbc6080/modbus.git"
	BuildModbusMapperProject  = "cp -r /home/runner/work/kubeedge/modbus/driver/*  /home/runner/work/kubeedge/kubeedge/staging/src/github.com/kubeedge/modbus/driver/ ;cp /home/runner/work/kubeedge/modbus/config.yaml  /home/runner/work/kubeedge/kubeedge/staging/src/github.com/kubeedge/modbus/"
	MakeModbusMapperImage     = "cd /home/runner/work/kubeedge/kubeedge/staging/src/github.com/kubeedge/modbus;docker build -t modbus-e2e-mapper:v1.0.0 ."
	CheckModbusMapperImage    = "docker images | grep modbus-e2e-mapper"
	DeleteModbusMapperImage   = "docker rmi modbus-e2e-mapper:v1.0.0"
	DeleteModbusMapperProject = "cd /home/runner/work/kubeedge/kubeedge/staging/src/github.com/kubeedge; rm -rf modbus/"
	DeleteModbusExampleCode   = "cd /home/runner/work/kubeedge;rm -rf modbus/"

	MakeModbusMapperContainer = "docker run -d -v /etc/kubeedge:/etc/kubeedge modbus-e2e-mapper:v1.0.0 ./main --v 4 --config-file config.yaml"
	GetModbusMapperContainer  = "docker ps | grep modbus-e2e-mapper:v1.0.0"
	DeleteMapperContainer     = "docker stop `docker ps |grep mapper |awk '{print $1}'`; docker rm `docker ps -a|grep mapper |awk '{print $1}'`"
)

var (
	// KubeEdgeE2ELabel labels resources created during e2e testing
	KubeEdgeE2ELabel = map[string]string{
		E2ELabelKey: E2ELabelValue,
	}
)
