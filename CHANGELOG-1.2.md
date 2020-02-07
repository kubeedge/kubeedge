
   * [v1.2.0](#v120)
      * [Downloads for v1.2.0](#downloads-for-v120)
         * [KubeEdge Binaries](#kubeedge-binaries)
         * [Installer Binaries](#installer-binaries)
         * [EdgeSite Binaries](#edgesite-binaries)
      * [KubeEdge v1.2 Release Notes](#kubeedge-v12-release-notes)
         * [1.2 What's New](#12-whats-new)
         * [Known Issues](#known-issues)
         * [Other notable changes](#other-notable-changes)

# KubeEdge v1.2 Release Notes

## 1.2 What's New

**Reliable message delivery from cloud to edge**

This feature improved the reliable message delivery mechanism from cloud to edge. If cloudcore or edgecore
being restarted or offline for a while, it can ensure that the messages sent to the edge are not lost, and 
avoid inconsistency between cloud and edge.
([#1343](https://github.com/kubeedge/kubeedge/pull/1343), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng), [@fisherxu](https://github.com/fisherxu), [@SpaghettiAndSalmon](https://github.com/SpaghettiAndSalmon))

**KubeEdge Component Config**

The configuration information of all KubeEdge components is integrated into the unified API, 
and users can view all configuration information and their default values through the API.
([#1172](https://github.com/kubeedge/kubeedge/pull/1172), [@kadisi](https://github.com/kadisi))

**Kubernetes dependencies Upgrade**

Upgrade the venderod kubernetes version to v1.17.1, so users can use the feature of new version 
on the cloud and on the edge side.
([#1349](https://github.com/kubeedge/kubeedge/pull/1349), [@fisherxu](https://github.com/fisherxu))

**Auto registration of edge node**

Users can set the `register-node` option to `true` in EdgeCore so that edge nodes will
automatically register node info to K8s master in the cloud.
([#1401](https://github.com/kubeedge/kubeedge/pull/1401), [@kuramal](https://github.com/kuramal))

### Known Issues

- High Available of CloudCore is missing.

- Metrics at edge is missing.

### Other notable changes

- Move beehive code intree. ([#1157](https://github.com/kubeedge/kubeedge/pull/1157), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))

- Move viaduct code intree. ([#1158](https://github.com/kubeedge/kubeedge/pull/1158), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))

- Component Config: Add edgecore,cloudcore,edgesite config apis. ([#1212](https://github.com/kubeedge/kubeedge/pull/1212), [@kadisi](https://github.com/kadisi))

- Bugfix: Remove redundant logs when CloudCore exits. ([#1215](https://github.com/kubeedge/kubeedge/pull/1215), [@kadisi](https://github.com/kadisi))

- Bugfix: Remove redundant logs when EdgeCore exits. ([#1223](https://github.com/kubeedge/kubeedge/pull/1223), [@kadisi](https://github.com/kadisi))

- Optimize the use of beehive context. ([#1262](https://github.com/kubeedge/kubeedge/pull/1262), [@kadisi](https://github.com/kadisi))

- Add default initialization method for each module. ([#1267](https://github.com/kubeedge/kubeedge/pull/1267), [@kadisi](https://github.com/kadisi))

- Dns query from container can not correctly return back when using edgemesh. ([#1281](https://github.com/kubeedge/kubeedge/pull/1281), [@cwl233](https://github.com/cwl233))

- Add compatibility matrix for K8s and Golang. ([#1300](https://github.com/kubeedge/kubeedge/pull/1300), [@fisherxu](https://github.com/fisherxu))

- Check the running environment before start edge core. ([#1341](https://github.com/kubeedge/kubeedge/pull/1341), [@kuramal](https://github.com/kuramal))

- Add reliable sync API to store the object resourceVersion that was successfully persisted to the edge node. ([#1373](https://github.com/kubeedge/kubeedge/pull/1373), [@fisherxu](https://github.com/fisherxu))

- Add synccontroller for reliable message delivery. ([#1385](https://github.com/kubeedge/kubeedge/pull/1385), [@fisherxu](https://github.com/fisherxu))

- kubeedge Component use new config api, and use new config file. ([#1393](https://github.com/kubeedge/kubeedge/pull/1393), [@kadisi](https://github.com/kadisi))

- Fix edgecore cpu usage issue of running lot of pods on the edge. ([#1396](https://github.com/kubeedge/kubeedge/pull/1396), [@fisherxu](https://github.com/fisherxu))

- Bump k8s dependencies to 1.17.1. ([#1402](https://github.com/kubeedge/kubeedge/pull/1402), [@fisherxu](https://github.com/fisherxu))

- Create socket address directory if not exist. ([#1412](https://github.com/kubeedge/kubeedge/pull/1412), [@chendave](https://github.com/chendave))

- Add reliable message delivery implementation. ([#1416](https://github.com/kubeedge/kubeedge/pull/1416), [@fisherxu](https://github.com/fisherxu))
