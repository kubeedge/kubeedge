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

package restful

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"

	"github.com/kubeedge/kubeedge/edge/pkg/edged/config"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/common"
)

const (
	CONTAINERLOGS string = "/containerLogs"
)

type Request struct {
	Method string
	Path   string
	Body   io.Reader
}

func LogsRequest(namespace string, podName string, containerName string, logsInfo common.LogsInfo) *Request {
	queryString, err := structToQueryString(logsInfo)
	if err != nil {
		return nil
	}
	return &Request{
		Method: http.MethodGet,
		Path:   "/" + CONTAINERLOGS + "/" + namespace + "/" + podName + "/" + containerName + "?" + queryString,
	}
}

// structToQueryString converts a struct to a query string
func structToQueryString(s interface{}) (string, error) {
	v := reflect.ValueOf(s)
	t := reflect.TypeOf(s)

	values := url.Values{}
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("query")

		if fieldType.Name == "Namespace" || fieldType.Name == "PodName" || fieldType.Name == "ContainerName" {
			continue
		}

		if field.IsZero() {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			values.Add(tag, field.String())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			values.Add(tag, strconv.FormatInt(field.Int(), 10))
		}
	}
	return values.Encode(), nil
}

func (req *Request) RestfulRequest() (*http.Response, error) {
	var client http.Client
	edgedAddress := config.Config.Edged.TailoredKubeletConfig.Address
	edgedPort := config.Config.Edged.TailoredKubeletConfig.ReadOnlyPort
	port := strconv.Itoa(int(edgedPort))

	url := "http://" + edgedAddress + ":" + port
	client = http.Client{}

	request, err := http.NewRequest(req.Method, url+req.Path, req.Body)
	if err != nil {
		return nil, fmt.Errorf("restful format failed with err: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("restful request failed with err: %v", err)
	}
	return response, nil
}
