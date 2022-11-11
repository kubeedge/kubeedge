package util

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.expected, ValidateNodeIP(c.ip))
		})
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
		t.Run(c.name, func(t *testing.T) {
			_, err := Command(c.command, nil)
			require.Equal(t, c.expected, err == nil)
		})
	}
}

func TestGetCurPath(t *testing.T) {
	require.NotEqual(t, "", GetCurPath(), "failed to get current path")
}

func TestGetHostname(t *testing.T) {
	require.NotEqual(t, "", GetHostname(), "get host name failed")
}

func TestGetLocalIP(t *testing.T) {
	_, err := GetLocalIP(GetHostname())
	require.NoError(t, err)
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

	require.Zero(t, strings.Index(sliceOutput, head))
	require.Equal(t, len(head), strings.Index(sliceOutput, line1))
	require.Equal(t, len(head+line1), strings.Index(sliceOutput, line2))
	require.Equal(t, len(head+line1+line2), strings.Index(sliceOutput, line3))
	require.Equal(t, len(head+line1+line2+line3), strings.Index(sliceOutput, tail))

	require.Empty(t, SpliceErrors([]error{}))
	require.Empty(t, SpliceErrors(nil))
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

	for i, c := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			require.Equal(t, c.expect, ConcatStrings(c.args...))
		})
	}
}
