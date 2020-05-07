# FAQs

This page contains a few commonly occurring questions.
For further support please contact us using the [support page](../getting-started/support.md)

## Container keeps pending/ terminating

1. Check the output of `kubectl get nodes`
2. Check the output of `kubectl describe pod <your-pod>`
3. Check the `edgecore` logs for any errors.
4. Check the architecture of the node running `edgecore` and make sure that container image you are trying to run is of the same architecture.
   For example, if you are running `edgecore` on Raspberry Pi 4, which is of `arm64v8` architecture, the nginx image to be executed would be `arm64v8/nginx` from the docker hub.

5. Also, check that the `podSandboxImage` is correctly set as defined in [Modification in edgecore.yaml](https://github.com/kubeedge/kubeedge/blob/master/docs/setup/kubeedge_configure.md#modification-in-edgecoreyaml).

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
