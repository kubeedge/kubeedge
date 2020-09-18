# FAQs

This page contains a few commonly occurring questions.


## keadm init failed to download release

If you have issue about connection to github, please follow below guide with proxy before do setup, take `v1.4.0` as example:

- download release pkgs from [release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.4.0)
- download crds yamls matches the release version you downloaded, links as below:
  - [devices_v1alpha1_device.yaml](https://raw.githubusercontent.com/kubeedge/kubeedge/v1.4.0/build/crds/devices/devices_v1alpha2_device.yaml)
  - [devices_v1alpha1_devicemodel.yaml](https://raw.githubusercontent.com/kubeedge/kubeedge/v1.4.0/build/crds/devices/devices_v1alpha2_devicemodel.yaml)
  - [cluster_objectsync_v1alpha1.yaml](https://raw.githubusercontent.com/kubeedge/kubeedge/v1.4.0/build/crds/reliablesyncs/cluster_objectsync_v1alpha1.yaml)
  - [objectsync_v1alpha1.yaml](https://raw.githubusercontent.com/kubeedge/kubeedge/v1.4.0/build/crds/reliablesyncs/objectsync_v1alpha1.yaml)
- put them under `/etc/kubeedge` as below:
    ```bash
    $ tree -L 3
    .
    ├── crds
    │   ├── devices
    │   │   ├── devices_v1alpha2_devicemodel.yaml
    │   │   └── devices_v1alpha2_device.yaml
    │   └── reliablesyncs
    │       ├── cluster_objectsync_v1alpha1.yaml
    │       └── objectsync_v1alpha1.yaml
    └── kubeedge-v1.4.0-linux-amd64.tar.gz

    3 directories, 5 files

    ```

Then you can do setup without any network issue, `keadm` would detect them and not download again(make sure you specify `v1.4.0` with option `--kubeedge-version 1.4.0`).

## Container keeps pending/ terminating

1. Check the output of `kubectl get nodes`, whether the node is running healthy. Note that nodes in unreachable, offline status cannot complete graceful/soft pod deletion until they come back to normal.
2. Check the output of `kubectl describe pod <your-pod>`, whether the pod is scheduled successfully.
3. Check the `edgecore` logs for any errors.
4. Check the architecture of the node running `edgecore` and make sure that container image you are trying to run is of the same architecture.
   For example, if you are running `edgecore` on Raspberry Pi 4, which is of `arm64v8` architecture, the nginx image to be executed would be `arm64v8/nginx` from the docker hub.

5. Also, check that the `podSandboxImage` is correctly set as defined in [Modification in edgecore.yaml](./configuration/kubeedge.md#modification-in-edgecoreyaml).

6. If all of the above is correctly set, login manually to your edge node and run your docker image manually by

   ```shell
    docker run <your-container-image>
   ```

7. If the docker container image is not pulled from the docker hub, please check that there is enough space on the edge node.

## Where do we find cloudcore/edgecore logs

This depends on the how cloudcore/ edgecore has been executed.

1. If `systemd` was used to start the cloudcore/ edgecore? then use `journalctl --unit <name of the service probably edgecore.service>` to view the logs.
2. If `nohup` was used to start the cloudcore/ edgecore, either a path would have been added where the log is located, Otherwise, if the log file wasn't provided, the logs would be written to stdout.

## Where do we find the pod logs

Connect to the edge node and then either

- use the log file located in `/var/log/pods` or
- use commands like `docker logs <container id>`

**kubectl logs** is not yet supported by KubeEdge.
