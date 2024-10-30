# Mapper Framework-Java
The Mapper module acts as a "translator" between KubeEdge and devices, enabling KubeEdge to interact with devices using various protocols, retrieve device status, read necessary data from devices, and control edge devices.\
Here is a KubeEdge Device Mapper framework written in Java 11, users could quickly complete a custom protocol mapper project based on this framework.

# How to create your own mappers
## 1. Design the device model and device instance CRDs
If you don't know how to use device model, device instance APIs, please get more details in the [page](https://github.com/kubeedge/kubeedge/blob/master/docs/proposals/device-crd-v1beta1.md).

## 2. Generate a new mapper project
The command below will help user generate a new mapper project based on the Mapper Framework-Java. If MapperName is not provided, the default value 'mapper_default' is used
```shell
make generate [MapperName]
```
The resulting project structure tree is shown below:
```
mapper_default
├── src
│ └── main
│  ├── java
│  │ ├── Main.java ------------------ Main process
│  │ ├── config --------------------- Parse config files
│  │ ├── data ----------------------- Push data to 3rd app, save to database implementation layer
│  │ ├── devicepanel ---------------- Implementation of devicepanel layer, managing the device lifecycle
│  │ ├── driver --------------------- Device driver layer, reading and writing device data, then converts it through the customized protocol
│  │ ├── grpc ----------------------- Message interaction between Edgecore and mapper through DMI
│  │ ├── http ----------------------- Create HTTP server to provide API services, supporting directly obtaining device data from the device
│  │ ├── dmi ------------------------ Java version of device manage interface definition
│  │ ├── model ---------------------- Definition of complex variables
│  │ └── service -------------------- Definition of interfaces
│  └── resources
│    ├── logback.xml ---------------- Log configuration
│    ├── config.yaml ---------------- Global Configuration
├── hack
│ └── make-rules
│     ├── generate.sh
│     └── build.sh
├── Dockerfile
├── Makefile
└── pom.xml
```
## 3. Custom your own project
To implement your own protocol, user need to override the functions in the driver module that use the TODO flag, such as init/stop device, get/set device data, protocol config data, visitor config data. Then, user could use the following command to make their own mapper image based on the Dockerfile and deploy the mapper in the cluster through deployment and other methods.
```shell
make build [ImageName]
```
If ImageName is not provided, the name of the current project is used as the image name.
