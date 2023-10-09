# Mapper Generator
Golang mapper-generator used to generate [mappers](https://github.com/kubeedge/mappers-go).

# How to create your own mappers

## 1. Design the device model and device instance CRDs
If you don't know how to use device model, device instance APIs, please get more details in the [page](https://kubeedge.io/docs/developer/device_crd/).

## 2. Generate the code template
The mapper template is to generate a framework for the customized mapper. Run the command and input your mapper's name:
```shell
make generate
Please input the mapper name (like 'Bluetooth', 'BLE'): foo
```
A project named as your input will be generated. The file tree is as below:
```
mapper
├── cmd ------------------------ Main process.
│ └── main.go ------------------ Almost need not change.
├── config.yaml ---------------- Configuration file including DMI's grpc settting
├── data ----------------------- Publish data and database implementation layer, almost need not change
│ ├── dbmethod ----------------- Provider implement database interfaces to save data and provide REST API
│ │ ├── influx ----------------- Implementation of Time Series Database(InfluxDB)
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

# Where does it come from?
mapper-generator is synced from https://github.com/kubeedge/kubeedge/tree/master/staging/src/github.com/kubeedge/mapper-generator. Code changes are made in that location, merged into kubeedge and later synced here.
