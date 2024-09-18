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
Prevent a hacker from masquerading an image and introducing an untrusted binary by validating the image's digest before the upgrade process begins.
#### Steps
- When the NodeUpgradeJob CRD is used to initiate the upgrade, implement a mechanism to fetch the image's digest from a trusted source (e.g., a secure image registry).
**Example**:Docker API
  ```go
  //Create a Docker Client
  func getDockerClient() (*client.Client, error) {
      cli, err := client.NewClientWithOpts(client.FromEnv,client.WithAPIVersionNegotiation())
      if err != nil {
          return nil, err
      }
      return cli, nil
    }
  //Inspect the Image to Retrieve the Digest
  func getImageDigest(cli *client.Client, imageName string) (string, error) {
      ctx := context.Background()
      imageInspect, _, err := cli.ImageInspectWithRaw(ctx, imageName)
      if err != nil {
          return "", err
      }

      // RepoDigests contains the digest info
      if len(imageInspect.RepoDigests) > 0 {
          return imageInspect.RepoDigests[0], nil
      }
      return "", fmt.Errorf("no digest found for image %s", imageName)
    }
  ```
- Release image digest:Pass the obtained image digest to the edge using Hub by calling this `nodetask.TransferImageToEdge` method.
  ```go
  type NodeUpgradeJobRequest struct {
    UpgradeID   string
    HistoryID   string
    Version     string
    UpgradeTool string
    Image       string
    ImageDigest string
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

