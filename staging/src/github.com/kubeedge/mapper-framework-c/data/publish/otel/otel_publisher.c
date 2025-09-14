#include "otel_publisher.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <cjson/cJSON.h>

// Parse OpenTelemetry configuration
int otel_parse_config(const char *json, OtelPublishConfig *config) {
    if (!json || !config) return -1;
    
    memset(config, 0, sizeof(OtelPublishConfig));
    
    cJSON *root = cJSON_Parse(json);
    if (!root) {
        log_error("Failed to parse OpenTelemetry config JSON");
        return -1;
    }
    
    cJSON *endpoint = cJSON_GetObjectItem(root, "endpoint");
    cJSON *service_name = cJSON_GetObjectItem(root, "serviceName");
    cJSON *service_version = cJSON_GetObjectItem(root, "serviceVersion");
    cJSON *timeout = cJSON_GetObjectItem(root, "timeout");
    
    config->endpoint = endpoint ? strdup(endpoint->valuestring) : strdup("http://localhost:4318/v1/metrics");
    config->service_name = service_name ? strdup(service_name->valuestring) : strdup("kubeedge-mapper");
    config->service_version = service_version ? strdup(service_version->valuestring) : strdup("1.0.0");
    config->timeout_ms = timeout ? timeout->valueint : 10000;
    
    cJSON_Delete(root);
    return 0;
}

// Free OpenTelemetry configuration
void otel_free_config(OtelPublishConfig *config) {
    if (!config) return;
    
    free(config->endpoint);
    free(config->service_name);
    free(config->service_version);
    memset(config, 0, sizeof(OtelPublishConfig));
}

// Create an OpenTelemetry publisher
OtelPublisher *otel_publisher_new(const char *config_json) {
    if (!config_json) return NULL;
    
    OtelPublisher *publisher = calloc(1, sizeof(OtelPublisher));
    if (!publisher) return NULL;
    
    // Parse configuration
    if (otel_parse_config(config_json, &publisher->config) != 0) {
        free(publisher);
        return NULL;
    }
    
    // Initialize libcurl
    publisher->curl = curl_easy_init();
    if (!publisher->curl) {
        otel_free_config(&publisher->config);
        free(publisher);
        return NULL;
    }
    
    // Set basic options
    curl_easy_setopt(publisher->curl, CURLOPT_TIMEOUT_MS, publisher->config.timeout_ms);
    curl_easy_setopt(publisher->curl, CURLOPT_FOLLOWLOCATION, 1L);
    
    // Set request headers
    publisher->headers = curl_slist_append(publisher->headers, "Content-Type: application/json");
    curl_easy_setopt(publisher->curl, CURLOPT_HTTPHEADER, publisher->headers);
    
    log_info("OpenTelemetry publisher created for endpoint: %s", publisher->config.endpoint);
    return publisher;
}

// Free an OpenTelemetry publisher
void otel_publisher_free(OtelPublisher *publisher) {
    if (!publisher) return;
    
    if (publisher->curl) {
        curl_easy_cleanup(publisher->curl);
    }
    
    if (publisher->headers) {
        curl_slist_free_all(publisher->headers);
    }
    
    otel_free_config(&publisher->config);
    free(publisher);
}

// Get the current timestamp in nanoseconds
static uint64_t get_nanoseconds() {
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    return (uint64_t)ts.tv_sec * 1000000000ULL + ts.tv_nsec;
}

// Publish metrics data to OpenTelemetry (synchronous)
int otel_publisher_publish(OtelPublisher *publisher, const DataModel *data) {
    if (!publisher || !data) return -1;
    
    uint64_t now_ns = get_nanoseconds();
    
    // Build OpenTelemetry metrics data
    cJSON *root = cJSON_CreateObject();
    cJSON *resource_metrics = cJSON_CreateArray();
    cJSON_AddItemToObject(root, "resourceMetrics", resource_metrics);
    
    cJSON *resource_metric = cJSON_CreateObject();
    cJSON_AddItemToArray(resource_metrics, resource_metric);
    
    // Resource information
    cJSON *resource = cJSON_CreateObject();
    cJSON_AddItemToObject(resource_metric, "resource", resource);
    
    cJSON *attributes = cJSON_CreateArray();
    cJSON_AddItemToObject(resource, "attributes", attributes);
    
    // Service name attribute
    cJSON *service_name_attr = cJSON_CreateObject();
    cJSON_AddItemToArray(attributes, service_name_attr);
    cJSON_AddStringToObject(service_name_attr, "key", "service.name");
    cJSON *service_name_value = cJSON_CreateObject();
    cJSON_AddItemToObject(service_name_attr, "value", service_name_value);
    cJSON_AddStringToObject(service_name_value, "stringValue", publisher->config.service_name);
    
    // Metrics data
    cJSON *scope_metrics = cJSON_CreateArray();
    cJSON_AddItemToObject(resource_metric, "scopeMetrics", scope_metrics);
    
    cJSON *scope_metric = cJSON_CreateObject();
    cJSON_AddItemToArray(scope_metrics, scope_metric);
    
    // Scope information
    cJSON *scope = cJSON_CreateObject();
    cJSON_AddItemToObject(scope_metric, "scope", scope);
    cJSON_AddStringToObject(scope, "name", "github.com/kubeedge/mapper-framework-c/data/publish/otel");
    
    // Metrics
    cJSON *metrics = cJSON_CreateArray();
    cJSON_AddItemToObject(scope_metric, "metrics", metrics);
    
    cJSON *metric = cJSON_CreateObject();
    cJSON_AddItemToArray(metrics, metric);
    
    cJSON_AddStringToObject(metric, "name", data->propertyName ? data->propertyName : "device_value");
    cJSON_AddStringToObject(metric, "description", "Device property value");
    
    // Gauge data
    cJSON *gauge = cJSON_CreateObject();
    cJSON_AddItemToObject(metric, "gauge", gauge);
    
    cJSON *data_points = cJSON_CreateArray();
    cJSON_AddItemToObject(gauge, "dataPoints", data_points);
    
    cJSON *data_point = cJSON_CreateObject();
    cJSON_AddItemToArray(data_points, data_point);
    
    // Device attributes
    cJSON *dp_attributes = cJSON_CreateArray();
    cJSON_AddItemToObject(data_point, "attributes", dp_attributes);
    
    // Device name attribute
    cJSON *device_name_attr = cJSON_CreateObject();
    cJSON_AddItemToArray(dp_attributes, device_name_attr);
    cJSON_AddStringToObject(device_name_attr, "key", "device_name");
    cJSON *device_name_value = cJSON_CreateObject();
    cJSON_AddItemToObject(device_name_attr, "value", device_name_value);
    cJSON_AddStringToObject(device_name_value, "stringValue", data->deviceName ? data->deviceName : "unknown");
    
    // Timestamp and value
    char timestamp_str[32];
    snprintf(timestamp_str, sizeof(timestamp_str), "%lu", now_ns);
    cJSON_AddStringToObject(data_point, "timeUnixNano", timestamp_str);
    
    if (data->value) {
        char *endptr;
        double numeric_value = strtod(data->value, &endptr);
        if (*endptr == '\0') {
            cJSON_AddNumberToObject(data_point, "asDouble", numeric_value);
        } else {
            cJSON_AddNumberToObject(data_point, "asDouble", (double)strlen(data->value));
        }
    } else {
        cJSON_AddNumberToObject(data_point, "asDouble", 0.0);
    }
    
    char *json_string = cJSON_PrintUnformatted(root);
    cJSON_Delete(root);
    
    if (!json_string) {
        log_error("Failed to create OpenTelemetry JSON data");
        return -1;
    }
    
    // Set URL and data
    curl_easy_setopt(publisher->curl, CURLOPT_URL, publisher->config.endpoint);
    curl_easy_setopt(publisher->curl, CURLOPT_POSTFIELDS, json_string);
    curl_easy_setopt(publisher->curl, CURLOPT_POST, 1L);
    
    // Perform the request
    CURLcode res = curl_easy_perform(publisher->curl);
    
    if (res == CURLE_OK) {
        long response_code;
        curl_easy_getinfo(publisher->curl, CURLINFO_RESPONSE_CODE, &response_code);
        
        if (response_code >= 200 && response_code < 300) {
            log_debug("OpenTelemetry publish success: %ld", response_code);
            free(json_string);
            return 0;
        } else {
            log_warn("OpenTelemetry publish failed with code: %ld", response_code);
        }
    } else {
        log_error("OpenTelemetry publish failed: %s", curl_easy_strerror(res));
    }
    
    free(json_string);
    return -1;
}