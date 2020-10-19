package configmap

import (
	"fmt"

	"github.com/kubeedge/kubeedge/mappers/common"
	"github.com/kubeedge/kubeedge/mappers/modbus/src/configmap"
)

func test_parse() {
	var dp common.DeviceProfile

	err := configmap.Parse("/home/wei/go/src/github.com/kubeedge/kubeedge/mappers/modbus/configmap/configmap_test.json", &dp)
	if err != nil {
		fmt.Print(err)
	} else {
		fmt.Printf("%+v\n", dp)
	}
}
