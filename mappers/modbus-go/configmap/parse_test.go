package configmap

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	mappercommon "github.com/kubeedge/kubeedge/mappers/common"
	. "github.com/kubeedge/kubeedge/mappers/modbus-go/globals"
)

func TestParse(t *testing.T) {
	var devices map[string]*ModbusDev
	var models map[string]mappercommon.DeviceModel
	var protocols map[string]mappercommon.Protocol

	devices = make(map[string]*ModbusDev)
	models = make(map[string]mappercommon.DeviceModel)
	protocols = make(map[string]mappercommon.Protocol)

	assert.Nil(t, Parse("./configmap_test.json", devices, models, protocols))
	for _, device := range devices {
		var pcc ModbusProtocolCommonConfig
		assert.Nil(t, json.Unmarshal([]byte(device.Instance.PProtocol.ProtocolCommonConfig), &pcc))
		assert.Equal(t, "RS485", pcc.CustomizedValues["serialType"])
	}
}

func TestParseNeg(t *testing.T) {
	var devices map[string]*ModbusDev
	var models map[string]mappercommon.DeviceModel
	var protocols map[string]mappercommon.Protocol

	devices = make(map[string]*ModbusDev)
	models = make(map[string]mappercommon.DeviceModel)
	protocols = make(map[string]mappercommon.Protocol)

	assert.NotNil(t, Parse("./configmap_negtest.json", devices, models, protocols))
}
