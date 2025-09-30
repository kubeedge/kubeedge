#include "http_publisher.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <cjson/cJSON.h>

// Parse HTTP configuration
int http_parse_config(const char *json, HttpPublishConfig *config)
{
    if (!json || !config)
        return -1;

    memset(config, 0, sizeof(HttpPublishConfig));

    cJSON *root = cJSON_Parse(json);
    if (!root)
    {
        log_error("Failed to parse HTTP config JSON");
        return -1;
    }

    cJSON *endpoint = cJSON_GetObjectItem(root, "endpoint");
    cJSON *method = cJSON_GetObjectItem(root, "method");
    cJSON *auth_token = cJSON_GetObjectItem(root, "authToken");
    cJSON *content_type = cJSON_GetObjectItem(root, "contentType");
    cJSON *timeout = cJSON_GetObjectItem(root, "timeout");
    cJSON *retry = cJSON_GetObjectItem(root, "retryCount");

    config->endpoint = endpoint ? strdup(endpoint->valuestring) : strdup("http://localhost:8080/api/data");
    config->method = method ? strdup(method->valuestring) : strdup("POST");
    config->auth_token = auth_token ? strdup(auth_token->valuestring) : NULL;
    config->content_type = content_type ? strdup(content_type->valuestring) : strdup("application/json");
    config->timeout_ms = timeout ? timeout->valueint : 10000;
    config->retry_count = retry ? retry->valueint : 3;

    cJSON_Delete(root);
    return 0;
}

// Free HTTP configuration
void http_free_config(HttpPublishConfig *config)
{
    if (!config)
        return;

    free(config->endpoint);
    free(config->method);
    free(config->auth_token);
    free(config->content_type);
    memset(config, 0, sizeof(HttpPublishConfig));
}

// HTTP response callback
static size_t http_response_callback(void *contents, size_t size, size_t nmemb, void *userp)
{
    size_t realsize = size * nmemb;
    return realsize;
}

// Create HTTP publisher
HttpPublisher *http_publisher_new(const char *config_json)
{
    if (!config_json)
        return NULL;

    HttpPublisher *publisher = calloc(1, sizeof(HttpPublisher));
    if (!publisher)
        return NULL;

    if (http_parse_config(config_json, &publisher->config) != 0)
    {
        free(publisher);
        return NULL;
    }

    publisher->curl = curl_easy_init();
    if (!publisher->curl)
    {
        http_free_config(&publisher->config);
        free(publisher);
        return NULL;
    }

    curl_easy_setopt(publisher->curl, CURLOPT_TIMEOUT_MS, publisher->config.timeout_ms);
    curl_easy_setopt(publisher->curl, CURLOPT_FOLLOWLOCATION, 1L);
    curl_easy_setopt(publisher->curl, CURLOPT_WRITEFUNCTION, http_response_callback);

    char content_type_header[256];
    snprintf(content_type_header, sizeof(content_type_header), "Content-Type: %s",
             publisher->config.content_type);
    publisher->headers = curl_slist_append(publisher->headers, content_type_header);

    if (publisher->config.auth_token)
    {
        char auth_header[512];
        snprintf(auth_header, sizeof(auth_header), "Authorization: Bearer %s",
                 publisher->config.auth_token);
        publisher->headers = curl_slist_append(publisher->headers, auth_header);
    }

    curl_easy_setopt(publisher->curl, CURLOPT_HTTPHEADER, publisher->headers);
    log_info("HTTP publisher created for endpoint: %s", publisher->config.endpoint);
    return publisher;
}

// Free HTTP publisher
void http_publisher_free(HttpPublisher *publisher)
{
    if (!publisher)
        return;

    if (publisher->curl)
    {
        curl_easy_cleanup(publisher->curl);
    }

    if (publisher->headers)
    {
        curl_slist_free_all(publisher->headers);
    }

    http_free_config(&publisher->config);
    free(publisher);
}

// Publish data to HTTP (synchronous)
int http_publisher_publish(HttpPublisher *publisher, const DataModel *data)
{
    if (!publisher || !data)
        return -1;

    cJSON *json = cJSON_CreateObject();
    cJSON_AddStringToObject(json, "deviceName", data->deviceName ? data->deviceName : "");
    cJSON_AddStringToObject(json, "namespace", data->namespace_ ? data->namespace_ : "");
    cJSON_AddStringToObject(json, "propertyName", data->propertyName ? data->propertyName : "");
    cJSON_AddStringToObject(json, "value", data->value ? data->value : "");
    cJSON_AddStringToObject(json, "type", data->type ? data->type : "string");
    cJSON_AddNumberToObject(json, "timestamp", data->timeStamp);

    char *json_string = cJSON_PrintUnformatted(json);
    cJSON_Delete(json);

    if (!json_string)
    {
        log_error("Failed to create JSON data");
        return -1;
    }

    curl_easy_setopt(publisher->curl, CURLOPT_URL, publisher->config.endpoint);
    curl_easy_setopt(publisher->curl, CURLOPT_POSTFIELDS, json_string);

    if (strcmp(publisher->config.method, "PUT") == 0)
    {
        curl_easy_setopt(publisher->curl, CURLOPT_CUSTOMREQUEST, "PUT");
    }
    else
    {
        curl_easy_setopt(publisher->curl, CURLOPT_POST, 1L);
    }

    CURLcode res;
    int retry_count = 0;
    do
    {
        res = curl_easy_perform(publisher->curl);
        if (res == CURLE_OK)
        {
            long response_code;
            curl_easy_getinfo(publisher->curl, CURLINFO_RESPONSE_CODE, &response_code);

            if (response_code >= 200 && response_code < 300)
            {
                log_debug("HTTP publish success: %ld", response_code);
                free(json_string);
                return 0;
            }
            else
            {
                log_warn("HTTP publish failed with code: %ld", response_code);
            }
        }
        else
        {
            log_warn("HTTP publish failed: %s (attempt %d/%d)",
                     curl_easy_strerror(res), retry_count + 1, publisher->config.retry_count);
        }

        retry_count++;
    } while (retry_count < publisher->config.retry_count);

    free(json_string);
    log_error("HTTP publish failed after %d attempts", publisher->config.retry_count);
    return -1;
}