/*
Copyright 2025 The KubeEdge Authors.

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

package util

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	nodeutil "k8s.io/component-helpers/node/util"
)

func TestValidateNodeIP(t *testing.T) {
	assert := assert.New(t)

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
		assert.Equal(c.expected, err, c.name)
	}

	patches := gomonkey.ApplyFunc(net.InterfaceAddrs, func() ([]net.Addr, error) {
		return nil, fmt.Errorf("mock interface addrs error")
	})
	defer patches.Reset()

	err := ValidateNodeIP(net.IPv4(192, 168, 1, 1))
	assert.Error(err)
	assert.Contains(err.Error(), "mock interface addrs error")
}

func TestCommand(t *testing.T) {
	assert := assert.New(t)

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
		assert.Equal(c.expected, isSuccess, c.name)
	}

	output, err := Command("echo", []string{"-n", "hello\n"})
	assert.NoError(err)
	assert.Equal("hello", output)
}

func TestGetCurPath(t *testing.T) {
	assert := assert.New(t)

	path := GetCurPath()
	assert.NotEmpty(path)
}

func TestGetHostname(t *testing.T) {
	assert := assert.New(t)

	name := GetHostname()
	assert.NotEmpty(name)

	patches := gomonkey.ApplyFunc(nodeutil.GetHostname, func(hostnameOverride string) (string, error) {
		return "", fmt.Errorf("hostname error")
	})
	defer patches.Reset()

	name = GetHostname()
	assert.Equal("default-edge-node", name)
}

func TestGetLocalIP(t *testing.T) {
	assert := assert.New(t)

	ip, err := GetLocalIP(GetHostname())
	assert.NoError(err)
	assert.NotEmpty(ip)

	chooseHostPatch := gomonkey.ApplyFunc(utilnet.ChooseHostInterface, func() (net.IP, error) {
		return nil, fmt.Errorf("no interface found")
	})
	defer chooseHostPatch.Reset()

	patches := gomonkey.ApplyFunc(net.LookupIP, func(host string) ([]net.IP, error) {
		return []net.IP{
			net.ParseIP("fe80::1"),
			net.ParseIP("2001:db8::1"),
			net.ParseIP("127.0.0.1"),
			net.ParseIP("192.168.1.1"),
		}, nil
	})
	defer patches.Reset()

	validatePatch := gomonkey.ApplyFunc(ValidateNodeIP, func(ip net.IP) error {
		if ip.Equal(net.ParseIP("127.0.0.1")) || ip.Equal(net.ParseIP("fe80::1")) {
			return fmt.Errorf("invalid IP")
		}
		return nil
	})
	defer validatePatch.Reset()

	ip, err = GetLocalIP("test-host")
	assert.NoError(err)
	assert.Equal("192.168.1.1", ip)

	patches.Reset()
	patches = gomonkey.ApplyFunc(net.LookupIP, func(host string) ([]net.IP, error) {
		return []net.IP{
			net.ParseIP("fe80::1"),
			net.ParseIP("2001:db8::1"),
		}, nil
	})

	validatePatch.Reset()
	validatePatch = gomonkey.ApplyFunc(ValidateNodeIP, func(ip net.IP) error {
		if ip.Equal(net.ParseIP("fe80::1")) {
			return fmt.Errorf("invalid IP")
		}
		return nil
	})

	ip, err = GetLocalIP("test-host")
	assert.NoError(err)
	assert.Equal("2001:db8::1", ip)

	patches.Reset()
	validatePatch.Reset()

	lookupPatch := gomonkey.ApplyFunc(net.LookupIP, func(host string) ([]net.IP, error) {
		return []net.IP{}, nil
	})
	defer lookupPatch.Reset()
}

func TestSpliceErrors(t *testing.T) {
	assert := assert.New(t)

	err1 := errors.New("this is error 1")
	err2 := errors.New("this is error 2")
	err3 := errors.New("this is error 3")

	const head = "[\n"
	var line1 = fmt.Sprintf("  %s\n", err1)
	var line2 = fmt.Sprintf("  %s\n", err2)
	var line3 = fmt.Sprintf("  %s\n", err3)
	const tail = "]\n"

	sliceOutput := SpliceErrors([]error{err1, err2, err3})
	assert.True(strings.HasPrefix(sliceOutput, head))
	assert.True(strings.Contains(sliceOutput, line1))
	assert.True(strings.Contains(sliceOutput, line2))
	assert.True(strings.Contains(sliceOutput, line3))
	assert.True(strings.HasSuffix(sliceOutput, tail))

	assert.Equal("", SpliceErrors([]error{}))
	assert.Equal("", SpliceErrors(nil))
}

func TestConcatStrings(t *testing.T) {
	assert := assert.New(t)

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
	for _, testcase := range cases {
		s := ConcatStrings(testcase.args...)
		assert.Equal(testcase.expect, s)
	}
}

func TestGetResourceID(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		namespace string
		name      string
		expected  string
	}{
		{
			namespace: "default",
			name:      "pod",
			expected:  "default/pod",
		},
		{
			namespace: "",
			name:      "pod",
			expected:  "/pod",
		},
		{
			namespace: "default",
			name:      "",
			expected:  "default/",
		},
		{
			namespace: "",
			name:      "",
			expected:  "/",
		},
	}

	for _, c := range cases {
		result := GetResourceID(c.namespace, c.name)
		assert.Equal(c.expected, result)
	}
}
