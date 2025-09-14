#ifndef COMMON_EVENT_H
#define COMMON_EVENT_H

#include <stdint.h>

// Metadata for type
typedef struct {
    char *type;
} TypeMetadata;

// Twin value
typedef struct {
    char *value;
} TwinValue;

// Twin property
typedef struct {
    TwinValue *actual;
    TwinValue *expected;
    TypeMetadata *metadata;
} MsgTwin;

// Base message with timestamp
typedef struct {
    int64_t timestamp;
} BaseMessage;

// DeviceTwinUpdate message
typedef struct {
    BaseMessage baseMessage;
    char *twinName;
    MsgTwin twin;
} DeviceTwinUpdate;

// Get current timestamp in ms
int64_t get_timestamp_ms(void);

// Create twin update message, output as JSON string
// Caller is responsible for freeing the returned string
char *create_message_twin_update(const char *name, const char *valueType, const char *value, const char *expectValue);

#endif // EVENT_H