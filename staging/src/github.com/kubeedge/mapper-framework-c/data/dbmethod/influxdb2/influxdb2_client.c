#include "influxdb2_client.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <curl/curl.h>
#include <cjson/cJSON.h>

// Parses the InfluxDB2 client configuration
int influxdb2_parse_client_config(const char *json, Influxdb2ClientConfig *out) {
    cJSON *root = cJSON_Parse(json);
    if (!root) return -1;
    cJSON *url = cJSON_GetObjectItem(root, "url");
    cJSON *org = cJSON_GetObjectItem(root, "org");
    cJSON *bucket = cJSON_GetObjectItem(root, "bucket");
    out->url = url ? strdup(url->valuestring) : NULL;
    out->org = org ? strdup(org->valuestring) : NULL;
    out->bucket = bucket ? strdup(bucket->valuestring) : NULL;
    // Token is read from the environment variable
    char *token_env = getenv("TOKEN");
    out->token = token_env ? strdup(token_env) : NULL;
    cJSON_Delete(root);
    return 0;
}

// Parses the InfluxDB2 data configuration
int influxdb2_parse_data_config(const char *json, Influxdb2DataConfig *out) {
    cJSON *root = cJSON_Parse(json);
    if (!root) return -1;
    cJSON *measurement = cJSON_GetObjectItem(root, "measurement");
    cJSON *fieldKey = cJSON_GetObjectItem(root, "fieldKey");
    out->measurement = measurement ? strdup(measurement->valuestring) : NULL;
    out->fieldKey = fieldKey ? strdup(fieldKey->valuestring) : NULL;
    // Parse tags as key-value pairs
    cJSON *tag = cJSON_GetObjectItem(root, "tag");
    if (tag && cJSON_IsObject(tag)) {
        int count = cJSON_GetArraySize(tag);
        out->tag_keys = calloc(count, sizeof(char*));
        out->tag_values = calloc(count, sizeof(char*));
        out->tag_count = 0;
        cJSON *item = NULL;
        cJSON_ArrayForEach(item, tag) {
            out->tag_keys[out->tag_count] = strdup(item->string);
            out->tag_values[out->tag_count] = strdup(item->valuestring);
            out->tag_count++;
        }
    } else {
        out->tag_keys = NULL;
        out->tag_values = NULL;
        out->tag_count = 0;
    }
    cJSON_Delete(root);
    return 0;
}

// Initializes the InfluxDB2 client
int influxdb2_init_client(const Influxdb2ClientConfig *cfg, Influxdb2Client *client) {
    client->curl = curl_easy_init();
    return client->curl ? 0 : -1;
}

// Closes the InfluxDB2 client
void influxdb2_close_client(Influxdb2Client *client) {
    if (client->curl) curl_easy_cleanup(client->curl);
    client->curl = NULL;
}

// Writes data to InfluxDB2
int influxdb2_add_data(const Influxdb2ClientConfig *client_cfg, const Influxdb2DataConfig *data_cfg, Influxdb2Client *client, const DataModel *data) {
    if (!client || !client->curl || !client_cfg || !data_cfg || !data) return -1;
    // Assemble the line protocol
    char line[1024] = {0};
    int offset = 0;
    offset += snprintf(line + offset, sizeof(line) - offset, "%s", data_cfg->measurement ? data_cfg->measurement : "measurement");
    for (int i = 0; i < data_cfg->tag_count; ++i) {
        offset += snprintf(line + offset, sizeof(line) - offset, ",%s=%s", data_cfg->tag_keys[i], data_cfg->tag_values[i]);
    }
    offset += snprintf(line + offset, sizeof(line) - offset, " %s=\"%s\"", data_cfg->fieldKey ? data_cfg->fieldKey : "value", data->value ? data->value : "");
    // Send HTTP request
    char url[512];
    snprintf(url, sizeof(url), "%s/api/v2/write?org=%s&bucket=%s&precision=ns", client_cfg->url, client_cfg->org, client_cfg->bucket);
    struct curl_slist *headers = NULL;
    char auth_header[256];
    snprintf(auth_header, sizeof(auth_header), "Authorization: Token %s", client_cfg->token ? client_cfg->token : "");
    headers = curl_slist_append(headers, auth_header);
    headers = curl_slist_append(headers, "Content-Type: text/plain");
    curl_easy_setopt(client->curl, CURLOPT_URL, url);
    curl_easy_setopt(client->curl, CURLOPT_POSTFIELDS, line);
    curl_easy_setopt(client->curl, CURLOPT_HTTPHEADER, headers);
    CURLcode res = curl_easy_perform(client->curl);
    curl_slist_free_all(headers);
    if (res != CURLE_OK) {
        log_error("InfluxDB write failed: %s", curl_easy_strerror(res));
        return -1;
    }
    return 0;
}