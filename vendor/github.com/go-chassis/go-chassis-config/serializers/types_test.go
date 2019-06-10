package serializers

import (
	"github.com/go-chassis/go-chassis-config/serializers/json"
	"testing"
)

type Test struct {
	Team string `json:"team"`
}

func Test_Encode1(t *testing.T) {
	t.Log("Testing serializer encoding function for valid serializer")
	availableSerializers = make(map[string]Serializer)
	availableSerializers[JsonEncoder] = json.JsonSerializer{}

	test := &Test{Team: "data"}
	data, _ := Encode(JsonEncoder, test)

	stringData := `{"team":"data"}`
	if string(data) != stringData {
		t.Error("error is encoding the data")
	}
}

func Test_Encode2(t *testing.T) {
	t.Log("Testing serializer encoding function for invalid serializer")
	availableSerializers = make(map[string]Serializer)
	availableSerializers[JsonEncoder] = json.JsonSerializer{}

	test := &Test{Team: "data"}
	_, err := Encode("Invalidserializer", test)
	if err == nil {
		t.Error("Encoder is encoding invalid type of serilizer format")
	}
}

func Test_Decode(t *testing.T) {
	t.Log("Testing serializer decode function")
	availableSerializers = make(map[string]Serializer)
	availableSerializers[JsonEncoder] = json.JsonSerializer{}
	test := &Test{Team: "data"}

	data, _ := Encode(JsonEncoder, test)
	err := Decode(JsonEncoder, data, test)

	if err != nil {
		t.Error("error in decoding data")
	}
}
