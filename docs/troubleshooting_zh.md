# FAQs

本页展示了一些常见的问题。


## keadm init 下载版本失败

如果您遇到了github的连接问题，请在安装前按照下面的代理指南进行操作，以`v1.4.0`为例：

- 从[发布页面](https://github.com/kubeedge/kubeedge/releases/tag/v1.4.0)下载发布包
- 下载与您下载的版本相匹配的crds yamls，链接如下：
  - [devices_v1alpha1_device.yaml](https://raw.githubusercontent.com/kubeedge/kubeedge/v1.4.0/build/crds/devices/devices_v1alpha2_device.yaml)
  - [devices_v1alpha1_devicemodel.yaml](https://raw.githubusercontent.com/kubeedge/kubeedge/v1.4.0/build/crds/devices/devices_v1alpha2_devicemodel.yaml)
  - [cluster_objectsync_v1alpha1.yaml](https://raw.githubusercontent.com/kubeedge/kubeedge/v1.4.0/build/crds/reliablesyncs/cluster_objectsync_v1alpha1.yaml)
  - [objectsync_v1alpha1.yaml](https://raw.githubusercontent.com/kubeedge/kubeedge/v1.4.0/build/crds/reliablesyncs/objectsync_v1alpha1.yaml)
- 把它们放在 `/etc/kubeedge` 下面：
    ```bash
    $ tree -L 3
    .
    ├── crds
    │   ├── devices
    │   │   ├── devices_v1alpha2_devicemodel.yaml
    │   │   └── devices_v1alpha2_device.yaml
    │   └── reliablesyncs
    │       ├── cluster_objectsync_v1alpha1.yaml
    │       └── objectsync_v1alpha1.yaml
    └── kubeedge-v1.4.0-linux-amd64.tar.gz

    3 directories, 5 files

    ```

然后你可以在没有任何网络问题的情况下进行安装，`keadm` 会检测到它们，并且不会再次下载(请确保您指定`v1.4.0`作为您的版本`--kubeedge-version 1.4.0`).

## 容器一直处于挂起/终止状态

1. 检查 `kubectl get nodes` 的输出，确认该节点是否运行正常。检查“kubectl get nodes”的输出，该节点是否运行正常。请注意，处于无法访问、脱机状态的节点在恢复正常之前无法完成非侵害式的pod删除操作的删除操作。
2. 检查 `kubectl describe pod <your-pod>`的输出，确认pod是否调度成功。
3. 检查 `edgecore` 日志是否有任何错误。
4. 检查运行 `edgecore` 的节点的体系结构，并确保尝试运行的容器映像具有相同的体系结构。

   例如，如果您在体系结构为 `arm64v8`的 Raspberry Pi 4上运行 `edgecore` ，对应要执行的nginx镜像应该是docker hub镜像仓的 `arm64v8/nginx` 。

5. 另外，请检查 `podSandboxImage` 的设置是否按照[Modification in edgecore.yaml](./configuration/kubeedge.md#modification-in-edgecoreyaml)进行正确的配置

6. 如果以上所有设置都正确，请手动登录到边缘节点，并通过手动运行docker映像

   ```shell
    docker run <your-container-image>
   ```

7. 如果docker容器镜像未从docker hub中拉取，请检查边缘节点上是否有足够的空间。

## 我们在哪里可以找到cloudcore/edgecore日志

这取决于cloudcore/edgecore的执行方式。

1. 如果使用 `systemd` 启动 cloudcore/edgecore，使用 `journalctl --unit <name of the service probably edgecore.service>` 查看日志。
2. 如果使用 `nohup` 启动 cloudcore/edgecore，会默认在日志所在的位置添加一个路径，如果没有提供日志文件，则会将日志写入stdout。

## 我们在哪里找到pod日志

连接到边缘节点，然后

- 查看位于`/var/log/pods` 中的日志文件 或者
- 使用如同 `docker logs <container id>` 的命令来操作