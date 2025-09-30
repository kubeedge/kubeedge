#ifndef COMMON_EVENTTYPE_H
#define COMMON_EVENTTYPE_H

#include <stdint.h>

// BaseMessage: the base structure of event message.
typedef struct {
    char *event_id;
    int64_t timestamp;
} BaseMessage;

// ValueMetadata: the meta of value.
typedef struct {
    char *timestamp;
} ValueMetadata;

// TwinValue: the structure of twin value.
typedef struct {
    char *value; 
    ValueMetadata metadata;
} TwinValue;

// TypeMetadata: the meta of value type.
typedef struct {
    char *type; 
} TypeMetadata;

// TwinVersion: twin version.
typedef struct {
    int64_t cloud_version;
    int64_t edge_version;
} TwinVersion;

// MsgTwin: the structure of device twin.
typedef struct {
    TwinValue *expected;         
    TwinValue *actual;           
    int *optional;              
    TypeMetadata *metadata;     
    TwinVersion *expected_version; 
    TwinVersion *actual_version;   
} MsgTwin;

// DeviceTwinUpdate: the structure of device twin update.
typedef struct {
    BaseMessage base_message;
    char **twin_names;
    MsgTwin **twins;
    int twin_count;
} DeviceTwinUpdate;

// DeviceTwinResult: device get result.
typedef struct {
    BaseMessage base_message;
    char **twin_names;
    MsgTwin **twins;
    int twin_count;
} DeviceTwinResult;

// DeviceTwinDelta: twin delta.
typedef struct {
    BaseMessage base_message;
    char **twin_names;
    MsgTwin **twins;
    int twin_count;
    char **delta_keys;
    char **delta_values;
    int delta_count;
} DeviceTwinDelta;

// DataMetadata: data metadata.
typedef struct {
    int64_t timestamp;
    char *type;
} DataMetadata;

// DataValue: data value.
typedef struct {
    char *value;
    DataMetadata metadata;
} DataValue;

// DeviceData: device data structure.
typedef struct {
    BaseMessage base_message;
    char **data_names;
    DataValue **data_values;
    int data_count;
} DeviceData;

// MsgAttr: the struct of device attr
typedef struct {
    char *value;
    int *optional;           
    TypeMetadata *metadata;  
} MsgAttr;

// DeviceUpdate: device update.
typedef struct {
    BaseMessage base_message;
    char *state;
    char **attr_names;
    MsgAttr **attributes;
    int attr_count;
} DeviceUpdate;

#endif // EVENTTYPE_H