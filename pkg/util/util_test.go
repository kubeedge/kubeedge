package util

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"
	"testing"

	"github.com/kubeedge/kubeedge/common/constants"
)

func TestValidateNodeIP(t *testing.T) {
	hostnameOverride := GetHostname()
	localIP, _ := GetLocalIP(hostnameOverride)

	cases := []struct {
		name     string
		ip       net.IP
		expected error
	}{
		{
			name:     "case1",
			ip:       nil,
			expected: fmt.Errorf("nodeIP must be a valid IP address"),
		},
		{
			name:     "case2",
			ip:       net.IPv4(127, 0, 0, 1),
			expected: fmt.Errorf("nodeIP can't be loopback address"),
		},
		{
			name:     "case3",
			ip:       net.IPv4(239, 0, 0, 254),
			expected: fmt.Errorf("nodeIP can't be a multicast address"),
		},
		{
			name:     "case4",
			ip:       net.IPv4(169, 254, 0, 0),
			expected: fmt.Errorf("nodeIP can't be a link-local unicast address"),
		},
		{
			name:     "case5",
			ip:       net.IPv4(0, 0, 0, 0),
			expected: fmt.Errorf("nodeIP can't be an all zeros address"),
		},
		{
			name:     "case 6",
			ip:       net.ParseIP(localIP),
			expected: nil,
		},
		{
			name:     "case 7",
			ip:       net.IPv4(114, 114, 114, 114),
			expected: fmt.Errorf("node IP: %q not found in the host's network interfaces", "114.114.114.114"),
		},
	}
	for _, c := range cases {
		err := ValidateNodeIP(c.ip)
		if !reflect.DeepEqual(err, c.expected) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.expected, err)
		}
	}
}

func TestCommand(t *testing.T) {
	cases := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "case1",
			command:  "fake_command_test",
			expected: false,
		},
		{
			name:     "case2",
			command:  "ls",
			expected: true,
		},
	}
	for _, c := range cases {
		_, err := Command(c.command, nil)
		isSuccess := err == nil
		if isSuccess != c.expected {
			t.Errorf("%v: expected %v, but got %v", c.name, c.expected, isSuccess)
		}
	}
}

func TestGetCurPath(t *testing.T) {
	path := GetCurPath()
	if path == "" {
		t.Errorf("failed to get current path")
	}
}

func TestGetHostname(t *testing.T) {
	name := GetHostname()
	if name == "" {
		t.Errorf("get host name failed")
	}
}

func TestGetLocalIP(t *testing.T) {
	_, err := GetLocalIP(GetHostname())
	if err != nil {
		t.Errorf("get local ip failed")
	}
}

func TestGetPodSandboxImage(t *testing.T) {
	image := GetPodSandboxImage()
	if image != constants.DefaultPodSandboxImage {
		t.Errorf("get pod sandbox image failed, get %v, expected %v", image, constants.DefaultPodSandboxImage)
	}
}

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
