package cast

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestCast(t *testing.T) {

	var configvalue interface{}
	var err error

	t.Log("Test core/cast/cast.go")
	assert.NotEqual(t, nil, NewValue("testkey", nil))

	t.Log("converting the data into int type by ToIntX methods and verifying")
	configvalue, err = NewValue(10, errors.New("error")).ToInt64()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(10, nil).ToInt64()
	assert.Equal(t, int64(10), configvalue)
	assert.Equal(t, nil, err)

	configvalue, err = NewValue(10, errors.New("error")).ToInt32()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(10, nil).ToInt32()
	assert.Equal(t, int32(10), configvalue)
	assert.Equal(t, nil, err)

	configvalue, err = NewValue(10, errors.New("error")).ToInt16()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(10, nil).ToInt16()
	assert.Equal(t, int16(10), configvalue)
	assert.Equal(t, nil, err)

	configvalue, err = NewValue(10, errors.New("error")).ToInt8()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(10, nil).ToInt8()
	assert.Equal(t, int8(10), configvalue)
	assert.Equal(t, nil, err)

	configvalue, err = NewValue(10, errors.New("error")).ToInt()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(10, nil).ToInt()
	assert.Equal(t, int(10), configvalue)
	assert.Equal(t, 10, configvalue)
	assert.Equal(t, nil, err)

	configvalue, err = NewValue(10, errors.New("error")).ToUint()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(10, nil).ToUint()
	assert.Equal(t, uint(10), configvalue)
	assert.Equal(t, nil, err)

	configvalue, err = NewValue(10, errors.New("error")).ToUint64()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(10, nil).ToUint64()
	assert.Equal(t, uint64(10), configvalue)
	assert.Equal(t, nil, err)

	configvalue, err = NewValue(10, errors.New("error")).ToUint32()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(10, nil).ToUint32()
	assert.Equal(t, uint32(10), configvalue)
	assert.Equal(t, nil, err)

	configvalue, err = NewValue(10, errors.New("error")).ToUint16()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(10, nil).ToUint16()
	assert.Equal(t, uint16(10), configvalue)
	assert.Equal(t, nil, err)

	configvalue, err = NewValue(10, errors.New("error")).ToUint8()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(10, nil).ToUint8()
	assert.Equal(t, uint8(10), configvalue)
	assert.Equal(t, nil, err)

	t.Log("verifying ToString method")
	configvalue, err = NewValue("hello", errors.New("error")).ToString()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue("hello", nil).ToString()
	assert.Equal(t, "hello", configvalue)
	assert.Equal(t, nil, err)
	configvalue, err = NewValue(10, nil).ToString()
	assert.Equal(t, "10", configvalue)
	assert.Equal(t, nil, err)
	configvalue, err = NewValue(true, nil).ToString()
	assert.Equal(t, "true", configvalue)
	assert.Equal(t, nil, err)

	t.Log("verifying ToStringMapStringSlice method in different scenarios")
	testmap := map[string]string{"testkey1": "test1", "testkey2": "test2"}
	expectedmap := make(map[string][]string)
	configvalue, err = NewValue(testmap, errors.New("error")).ToStringMapStringSlice()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(testmap, nil).ToStringMapStringSlice()
	assert.Equal(t, nil, err)
	if reflect.TypeOf(configvalue) != reflect.TypeOf(expectedmap) {
		t.Error("Fileed to get the data in expected(map[string][]string) format")
	}

	t.Log("verifying ToStringMapBool method")
	testmap2 := map[string]interface{}{"testkey1": "test1", "testkey2": "test2"}
	expectedmap2 := make(map[string]bool)
	configvalue, err = NewValue(testmap2, errors.New("error")).ToStringMapBool()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(testmap2, nil).ToStringMapBool()
	assert.Equal(t, nil, err)
	if reflect.TypeOf(configvalue) != reflect.TypeOf(expectedmap2) {
		t.Error("Fileed to get the data in expected(map[string]bool) format")
	}

	t.Log("verifying ToStringMap method")
	testmap3 := map[interface{}]interface{}{"testkey1": "test1", "testkey2": "test2"}
	expectedmap3 := make(map[string]interface{})
	configvalue, err = NewValue(testmap3, errors.New("error")).ToStringMap()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(testmap3, nil).ToStringMap()
	assert.Equal(t, nil, err)
	if reflect.TypeOf(configvalue) != reflect.TypeOf(expectedmap3) {
		t.Error("Fileed to get the data in expected(map[string]interface{}) format")
	}

	t.Log("verifying ToSlice method")
	configvalue, err = NewValue("hello", errors.New("error")).ToSlice()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(testmap3, nil).ToSlice()
	assert.NotEqual(t, nil, configvalue)
	assert.NotEqual(t, nil, err)

	t.Log("verifying ToBoolSlice method")
	testarray := []int{1, 2, 3}
	configvalue, err = NewValue(testarray, errors.New("error")).ToBoolSlice()
	assert.Equal(t, errors.New("error"), err)
	configvalue2, err := NewValue(testarray, nil).ToBoolSlice()
	assert.Equal(t, nil, err)
	if configvalue2[0] != true || configvalue2[1] != true || configvalue2[2] != true {
		t.Error("Fileed to get the data in expected([]bool) format")
	}

	t.Log("verifying ToStringSlice method")
	teststring := "hello"
	var expectedstring []string
	configvalue, err = NewValue(teststring, errors.New("error")).ToStringSlice()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(teststring, nil).ToStringSlice()
	assert.Equal(t, nil, err)
	if reflect.TypeOf(configvalue) != reflect.TypeOf(expectedstring) {
		t.Error("Fileed to get the data in expected([]string) format")
	}

	t.Log("verifying ToIntSlice method")
	teststring = "hello"
	configvalue, err = NewValue(teststring, errors.New("error")).ToIntSlice()
	assert.Equal(t, errors.New("error"), err)
	configvalue, err = NewValue(teststring, nil).ToIntSlice()
	assert.NotEqual(t, nil, configvalue)
	assert.NotEqual(t, nil, err)

	t.Log("verifying ToBool method in different scenarios")
	var boolkey = false
	configvalue, err = NewValue(boolkey, nil).ToBool()
	assert.Equal(t, nil, err)
	assert.Equal(t, false, configvalue)
	configvalue, err = NewValue(10, nil).ToBool()
	assert.Equal(t, nil, err)
	assert.Equal(t, true, configvalue)
	configvalue, err = NewValue(0, nil).ToBool()
	assert.Equal(t, nil, err)
	assert.Equal(t, false, configvalue)
	configvalue, err = NewValue("true", nil).ToBool()
	assert.Equal(t, nil, err)
	assert.Equal(t, true, configvalue)
	configvalue, err = NewValue("f", nil).ToBool()
	assert.Equal(t, nil, err)
	assert.Equal(t, false, configvalue)
	configvalue, err = NewValue("improperstring", nil).ToBool()
	assert.Equal(t, nil, err)
	assert.Equal(t, false, configvalue)
	configvalue, err = NewValue(testmap3, nil).ToBool()
	assert.NotEqual(t, nil, err)
	assert.Equal(t, false, configvalue)

	t.Log("converting the data into float type by ToFloat64 method and verifying")
	configvalue, err = NewValue(float64(10), nil).ToFloat64()
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(10), configvalue)
	configvalue, err = NewValue(10, nil).ToFloat64()
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(10), configvalue)
	configvalue, err = NewValue(int64(10), nil).ToFloat64()
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(10), configvalue)
	configvalue, err = NewValue(int32(10), nil).ToFloat64()
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(10), configvalue)
	configvalue, err = NewValue(int16(10), nil).ToFloat64()
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(10), configvalue)
	configvalue, err = NewValue(int8(10), nil).ToFloat64()
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(10), configvalue)
	configvalue, err = NewValue(uint(10), nil).ToFloat64()
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(10), configvalue)
	configvalue, err = NewValue(uint64(10), nil).ToFloat64()
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(10), configvalue)
	configvalue, err = NewValue(uint32(10), nil).ToFloat64()
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(10), configvalue)
	configvalue, err = NewValue(uint16(10), nil).ToFloat64()
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(10), configvalue)
	configvalue, err = NewValue(uint8(10), nil).ToFloat64()
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(10), configvalue)
	configvalue, err = NewValue(nil, nil).ToFloat64()
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(0), configvalue)
	configvalue, err = NewValue("10", nil).ToFloat64()
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(10), configvalue)
	configvalue, err = NewValue("hello", nil).ToFloat64()
	assert.NotEqual(t, nil, err)
	assert.Equal(t, float64(0), configvalue)
	configvalue, err = NewValue(testmap3, nil).ToFloat64()
	assert.NotEqual(t, nil, err)
	assert.Equal(t, float64(0), configvalue)
}
