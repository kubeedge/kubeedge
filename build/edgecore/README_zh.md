## 使用容器部署kubeedge的边缘节点

此方式将在容器中运行 edge 端和mqtt broker，所以需要确认 docker engine 监听在
`/var/run/docker.sock`，这个之后需要挂载到容器中。

+ 检查容器运行环境
  ```
  cd $GOPATH/src/github.com/kubeedge/kubeedge/build/edge
  ./run_daemon.sh prepare
  ```

+ 设置容器参数

  以下参数如果不用修改则无需设置

  | 参数名称            | 默认值                       | 备注                     |
  | ------------------- | ---------------------------- | ------------------------ |
  | cloudhub            | 0.0.0.0:10000                |                          |
  | edgename            | edge-node                    |                          |
  | edgecore_image     | kubeedge/edgecore:latest     |                          |
  | arch                | amd64                        | 可选值：amd64 \| arm64v8 \| arm32v7 \| i386 \| s390x |
  | qemu_arch           | x86_64                       | 可选值：x86_64 \| aarch64 \| arm \| i386 \| s390x  |
  | certpath            | /etc/kubeedge/certs          |                          |
  | certfile            | /etc/kubeedge/certs/edge.crt |                          |
  | keyfile             | /etc/kubeedge/certs/edge.key |                          |

  ```shell
  ./run_daemon.sh set \
          cloudhub=0.0.0.0:10000 \
          edgename=edge-node \
          edgecore_image="kubeedge/edgecore:latest" \
          arch=amd64 \
          qemu_arch=x86_64 \
          certpath=/etc/kubeedge/certs \
          certfile=/etc/kubeedge/certs/edge.crt \
          keyfile=/etc/kubeedge/certs/edge.key
  ````

+ 编译容器镜像

  ```
  ./run_daemon.sh build
  ```

+ **(可选)** 如果edge的性能不够，可以在cloud上交叉编译edge的镜像，在edge端加载镜像
  - 设置CPU类型

    ```
    ./run_daemon.sh set arch=arm64v8 qemu_arch=aarch64
    ```

  - 编译镜像
    ```
    ./run_daemon.sh build
    ```

  - 保存镜像
    ```
    ./run_daemon.sh save
    ```

+ 启动容器
  ```
  ./run_daemon.sh up
  ```
