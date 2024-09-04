package util

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
}

func TestGetLocalIP(t *testing.T) {
	assert := assert.New(t)

	_, err := GetLocalIP(GetHostname())
	assert.NoError(err)
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
