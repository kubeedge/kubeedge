#ifndef HTTP_PUBLISHER_H
#define HTTP_PUBLISHER_H

#include "common/datamodel.h"
#include <curl/curl.h>

// HTTP publish configuration
typedef struct
{
    char *endpoint;
    char *method;
    char *auth_token;
    char *content_type;
    int timeout_ms;
    int retry_count;
} HttpPublishConfig;

// HTTP publisher client
typedef struct
{
    HttpPublishConfig config;
    CURL *curl;
    struct curl_slist *headers;
} HttpPublisher;

// Function declarations
int http_parse_config(const char *json, HttpPublishConfig *config);
void http_free_config(HttpPublishConfig *config);

HttpPublisher *http_publisher_new(const char *config_json);
void http_publisher_free(HttpPublisher *publisher);
int http_publisher_publish(HttpPublisher *publisher, const DataModel *data);

#endif // HTTP_PUBLISHER_H