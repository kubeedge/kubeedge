#ifndef COMMON_EVENTTYPE_H
#define COMMON_EVENTTYPE_H

#include <stdint.h>

// BaseMessage: the base structure of event message.
typedef struct {
    char *event_id;   // Event ID
    int64_t timestamp; // Timestamp
} BaseMessage;

// ValueMetadata: the meta of value.
typedef struct {
    char *timestamp; // Timestamp as string
} ValueMetadata;

// TwinValue: the structure of twin value.
typedef struct {
    char *value;           // Value (nullable)
    ValueMetadata metadata; // Metadata
} TwinValue;

// TypeMetadata: the meta of value type.
typedef struct {
    char *type; // Type name
} TypeMetadata;

// TwinVersion: twin version.
typedef struct {
    int64_t cloud_version;
    int64_t edge_version;
} TwinVersion;

// MsgTwin: the structure of device twin.
typedef struct {
    TwinValue *expected;         // Pointer, can be NULL
    TwinValue *actual;           // Pointer, can be NULL
    int *optional;              // Pointer, can be NULL
    TypeMetadata *metadata;      // Pointer, can be NULL
    TwinVersion *expected_version; // Pointer, can be NULL
    TwinVersion *actual_version;   // Pointer, can be NULL
} MsgTwin;

// DeviceTwinUpdate: the structure of device twin update.
typedef struct {
    BaseMessage base_message;
    // Map from twin name to MsgTwin pointer (use array + count in C)
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
    // Delta: map from string to string (use array + count)
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
    // Map from data name to DataValue pointer (use array + count)
    char **data_names;
    DataValue **data_values;
    int data_count;
} DeviceData;

// MsgAttr: the struct of device attr
typedef struct {
    char *value;
    int *optional;           // Pointer, can be NULL
    TypeMetadata *metadata;   // Pointer, can be NULL
} MsgAttr;

// DeviceUpdate: device update.
typedef struct {
    BaseMessage base_message;
    char *state; // Optional
    // Map from attribute name to MsgAttr pointer (use array + count)
    char **attr_names;
    MsgAttr **attributes;
    int attr_count;
} DeviceUpdate;

#endif // EVENTTYPE_H