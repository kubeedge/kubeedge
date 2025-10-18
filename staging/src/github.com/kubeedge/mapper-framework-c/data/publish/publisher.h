#ifndef PUBLISHER_H
#define PUBLISHER_H

#include "common/datamodel.h"
#include "driver/driver.h"

// Enum for publish method types
typedef enum
{
    PUBLISH_METHOD_HTTP = 0,
    PUBLISH_METHOD_MQTT,
    PUBLISH_METHOD_OTEL,
    PUBLISH_METHOD_UNKNOWN
} PublishMethodType;

// Generic publisher interface
typedef struct
{
    PublishMethodType type;
    char *config_json;
    void *client_handle;
} Publisher;

// Publisher interface functions
Publisher *publisher_new(PublishMethodType type, const char *config_json);
void publisher_free(Publisher *publisher);
int publisher_publish_data(Publisher *publisher, const DataModel *data);
int publisher_publish_dynamic(const char *methodName, const char *methodConfigJson, const DataModel *data);

// Helper functions
PublishMethodType publisher_get_type_from_string(const char *method_name);
const char *publisher_get_type_string(PublishMethodType type);

#ifdef __cplusplus
extern "C"
{
#endif

#ifdef __cplusplus
}
#endif

#endif // PUBLISHER_H