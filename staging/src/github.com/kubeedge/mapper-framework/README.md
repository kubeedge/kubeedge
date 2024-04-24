# Mapper Framework
Mapper-framework is a framework to make writing [mappers](https://github.com/kubeedge/mappers-go) easier, by providing mapper runtime libs and tools for scaffolding and code generation to bootstrap a new mapper project.

# How to create your own mappers

## 1. Design the device model and device instance CRDs
If you don't know how to use device model, device instance APIs, please get more details in the [page](https://kubeedge.io/docs/developer/device_crd/).

## 2. Generate the mapper project
The command below will generate a framework for the customized mapper. Run the command and input your mapper's name:
```shell
make generate
Please input the mapper name (like 'Bluetooth', 'BLE'): foo
Please input the build method (like 'stream', 'nostream'): nostream
```
A project named as your input will be generated. The file tree is as below:
```
mapper
├── cmd ------------------------ Main process.
│ └── main.go ------------------ Almost need not change.
├── config.yaml ---------------- Configuration file including DMI's grpc settting
├── data ----------------------- Publish data and database implementation layer, almost need not change
│ ├── dbmethod ----------------- Provider implement database interfaces to save data
│ │ ├── influxdb2 -------------- Implementation of Time Series Database(InfluxDB)
│ │ │ └── client.go ------------ WIP
│ │ └── redis  ----------------- Implementation of K/V Database(Redis)
│ │     └── client.go ---------- WIP
│ └── publish ------------------ Publisher implement push interfaces to push data,will add more protocols in the future
│     ├── http ----------------- HTTP client will push data to server
│     │ └── client.go  --------- WIP
│     └── mqtt ----------------- MQTT client will push data to broker
│         └── client.go  ------- WIP
├── device --------------------- Implementation device layer, almost need not change
│ ├── device.go ---------------- Device control, almost need not change
│ └── devicetwin.go ------------ Push twin data to EdgeCore, almost need not change
├── Dockerfile
├── driver --------------------- Device driver layer, complete TODO item in this 
│ ├── devicetype.go ------------ Refine the struct as your CRD
│ └── driver.go ---------------- Fill in the functions like getting data/setting register.
├── hack
│ └── make-rules
│     └── mapper.sh
└── Makefile
```

## 2. Generate the mapper project
After generating the mapper project and filling driver folder, users can make their own mapper image 
based on the Dockerfile file and deploy the mapper in the cluster through deployment and other methods.
If your mapper is aimed to processing streaming data
```shell
  docker build -f Dockerfile_stream -t [YOUR MAPPER IMAGE NAME] .
```
If not, Use the following command:
```shell
  docker build -f Dockerfile_nostream -t [YOUR MAPPER IMAGE NAME] .
```

# Where does it come from?
mapper-framework is synced from https://github.com/kubeedge/kubeedge/tree/master/staging/src/github.com/kubeedge/mapper-framework.
Code changes are made in that location, merged into kubeedge and later synced here.