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
```
A project named as your input will be generated.
For more information, you can refer to [the docs](https://kubeedge.io/docs/developer/mappers/#how-to-create-your-own-mappers).


# Where does it come from?
mapper-framework is synced from https://github.com/kubeedge/kubeedge/tree/master/staging/src/github.com/kubeedge/mapper-framework.
Code changes are made in that location, merged into kubeedge and later synced here.
