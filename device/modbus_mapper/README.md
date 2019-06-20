# Modbus Mapper


## Introduction

Mapper is an application that is used to connect and control devices. This is an implementation of mapper for 
Modbus protocol. The aim is to create an application through which users can easily operate devices using ModbusTCP/ModbusRTU protocol for communication to the KubeEdge platform. The user is required to provide the mapper with the information required to control their device through the dpl configuration file. These can be changed at runtime by updating configmap.

## Running the mapper

  1. Please ensure that Modbus device is connected to your edge node
  2. Set 'modbus=true' label for the node (This label is a prerequisite for the scheduler to schedule modbus_mapper pod on the node)

      ```shell
      kubectl label nodes <name-of-node> modbus=true
      ```

  3. Build and deploy the mapper by following the steps given below.

### Building the modbus mapper

 ```shell
cd $GOPATH/src/github.com/kubeedge/kubeedge/device/modbus_mapper
make # or `make modbus_mapper`
docker tag modbus_mapper:v1.0 <your_dockerhub_username>/modbus_mapper:v1.0
docker push <your_dockerhub_username>/modbus_mapper:v1.0

Note: Before trying to push the docker image to the remote repository please ensure that you have signed into docker from your node, if not please type the followig command to sign in
 docker login
 # Please enter your username and password when prompted
```

### Deploying modbus mapper application

```shell
cd $GOPATH/src/github.com/kubeedge/kubeedge/device/modbus_mapper

# Please enter the following details in the deployment.yaml :-
#    1. Replace <edge_node_name> with the name of your edge node at spec.template.spec.voluems.configMap.name
#    2. Replace <your_dockerhub_username> with your dockerhub username at spec.template.spec.containers.image

kubectl create -f deployment.yaml
```

## Modules

The modbus mapper consists of the following four major modules :-

 1. Controller
 2. Modbus Manager
 3. Devicetwin Manager
 4. File Watcher

 ### Controller

 The main entry is index.js. The controller module is responsible for subscribing edge MQTT devicetwin topic and perform check/modify operation on connected modbus devices. The controller is also responsible for loading the configuration and starting the other modules. The controller first connects the MQTT client to the broker to receive message of expected devicetwin value (using the mqtt configurations in conf.json), it then connects to the devices and check all the properties of devices every 2 seconds (based on dpl configuration provided in the configuration file) and the file watcher runs parallelly to check whether the dpl configuration file is changed.

 ### Modbus Manager
 
 Modbus Manager is a component which can perform an read or write action on modbus device. The following are the main responsibilities of this component: 
 a) When controller receives message of expected devicetwin value, Modbus Manager will connect to the device and change the registers to make actual state equal to expected. 

 b) When controller checks all the properties of devices, Modbus Manager will connect to the device and read the actual value in registers accroding to the dpl configuration.

 ### Devicetwin Manager

 Devicetwin Manager is a component which can transfer the edge devicetwin message. The following are the main responsibilities of this component: 
 a) To receive the edge devicetwin message from edge mqtt broker and parse message.

 b) To report the actual value of device properties in devicetwin format to the cloud.
                  
 ### File Watcher
 
 File Watcher is a component which can load dpl and mqtt configuration from configuration files.The following are the main responsibilities of this component: 
 a) To monitor the dpl configuration file. If this file changed, file watcher will reload the dpl configuration to the mapper.

 b) To load dpl and mqtt configuration when mapper starts first time.

