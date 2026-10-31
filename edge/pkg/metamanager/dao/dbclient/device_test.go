/*
Copyright 2026 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dbclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

const (
	testDevice1     = "device-1"
	testDevice2     = "device-2"
	testTemperature = "temperature"
	testHumidity    = "humidity"
	testPressure    = "pressure"
	testPower       = "power"
	testSpeed       = "speed"

	// Columns and values of the device tables used by the tests.
	colState       = "state"
	colDescription = "description"
	colExpected    = "expected"
	colNoSuch      = "no_such_column"
	colValue       = "value"
	updatedValue   = "updated"
	stateOnline    = "online"
)

// deviceTables are the tables the device service works on.
var deviceTables = []interface{}{&models.Device{}, &models.DeviceAttr{}, &models.DeviceTwin{}}

func TestDeviceService(t *testing.T) {
	t.Run("uses the edgecore database", func(t *testing.T) {
		s := NewDeviceService()
		require.NoError(t, s.SaveDevice(&models.Device{ID: "test-device"}))
		t.Cleanup(func() {
			require.NoError(t, s.DeleteDeviceByID("test-device"))
		})

		devices, err := s.QueryDevice("id", "test-device")
		require.NoError(t, err)
		assert.Len(t, devices, 1)
	})

	t.Run("save, query, update and delete devices", func(t *testing.T) {
		s := newTestDeviceService(t, deviceTables...)
		require.NoError(t, s.SaveDevice(&models.Device{ID: testDevice1, Name: "sensor-1", State: stateOnline}))
		require.NoError(t, s.SaveDevice(&models.Device{ID: testDevice2, Name: "sensor-2", State: stateOnline}))
		// The device ID is the primary key.
		assert.Error(t, s.SaveDevice(&models.Device{ID: testDevice1, Name: "duplicate"}))

		devices, err := s.QueryDevice(colState, stateOnline)
		require.NoError(t, err)
		assert.Len(t, devices, 2)

		require.NoError(t, s.UpdateDeviceField(testDevice1, colState, "offline"))
		require.NoError(t, s.UpdateDeviceFields(testDevice2, map[string]interface{}{
			colState:      "unknown",
			"last_online": "2026-10-27T00:00:00Z",
		}))
		require.NoError(t, s.UpdateDeviceMulti([]models.DeviceUpdate{
			{DeviceID: testDevice1, Cols: map[string]interface{}{colDescription: updatedValue}},
		}))
		assert.Equal(t, models.Device{ID: testDevice1, Name: "sensor-1", State: "offline", Description: updatedValue},
			queryOneDevice(t, s, testDevice1))
		assert.Equal(t, models.Device{ID: testDevice2, Name: "sensor-2", State: "unknown", LastOnline: "2026-10-27T00:00:00Z"},
			queryOneDevice(t, s, testDevice2))

		require.NoError(t, s.DeleteDeviceByID(testDevice1))
		devices, err = s.QueryDeviceAll()
		require.NoError(t, err)
		require.Len(t, devices, 1)
		assert.Equal(t, testDevice2, devices[0].ID)
	})

	t.Run("multiple updates stop at the first failure", func(t *testing.T) {
		s := newTestDeviceService(t, deviceTables...)
		require.NoError(t, s.SaveDevice(&models.Device{ID: testDevice1, State: stateOnline}))

		err := s.UpdateDeviceMulti([]models.DeviceUpdate{
			{DeviceID: testDevice1, Cols: map[string]interface{}{colNoSuch: colValue}},
			{DeviceID: testDevice1, Cols: map[string]interface{}{colState: "offline"}},
		})
		assert.ErrorContains(t, err, errNoSuchColumn)
		assert.Equal(t, stateOnline, queryOneDevice(t, s, testDevice1).State)
	})

	t.Run("database errors", func(t *testing.T) {
		// The database has no device tables, so every query fails.
		s := newTestDeviceService(t)

		assert.ErrorContains(t, s.SaveDevice(&models.Device{ID: testDevice1}), errNoSuchTable)
		assert.ErrorContains(t, s.DeleteDeviceByID(testDevice1), errNoSuchTable)
		assert.ErrorContains(t, s.UpdateDeviceField(testDevice1, colState, stateOnline), errNoSuchTable)
		assert.ErrorContains(t, s.UpdateDeviceMulti([]models.DeviceUpdate{
			{DeviceID: testDevice1, Cols: map[string]interface{}{colState: stateOnline}},
		}), errNoSuchTable)
		devices, err := s.QueryDevice("id", testDevice1)
		assert.ErrorContains(t, err, errNoSuchTable)
		assert.Nil(t, devices)
		devices, err = s.QueryDeviceAll()
		assert.ErrorContains(t, err, errNoSuchTable)
		assert.Nil(t, devices)
	})
}

func TestDeviceServiceDeviceTrans(t *testing.T) {
	devices := []models.Device{{ID: testDevice1}, {ID: testDevice2}}
	attrs := []models.DeviceAttr{
		{DeviceID: testDevice1, Name: testTemperature},
		{DeviceID: testDevice2, Name: testTemperature},
	}
	twins := []models.DeviceTwin{
		{DeviceID: testDevice1, Name: testPower},
		{DeviceID: testDevice2, Name: testPower},
	}

	t.Run("add and delete devices with their attributes and twins", func(t *testing.T) {
		s := newTestDeviceService(t, deviceTables...)
		require.NoError(t, s.AddDeviceTrans(devices, attrs, twins))
		assertDeviceRows(t, s, testDevice1, 1, 1, 1)
		assertDeviceRows(t, s, testDevice2, 1, 1, 1)

		require.NoError(t, s.DeleteDeviceTrans([]string{testDevice1}))
		assertDeviceRows(t, s, testDevice1, 0, 0, 0)
		assertDeviceRows(t, s, testDevice2, 1, 1, 1)
	})

	// Each case leaves out a table so that one step of the transaction fails.
	failureCases := []struct {
		name   string
		tables []interface{}
	}{
		{name: "device table is missing"},
		{name: "attribute table is missing", tables: []interface{}{&models.Device{}}},
		{name: "twin table is missing", tables: []interface{}{&models.Device{}, &models.DeviceAttr{}}},
	}

	for _, c := range failureCases {
		t.Run("adding devices is rolled back when "+c.name, func(t *testing.T) {
			s := newTestDeviceService(t, c.tables...)

			err := s.AddDeviceTrans(devices, attrs, twins)
			assert.ErrorContains(t, err, errNoSuchTable)
			// The rows added before the failure are not kept.
			for _, table := range c.tables {
				assert.Zero(t, countRows(t, s.db, table), "%T", table)
			}
		})
	}

	for _, c := range failureCases {
		t.Run("deleting devices is rolled back when "+c.name, func(t *testing.T) {
			s := newTestDeviceService(t, c.tables...)
			rows := []interface{}{
				&models.Device{ID: testDevice1},
				&models.DeviceAttr{DeviceID: testDevice1, Name: testTemperature},
			}
			for i := range c.tables {
				require.NoError(t, s.db.Create(rows[i]).Error)
			}

			err := s.DeleteDeviceTrans([]string{testDevice1})
			assert.ErrorContains(t, err, errNoSuchTable)
			// The rows deleted before the failure are restored.
			for _, table := range c.tables {
				assert.Equal(t, int64(1), countRows(t, s.db, table), "%T", table)
			}
		})
	}
}

func TestDeviceServiceAttrs(t *testing.T) {
	t.Run("save, query, update and delete attributes", func(t *testing.T) {
		s := newTestDeviceService(t, deviceTables...)
		require.NoError(t, s.SaveDeviceAttr(&models.DeviceAttr{DeviceID: testDevice1, Name: testTemperature, Value: "20"}))
		require.NoError(t, s.SaveDeviceAttr(&models.DeviceAttr{DeviceID: testDevice1, Name: testHumidity, Value: "40"}))
		require.NoError(t, s.SaveDeviceAttr(&models.DeviceAttr{DeviceID: testDevice2, Name: testTemperature, Value: "25"}))

		require.NoError(t, s.UpdateDeviceAttrField(testDevice1, testTemperature, colValue, "21"))
		require.NoError(t, s.UpdateDeviceAttrFields(testDevice1, testHumidity, map[string]interface{}{
			colValue:   "41",
			"optional": true,
		}))
		require.NoError(t, s.UpdateDeviceAttrMulti([]models.DeviceAttrUpdate{
			{DeviceID: testDevice2, Name: testTemperature, Cols: map[string]interface{}{colDescription: updatedValue}},
		}))

		device1 := attrsByName(t, s, testDevice1)
		require.Len(t, device1, 2)
		assert.Equal(t, "21", device1[testTemperature].Value)
		assert.Equal(t, "41", device1[testHumidity].Value)
		assert.True(t, device1[testHumidity].Optional)
		// An update only changes the attribute with both the device ID and the name.
		device2 := attrsByName(t, s, testDevice2)
		assert.Equal(t, "25", device2[testTemperature].Value)
		assert.Equal(t, updatedValue, device2[testTemperature].Description)

		require.NoError(t, s.DeleteDeviceAttr(testDevice1, testTemperature))
		assert.Len(t, attrsByName(t, s, testDevice1), 1)
		assert.Contains(t, attrsByName(t, s, testDevice2), testTemperature)
	})

	t.Run("multiple updates stop at the first failure", func(t *testing.T) {
		s := newTestDeviceService(t, deviceTables...)
		require.NoError(t, s.SaveDeviceAttr(&models.DeviceAttr{DeviceID: testDevice1, Name: testTemperature, Value: "20"}))

		err := s.UpdateDeviceAttrMulti([]models.DeviceAttrUpdate{
			{DeviceID: testDevice1, Name: testTemperature, Cols: map[string]interface{}{colNoSuch: colValue}},
			{DeviceID: testDevice1, Name: testTemperature, Cols: map[string]interface{}{colValue: "21"}},
		})
		assert.ErrorContains(t, err, errNoSuchColumn)
		assert.Equal(t, "20", attrsByName(t, s, testDevice1)[testTemperature].Value)
	})

	t.Run("add, delete and update attributes in one transaction", func(t *testing.T) {
		s := newTestDeviceService(t, deviceTables...)
		require.NoError(t, s.SaveDeviceAttr(&models.DeviceAttr{DeviceID: testDevice1, Name: testTemperature, Value: "20"}))
		require.NoError(t, s.SaveDeviceAttr(&models.DeviceAttr{DeviceID: testDevice1, Name: testHumidity, Value: "40"}))

		err := s.DeviceAttrTrans(
			[]models.DeviceAttr{{DeviceID: testDevice1, Name: testPressure, Value: "1"}},
			[]models.DeviceDelete{{DeviceID: testDevice1, Name: testTemperature}},
			[]models.DeviceAttrUpdate{{DeviceID: testDevice1, Name: testHumidity, Cols: map[string]interface{}{colValue: "42"}}},
		)
		require.NoError(t, err)
		attrs := attrsByName(t, s, testDevice1)
		require.Len(t, attrs, 2)
		assert.Equal(t, "1", attrs[testPressure].Value)
		assert.Equal(t, "42", attrs[testHumidity].Value)
	})

	t.Run("attribute transaction is rolled back on failure", func(t *testing.T) {
		s := newTestDeviceService(t, deviceTables...)
		require.NoError(t, s.SaveDeviceAttr(&models.DeviceAttr{DeviceID: testDevice1, Name: testTemperature, Value: "20"}))

		err := s.DeviceAttrTrans(
			[]models.DeviceAttr{{DeviceID: testDevice1, Name: testPressure}},
			[]models.DeviceDelete{{DeviceID: testDevice1, Name: testTemperature}},
			[]models.DeviceAttrUpdate{{DeviceID: testDevice1, Name: testTemperature, Cols: map[string]interface{}{colNoSuch: colValue}}},
		)
		assert.ErrorContains(t, err, errNoSuchColumn)
		// The add and the delete before the failed update are rolled back.
		attrs := attrsByName(t, s, testDevice1)
		require.Len(t, attrs, 1)
		assert.Equal(t, "20", attrs[testTemperature].Value)
	})

	t.Run("database errors", func(t *testing.T) {
		// The database has no device tables, so every query fails.
		s := newTestDeviceService(t)
		update := models.DeviceAttrUpdate{DeviceID: testDevice1, Name: testTemperature, Cols: map[string]interface{}{colValue: "21"}}

		assert.ErrorContains(t, s.SaveDeviceAttr(&models.DeviceAttr{DeviceID: testDevice1, Name: testTemperature}), errNoSuchTable)
		assert.ErrorContains(t, s.DeleteDeviceAttr(testDevice1, testTemperature), errNoSuchTable)
		assert.ErrorContains(t, s.UpdateDeviceAttrField(testDevice1, testTemperature, colValue, "21"), errNoSuchTable)
		assert.ErrorContains(t, s.UpdateDeviceAttrMulti([]models.DeviceAttrUpdate{update}), errNoSuchTable)
		attrs, err := s.QueryDeviceAttr("deviceid", testDevice1)
		assert.ErrorContains(t, err, errNoSuchTable)
		assert.Nil(t, attrs)

		assert.ErrorContains(t, s.DeviceAttrTrans(
			[]models.DeviceAttr{{DeviceID: testDevice1, Name: testTemperature}}, nil, nil), errNoSuchTable)
		assert.ErrorContains(t, s.DeviceAttrTrans(
			nil, []models.DeviceDelete{{DeviceID: testDevice1, Name: testTemperature}}, nil), errNoSuchTable)
		assert.ErrorContains(t, s.DeviceAttrTrans(
			nil, nil, []models.DeviceAttrUpdate{update}), errNoSuchTable)
	})
}

func TestDeviceServiceTwins(t *testing.T) {
	t.Run("save, query, update and delete twins", func(t *testing.T) {
		s := newTestDeviceService(t, deviceTables...)
		require.NoError(t, s.SaveDeviceTwin(&models.DeviceTwin{DeviceID: testDevice1, Name: testPower, Expected: "on"}))
		require.NoError(t, s.SaveDeviceTwin(&models.DeviceTwin{DeviceID: testDevice1, Name: testSpeed, Expected: "1"}))
		require.NoError(t, s.SaveDeviceTwin(&models.DeviceTwin{DeviceID: testDevice2, Name: testPower, Expected: "off"}))

		require.NoError(t, s.UpdateDeviceTwinField(testDevice1, testPower, "actual", "on"))
		require.NoError(t, s.UpdateDeviceTwinFields(testDevice1, testSpeed, map[string]interface{}{
			colExpected: "2",
			"optional":  true,
		}))
		require.NoError(t, s.UpdateDeviceTwinMulti([]models.DeviceTwinUpdate{
			{DeviceID: testDevice2, Name: testPower, Cols: map[string]interface{}{colDescription: updatedValue}},
		}))

		device1 := twinsByName(t, s, testDevice1)
		require.Len(t, device1, 2)
		assert.Equal(t, "on", device1[testPower].Expected)
		assert.Equal(t, "on", device1[testPower].Actual)
		assert.Equal(t, "2", device1[testSpeed].Expected)
		assert.True(t, device1[testSpeed].Optional)
		// An update only changes the twin with both the device ID and the name.
		device2 := twinsByName(t, s, testDevice2)
		assert.Equal(t, "off", device2[testPower].Expected)
		assert.Empty(t, device2[testPower].Actual)
		assert.Equal(t, updatedValue, device2[testPower].Description)

		require.NoError(t, s.DeleteDeviceTwin(testDevice1, testPower))
		assert.Len(t, twinsByName(t, s, testDevice1), 1)
		assert.Contains(t, twinsByName(t, s, testDevice2), testPower)
	})

	t.Run("multiple updates stop at the first failure", func(t *testing.T) {
		s := newTestDeviceService(t, deviceTables...)
		require.NoError(t, s.SaveDeviceTwin(&models.DeviceTwin{DeviceID: testDevice1, Name: testPower, Expected: "on"}))

		err := s.UpdateDeviceTwinMulti([]models.DeviceTwinUpdate{
			{DeviceID: testDevice1, Name: testPower, Cols: map[string]interface{}{colNoSuch: colValue}},
			{DeviceID: testDevice1, Name: testPower, Cols: map[string]interface{}{colExpected: "off"}},
		})
		assert.ErrorContains(t, err, errNoSuchColumn)
		assert.Equal(t, "on", twinsByName(t, s, testDevice1)[testPower].Expected)
	})

	t.Run("add, delete and update twins in one transaction", func(t *testing.T) {
		s := newTestDeviceService(t, deviceTables...)
		require.NoError(t, s.SaveDeviceTwin(&models.DeviceTwin{DeviceID: testDevice1, Name: testPower, Expected: "on"}))
		require.NoError(t, s.SaveDeviceTwin(&models.DeviceTwin{DeviceID: testDevice1, Name: testSpeed, Expected: "1"}))

		err := s.DeviceTwinTrans(
			[]models.DeviceTwin{{DeviceID: testDevice1, Name: testPressure, Expected: "1"}},
			[]models.DeviceDelete{{DeviceID: testDevice1, Name: testPower}},
			[]models.DeviceTwinUpdate{{DeviceID: testDevice1, Name: testSpeed, Cols: map[string]interface{}{colExpected: "2"}}},
		)
		require.NoError(t, err)
		twins := twinsByName(t, s, testDevice1)
		require.Len(t, twins, 2)
		assert.Equal(t, "1", twins[testPressure].Expected)
		assert.Equal(t, "2", twins[testSpeed].Expected)
	})

	t.Run("twin transaction is rolled back on failure", func(t *testing.T) {
		s := newTestDeviceService(t, deviceTables...)
		require.NoError(t, s.SaveDeviceTwin(&models.DeviceTwin{DeviceID: testDevice1, Name: testPower, Expected: "on"}))

		err := s.DeviceTwinTrans(
			[]models.DeviceTwin{{DeviceID: testDevice1, Name: testPressure}},
			[]models.DeviceDelete{{DeviceID: testDevice1, Name: testPower}},
			[]models.DeviceTwinUpdate{{DeviceID: testDevice1, Name: testPower, Cols: map[string]interface{}{colNoSuch: colValue}}},
		)
		assert.ErrorContains(t, err, errNoSuchColumn)
		// The add and the delete before the failed update are rolled back.
		twins := twinsByName(t, s, testDevice1)
		require.Len(t, twins, 1)
		assert.Equal(t, "on", twins[testPower].Expected)
	})

	t.Run("database errors", func(t *testing.T) {
		// The database has no device tables, so every query fails.
		s := newTestDeviceService(t)
		update := models.DeviceTwinUpdate{DeviceID: testDevice1, Name: testPower, Cols: map[string]interface{}{colExpected: "on"}}

		assert.ErrorContains(t, s.SaveDeviceTwin(&models.DeviceTwin{DeviceID: testDevice1, Name: testPower}), errNoSuchTable)
		assert.ErrorContains(t, s.DeleteDeviceTwin(testDevice1, testPower), errNoSuchTable)
		assert.ErrorContains(t, s.UpdateDeviceTwinField(testDevice1, testPower, colExpected, "on"), errNoSuchTable)
		assert.ErrorContains(t, s.UpdateDeviceTwinMulti([]models.DeviceTwinUpdate{update}), errNoSuchTable)
		twins, err := s.QueryDeviceTwin("deviceid", testDevice1)
		assert.ErrorContains(t, err, errNoSuchTable)
		assert.Nil(t, twins)

		assert.ErrorContains(t, s.DeviceTwinTrans(
			[]models.DeviceTwin{{DeviceID: testDevice1, Name: testPower}}, nil, nil), errNoSuchTable)
		assert.ErrorContains(t, s.DeviceTwinTrans(
			nil, []models.DeviceDelete{{DeviceID: testDevice1, Name: testPower}}, nil), errNoSuchTable)
		assert.ErrorContains(t, s.DeviceTwinTrans(
			nil, nil, []models.DeviceTwinUpdate{update}), errNoSuchTable)
	})
}

func newTestDeviceService(t *testing.T, tables ...interface{}) *DeviceService {
	return &DeviceService{db: newTestDB(t, tables...)}
}

func queryOneDevice(t *testing.T, s *DeviceService, deviceID string) models.Device {
	t.Helper()
	devices, err := s.QueryDevice("id", deviceID)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	return devices[0]
}

func attrsByName(t *testing.T, s *DeviceService, deviceID string) map[string]models.DeviceAttr {
	t.Helper()
	attrs, err := s.QueryDeviceAttr("deviceid", deviceID)
	require.NoError(t, err)
	byName := make(map[string]models.DeviceAttr, len(*attrs))
	for _, attr := range *attrs {
		byName[attr.Name] = attr
	}
	return byName
}

func twinsByName(t *testing.T, s *DeviceService, deviceID string) map[string]models.DeviceTwin {
	t.Helper()
	twins, err := s.QueryDeviceTwin("deviceid", deviceID)
	require.NoError(t, err)
	byName := make(map[string]models.DeviceTwin, len(*twins))
	for _, twin := range *twins {
		byName[twin.Name] = twin
	}
	return byName
}

// assertDeviceRows checks how many device, attribute and twin rows are stored for the device.
func assertDeviceRows(t *testing.T, s *DeviceService, deviceID string, wantDevices, wantAttrs, wantTwins int) {
	t.Helper()
	devices, err := s.QueryDevice("id", deviceID)
	require.NoError(t, err)
	assert.Len(t, devices, wantDevices, "devices of %s", deviceID)
	assert.Len(t, attrsByName(t, s, deviceID), wantAttrs, "attributes of %s", deviceID)
	assert.Len(t, twinsByName(t, s, deviceID), wantTwins, "twins of %s", deviceID)
}

func countRows(t *testing.T, db *gorm.DB, table interface{}) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(table).Count(&count).Error)
	return count
}
