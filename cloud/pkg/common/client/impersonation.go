/*
Copyright 2024 The KubeEdge Authors.

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

package client

import (
	"net/http"
	"strings"

	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	crdClientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
)

func newForK8sConfigOrDie(c *rest.Config, enableImpersonation bool) *kubernetes.Clientset {
	configShallowCopy := *c

	if configShallowCopy.UserAgent == "" {
		configShallowCopy.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	httpClient, err := httpClientFor(&configShallowCopy, enableImpersonation)
	if err != nil {
		panic(err)
	}

	cs, err := kubernetes.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		panic(err)
	}
	return cs
}

func newForDynamicConfigOrDie(c *rest.Config, enableImpersonation bool) *dynamic.DynamicClient {
	configShallowCopy := dynamic.ConfigFor(c)
	httpClient, err := httpClientFor(configShallowCopy, enableImpersonation)
	if err != nil {
		panic(err)
	}

	cs, err := dynamic.NewForConfigAndClient(configShallowCopy, httpClient)
	if err != nil {
		panic(err)
	}
	return cs
}

func newForCrdConfigOrDie(c *rest.Config, enableImpersonation bool) *crdClientset.Clientset {
	configShallowCopy := *c

	if configShallowCopy.UserAgent == "" {
		configShallowCopy.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	httpClient, err := httpClientFor(&configShallowCopy, enableImpersonation)
	if err != nil {
		panic(err)
	}

	cs, err := crdClientset.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		panic(err)
	}
	return cs
}

func httpClientFor(c *rest.Config, enableImpersonation bool) (*http.Client, error) {
	transport, err := rest.TransportFor(c)
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Transport: &impersonationRoundTripper{
			enable: enableImpersonation,
			rt:     transport,
		},
		Timeout: c.Timeout,
	}, nil
}

type impersonationRoundTripper struct {
	enable bool
	rt     http.RoundTripper
}

func (r *impersonationRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// extract user and group from context and set impersonation headers
	var userStr, groupStr string
	user := req.Context().Value(authenticationv1.ImpersonateUserHeader)
	if user != nil && r.enable {
		userStr = user.(string)
		req.Header.Set(authenticationv1.ImpersonateUserHeader, userStr)
	}
	group := req.Context().Value(authenticationv1.ImpersonateGroupHeader)
	if group != nil && r.enable {
		groupStr = group.(string)
		for _, g := range strings.Split(groupStr, "|") {
			req.Header.Set(authenticationv1.ImpersonateGroupHeader, g)
		}
	}

	klog.V(4).Infof("KubeClient: request.method=%s, request.path=%s, user=%q, group= %q", req.Method, req.URL.Path, userStr, groupStr)
	return r.rt.RoundTrip(req)
}
