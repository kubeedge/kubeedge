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

package constants

const (
	CloudCoreConfigFile = "/tmp/cloudcore.yaml"
	EdgeCoreConfigFile  = "/tmp/edgecore.yaml"
	EdgeSiteConfigFile  = "/tmp/edgesite.yaml"

	CatCloudCoreConfigFile = "cat " + CloudCoreConfigFile
	CatEdgeCoreConfigFile  = "cat " + EdgeCoreConfigFile
	CatEdgeSiteConfigFile  = "cat " + EdgeSiteConfigFile

	RunCloudcore = "cd ${GOPATH}/src/github.com/kubeedge/kubeedge/_output/local/bin/; sudo nohup ./cloudcore --config=" + CloudCoreConfigFile + " > cloudcore.log 2>&1 &"
	RunEdgecore  = "cd ${GOPATH}/src/github.com/kubeedge/kubeedge/_output/local/bin/; sudo nohup ./edgecore --config=" + EdgeCoreConfigFile + " > edgecore.log 2>&1 &"
	RunEdgeSite  = "cd ${GOPATH}/src/github.com/kubeedge/kubeedge/_output/local/bin/; sudo nohup ./edgesite --config=" + EdgeSiteConfigFile + " > edgesite.log 2>&1 &"

	CheckCloudcore = "sudo pgrep cloudcore"
	CheckEdgecore  = "sudo pgrep edgecore"
	CheckEdgesite  = "sudo pgrep edgesite"

	CatCloudcoreLog = "cd ${GOPATH}/src/github.com/kubeedge/kubeedge/_output/local/bin/; cat cloudcore.log"
	CatEdgecoreLog  = "cd ${GOPATH}/src/github.com/kubeedge/kubeedge/_output/local/bin/; cat edgecore.log"
	CatEdgeSiteLog  = "cd ${GOPATH}/src/github.com/kubeedge/kubeedge/_output/local/bin/; cat  edgesite.log"

	AppHandler        = "/api/v1/namespaces/default/pods"
	NodeHandler       = "/api/v1/nodes"
	DeploymentHandler = "/apis/apps/v1/namespaces/default/deployments"
)
