package controller

import (
	"os"
	"os/exec"
)

const (
	defBinary = "/usr/local/nginx/sbin/nginx"
	cfgPath   = "/etc/nginx/nginx.conf"
)

// NginxExecTester defines the interface to execute
// command like reload or test configuration
type NginxExecTester interface {
	ExecCommand(args ...string) *exec.Cmd
	Test(cfg string) ([]byte, error)
}

// NginxCommand stores context around a given nginx executable path
type NginxCommand struct {
	Binary string
}

// ExecCommand instantiates an exec.Cmd object to call nginx program
func (nc NginxCommand) ExecCommand(args ...string) *exec.Cmd {
	var cmdArgs []string

	cmdArgs = append(cmdArgs, "-c", cfgPath)
	cmdArgs = append(cmdArgs, args...)
	return exec.Command(nc.Binary, cmdArgs...)
}

// Test checks if config file is a syntax valid nginx configuration
func (nc NginxCommand) Test(cfg string) ([]byte, error) {
	return exec.Command(nc.Binary, "-c", cfg, "-t").CombinedOutput()
}

// NewNginxCommand returns a new NginxCommand from which path
// has been detected from environment variable NGINX_BINARY or default
func NewNginxCommand() NginxCommand {
	command := NginxCommand{
		Binary: defBinary,
	}

	binary := os.Getenv("NGINX_BINARY")
	if binary != "" {
		command.Binary = binary
	}

	return command
}