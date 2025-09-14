#include "common/event.h"
#include <stdlib.h>
#include <stdio.h>
#include <time.h>
#include <cjson/cJSON.h>

// Get current timestamp in milliseconds
#include <sys/time.h>
int64_t get_timestamp_ms(void) {
    struct timeval tv;
    gettimeofday(&tv, NULL);
    return (int64_t)tv.tv_sec * 1000 + tv.tv_usec / 1000;
}

// Create twin update message as JSON string
char *create_message_twin_update(const char *name, const char *valueType, const char *value, const char *expectValue) {
    int64_t ts = get_timestamp_ms();

    cJSON *root = cJSON_CreateObject();
    cJSON *baseMsg = cJSON_CreateObject();
    cJSON *twinObj = cJSON_CreateObject();
    cJSON *twinItem = cJSON_CreateObject();
    cJSON *actual = cJSON_CreateObject();
    cJSON *expected = cJSON_CreateObject();
    cJSON *metadata = cJSON_CreateObject();

    cJSON_AddNumberToObject(baseMsg, "Timestamp", ts);
    cJSON_AddItemToObject(root, "BaseMessage", baseMsg);

    cJSON_AddStringToObject(actual, "Value", value);
    cJSON_AddItemToObject(twinItem, "Actual", actual);

    cJSON_AddStringToObject(expected, "Value", expectValue);
    cJSON_AddItemToObject(twinItem, "Expected", expected);

    cJSON_AddStringToObject(metadata, "Type", valueType);
    cJSON_AddItemToObject(twinItem, "Metadata", metadata);

    cJSON_AddItemToObject(twinObj, name, twinItem);
    cJSON_AddItemToObject(root, "Twin", twinObj);

    char *json_str = cJSON_PrintUnformatted(root);

    cJSON_Delete(root);
    return json_str; // caller must free
}