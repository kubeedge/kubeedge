# Bluetooth Mapper


## Introduction

Mapper is an application that is used to connect and control devices. This is an implementation of mapper for 
bluetooth protocol. The aim is to create an application through which users can easily operate devices using bluetooth protocol for communication to the KubeEdge platform. The user is required to provide the mapper with the information required to control their device through the configuration file. These can be changed at runtime by providing the input through the MQTT broker.

## Running the mapper

While running the mapper please keep in mind the following points :-

   1. Please run the binary as root or with root permissions.
   2. Please ensure that bluetooth service of your device is ON
   3. Please remember to give one of the following command line parameters for the logging service:
    
     -logtostderr=true      	   // Logs are written to standard error instead of to files.
     -alsologtostderr=true             // Logs are written to standard error as well as to files.
     -stderrthreshold=ERROR		   // Log events at or above this severity are logged to standard error as well as to files.
     -log_dir=""          		   // Log files will be written to this directory instead of the default temporary directory.

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
           enable: <true/false>
           operation:
             action: <Write/Read>
             characteristic-uuid: <UUID of the characteristic example:- f000aa0304514000b000000000000000>
             value: [1]              # value in bytes to be written by default if converter for this is not mentioned
         - .......
           .......

1. Multiple actions can be added in the action manager module. Each of these actions can either be executed by the action manager of invoked by other modules of 
the mapper like scheduler and watcher.

2. Name of each action should be unique, it is using this name that the other modules like the scheduler or watcher can invoke which action to perform.

3. Enable field of the action manager tells the action manager whether it is supposed to perform the action or not, if it set to true then the action maanger will 
perform the event once.

4. Each action is associated with an operation, which contains the implementation details of the action.

5. Action field under the operation specifies whether it is a read/write event 

6. Characteristic-UUID is an entity exposed by the bluetooth device which contains maps internally to the register at which the value is to written into or read from.

7. Value contains the value in bytes that is to be written into the register, in case of a write operation and in the case of read operation, the read value is stored here.  


 ### Scheduler
 
 Scheduler is a component which can perform an action or a set of actions at regular intervals of time. They will make use of the actions previously defined in the action manager module,
 it has to be ensured that before the execution of the schedule the action should be defined, otherwise it would lead to an error. The schedule can be configured to run for a specified number of times
 or run infinitely. The scheduler is an optional module and need not be specified if not required by the user. The user can provide input to the scheduler through configuration file or 
 through MQTT at runtime. The values specified initially by the user through the configuration file can be modified at runtime through MQTT. Given below is a guide to provide input to scheduler 
 through the configuration file. 
 
          scheduler:
            schedules:
              - enable: <true/false>
                name: <name of schedule>
                event-frequency: <time in milliseconds>
                occurrence-limit: <number of times to be executed>            # if it is 0, then the event will execute infinitely
                actions:
                  - <action name>
                  - <action name>
              - ......
                ......
 
 1. Multiple schedules can be defined by the user by providing an array as input though the configuration file.

 2. Enable specifies whether the scheduler is enabled or not, if enable is set to true then the schedule will be executed by the scheduler.  

 3. Name specifies the name of the schedule to be executed, each schedule must have a unique name as it is used as a method of identification by the scheduler.

 4. Event-frequency refers to the time interval at which the schedule is meant to be repeated. The user is expected to provide the input in milliseconds.  

 5. Occurrence-limit refers to the number of times the action(s) is supposed to occur. If the user wants the event to run infinitely then it can be set to 0 or the field can be skipped.

 6. Actions refer to the action names which are supposed to be executed in the schedule. The actions will be defined in the same order in which they are mentioned here.

 7. The user is expected to provide the names of the actions to be performed in the schedule, in the same order that they are to be executed. 


 ### Watcher

 The following are the main responsibilities of the watcher component: 
 a) To scan for bluetooth devices and connect to the correct device once it is Online/In-Range. 

 b) Keep a watch on the expected state of the twin-attributes of the device and perform the action(s) to make actual state equal to expected.

 c) To report the actual state of twin attributes back to the cloud.
  
 The watcher is an optional component and need not be defined or used by the user if not necessary. The input to the watcher can be provided through the configuration file or through 
 mqtt at runtime. The values that are defined through the configuration file can be changed at runtime through MQTT. Given below is a guide to provide input to the watcher through the configuration file.

          watcher:
            enable: <true/false>
            attributes :
              - name: <name of attribute>                        # the twin attribute name defined while creating device 
                actions:        # Multiple actions can be added
                  - <action name>
                  - <action name>
              - ......
                ......   
 
 1. Enable specifies whether the watcher is enabled or not. If it is set to true then the watcher will start watching on the expected state.

 2. Name refers to the attribute name that was given when creating the device. It is using this name that the watcher watches for any change in expected state.

 3. Actions refers to a list of action names, these are the names of the actions using which we can convert the actual state to the expected state. 

 4. The names of the actions being provided must have been defined using the action manager before the mapper begins execution. Also the action names should be mentioned in the same order in which they have 
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
          device:
            name: <device name>    # name of the device to be connected
            id: <deviceID>         #deviceID received while registering device with the cloud


 ### Data Converter

 The data converter module is responsible to convert the data going into and coming from the device into a suitable form. Data received from the devices can be in complex formats. eg: HexDecimal with bytes shuffled, this data cannot be directly understood by KubeEdge.
 It is the responsibility of the data-converter module to convert the readings into a format understood by KubeEdge. The operations required to be performed to convert the data to a suitable form are specified under the read section of the data converter part of the configuration 
 file. In the same manner the data coming from the platform that are to be written into the device, will not be in a form that is understandable to the device. Therefore, it is the responsibility of the data convert the data into a form that can be understood by the device.    
 The data converter acts as an adapter between the platform and the device and converts the data in such a manner that they are understandable to each other.
 Given below is a guide to provide input to the data converter through the configuration file.
  
         data-converter:
           write:  
             attributes:     
               - name: <name of attribute>                   # the twin attribute name defined while creating device  
                 operations :     
                   <action name> : 
                     data-map:                   # to be given in byte format
                       <value coming from platform>: <byte value>
                       <value coming from platform>: <byte value>
               - .......
                 .......        
           read:
             actions:
               - action-name: <name of action> # name of action for which conversion is to be performed   
                 conversion-operation:         # Only applicable operations/Values to be given, everything else is taken as nil
                   start-index: <start index of the incoming byte stream>        # ex:- start-index:2, end-index:3 concatenates the value present at second and third index of the incoming byte stream. If we want to reverse the order we can give it as start-index:3, end-index:2        
                   end-index: <end index of incoming byte stream>                 # the value specified should be inclusive for example if 3 is specified it includes the third index. 
                   shift-left: <number of bits to shift left, if necessary>
                   shift-right: <number of bits to shift right, if necessary> 
                   multiply: <value to be multiplied by, if necessary>
                   divide: <value to be divided by, if necessary>
                   add: <value to be added, if necessary> 
                   subtract: <value to be subtracted, if necessary>
                   order-of-execution:             # this refers to the order in which the add, subtract, multiply or divide operation is to be performed  
                     - <add/subtract/multiply/divide>
                     - <add/subtract/multiply/divide>
               - .....
                 .....
                       
 1. The data-converter input in the configuration file consists of two sections, read section and write section. The read section is responsible for converting the data being read from the device into a form that is understandable by the platform, the write section is responsible for 
 converting the data coming from the platform into a form that is understandable by the device.

 2. Under the write section of the data converter multiple attributes can be defined:
    2.1 Each attribute contains the name of the attribute (i.e the twin name defined while creating the device) and operations
    2.2 Under operations we provide the action name (i.e. the action, defined in action manager, for which this conversion is to be performed) which contains a data map. The data map refers to a map of values with the values coming from the platform as key 
    and the value to be written to the device (in byte format) as values    

 3. Under the read section of data converter multiple actions can be defined:
    3.1 Each action contains an action name (the name of the action, defined in the action manager) as well as conversion operation.
    3.2 Conversion operation consists of the following value: (Please note that only applicable values are to be filled. Any field, except start and end index, that is not necessary for conversion can be omitted)      
        3.2.1 Start-index and end-index refers to the starting index and ending index from which the incoming byte stream is to be considered for conversion, all values from the start index to the end index are concatenated together. 
        For example: if 4 values are going to be arriving and you want to concatenate the 2nd and 3rd value then start and end index will be
        1 and 2 respectively. In case you require the value in reverse order then the end-index < start-index, for example, if we need the 3rd and 2nd value to be concatenated then the start and end index will be 2 and 1 respectively.   
    3.3 Shift-left indicates the number of bits to which the left shift operation is to be performed, this can be omitted if this operation is not required.
    3.4 Shift-right indicates the number of bits to which the right shift operation is to be performed, this can be omitted if this operation is not required.
    3.5 Multiply indicates the value to be multiplied by, this field could be left empty if not required
    3.6 Divide indicates the value to be divided by, this field could be left empty if not required 
    3.7 Add indicates the value to added with, this field could be left empty if not required
    3.8 Subtract indicates the value to subtracted by, this field could be left empty if not required
    3.9 Order-of-execution refers to order in which the add, subtract, divide or multiply operations are to be performed. The user is expected to provide "add", "subtract", "multiply" or "divide" as input in the same order that they are to be executed.


 ## Usage
 
 ### Configuration File
 
 The user can give the configurations specific to the bluetooth device using configurations provided in the configuration  file.
 The user can enable or disable modules through the configuration files. The details provided in the configuration file are used by 
 action-manager module, scheduler module, watcher module, the data-converter module and the controller.  
 
 **Example:** Below is an example for the configuration file for the temperature sensor and I/O control of Texas Instruments CC2650 device,
  Similar to this user can create their own configuration file.  
 
         mqtt:
           mode: 0       # 0 -internal mqtt broker  1 - external mqtt broker
           server: tcp://127.0.0.1:1883 # external mqtt broker url.
           internal-server: tcp://127.0.0.1:1884 # internal mqtt broker url.
         device:
           name: <device name>
           id: <deviceID>         #deviceID received while registering device with the cloud
         action-manager:
           actions:          # Multiple actions can be added
             - name: <name of the action>
               enable: <true/false>
               operation:
                 action: <Write/Read>
                 characteristic-uuid: <UUID of the characteristic example:- f000aa0304514000b000000000000000>
                 value: [1]              # value in bytes to be written by default if converter for this is not mentioned                  
         scheduler:
           schedules:
             - enable: <true/false>
               name: <name of schedule>
               event-frequency: <time in milliseconds>
               occurrence-limit: <number of times to be executed>            # if it is 0, then the event will execute infinitely
               actions:                        
                 - <action name>
                 - <action name> 
         watcher:
           enable: <true/false>
           attributes :
             - name: <name of attribute>                        # the twin attribute name defined while creating device 
               actions:        # Multiple actions can be added
                 - <action name>
                 - <action name>
         data-converter:
           write:  
             attributes:     
               - name: <name of attribute>                   # the twin attribute name defined while creating device  
                 operations :     
                   <action name> :
                     data-map:                   # to be given in byte format
                       <value coming from platform>: <byte value>
                       <value coming from platform>: <byte value>
           read:
             actions:
               - action-name: <name of action> # name of action for which conversion is to be performed   
                 conversion-operation:         # Only applicable operations/Values to be given, everything else is taken as nil
                   start-index: <start index of the incoming byte stream>        # ex:- start-index:2, end-index:3 concatenates the value present at second and third index of the incoming byte stream. If we want to reverse the order we can give it as start-index:3, end-index:2        
                   end-index: <end index of incoming byte stream>                 # the value specified should be inclusive for example if 3 is specified it includes the third index. 
                   shift-left: <number of bits to shift left, if necessary>
                   shift-right: <number of bits to shift right, if necessary> 
                   multiply: <value to be multiplied by, if necessary>
                   divide: <value to be divided by, if necessary>
                   add: <value to be added with, if necessary>
                   subtract: <value to be subtracted, if necessary>
                   order-of-execution:             # this refers to the order in which the add, subtract, multiply or divide operation is to be performed  
                     - <add/subtract/multiply/divide>
                     - <add/subtract/multiply/divide>


                
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

**Topic:** $ke/mappers/bluetooth-mapper/< deviceID >/watcher/create

**Message:**

             {
              "enable": true,          # whether the watch is to be started immediately or not
              "attributes": [
                {
                  "name": "IOControl",               # name of twin attribute defined in cloud
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

**Topic:** $ke/mappers/bluetooth-mapper/< deviceID >/action-manager/create

**Message:**

        [
          {
            "name": "IRTemperatureConfiguration",          # name of action
            "enable": true,                                   # whether the action is to performed immediately or not
            "operation": {
           "action": "Write",                             # Read or Write operation
              "characteristic-uuid": "f000aa0304514000b000000000000000",         # Characteristic UUID from where value should be read from or written into 
              "value": [1]                              # value to be written 
            }
          },
          {
            "name": "IRTemperatureData",
            "enable": true,
            "operation": {
              "action": "Read",
              "characteristic-uuid": "f000aa0304514000b000000000000000"
            }
          }
        ]

##### Action Delete

The users can delete a set of actions that were previously defined for the device. If the action mentioned in the list does not exist
then it returns an error message.

**Topic:** $ke/mappers/bluetooth-mapper/< deviceID >/action-manager/delete

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
 
**Topic:** $ke/mappers/bluetooth-mapper/< deviceID >/scheduler/create

**Message:**
    
    [
      {
        "enable": true,                                     #  whether the schedule  is to executed or not   
        "name": "temperature",            # name of schedule
        "event-frequency": 3000,           # frequency of the actions to be executed (in milliseconds)
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

**Topic:** $ke/mappers/bluetooth-mapper/< deviceID >/scheduler/delete

**Message:**

        [
          {
            "name": "temperature"                  #name of schedule to be deleted
          }
        ]