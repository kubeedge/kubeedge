#ifndef COMMON_CONFIGMAPTYPE_H
#define COMMON_CONFIGMAPTYPE_H

#include <stdint.h>
#include <stdbool.h>

// ProtocolConfig stores protocol information in device.
typedef struct {
    char *protocolName;  
    char *configData;     
} ProtocolConfig;

// DeviceMethod stores method information in device.
typedef struct {
    char *name;            
    char *description;
    char **propertyNames;   
    int propertyNamesCount; 
} DeviceMethod;

// DeviceStatus stores parameters for device status reporting.
typedef struct {
    bool reportToCloud; 
    int64_t reportCycle;
    char *status;
    char *lastStatus;
    long long lastUpdateTime;
    int  healthCheckInterval;
    int  statusChangeCount;
} DeviceStatus;

// ModelProperty stores device model property information.
typedef struct {
    char *name;
    char *dataType;
    char *description;
    char *accessMode;
    char *minimum; 
    char *maximum;
    char *unit;
} ModelProperty;

// DBConfig stores database configuration.
typedef struct DBConfig {
    char *influxdb2ClientConfig;
    char *influxdb2DataConfig;
    char *redisClientConfig;
    char *tdengineClientConfig; 
    char *mysqlClientConfig;    
} DBConfig;

// DBMethodConfig stores database method configuration.
typedef struct {
    char *dbMethodName;     
    DBConfig *dbConfig;     
} DBMethodConfig;

// PushMethodConfig stores push configuration.
typedef struct {
    char *methodName;       
    char *methodConfig;      
    DBMethodConfig *dbMethod;
} PushMethodConfig;

// DeviceProperty stores property visitor information in device.
typedef struct {
    char *name;              
    char *propertyName;      
    char *modelName;         
    char *protocol;          
    char *visitors;          
    bool reportToCloud;      
    int64_t collectCycle;    
    int64_t reportCycle;     
    PushMethodConfig *pushMethod;
    ModelProperty *pProperty;    
} DeviceProperty;

// Metadata is the metadata for data.
typedef struct {
    char *timestamp; 
    char *type;      
} Metadata;

// TwinProperty is the value and metadata for a twin property.
typedef struct {
    char *value;     
    Metadata metadata;
} TwinProperty;

// Twin is the set/get pair to one register.
typedef struct Twin {
    char *propertyName;  
    DeviceProperty *property;
    TwinProperty observedDesired;
    TwinProperty reported;       
} Twin;

// DeviceInstance stores detailed information about the device in the mapper.
typedef struct {
    char *id;           
    char *name;           
    char *namespace_;   
    char *protocolName;     
    ProtocolConfig pProtocol; 
    char *model;         
    Twin *twins;             
    int twinsCount;           
    DeviceProperty *properties;  
    int propertiesCount;         
    DeviceMethod *methods;      
    int methodsCount;           
    DeviceStatus status;    
} DeviceInstance;

// DeviceModel stores detailed information about the device model in the mapper.
typedef struct {
    char *id;                   
    char *name;                 
    char *namespace_;       
    char *description;      
    ModelProperty *properties;  
    int propertiesCount;        
} DeviceModel;

#endif // COMMON_CONFIGMAPTYPE_H