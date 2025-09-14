#include "common/dataconverter.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include "google/protobuf/wrappers.pb-c.h"

// Convert string to int64
int convert_to_int64(const char *value, int64_t *out) {
    if (!value || !out) return -1;
    char *endptr = NULL;
    *out = strtoll(value, &endptr, 10);
    return (*endptr == '\0') ? 0 : -1;
}

// Convert string to double
int convert_to_double(const char *value, double *out) {
    if (!value || !out) return -1;
    char *endptr = NULL;
    *out = strtod(value, &endptr);
    return (*endptr == '\0') ? 0 : -1;
}

// Convert string to bool
int convert_to_bool(const char *value, bool *out) {
    if (!value || !out) return -1;
    if (strcmp(value, "true") == 0 || strcmp(value, "1") == 0) {
        *out = true;
        return 0;
    }
    if (strcmp(value, "false") == 0 || strcmp(value, "0") == 0) {
        *out = false;
        return 0;
    }
    return -1;
}

// Convert int64 to string
int int64_to_string(int64_t value, char *buf, size_t bufsize) {
    if (!buf) return -1;
    int n = snprintf(buf, bufsize, "%lld", (long long)value);
    return (n > 0 && (size_t)n < bufsize) ? 0 : -1;
}

// Convert double to string
int double_to_string(double value, char *buf, size_t bufsize) {
    if (!buf) return -1;
    int n = snprintf(buf, bufsize, "%f", value);
    return (n > 0 && (size_t)n < bufsize) ? 0 : -1;
}

// Convert bool to string
int bool_to_string(bool value, char *buf, size_t bufsize) {
    if (!buf) return -1;
    int n = snprintf(buf, bufsize, "%s", value ? "true" : "false");
    return (n > 0 && (size_t)n < bufsize) ? 0 : -1;
}

// Get message type name from type_url
const char *get_message_type_name(const char *type_url) {
    const char *slash = strrchr(type_url, '/');
    if (slash && *(slash + 1)) {
        return slash + 1;
    }
    return "";
}

// Decode protobuf Any value (only support Int32Value, StringValue, FloatValue, BoolValue, Int64Value)

int decode_any_value(const Google__Protobuf__Any *any, AnyValueType *type, void *out_value) {
    if (!any || !type || !out_value) return -1;
    const char *type_name = get_message_type_name(any->type_url);

    if (strcmp(type_name, "Int32Value") == 0) {
        Google__Protobuf__Int32Value *val = google__protobuf__int32_value__unpack(NULL, any->value.len, any->value.data);
        if (!val) return -1;
        *(int32_t *)out_value = val->value;
        *type = ANY_TYPE_INT32;
        google__protobuf__int32_value__free_unpacked(val, NULL);
        return 0;
    } else if (strcmp(type_name, "StringValue") == 0) {
        Google__Protobuf__StringValue *val = google__protobuf__string_value__unpack(NULL, any->value.len, any->value.data);
        if (!val) return -1;
        *(char **)out_value = strdup(val->value);
        *type = ANY_TYPE_STRING;
        google__protobuf__string_value__free_unpacked(val, NULL);
        return 0;
    } else if (strcmp(type_name, "FloatValue") == 0) {
        Google__Protobuf__FloatValue *val = google__protobuf__float_value__unpack(NULL, any->value.len, any->value.data);
        if (!val) return -1;
        *(float *)out_value = val->value;
        *type = ANY_TYPE_FLOAT;
        google__protobuf__float_value__free_unpacked(val, NULL);
        return 0;
    } else if (strcmp(type_name, "BoolValue") == 0) {
        Google__Protobuf__BoolValue *val = google__protobuf__bool_value__unpack(NULL, any->value.len, any->value.data);
        if (!val) return -1;
        *(bool *)out_value = val->value;
        *type = ANY_TYPE_BOOL;
        google__protobuf__bool_value__free_unpacked(val, NULL);
        return 0;
    } else if (strcmp(type_name, "Int64Value") == 0) {
        Google__Protobuf__Int64Value *val = google__protobuf__int64_value__unpack(NULL, any->value.len, any->value.data);
        if (!val) return -1;
        *(int64_t *)out_value = val->value;
        *type = ANY_TYPE_INT64;
        google__protobuf__int64_value__free_unpacked(val, NULL);
        return 0;
    }
    *type = ANY_TYPE_UNKNOWN;
    return -1;
}