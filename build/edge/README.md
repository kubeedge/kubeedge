## Deploy kubeedge's edge nodes using containers

This method will guide you to deploy the edge part running in docker
container and MQTT Broker, so make sure that docker engine listening on
`/var/run/docker.sock` which will then mount into the edge container.

+ Check the container runtime environment
  ```
  ./build/edge/run_daemon.sh prepare
  ```

+ Set container parameters

  The following parameters do not need to be set if they are not modified.

  | name            | default                           | note                      |
  | --------------- | --------------------------------- | ------------------------- |
  | cloudhub        | 0.0.0.0:10000                     |                           |
  | edgename        | edge-node                         |                           |
  | edge_core_image | kubeedge/edgecore:latest          |                           |
  | arch            | amd64                             | Optional: amd64 \|arm64v8 |
  | qemu_arch       | x86_64                            | Optional: x86_64 \| aarch |
  | certpath        | /etc/kubeedge/edge/certs          |                           |
  | certfile        | /etc/kubeedge/edge/certs/edge.crt |                           |
  | keyfile         | /etc/kubeedge/edge/certs/edge.key |                           |

  ```shell
  ./build/edge/run_daemon.sh set \
  		    cloudhub=0.0.0.0:10000 \
          edgename=edgeNode \
          edge_core_image="kubeedge/edgecore:latest" \
          arch=amd64 \
          qemu_arch=x86_64 \
          certpath=/etc/kubeedge/edge/certs \
          certfile=/etc/kubeedge/edge/certs/edge.crt \
          keyfile=/etc/kubeedge/edge/certs/edge.ke
  ```

+ Build image

  ```
  ./build/edge/run_daemon.sh build
  ```

+ **(Optional)** If the performance of the edge is not enough, you can cross-compile the image of the edge on the cloud and load the image on the edge.

  - Set the CPU type

    ```
    ./build/edge/run_daemon.sh set arch=arm64v8 qemu_arch=aarch
    ```
  - Build image
    ```
    ./build/edge/run_daemon.sh build
    ```
  - Save image
    ```
    ./build/edge/run_daemon.sh save 
    ```

+ Start container
  ```
  ./build/edge/run_daemon.sh up
  ```
