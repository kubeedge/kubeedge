package util

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestSpliceErrors(t *testing.T) {
	err1 := errors.New("this is error 1")
	err2 := errors.New("this is error 2")
	err3 := errors.New("this is error 3")

	const head = "[\n"
	var line1 = fmt.Sprintf("  %s\n", err1)
	var line2 = fmt.Sprintf("  %s\n", err2)
	var line3 = fmt.Sprintf("  %s\n", err3)
	const tail = "]\n"

	sliceOutput := SpliceErrors([]error{err1, err2, err3})
	if strings.Index(sliceOutput, head) != 0 ||
		strings.Index(sliceOutput, line1) != len(head) ||
		strings.Index(sliceOutput, line2) != len(head+line1) ||
		strings.Index(sliceOutput, line3) != len(head+line1+line2) ||
		strings.Index(sliceOutput, tail) != len(head+line1+line2+line3) {
		t.Error("the func format the multiple elements error slice unexpected")
		return
	}

	if SpliceErrors([]error{}) != "" || SpliceErrors(nil) != "" {
		t.Error("the func format the zero-length error slice unexpected")
		return
	}
}

func TestConcatStrings(t *testing.T) {
	cases := []struct {
		args   []string
		expect string
	}{
		{
			args:   []string{},
			expect: "",
		},
		{
			args:   nil,
			expect: "",
		},
		{
			args:   []string{"a", "", "b"},
			expect: "ab",
		},
	}
	var s string
	for _, c := range cases {
		s = ConcatStrings(c.args...)
		if s != c.expect {
			t.Errorf("the func return failed. expect: %s, actual: %s\n", c.expect, s)
			return
		}
	}
}
