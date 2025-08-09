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

    crdClientset "github.com/kubeedge/api/client/clientset/versioned"
)

func newForK8sConfig(c *rest.Config, enableImpersonation bool) (*kubernetes.Clientset, error) {
	configShallowCopy := *c

	if configShallowCopy.UserAgent == "" {
		configShallowCopy.UserAgent = rest.DefaultKubernetesUserAgent()
	}

    httpClient, err := httpClientFor(&configShallowCopy, enableImpersonation)
    if err != nil {
        return nil, err
    }

    cs, err := kubernetes.NewForConfigAndClient(&configShallowCopy, httpClient)
    if err != nil {
        return nil, err
    }
    return cs, nil
}

func newForDynamicConfig(c *rest.Config, enableImpersonation bool) (*dynamic.DynamicClient, error) {
	configShallowCopy := dynamic.ConfigFor(c)
    httpClient, err := httpClientFor(configShallowCopy, enableImpersonation)
    if err != nil {
        return nil, err
    }

    cs, err := dynamic.NewForConfigAndClient(configShallowCopy, httpClient)
    if err != nil {
        return nil, err
    }
    return cs, nil
}

func newForCrdConfig(c *rest.Config, enableImpersonation bool) (*crdClientset.Clientset, error) {
	configShallowCopy := *c

	if configShallowCopy.UserAgent == "" {
		configShallowCopy.UserAgent = rest.DefaultKubernetesUserAgent()
	}

    httpClient, err := httpClientFor(&configShallowCopy, enableImpersonation)
    if err != nil {
        return nil, err
    }

    cs, err := crdClientset.NewForConfigAndClient(&configShallowCopy, httpClient)
    if err != nil {
        return nil, err
    }
    return cs, nil
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
	var user, group string
	if r.enable {
		if v := req.Context().Value(authenticationv1.ImpersonateUserHeader); v != nil {
			user = v.(string)
			req.Header.Set(authenticationv1.ImpersonateUserHeader, user)
		}
		if v := req.Context().Value(authenticationv1.ImpersonateGroupHeader); v != nil {
			group = v.(string)
			req.Header[authenticationv1.ImpersonateGroupHeader] = strings.Split(group, "|")
		}
	}
	klog.V(4).Infof("KubeClient: request.method=%s, request.path=%s, user=%q, group= %q",
		req.Method, req.URL.Path, user, group)
	return r.rt.RoundTrip(req)
}
