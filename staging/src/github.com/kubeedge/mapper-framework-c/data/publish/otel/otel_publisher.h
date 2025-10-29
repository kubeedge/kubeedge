#ifndef OTEL_PUBLISHER_H
#define OTEL_PUBLISHER_H

#include "common/datamodel.h"
#include <curl/curl.h>

// OpenTelemetry publish configuration
typedef struct
{
    char *endpoint;
    char *service_name;
    char *service_version;
    int timeout_ms;
} OtelPublishConfig;

// OpenTelemetry publisher
typedef struct
{
    OtelPublishConfig config;
    CURL *curl;
    struct curl_slist *headers;
} OtelPublisher;

// Function declarations
int otel_parse_config(const char *json, OtelPublishConfig *config);
void otel_free_config(OtelPublishConfig *config);

OtelPublisher *otel_publisher_new(const char *config_json);
void otel_publisher_free(OtelPublisher *publisher);
int otel_publisher_publish(OtelPublisher *publisher, const DataModel *data);

#endif // OTEL_PUBLISHER_H