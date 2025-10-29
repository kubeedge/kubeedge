#include "influxdb2_client.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <curl/curl.h>
#include <cjson/cJSON.h>

// Parses the InfluxDB2 client configuration
int influxdb2_parse_client_config(const char *json, Influxdb2ClientConfig *out)
{
    if (!out)
        return -1;

    if (json && *json)
    {
        cJSON *root = cJSON_Parse(json);
        if (root)
        {
            cJSON *jurl = cJSON_GetObjectItem(root, "url");
            cJSON *jorg = cJSON_GetObjectItem(root, "org");
            cJSON *jbucket = cJSON_GetObjectItem(root, "bucket");
            cJSON *jtoken = cJSON_GetObjectItem(root, "token");

            out->url = jurl && cJSON_IsString(jurl) ? strdup(jurl->valuestring) : NULL;
            out->org = jorg && cJSON_IsString(jorg) ? strdup(jorg->valuestring) : NULL;
            out->bucket = jbucket && cJSON_IsString(jbucket) ? strdup(jbucket->valuestring) : NULL;
            out->token = jtoken && cJSON_IsString(jtoken) ? strdup(jtoken->valuestring) : NULL;

            cJSON_Delete(root);

            /* env fallback for missing fields */
            if (!out->url)
                out->url = getenv("INFLUXDB_URL") ? strdup(getenv("INFLUXDB_URL")) : NULL;
            if (!out->org)
                out->org = getenv("INFLUXDB_ORG") ? strdup(getenv("INFLUXDB_ORG")) : NULL;
            if (!out->bucket)
                out->bucket = getenv("INFLUXDB_BUCKET") ? strdup(getenv("INFLUXDB_BUCKET")) : NULL;
            if (!out->token)
                out->token = getenv("TOKEN") ? strdup(getenv("TOKEN")) : NULL;
            return 0;
        }
        else
        {
            log_warn("influxdb2_parse_client_config: invalid JSON, falling back to env");
        }
    }

    const char *url = getenv("INFLUXDB_URL");
    const char *org = getenv("INFLUXDB_ORG");
    const char *bucket = getenv("INFLUXDB_BUCKET");
    const char *token = getenv("TOKEN");

    out->url = url ? strdup(url) : NULL;
    out->org = org ? strdup(org) : NULL;
    out->bucket = bucket ? strdup(bucket) : NULL;
    out->token = token ? strdup(token) : NULL;
    return 0;
}

// Parses the InfluxDB2 data configuration
int influxdb2_parse_data_config(const char *json, Influxdb2DataConfig *out)
{
    if (!out)
        return -1;

    if (json && *json)
    {
        cJSON *root = cJSON_Parse(json);
        if (root)
        {
            cJSON *jmeasurement = cJSON_GetObjectItem(root, "measurement");
            cJSON *jfieldKey = cJSON_GetObjectItem(root, "fieldKey");
            cJSON *jtag = cJSON_GetObjectItem(root, "tag");

            out->measurement = jmeasurement && cJSON_IsString(jmeasurement) ? strdup(jmeasurement->valuestring) : NULL;
            out->fieldKey = jfieldKey && cJSON_IsString(jfieldKey) ? strdup(jfieldKey->valuestring) : NULL;

            out->tag_keys = NULL;
            out->tag_values = NULL;
            out->tag_count = 0;
            if (jtag && cJSON_IsObject(jtag))
            {
                int count = cJSON_GetArraySize(jtag);
                out->tag_keys = calloc((size_t)count, sizeof(char *));
                out->tag_values = calloc((size_t)count, sizeof(char *));
                cJSON *item = NULL;
                int idx = 0;
                cJSON_ArrayForEach(item, jtag)
                {
                    if (item && item->string && cJSON_IsString(item))
                    {
                        out->tag_keys[idx] = strdup(item->string);
                        out->tag_values[idx] = strdup(item->valuestring);
                        idx++;
                    }
                }
                out->tag_count = idx;
            }
            cJSON_Delete(root);

            if (!out->measurement)
                out->measurement = getenv("INFLUXDB_MEASUREMENT") ? strdup(getenv("INFLUXDB_MEASUREMENT")) : NULL;
            if (!out->fieldKey)
                out->fieldKey = getenv("INFLUXDB_FIELDKEY") ? strdup(getenv("INFLUXDB_FIELDKEY")) : NULL;
            if (out->tag_count == 0)
            {
                const char *tags = getenv("INFLUXDB_TAGS");
                if (tags && *tags)
                {
                    int count = 1;
                    for (const char *p = tags; *p; ++p)
                        if (*p == ',')
                            ++count;
                    out->tag_keys = calloc((size_t)count, sizeof(char *));
                    out->tag_values = calloc((size_t)count, sizeof(char *));
                    char *tmp = strdup(tags);
                    char *saveptr1 = NULL;
                    char *pair = strtok_r(tmp, ",", &saveptr1);
                    while (pair)
                    {
                        char *saveptr2 = NULL;
                        char *k = strtok_r(pair, "=", &saveptr2);
                        char *v = strtok_r(NULL, "=", &saveptr2);
                        out->tag_keys[out->tag_count] = k ? strdup(k) : strdup("");
                        out->tag_values[out->tag_count] = v ? strdup(v) : strdup("");
                        out->tag_count++;
                        pair = strtok_r(NULL, ",", &saveptr1);
                    }
                    free(tmp);
                }
            }
            return 0;
        }
        else
        {
            log_warn("influxdb2_parse_data_config: invalid JSON, falling back to env");
        }
    }

    /* fallback to env */
    const char *measurement = getenv("INFLUXDB_MEASUREMENT");
    const char *fieldKey = getenv("INFLUXDB_FIELDKEY");
    const char *tags = getenv("INFLUXDB_TAGS");

    out->measurement = measurement ? strdup(measurement) : NULL;
    out->fieldKey = fieldKey ? strdup(fieldKey) : NULL;
    out->tag_keys = NULL;
    out->tag_values = NULL;
    out->tag_count = 0;
    if (tags && *tags)
    {
        int count = 1;
        for (const char *p = tags; *p; ++p)
            if (*p == ',')
                ++count;
        out->tag_keys = calloc((size_t)count, sizeof(char *));
        out->tag_values = calloc((size_t)count, sizeof(char *));
        char *tmp = strdup(tags);
        char *saveptr1 = NULL;
        char *pair = strtok_r(tmp, ",", &saveptr1);
        while (pair)
        {
            char *saveptr2 = NULL;
            char *k = strtok_r(pair, "=", &saveptr2);
            char *v = strtok_r(NULL, "=", &saveptr2);
            out->tag_keys[out->tag_count] = k ? strdup(k) : strdup("");
            out->tag_values[out->tag_count] = v ? strdup(v) : strdup("");
            out->tag_count++;
            pair = strtok_r(NULL, ",", &saveptr1);
        }
        free(tmp);
    }
    return 0;
}

// Initializes the InfluxDB2 client
int influxdb2_init_client(const Influxdb2ClientConfig *cfg, Influxdb2Client *client)
{
    client->curl = curl_easy_init();
    return client->curl ? 0 : -1;
}

// Closes the InfluxDB2 client
void influxdb2_close_client(Influxdb2Client *client)
{
    if (client->curl)
        curl_easy_cleanup(client->curl);
    client->curl = NULL;
}

// Writes data to InfluxDB2
int influxdb2_add_data(const Influxdb2ClientConfig *client_cfg, const Influxdb2DataConfig *data_cfg, Influxdb2Client *client, const DataModel *data)
{
    if (!client || !client->curl || !client_cfg || !data_cfg || !data)
        return -1;
    char line[1024] = {0};
    int offset = 0;
    offset += snprintf(line + offset, sizeof(line) - offset, "%s", data_cfg->measurement ? data_cfg->measurement : "measurement");
    for (int i = 0; i < data_cfg->tag_count; ++i)
    {
        offset += snprintf(line + offset, sizeof(line) - offset, ",%s=%s", data_cfg->tag_keys[i], data_cfg->tag_values[i]);
    }
    offset += snprintf(line + offset, sizeof(line) - offset, " %s=\"%s\"", data_cfg->fieldKey ? data_cfg->fieldKey : "value", data->value ? data->value : "");
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
    if (res != CURLE_OK)
    {
        log_error("InfluxDB write failed: %s", curl_easy_strerror(res));
        return -1;
    }
    return 0;
}