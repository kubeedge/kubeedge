# Bluetooth Mapper


## Introduction

Mapper is an application that is used to connect and control devices. This is an implementation of mapper for 
bluetooth protocol. The aim is to create an application through which users can easily operate devices using bluetooth protocol for communication to the KubeEdge platform. The user is required to provide the mapper with the information required to control their device through the configuration file. These can be changed at runtime by providing the input through the MQTT broker.

## Running the mapper

  1. Please ensure that bluetooth service of your device is ON
  2. Set 'bluetooth=true' label for the node (This label is a prerequisite for the scheduler to schedule bluetooth_mapper pod on the node)

      ```shell
      kubectl label nodes <name-of-node> bluetooth=true
      ```

  3. Build and deploy the mapper by following the steps given below.

### Building the bluetooth mapper

 ```shell
cd $GOPATH/src/github.com/kubeedge/kubeedge/device/bluetooth_mapper
make bluetooth_mapper_image
docker tag bluetooth_mapper:v1.0 <your_dockerhub_username>/bluetooth_mapper:v1.0
docker push <your_dockerhub_username>/bluetooth_mapper:v1.0

Note: Before trying to push the docker image to the remote repository please ensure that you have signed into docker from your node, if not please type the followig command to sign in
 docker login
 # Please enter your username and password when prompted
```

### Deploying bluetooth mapper application

```shell
cd $GOPATH/src/github.com/kubeedge/kubeedge/device/bluetooth_mapper
    
# Please enter the following details in the deployment.yaml :-
#    1. Replace <edge_node_name> with the name of your edge node at spec.template.spec.voluems.configMap.name
#    2. Replace <your_dockerhub_username> with your dockerhub username at spec.template.spec.containers.image

kubectl create -f deployment.yaml
```

## Modules

The bluetooth mapper consists of the following five major modules :-

 1. Action Manager
 2. Scheduler
 3. Watcher
 4. Controller
 5. Data Converter


 ### Action Manager

 A bluetooth device can be controlled by setting a specific value in physical register(s) of a device and readings can be acquired by 
 getting the value from specific register(s). We can define an Action as a group of read/write operations on a device. A device may support 
 multiple such actions. The registers are identified by characteristic values which are exposed by the device through entities called characteristic-uuids.
 Each of these actions should be supplied through config-file to action manager or at runtime through MQTT. The values specified initially through the configuration 
 file can be modified at runtime through MQTT. Given below is a guide to provide input to action manager through the configuration file.
   
    action-manager:
       actions:          # Multiple actions can be added
         - name: <name of the action>
           perform-immediately: <true/false>
           device-property-name: <property-name defined in the device model>
         - .......
           .......

1. Multiple actions can be added in the action manager module. Each of these actions can either be executed by the action manager of invoked by other modules of 
the mapper like scheduler and watcher.

2. Name of each action should be unique, it is using this name that the other modules like the scheduler or watcher can invoke which action to perform.

3. Perform-immediately field of the action manager tells the action manager whether it is supposed to perform the action immediately or not, if it set to true then the action manger will
perform the event once.

4. Each action is associated with a device-property-name, which is the property-name defined in the device CRD, which in turn contains the implementation details required by the action.



 ### Scheduler
 
 Scheduler is a component which can perform an action or a set of actions at regular intervals of time. They will make use of the actions previously defined in the action manager module,
 it has to be ensured that before the execution of the schedule the action should be defined, otherwise it would lead to an error. The schedule can be configured to run for a specified number of times
 or run infinitely. The scheduler is an optional module and need not be specified if not required by the user. The user can provide input to the scheduler through configuration file or 
 through MQTT at runtime. The values specified initially by the user through the configuration file can be modified at runtime through MQTT. Given below is a guide to provide input to scheduler 
 through the configuration file. 
 
          scheduler:
            schedules:
              - name: <name of schedule>
                interval: <time in milliseconds>
                occurrence-limit: <number of times to be executed>            # if it is 0, then the event will execute infinitely
                actions:
                  - <action name>
                  - <action name>
              - ......
                ......
 
 1. Multiple schedules can be defined by the user by providing an array as input though the configuration file.

 2. Name specifies the name of the schedule to be executed, each schedule must have a unique name as it is used as a method of identification by the scheduler.

 3. Interval refers to the time interval at which the schedule is meant to be repeated. The user is expected to provide the input in milliseconds.

 4. Occurrence-limit refers to the number of times the action(s) is supposed to occur. If the user wants the event to run infinitely then it can be set to 0 or the field can be skipped.

 5. Actions refer to the action names which are supposed to be executed in the schedule. The actions will be defined in the same order in which they are mentioned here.

 6. The user is expected to provide the names of the actions to be performed in the schedule, in the same order that they are to be executed.


 ### Watcher

 The following are the main responsibilities of the watcher component: 
 a) To scan for bluetooth devices and connect to the correct device once it is Online/In-Range. 

 b) Keep a watch on the expected state of the twin-attributes of the device and perform the action(s) to make actual state equal to expected.

 c) To report the actual state of twin attributes back to the cloud.
  
 The watcher is an optional component and need not be defined or used by the user if not necessary. The input to the watcher can be provided through the configuration file or through 
 mqtt at runtime. The values that are defined through the configuration file can be changed at runtime through MQTT. Given below is a guide to provide input to the watcher through the configuration file.

          watcher:
              device-twin-attributes :
              - device-property-name: <name of attribute>
                  - <action name>
                  - <action name>
              - ......
                ......   
 
 1. Device-property-name refers to the device twin attribute name that was given when creating the device. It is using this name that the watcher watches for any change in expected state.

 2. Actions refers to a list of action names, these are the names of the actions using which we can convert the actual state to the expected state.

 3. The names of the actions being provided must have been defined using the action manager before the mapper begins execution. Also the action names should be mentioned in the same order in which they have
 to be executed.
                
                  
 ### Controller
 
 The controller module is responsible for exposing MQTT APIs to perform CRUD operations on the watcher, scheduler and action manager. The controller is also responsible for starting the other modules like action manager, watcher and scheduler.
 The controller first connects the MQTT client to the broker (using the mqtt configurations, specified in the configuration file), it then initiates the watcher which will connect to the device (based on the configurations provided in the configuration file) and the 
 watcher runs parallelly, after this it starts the action manger which executes all the actions that have been enabled in it, after which the scheduler is started to run parallelly as well. Given below is a guide to provide input to the 
 controller through the configuration file. 
 
          mqtt:
            mode: 0       # 0 -internal mqtt broker  1 - external mqtt broker
            server: tcp://127.0.0.1:1883 # external mqtt broker url.
            internal-server: tcp://127.0.0.1:1884 # internal mqtt broker url.
          device-model-name: <device_model_name>


 ## Usage
 
 ### Configuration File
 
 The user can give the configurations specific to the bluetooth device using configurations provided in the configuration file present at $GOPATH/src/github.com/kubeedge/kubeedge/device/bluetooth_mapper/configuration/config.yaml.
 The details provided in the configuration file are used by action-manager module, scheduler module, watcher module, the data-converter module and the controller.
 
 **Example:** Given below is the instructions using which user can create their own configuration file, for their device.
 
         mqtt:
           mode: 0       # 0 -internal mqtt broker  1 - external mqtt broker
           server: tcp://127.0.0.1:1883 # external mqtt broker url.
           internal-server: tcp://127.0.0.1:1884 # internal mqtt broker url.
         device-model-name: <device_model_name>        #deviceID received while registering device with the cloud
         action-manager:
           actions:          # Multiple actions can be added
           - name: <name of the action>
             perform-immediately: <true/false>
             device-property-name: <property-name defined in the device model>
           - .......
             .......
         scheduler:
           schedules:
           - name: <name of schedule>
             interval: <time in milliseconds>
             occurrence-limit: <number of times to be executed>            # if it is 0, then the event will execute infinitely
             actions:
             - <action name>
             - <action name>
             - ......
           - ......
         watcher:
           device-twin-attributes :
           - device-property-name: <name of attribute>
             actions:        # Multiple actions can be added
             - <action name>
             - <action name>
             - ......
           - ......

                
### Runtime Configuration Modifications
 
 The configuration of the mapper as well as triggering of the modules of the mapper can be done during runtime. The user can do this by
 publishing messages on the respective MQTT topics of each module. Please note that we have to use the same MQTT broker that is being used by the mapper
 i.e. if the mapper is using the internal MQTT broker then the messages have to be published on the internal MQTT broker
 and if the mapper is using the external MQTT broker then the messages have to be published on the external MQTT broker.
                   
The following properties can be changed at runtime by publishing messages on MQTT topics of the MQTT broker:
  - Watcher
  - Action Manager
  - Scheduler
 
  
#### Watcher

The user can add or update the watcher properties of the mapper at runtime. It will overwrite the existing watcher configurations (if exists)

**Topic:** $ke/device/bluetooth-mapper/< deviceID >/watcher/create

**Message:**

             {
              "device-twin-attributes": [
                {
                  "device-property-name": "IOControl",
                  "actions": [                     # List of names of actions to be performed (actions should have been defined before watching)
                    "IOConfigurationInitialize",
                    "IODataInitialize",
                    "IOConfiguration",
                    "IOData"
                  ]
                }
              ]
            }

#### Action Manager

In the action manager module the user can perform two types of operations at runtime, i.e. :
    1. The user can add or update the actions to be performed on the bluetooth device.
    2. The user can delete the actions that were previously defined for the bluetooth device.

##### Action Add

The user can add a set of actions to be performed by the mapper. If an action with the same name as one of the actions in the list exists
 then it updates the action and if the action does not already exist then it is added to the existing set of actions. 

**Topic:** $ke/device/bluetooth-mapper/< deviceID >/action-manager/create

**Message:**

        [
          {
            "name": "IRTemperatureConfiguration",          # name of action
            "perform-immediately": true,                   # whether the action is to performed immediately or not
            "device-property-name": "temperature-enable"   #property-name defined in the device model
          },
          {
            "name": "IRTemperatureData",
            "perform-immediately": true,
            "device-property-name": "temperature"          #property-name defined in the device model
          }
        ]

##### Action Delete

The users can delete a set of actions that were previously defined for the device. If the action mentioned in the list does not exist
then it returns an error message.

**Topic:** $ke/device/bluetooth-mapper/< deviceID >/action-manager/delete

**Message:**
 
        [
          {
            "name": "IRTemperatureConfiguration"        #name of action to be deleted
          },
          {
            "name": "IRTemperatureData"
          },
          {
            "name": "IOConfigurationInitialize"
          },
          {
            "name": "IOConfiguration"
          }
        ]


#### Scheduler

In the scheduler module the user can perform two types of operations at runtime, i.e. :
    1. The user can add or update the schedules to be performed on the bluetooth device.
    2. The user can delete the schedules that were previously defined for the bluetooth device.

##### Schedule Add

The user can add a set of schedules to be performed by the mapper. If a schedule with the same name as one of the schedules in the list exists
 then it updates the schedule and if the action does not already exist then it is added to the existing set of schedules.
 
**Topic:** $ke/device/bluetooth-mapper/< deviceID >/scheduler/create

**Message:**
    
    [
      {
        "name": "temperature",            # name of schedule
        "interval": 3000,           # frequency of the actions to be executed (in milliseconds)
        "occurrence-limit": 25,         # Maximum number of times the event is to be executed, if not given then it runs infinitely 
        "actions": [                          # List of names of actions to be performed (actions should have been defined before execution of schedule) 
          "IRTemperatureConfiguration",
          "IRTemperatureData"
        ]
      }
    ]

##### Schedule Delete

The users can delete a set of schedules that were previously defined for the device. If the schedule mentioned in the list does not exist
then it returns an error message.

**Topic:** $ke/device/bluetooth-mapper/< deviceID >/scheduler/delete

**Message:**

        [
          {
            "name": "temperature"                  #name of schedule to be deleted
          }
        ]