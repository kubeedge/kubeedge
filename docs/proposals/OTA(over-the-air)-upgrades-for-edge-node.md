---
title: OTA(Over-The-Air) Upgrades For Edge Node
authors:
  - "@HT0403"
approvers:
creation-date: 2024-07-28
last-updated: 2024-07-28

---

# OTA(Over-The-Air) Upgrades For Edge Node

## Motivation

In order to make the edge node more convenient and rapid upgrade, we introduce a remote upgrade scheme OTA (Over-The-Air) into KubeEdge.In the main process of OTA(i.e. make the bundle, download the bundle, verify the bundle and firmware upgrade), we have realized most steps.Our release will generate a new image version called installation-package, then we use the NodeUpgradeJob CRD to obtain the installation tool keadm in the image and run the command to upgrade the edge node.During this process, if the hacker masquerades the image in the edge node, this will result in the untrusted binary keadm. We need to verify the digest of the image before the keadm executes the upgrade, which is the third step of OTA to verify the bundle.And in some business scenarios (Internet of vehicles, Internet of Things), we also need to provide an option to make the node wait for confirmation from a person with permission before upgrading the edge node.

## Goals

- Update the proposal of the NodeUpgradeJob
- During the edge node upgrade, the keadm can verify the image digest. If the image is invalid, the edge node cannot be upgraded and an error message is reported
- If the edge node upgrade confirmation is enabled, the upgrade cannot be executes before the confirmation and the job will always be waiting
- Users can call the metaserver API or execute command `keadm ctl ...` to confirm the upgrade

## Project framework diagram

![](../images/proposals/over-the-air-upgrades-for-edge-node.png)

## Implementation details
### Validation of the Image Digest Before Edge Node Upgrade

#### Objective
Ensure that the image being used for the upgrade is authentic and has not been tampered with.
#### Steps
- Get image digest(**Ensure Docker Daemon is Running**):When creating an installation package image, obtain the cryptographic hash (digest) of the image through docker's API.Docker provides an API that allows you to interact with the Docker daemon programmatically.*For example*:Use `curl` to interact with the Docker registry API. Replace your_registry, your_image, and tag with your actual values
```bash
curl -X GET -H "Accept: application/vnd.docker.distribution.manifest.v2+json"
\
https://your_registry/v2/your_image/manifests/tag
```
Response:
```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
  "config": {
    "mediaType": "application/vnd.docker.container.image.v1+json",
    "size": 7023,
    "digest": "sha256:abc123..."
  },
  "layers": [
    {
      "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
      "size": 32654,
      "digest": "sha256:def456..."
    }
  ]
}
```
- Release image digest:Pass the obtained image digest to the edge using Hub by calling this `commontypes.NodeUpgradeJobRequest` method.
```go
func (ndc *NodeUpgradeController) processUpgrade(upgrade *v1alpha1.NodeUpgradeJob) {
	// if users specify Image, we'll use upgrade Version as its image tag, even though Image contains tag.
	// if not, we'll use default image: kubeedge/installation-package:${Version}
	var repo string
	var err error
	repo = "kubeedge/installation-package"
	if upgrade.Spec.Image != "" {
		repo, err = util.GetImageRepo(upgrade.Spec.Image)
		if err != nil {
			klog.Errorf("Image format is not right: %v", err)
			return
		}
	}
	imageTag := upgrade.Spec.Version
  //Get image summary
  imageDigest := upgrade.Spec.ImageDigest
  image := fmt.Sprintf("%s:%s:%s", repo, imageTag,imageDigest)

	upgradeReq := commontypes.NodeUpgradeJobRequest{
		UpgradeID: upgrade.Name,
		HistoryID: uuid.New().String(),
		Version:   upgrade.Spec.Version,
    Image:     image,//String format：repo:version:imagedigest
	}

	tolerate, err := strconv.ParseFloat(upgrade.Spec.FailureTolerate, 64)
	if err != nil {
		klog.Errorf("convert FailureTolerate to float64 failed: %v", err)
		tolerate = 0.1
	}

	concurrency := upgrade.Spec.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	klog.V(4).Infof("deal task message: %v", upgrade)
	ndc.MessageChan <- util.TaskMessage{
		Type:            util.TaskUpgrade,
		CheckItem:       upgrade.Spec.CheckItems,
		Name:            upgrade.Name,
		TimeOutSeconds:  upgrade.Spec.TimeoutSeconds,
		Concurrency:     concurrency,
		FailureTolerate: tolerate,
		NodeNames:       upgrade.Spec.NodeNames,
		LabelSelector:   upgrade.Spec.LabelSelector,
		Status:          v1alpha1.TaskStatus{},
		Msg:             upgradeReq,
	}
}
```
- Fetch and Compare Digest:Before executing the keadm tool on the edge node, the edge node obtains the digest transmitted from the cloud through a `request` request. Calculate the digest of the locally available image and compare it with the obtained digest.
- Decision Making:If the digests match, proceed with the upgrade. If they do not match, abort the upgrade process and possibly trigger an alert or a log entry for investigation.

### Add a Field to Define Edge Node Upgrade Confirmation
#### Objective
Allow for manual confirmation before proceeding with the upgrade, especially in critical scenarios like Internet of Vehicles (IoV) or Internet of Things (IoT).
#### Steps
- Configuration Field：Introduce a new field in the NodeUpgradeJob CRD (Custom Resource Definition) to specify whether manual confirmation is required. This field could be named `requireConfirmation` or similar.

    - `yaml` configuration file
    ```yaml
    apiVersion: edge.kubeedge.io/v1alpha1
    kind: NodeUpgradeJob
    metadata:
      name: example-nodeupgradejob
    spec:
      image: "installation-package:latest"
      requireConfirmation: true  # new field
    ```
    - `NodeUpgradeJobSpec` structure definition
    ```go
    type NodeUpgradeJobSpec struct {
        ......
        ......
        ......
        // RequireConfirmation specifies whether you need to confirm the upgrade
        RequireConfirmation bool `json:"requireConfirmation,omitempty"`
    }
    ```
- Upgrade Logic Modification：Modify the upgrade logic to check the value of this new field before starting the upgrade. If requireConfirmation is set to true, the process should pause and wait for an external signal or API call to proceed.

### API in MetaService for Confirming Edge Node Upgrade
#### Objective
Provide a mechanism for authorized personnel to confirm the upgrade manually.
#### Steps
- API Endpoint：Develop a new API endpoint in the MetaService that can receive a confirmation signal.
- Integration with Upgrade Process：Integrate this API with the upgrade process so that upon receiving a valid confirmation, the upgrade process can resume.
### Command in `keadm ctl` for Confirming Upgrade
#### Objective
Provide a command-line tool for administrators to confirm the upgrade
#### Steps
- Subcommand Development:Add a new subcommand to `keadm ctl` that sends a confirmation signal to the MetaService API. This command could be named something like `confirm`.
- Usage Instructions:`keadm ctl confirm --node=<node_name>`

