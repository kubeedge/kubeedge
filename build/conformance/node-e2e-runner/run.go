/*
Copyright 2023 The KubeEdge Authors.

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

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/pkg/errors"

	"github.com/kubeedge/kubeedge/build/conformance/util"
)

const (
	parallelEnvKey      = "E2E_PARALLEL"
	dryRunEnvKey        = "E2E_DRYRUN"
	skipEnvKey          = "E2E_SKIP"
	focusEnvKey         = "E2E_FOCUS"
	ginkgoPathEnvKey    = "GINKGO_BIN"
	ginkgoTimeoutEnvKey = "GINKGO_TIMEOUT"
	testBinEnvKey       = "TEST_BIN"
	resultsDirEnvKey    = "RESULTS_DIR"
	reportPrefixEnvKey  = "REPORT_PREFIX"
	imageURL            = "IMAGE_URL"
	testWithDevice      = "TEST_WITH_DEVICE"
	kubeConfigEnvKey    = "KUBECONFIG"
	logFileName         = "e2e.log"
	defaultFocus        = "\\[sig-node\\].*Conformance"
	defaultTimeout      = "2h"
	extraArgsEnvKey     = "E2E_EXTRA_ARGS"
	defaultResultsDir   = "/tmp/results"
	defaultReportPrefix = "conformance"
	defaultGinkgoBinary = "/usr/local/bin/ginkgo"
	defaultTestBinary   = "/usr/local/bin/e2e.test"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		sig := <-c
		log.Printf("Received signal %v, exiting", sig)
		err := util.AfterRunConformance()
		if err != nil {
			log.Printf("failed to cleanup after conformance, err: %v\n", err)
		}
	}()

	if err := RunE2E(); err != nil {
		log.Fatal(err)
	}
}

func RunE2E() error {
	err := util.BeforeRunConformance()
	if err != nil {
		return fmt.Errorf("failed to prepare for run conformance, err: %v", err)
	}

	defer func() {
		err := util.AfterRunConformance()
		if err != nil {
			log.Printf("failed to cleanup after conformance, err: %v\n", err)
		}
	}()

	resultsDir := util.GetEnvWithDefault(resultsDirEnvKey, defaultResultsDir)

	// Print the output to stdout and a logfile which will be returned
	// as part of the results' tarball.
	logFilePath := filepath.Join(resultsDir, logFileName)
	logFile, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to create log file %v, err: %v", logFilePath, err)
	}

	mw := io.MultiWriter(os.Stdout, logFile)

	cmd, err := makeCmd(mw)
	if err != nil {
		return err
	}

	log.Printf("Running command:\n%v\n", util.CmdInfo(cmd))

	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, "starting command")
	}

	return errors.Wrap(cmd.Wait(), "running command")
}

func makeCmd(w io.Writer) (*exec.Cmd, error) {
	var ginkgoArgs []string

	skipCommands, err := util.SkipCommands()
	if err != nil {
		return nil, err
	}

	skipped := strings.Join(skipCommands, "|")

	ginkgoArgs = append(ginkgoArgs, "--skip="+skipped)

	skipEnvValue := util.GetEnvWithDefault(skipEnvKey, "")
	if len(skipEnvValue) > 0 {
		ginkgoArgs = append(ginkgoArgs, "--skip="+skipEnvValue)
	}

	focusEnvValue := util.GetEnvWithDefault(focusEnvKey, defaultFocus)
	ginkgoArgs = append(ginkgoArgs, "--focus="+focusEnvValue)
	ginkgoArgs = append(ginkgoArgs, "--no-color=true")

	timeoutEnvValue := util.GetEnvWithDefault(ginkgoTimeoutEnvKey, defaultTimeout)
	ginkgoArgs = append(ginkgoArgs, "--timeout="+timeoutEnvValue)

	if len(util.GetEnvWithDefault(dryRunEnvKey, "")) > 0 {
		ginkgoArgs = append(ginkgoArgs, "--dryRun=true")
	}

	if parallelEnvValue := util.GetEnvWithDefault(parallelEnvKey, ""); len(parallelEnvValue) > 0 {
		ginkgoArgs = append(ginkgoArgs, parallelEnvValue)
	}

	extraArgs := []string{
		"--report-dir=" + util.GetEnvWithDefault(resultsDirEnvKey, defaultResultsDir),
		"--report-prefix=" + util.GetEnvWithDefault(reportPrefixEnvKey, defaultReportPrefix),
		"--kubeconfig=" + util.GetEnvWithDefault(kubeConfigEnvKey, ""),
		"--image-url=" + util.GetEnvWithDefault(imageURL, "nginx"),
		"--test-with-device=" + util.GetEnvWithDefault(testWithDevice, "false"),
	}

	if len(util.GetEnvWithDefault(extraArgsEnvKey, "")) > 0 {
		extraArgs = append(extraArgs, strings.Split(util.GetEnvWithDefault(extraArgsEnvKey, ""), ",")...)
	}

	var args []string
	args = append(args, ginkgoArgs...)
	args = append(args, util.GetEnvWithDefault(testBinEnvKey, defaultTestBinary))
	args = append(args, "--")
	args = append(args, extraArgs...)

	cmd := exec.Command(util.GetEnvWithDefault(ginkgoPathEnvKey, defaultGinkgoBinary), args...)
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd, nil
}
