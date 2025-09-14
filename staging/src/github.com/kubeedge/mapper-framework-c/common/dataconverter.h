#ifndef COMMON_DATACONVERTER_H
#define COMMON_DATACONVERTER_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

// Convert string to int64, double, bool, etc.
int convert_to_int64(const char *value, int64_t *out);
int convert_to_double(const char *value, double *out);
int convert_to_bool(const char *value, bool *out);

// Convert various types to string
int int64_to_string(int64_t value, char *buf, size_t bufsize);
int double_to_string(double value, char *buf, size_t bufsize);
int bool_to_string(bool value, char *buf, size_t bufsize);

// Protobuf Any decode (only support Int32Value, StringValue, FloatValue, BoolValue, Int64Value)
#include "google/protobuf/any.pb-c.h"

typedef enum {
    ANY_TYPE_UNKNOWN = 0,
    ANY_TYPE_INT32,
    ANY_TYPE_STRING,
    ANY_TYPE_FLOAT,
    ANY_TYPE_BOOL,
    ANY_TYPE_INT64
} AnyValueType;

int decode_any_value(const Google__Protobuf__Any *any, AnyValueType *type, void *out_value);

// Get message type name from type_url
const char *get_message_type_name(const char *type_url);

#endif // DATACONVERTER_H