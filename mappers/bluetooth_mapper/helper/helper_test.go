package helper_test

import (
	"testing"

	"github.com/kubeedge/kubeedge/mappers/bluetooth_mapper/helper"
)

func strToPtr(val string) *string {
	return &val
}

func TestMsgTwinIsSynced(t *testing.T) {
	cases := []struct {
		Name   string
		t1     *helper.MsgTwin
		Expect bool
	}{
		{
			Name:   "nil value",
			t1:     nil,
			Expect: false,
		},
		{
			Name: "same value",
			t1: &helper.MsgTwin{
				Expected: &helper.TwinValue{
					Value: strToPtr("val_1"),
				},
				Actual: &helper.TwinValue{
					Value: strToPtr("val_1"),
				},
			},
			Expect: true,
		},
		{
			Name:   "nil value on both actual and expected",
			t1:     &helper.MsgTwin{},
			Expect: true,
		},
		{
			Name: "different values",
			t1: &helper.MsgTwin{
				Expected: &helper.TwinValue{
					Value: strToPtr("val_1"),
				},
			},
			Expect: false,
		},
		{
			Name: "one nil",
			t1: &helper.MsgTwin{
				Actual: &helper.TwinValue{
					Value: strToPtr("val_1"),
				},
			},
			Expect: false,
		},
		{
			Name: "other nil",
			t1: &helper.MsgTwin{
				Actual: &helper.TwinValue{
					Value: strToPtr("val_1"),
				},
				Expected: &helper.TwinValue{
					Value: strToPtr("val_2"),
				},
			},
			Expect: false,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			val := c.t1.IsSynced()
			if val != c.Expect {
				t.Errorf("Input %+v: Expected %t, got %t", c.t1, c.Expect, val)
			}
		})
	}
}
