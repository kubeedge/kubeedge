package common

import (
	"fmt"
	"testing"
)

func TestCloudInitUpdateBaseHasSets(t *testing.T) {
	key := "key"
	cases := []struct {
		sets []string
		want bool
	}{
		{sets: []string{""}, want: false},
		{sets: []string{"key"}, want: false},
		{sets: []string{"key="}, want: true},
		{sets: []string{"key=value"}, want: true},
		{sets: []string{"key=value=1"}, want: true},
		{sets: []string{"key2=value"}, want: false},
	}

	for _, c := range cases {
		obj := CloudInitUpdateBase{Sets: c.sets}
		t.Run(fmt.Sprint(c.sets), func(t *testing.T) {
			b := obj.HasSets(key)
			if b != c.want {
				t.Fatalf("failed to test HasSets, expected: %t, actual: %t", c.want, b)
			}
		})
	}
}

func TestCloudInitUpdateBaseGetValidSets(t *testing.T) {
	wantLen := 4
	sets := []string{
		"",
		"key1",
		"key2=",
		"key3=value",
		"key4=value=1",
		"key4=value=====",
	}
	obj := CloudInitUpdateBase{Sets: sets}
	res := obj.GetValidSets()
	if len(res) != wantLen {
		t.Fatalf("failed to test GetValidSets, expected sets len: %d,"+
			" actual sets len: %d", wantLen, len(res))
	}
}
