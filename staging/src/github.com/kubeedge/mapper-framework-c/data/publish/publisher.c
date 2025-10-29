#include "data/publish/publisher.h"
#include "data/publish/http/http_publisher.h"
#include "data/publish/mqtt/mqtt_publisher.h"
#include "data/publish/otel/otel_publisher.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include "../device/device.h"

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

typedef struct {
    char *key;        
    Publisher *pub;
} PubCacheEntry;
static PubCacheEntry g_pub_cache[8]; 

static int key_equal(const char *a, const char *b){ return a && b && strcmp(a,b)==0; }
static void cache_put(const char *key, Publisher *p) {
    for (int i = 0; i < (int)(sizeof(g_pub_cache)/sizeof(g_pub_cache[0])); ++i) {
        if (!g_pub_cache[i].key) {
            g_pub_cache[i].key = strdup(key);
            g_pub_cache[i].pub = p;
            return;
        }
    }
    if (g_pub_cache[0].pub) publisher_free(g_pub_cache[0].pub);
    free(g_pub_cache[0].key);
    g_pub_cache[0].key = strdup(key);
    g_pub_cache[0].pub = p;
}

int publisher_publish_dynamic(const char *methodName, const char *methodConfigJson, const DataModel *data) {
    if (!methodName || !*methodName || !data) return -1;
    char key[256]; snprintf(key, sizeof(key), "%s|%s", methodName, methodConfigJson ? methodConfigJson : "");
    for (int i = 0; i < (int)(sizeof(g_pub_cache)/sizeof(g_pub_cache[0])); ++i) {
        if (g_pub_cache[i].key && key_equal(g_pub_cache[i].key, key)) {
            return publisher_publish_data(g_pub_cache[i].pub, data);
        }
    }
    PublishMethodType t = publisher_get_type_from_string(methodName);
    Publisher *p = publisher_new(t, methodConfigJson);
    if (!p) {
        log_error("publish dynamic new publisher failed method=%s", methodName);
        return -1;
    }
    cache_put(key, p);
    return publisher_publish_data(p, data);
}

static DeviceProperty *find_property_in_device(const Device *device, const char *propName)
{
    if (!device || !propName) return NULL;
    for (int i = 0; i < device->instance.propertiesCount; ++i) {
        DeviceProperty *p = &device->instance.properties[i];
        if (p->propertyName && strcmp(p->propertyName, propName) == 0)
            return p;
    }
    return NULL;
}

int publisher_publish_from_device(const void *devptr, const char *propertyName, const char *value, long long timestamp)
{
    if (!devptr || !propertyName || !value) return -1;
    const Device *device = (const Device *)devptr;
    DeviceProperty *p = find_property_in_device(device, propertyName);
    if (!p) return -1;

    const char *ns = device->instance.namespace_ ? device->instance.namespace_ : "default";
    const char *dev = device->instance.name ? device->instance.name : "unknown";

    DataModel dm = {0};
    dm.namespace_   = (char *)ns;
    dm.deviceName   = (char *)dev;
    dm.propertyName = (char *)propertyName;
    dm.type         = (char *)"string";
    dm.value        = (char *)(value ? value : "");
    dm.timeStamp    = (int64_t)timestamp;

    if (p->pushMethod && p->pushMethod->methodName && p->pushMethod->methodConfig) {
        return publisher_publish_dynamic(p->pushMethod->methodName, p->pushMethod->methodConfig, &dm);
    }
    return -1;
}