## Deploy kubeedge's edge nodes using containers

This method will guide you to deploy the edge part running in docker
container and MQTT Broker, so make sure that docker engine listening on
`/var/run/docker.sock` which will then mount into the edge container.

+ Check the container runtime environment
  ```bash
  cd build/edge
  ./run_daemon.sh prepare
  ```

+ Set container parameters

  The following parameters do not need to be set if they are not modified.

  | name            | default                           | note                       |
  | --------------- | --------------------------------- | -------------------------- |
  | cloudhub        | 0.0.0.0:10000                     |                            |
  | edgename        | edge-node                         |                            |
  | edgecore_image | kubeedge/edgecore:latest          |                            |
  | arch            | amd64                             | Optional: amd64 \| arm64v8 \| arm32v7 \| i386 \| s390x |
  | qemu_arch       | x86_64                            | Optional: x86_64 \| aarch64 \| arm \| i386 \| s390x  |
  | certpath        | /etc/kubeedge/certs               |                            |
  | certfile        | /etc/kubeedge/certs/edge.crt      |                            |
  | keyfile         | /etc/kubeedge/certs/edge.key      |                            |

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
  ```

+ Build image

  ```
  ./run_daemon.sh build
  ```

+ **(Optional)** If the performance of the edge is not enough, you can cross-compile the image of the edge on the cloud and load the image on the edge.

  - Set the CPU type

    ```
    ./run_daemon.sh set arch=arm64v8 qemu_arch=aarch64
    ```
  - Build image
    ```
    ./run_daemon.sh build
    ```
  - Save image
    ```
    ./run_daemon.sh save
    ```

+ Start container
  ```
  ./run_daemon.sh up
  ```
