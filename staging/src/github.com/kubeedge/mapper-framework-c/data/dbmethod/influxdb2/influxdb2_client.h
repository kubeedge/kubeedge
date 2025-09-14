#ifndef INFLUXDB2_CLIENT_H
#define INFLUXDB2_CLIENT_H
#include <curl/curl.h>
#include "common/datamodel.h"

typedef struct {
    char *url;
    char *org;
    char *bucket;
    char *token;
} Influxdb2ClientConfig;

typedef struct {
    char *measurement;
    char **tag_keys;
    char **tag_values;
    int tag_count;
    char *fieldKey;
} Influxdb2DataConfig;

typedef struct {
    Influxdb2ClientConfig clientConfig;
    Influxdb2DataConfig dataConfig;
} Influxdb2DataBaseConfig;

typedef struct {
    CURL *curl;
} Influxdb2Client;

int influxdb2_parse_client_config(const char *json, Influxdb2ClientConfig *out);
int influxdb2_parse_data_config(const char *json, Influxdb2DataConfig *out);

int influxdb2_init_client(const Influxdb2ClientConfig *cfg, Influxdb2Client *client);
void influxdb2_close_client(Influxdb2Client *client);

int influxdb2_add_data(const Influxdb2ClientConfig *client_cfg, const Influxdb2DataConfig *data_cfg, Influxdb2Client *client, const DataModel *data);
#endif