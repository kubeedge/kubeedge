Now this directory contains images below, they all use the docker buildx to build multi-architecture docker images.

- `kubeedge/build-tools`
  
  It's used to build KubeEdge project, and the dependencies for building KubeEdge will be packaged into this image. 
  You can use [make-build-tools.sh](./build-tools/make-build-tools.sh) to build `kubeedge/build-tools` image.
	
- `kubeedge/installation-package`

  It's used to install edgecore and kedam from docker image to host as binary, and edgecore and keadm will be packaged 
  into this image. You can use [installation-package.sh](./installation-package/installation-package.sh) to build `kubeedge/installation-package` image.

# Building Multi-Architecture Docker Images With Buildx
The both two images are multi-architecture, and rely on docker Buildx. So if you want to build images by yourselves, 
you should enable docker Buildx. It just needs running some commands on you machine.

- Turn On Docker Experimental Features
  
    Either by setting an environment variable
    ```shell
    $ export DOCKER_CLI_EXPERIMENTAL=enabled
    ```
    or by turning the feature on in the config file `$HOME/.docker/config.json`:
    ```
    {
        …
        "experimental" : "enabled"
    }
    ```

- Install QEMU
    ```shell
    sudo apt-get install -y qemu-user-static
    ```

- Install update-binfmts Tool
    ```shell
    sudo apt-get install -y binfmt-support
    ```

- Run QEMU for docker
    ```shell
    docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
    ```

- Create a Buildx Builder
    ```shell
    docker buildx create --name mybuilder
    docker buildx use mybuilder
    docker buildx inspect --bootstrap
    ```

Now, congratulations, you can build images successfully using Buildx.

The above operations are from blog [Building Multi-Architecture Docker Images With Buildx](https://medium.com/@artur.klauser/building-multi-architecture-docker-images-with-buildx-27d80f7e2408). 
For more details about how to build Multi-Architecture docker images with Buildx, you can reference it. 
