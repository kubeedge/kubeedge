/*
Copyright 2019 The KubeEdge Authors.

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
package utils

import (
	"flag"
	"math/rand"
	"os"
	"time"

	"k8s.io/client-go/tools/clientcmd"
	cliflag "k8s.io/component-base/cli/flag"
)

//config.json decode struct
type Config struct {
	AppImageURL                    []string `json:"image_url"`
	K8SMasterForKubeEdge           string   `json:"k8smasterforkubeedge"`
	NumOfNodes                     int      `json:"node_num"`
	K8SMasterForProvisionEdgeNodes string   `json:"k8smasterforprovisionedgenodes"`
	CloudImageURL                  string   `json:"cloudimageurl"`
	EdgeImageURL                   string   `json:"edgeimageurl"`
	ControllerStubPort             int      `json:"controllerstubport"`
	Protocol                       string   `json:"protocol"`
	KubeConfigPath                 string   `json:"kubeconfigpath"`
}

// config struct
var config Config
var Flags = flag.NewFlagSet("", flag.ContinueOnError)

func RegisterFlags(flags *flag.FlagSet) {
	flags.StringVar(&config.KubeConfigPath, "kubeconfig", os.Getenv(clientcmd.RecommendedConfigPathEnvVar), "Path to kubeconfig containing embedded authinfo.")
	flags.Var(cliflag.NewStringSlice(&config.AppImageURL), "image-url", "image url list for e2e")
	flags.StringVar(&config.K8SMasterForKubeEdge, "kube-master", "", "the kubernetes master address")
}

func CopyFlags(source *flag.FlagSet, target *flag.FlagSet) {
	source.VisitAll(func(flag *flag.Flag) {
		target.Var(flag.Value, flag.Name, flag.Usage)
	})
}

// get config.json path
func LoadConfig() Config {
	return config
}

// function to Generate Random string
func GetRandomString(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}
