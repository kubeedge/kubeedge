#ifndef PUBLISHER_H
#define PUBLISHER_H

#include "common/datamodel.h"
#include "driver/driver.h"

// Enum for publish method types
typedef enum {
    PUBLISH_METHOD_HTTP = 0,
    PUBLISH_METHOD_MQTT,
    PUBLISH_METHOD_OTEL,
    PUBLISH_METHOD_UNKNOWN
} PublishMethodType;

// Generic publisher interface
typedef struct {
    PublishMethodType type;   // Publish method type
    char *config_json;        // Publish configuration in JSON format
    void *client_handle;      // Specific client handle
} Publisher;

// Publisher interface functions
Publisher *publisher_new(PublishMethodType type, const char *config_json);
void publisher_free(Publisher *publisher);
int publisher_publish_data(Publisher *publisher, const DataModel *data);

// Helper functions
PublishMethodType publisher_get_type_from_string(const char *method_name);
const char *publisher_get_type_string(PublishMethodType type);

#endif // PUBLISHER_H