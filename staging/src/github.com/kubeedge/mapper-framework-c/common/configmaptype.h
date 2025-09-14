#ifndef COMMON_CONFIGMAPTYPE_H
#define COMMON_CONFIGMAPTYPE_H

#include <stdint.h>
#include <stdbool.h>

// ProtocolConfig stores protocol information in device.
typedef struct {
    char *protocolName;   // Unique protocol name (Required)
    char *configData;     // Any config data, recommended as JSON string
} ProtocolConfig;

// DeviceMethod stores method information in device.
typedef struct {
    char *name;              // Device method name (Required, must be unique)
    char *description;       // Description of device method (Optional)
    char **propertyNames;    // List of device properties that this method can control (Required)
    int propertyNamesCount;  // Number of property names
} DeviceMethod;

// DeviceStatus stores parameters for device status reporting.
typedef struct {
    bool reportToCloud;   // Whether to report to the cloud
    int64_t reportCycle;  // Report cycle in seconds
    char *status;
    char *lastStatus;
    long long lastUpdateTime;
    int  healthCheckInterval;
    int  statusChangeCount;
} DeviceStatus;

// ModelProperty stores device model property information.
typedef struct {
    char *name;         // Property name
    char *dataType;     // Data type
    char *description;  // Description
    char *accessMode;   // Access mode
    char *minimum;      // Minimum value
    char *maximum;      // Maximum value
    char *unit;         // Unit
} ModelProperty;

// DBConfig stores database configuration.
typedef struct DBConfig {
    char *influxdb2ClientConfig;   // InfluxDB2 client config (JSON string)
    char *influxdb2DataConfig;     // InfluxDB2 data config (JSON string)
    char *redisClientConfig;       // Redis client config (JSON string)
    char *tdengineClientConfig;    // TDengine client config (JSON string)
    char *mysqlClientConfig;       // MySQL client config (JSON string)
} DBConfig;

// DBMethodConfig stores database method configuration.
typedef struct {
    char *dbMethodName;     // Database method name
    DBConfig *dbConfig;     // Pointer to database config
} DBMethodConfig;

// PushMethodConfig stores push configuration.
typedef struct {
    char *methodName;           // Push method name
    char *methodConfig;         // Push method config, recommended as JSON string
    DBMethodConfig *dbMethod;   // Pointer to database method config
} PushMethodConfig;

// DeviceProperty stores property visitor information in device.
typedef struct {
    char *name;                // Device property name
    char *propertyName;        // Property name
    char *modelName;           // Model name
    char *protocol;            // Protocol name
    char *visitors;            // Visitor config, recommended as JSON string
    bool reportToCloud;        // Whether to report to the cloud
    int64_t collectCycle;      // Collect cycle in seconds
    int64_t reportCycle;       // Report cycle in seconds
    PushMethodConfig *pushMethod; // Pointer to push method config
    ModelProperty *pProperty;     // Pointer to model property
} DeviceProperty;

// Metadata is the metadata for data.
typedef struct {
    char *timestamp;   // Timestamp
    char *type;        // Data type
} Metadata;

// TwinProperty is the value and metadata for a twin property.
typedef struct {
    char *value;      // Value for this property (Required)
    Metadata metadata; // Metadata (Optional)
} TwinProperty;

// Twin is the set/get pair to one register.
typedef struct Twin {
    char *propertyName;         // Twin property name
    DeviceProperty *property;   // Pointer to device property
    TwinProperty observedDesired; // Observed desired value
    TwinProperty reported;        // Reported value
} Twin;

// DeviceInstance stores detailed information about the device in the mapper.
typedef struct {
    char *id;                   // Device ID
    char *name;                 // Device name
    char *namespace_;           // Namespace
    char *protocolName;         // Protocol name
    ProtocolConfig pProtocol;   // Protocol config
    char *model;                // Model name
    Twin *twins;                // Array of twins
    int twinsCount;             // Number of twins
    DeviceProperty *properties; // Array of device properties
    int propertiesCount;        // Number of properties
    DeviceMethod *methods;      // Array of device methods
    int methodsCount;           // Number of methods
    DeviceStatus status;        // Device status
} DeviceInstance;

// DeviceModel stores detailed information about the device model in the mapper.
typedef struct {
    char *id;                   // Model ID
    char *name;                 // Model name
    char *namespace_;           // Namespace
    char *description;          // Description
    ModelProperty *properties;  // Array of model properties
    int propertiesCount;        // Number of properties
} DeviceModel;

#endif // COMMON_CONFIGMAPTYPE_H