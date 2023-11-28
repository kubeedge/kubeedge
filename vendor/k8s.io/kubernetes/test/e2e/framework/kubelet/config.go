/*
Copyright 2019 The Kubernetes Authors.

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

package kubelet

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/kubernetes/pkg/cluster/ports"
	kubeletconfig "k8s.io/kubernetes/pkg/kubelet/apis/config"
	kubeletconfigscheme "k8s.io/kubernetes/pkg/kubelet/apis/config/scheme"

	"k8s.io/kubernetes/test/e2e/framework"
	e2ekubectl "k8s.io/kubernetes/test/e2e/framework/kubectl"
)

// GetCurrentKubeletConfig fetches the current Kubelet Config for the given node
func GetCurrentKubeletConfig(ctx context.Context, nodeName, namespace string, useProxy bool, standaloneMode bool) (*kubeletconfig.KubeletConfiguration, error) {
	resp := pollConfigz(ctx, 5*time.Minute, 5*time.Second, nodeName, namespace, useProxy, standaloneMode)
	if len(resp) == 0 {
		return nil, fmt.Errorf("failed to fetch /configz from %q", nodeName)
	}
	kubeCfg, err := decodeConfigz(resp)
	if err != nil {
		return nil, err
	}
	return kubeCfg, nil
}

// returns a status 200 response from the /configz endpoint or nil if fails
func pollConfigz(ctx context.Context, timeout time.Duration, pollInterval time.Duration, nodeName, namespace string, useProxy bool, standaloneMode bool) []byte {
	endpoint := ""
	if useProxy {
		// start local proxy, so we can send graceful deletion over query string, rather than body parameter
		framework.Logf("Opening proxy to cluster")
		tk := e2ekubectl.NewTestKubeconfig(framework.TestContext.CertDir, framework.TestContext.Host, framework.TestContext.KubeConfig, framework.TestContext.KubeContext, framework.TestContext.KubectlPath, namespace)
		cmd := tk.KubectlCmd("proxy", "-p", "0")
		stdout, stderr, err := framework.StartCmdAndStreamOutput(cmd)
		framework.ExpectNoError(err)
		defer stdout.Close()
		defer stderr.Close()
		defer framework.TryKill(cmd)

		buf := make([]byte, 128)
		var n int
		n, err = stdout.Read(buf)
		framework.ExpectNoError(err)
		output := string(buf[:n])
		proxyRegexp := regexp.MustCompile("Starting to serve on 127.0.0.1:([0-9]+)")
		match := proxyRegexp.FindStringSubmatch(output)
		framework.ExpectEqual(len(match), 2)
		port, err := strconv.Atoi(match[1])
		framework.ExpectNoError(err)
		framework.Logf("http requesting node kubelet /configz")
		endpoint = fmt.Sprintf("http://127.0.0.1:%d/api/v1/nodes/%s/proxy/configz", port, nodeName)
	} else if !standaloneMode {
		endpoint = fmt.Sprintf("%s/api/v1/nodes/%s/proxy/configz", framework.TestContext.Host, framework.TestContext.NodeName)
	} else {
		endpoint = fmt.Sprintf("https://127.0.0.1:%d/configz", ports.KubeletPort)
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", endpoint, nil)
	framework.ExpectNoError(err)
	if !useProxy {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", framework.TestContext.BearerToken))
	}
	req.Header.Add("Accept", "application/json")

	var respBody []byte
	err = wait.PollImmediateWithContext(ctx, pollInterval, timeout, func(ctx context.Context) (bool, error) {
		resp, err := client.Do(req)
		if err != nil {
			framework.Logf("Failed to get /configz, retrying. Error: %v", err)
			return false, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			framework.Logf("/configz response status not 200, retrying. Response was: %+v", resp)
			return false, nil
		}

		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			framework.Logf("failed to read body from /configz response, retrying. Error: %v", err)
			return false, nil
		}

		return true, nil
	})
	framework.ExpectNoError(err, "Failed to get successful response from /configz")

	return respBody
}

// Decodes the http response from /configz and returns a kubeletconfig.KubeletConfiguration (internal type).
func decodeConfigz(respBody []byte) (*kubeletconfig.KubeletConfiguration, error) {
	// This hack because /configz reports the following structure:
	// {"kubeletconfig": {the JSON representation of kubeletconfigv1beta1.KubeletConfiguration}}
	type configzWrapper struct {
		ComponentConfig kubeletconfigv1beta1.KubeletConfiguration `json:"kubeletconfig"`
	}

	configz := configzWrapper{}
	kubeCfg := kubeletconfig.KubeletConfiguration{}

	err := json.Unmarshal(respBody, &configz)
	if err != nil {
		return nil, err
	}

	scheme, _, err := kubeletconfigscheme.NewSchemeAndCodecs()
	if err != nil {
		return nil, err
	}
	err = scheme.Convert(&configz.ComponentConfig, &kubeCfg, nil)
	if err != nil {
		return nil, err
	}

	return &kubeCfg, nil
}
