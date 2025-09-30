#include "data/publish/publisher.h"
#include "data/publish/http/http_publisher.h"
#include "data/publish/mqtt/mqtt_publisher.h"
#include "data/publish/otel/otel_publisher.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>

// Convert method name string to PublishMethodType
PublishMethodType publisher_get_type_from_string(const char *method_name)
{
    if (!method_name)
        return PUBLISH_METHOD_UNKNOWN;
    if (!strcasecmp(method_name, "http"))
        return PUBLISH_METHOD_HTTP;
    if (!strcasecmp(method_name, "mqtt"))
        return PUBLISH_METHOD_MQTT;
    if (!strcasecmp(method_name, "otel"))
        return PUBLISH_METHOD_OTEL;
    return PUBLISH_METHOD_UNKNOWN;
}

// Convert PublishMethodType to string
const char *publisher_get_type_string(PublishMethodType type)
{
    switch (type)
    {
    case PUBLISH_METHOD_HTTP:
        return "http";
    case PUBLISH_METHOD_MQTT:
        return "mqtt";
    case PUBLISH_METHOD_OTEL:
        return "otel";
    default:
        return "unknown";
    }
}

// Create a new publisher
Publisher *publisher_new(PublishMethodType type, const char *config_json)
{
    if (type == PUBLISH_METHOD_UNKNOWN || !config_json)
        return NULL;
    Publisher *p = (Publisher *)calloc(1, sizeof(Publisher));
    if (!p)
        return NULL;
    p->type = type;
    p->config_json = strdup(config_json);

    switch (type)
    {
    case PUBLISH_METHOD_HTTP:
        p->client_handle = (void *)http_publisher_new(config_json);
        break;
    case PUBLISH_METHOD_MQTT:
        p->client_handle = (void *)mqtt_publisher_new(config_json);
        break;
    case PUBLISH_METHOD_OTEL:
        p->client_handle = (void *)otel_publisher_new(config_json);
        break;
    default:
        break;
    }
    if (!p->client_handle)
    {
        free(p->config_json);
        free(p);
        return NULL;
    }
    log_info("Publisher created: type=%s", publisher_get_type_string(type));
    return p;
}

// Free a publisher
void publisher_free(Publisher *publisher)
{
    if (!publisher)
        return;
    if (publisher->client_handle)
    {
        switch (publisher->type)
        {
        case PUBLISH_METHOD_HTTP:
            http_publisher_free((HttpPublisher *)publisher->client_handle);
            break;
        case PUBLISH_METHOD_MQTT:
            mqtt_publisher_free((MqttPublisher *)publisher->client_handle);
            break;
        case PUBLISH_METHOD_OTEL:
            otel_publisher_free((OtelPublisher *)publisher->client_handle);
            break;
        default:
            break;
        }
    }
    free(publisher->config_json);
    free(publisher);
}

// Publish data using the specified publisher
int publisher_publish_data(Publisher *publisher, const DataModel *data)
{
    if (!publisher || !publisher->client_handle || !data)
        return -1;
    switch (publisher->type)
    {
    case PUBLISH_METHOD_HTTP:
        return http_publisher_publish((HttpPublisher *)publisher->client_handle, data);
    case PUBLISH_METHOD_MQTT:
        return mqtt_publisher_publish((MqttPublisher *)publisher->client_handle, data);
    case PUBLISH_METHOD_OTEL:
        return otel_publisher_publish((OtelPublisher *)publisher->client_handle, data);
    default:
        return -1;
    }
}