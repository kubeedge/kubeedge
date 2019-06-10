package json

import (
	"testing"
)

type Test struct {
	Team string `json:"team"`
}

func TestEncode(t *testing.T) {

	testSerilizer := &JsonSerializer{}
	test := &Test{Team: "data"}
	_, err := testSerilizer.Encode(test)

	if err != nil {
		t.Error("error in encoding")
	}
}
func TestDecode(t *testing.T) {

	testSerilizer := &JsonSerializer{}
	test := &Test{Team: "data"}

	data, _ := testSerilizer.Encode(test)
	err := testSerilizer.Decode(data, test)

	if err != nil {
		t.Error("error in decoding")
	}
}
