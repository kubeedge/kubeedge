#ifndef HTTP_PUBLISHER_H
#define HTTP_PUBLISHER_H

#include "common/datamodel.h"
#include <curl/curl.h>

// HTTP publish configuration
typedef struct {
    char *endpoint;      // HTTP endpoint URL
    char *method;        // HTTP method (POST/PUT)
    char *auth_token;    // Authentication token
    char *content_type;  // Content type
    int timeout_ms;      // Timeout in milliseconds
    int retry_count;     // Number of retry attempts
} HttpPublishConfig;

// HTTP publisher client
typedef struct {
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