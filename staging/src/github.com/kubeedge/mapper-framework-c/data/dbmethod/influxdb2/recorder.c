#include "recorder.h"
#include "influxdb2_client.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include "common/datamodel.h"
#include "common/string_util.h"

static Influxdb2ClientConfig g_client_cfg = {0};
static Influxdb2DataConfig g_data_cfg = {0};
static Influxdb2Client g_client = {0};
static int g_initialized = 0;
static pthread_mutex_t g_mutex = PTHREAD_MUTEX_INITIALIZER;

static void free_data_config(Influxdb2DataConfig *d)
{
    if (!d) return;
    if (d->measurement) free(d->measurement);
    if (d->fieldKey) free(d->fieldKey);
    if (d->tag_keys)
    {
        for (int i = 0; i < d->tag_count; ++i)
        {
            free(d->tag_keys[i]);
            free(d->tag_values[i]);
        }
        free(d->tag_keys);
        free(d->tag_values);
    }
    memset(d, 0, sizeof(*d));
}

static void free_client_config(Influxdb2ClientConfig *c)
{
    if (!c) return;
    if (c->url) free(c->url);
    if (c->org) free(c->org);
    if (c->bucket) free(c->bucket);
    if (c->token) free(c->token);
    memset(c, 0, sizeof(*c));
}

int influxdb2_recorder_set_db(const Influxdb2DataBaseConfig *cfg)
{
    if (!cfg) return -1;
    pthread_mutex_lock(&g_mutex);
    if (g_initialized)
    {
        influxdb2_close_client(&g_client);
        free_client_config(&g_client_cfg);
        free_data_config(&g_data_cfg);
        g_initialized = 0;
    }
    if (cfg->clientConfig.url) g_client_cfg.url = strdup(cfg->clientConfig.url);
    if (cfg->clientConfig.org) g_client_cfg.org = strdup(cfg->clientConfig.org);
    if (cfg->clientConfig.bucket) g_client_cfg.bucket = strdup(cfg->clientConfig.bucket);
    if (cfg->clientConfig.token) g_client_cfg.token = strdup(cfg->clientConfig.token);
    if (cfg->dataConfig.measurement) g_data_cfg.measurement = strdup(cfg->dataConfig.measurement);
    if (cfg->dataConfig.fieldKey) g_data_cfg.fieldKey = strdup(cfg->dataConfig.fieldKey);
    if (cfg->dataConfig.tag_count > 0 && cfg->dataConfig.tag_keys)
    {
        int n = cfg->dataConfig.tag_count;
        g_data_cfg.tag_keys = calloc(n, sizeof(char *));
        g_data_cfg.tag_values = calloc(n, sizeof(char *));
        for (int i = 0; i < n; ++i)
        {
            if (cfg->dataConfig.tag_keys[i]) g_data_cfg.tag_keys[g_data_cfg.tag_count] = strdup(cfg->dataConfig.tag_keys[i]);
            if (cfg->dataConfig.tag_values[i]) g_data_cfg.tag_values[g_data_cfg.tag_count] = strdup(cfg->dataConfig.tag_values[i]);
            g_data_cfg.tag_count++;
        }
    }
    if (influxdb2_init_client(&g_client_cfg, &g_client) != 0)
    {
        free_client_config(&g_client_cfg);
        free_data_config(&g_data_cfg);
        pthread_mutex_unlock(&g_mutex);
        return -1;
    }
    g_initialized = 1;
    pthread_mutex_unlock(&g_mutex);
    return 0;
}

static int ensure_influx_ready_locked(void)
{
    if (g_initialized) return 0;
    if (influxdb2_parse_client_config(NULL, &g_client_cfg) != 0) return -1;
    if (influxdb2_parse_data_config(NULL, &g_data_cfg) != 0) { free_client_config(&g_client_cfg); return -1; }
    if (influxdb2_init_client(&g_client_cfg, &g_client) != 0) { free_client_config(&g_client_cfg); free_data_config(&g_data_cfg); return -1; }
    g_initialized = 1;
    return 0;
}

int influxdb2_recorder_record(const char *ns, const char *device, const char *prop, const char *value, long long ts_ms)
{
    if (!ns || !device || !prop || !value) return -1;
    pthread_mutex_lock(&g_mutex);
    if (!g_initialized)
    {
        if (ensure_influx_ready_locked() != 0)
        {
            pthread_mutex_unlock(&g_mutex);
            return -1;
        }
    }
    char ns_s[128], dev_s[128], prop_s[128];
    sanitize_id(ns, ns_s, sizeof(ns_s), "default");
    sanitize_id(device, dev_s, sizeof(dev_s), "device");
    sanitize_id(prop, prop_s, sizeof(prop_s), "property");
    DataModel dm = {0};
    dm.namespace_ = (char *)ns;
    dm.deviceName = (char *)device;
    dm.propertyName = (char *)prop;
    dm.value = (char *)value;
    dm.timeStamp = (time_t)(ts_ms / 1000);
    Influxdb2DataConfig local_cfg = {0};
    if (!g_data_cfg.measurement)
    {
        char m[256];
        snprintf(m, sizeof(m), "%s_%s", ns_s, dev_s);
        local_cfg.measurement = strdup(m);
        local_cfg.fieldKey = g_data_cfg.fieldKey ? strdup(g_data_cfg.fieldKey) : strdup(prop_s);
        local_cfg.tag_count = g_data_cfg.tag_count;
        if (g_data_cfg.tag_count > 0)
        {
            local_cfg.tag_keys = calloc(g_data_cfg.tag_count, sizeof(char *));
            local_cfg.tag_values = calloc(g_data_cfg.tag_count, sizeof(char *));
            for (int i = 0; i < g_data_cfg.tag_count; ++i)
            {
                local_cfg.tag_keys[i] = strdup(g_data_cfg.tag_keys[i]);
                local_cfg.tag_values[i] = strdup(g_data_cfg.tag_values[i]);
            }
        }
    }
    else
    {
        local_cfg.measurement = strdup(g_data_cfg.measurement);
        local_cfg.fieldKey = g_data_cfg.fieldKey ? strdup(g_data_cfg.fieldKey) : strdup(prop_s);
        local_cfg.tag_count = g_data_cfg.tag_count;
        if (g_data_cfg.tag_count > 0)
        {
            local_cfg.tag_keys = calloc(g_data_cfg.tag_count, sizeof(char *));
            local_cfg.tag_values = calloc(g_data_cfg.tag_count, sizeof(char *));
            for (int i = 0; i < g_data_cfg.tag_count; ++i)
            {
                local_cfg.tag_keys[i] = strdup(g_data_cfg.tag_keys[i]);
                local_cfg.tag_values[i] = strdup(g_data_cfg.tag_values[i]);
            }
        }
    }
    int rc = influxdb2_add_data(&g_client_cfg, &local_cfg, &g_client, &dm);
    free_data_config(&local_cfg);
    pthread_mutex_unlock(&g_mutex);
    return rc;
}

void influxdb2_recorder_close(void)
{
    pthread_mutex_lock(&g_mutex);
    if (g_initialized)
    {
        influxdb2_close_client(&g_client);
        free_client_config(&g_client_cfg);
        free_data_config(&g_data_cfg);
        g_initialized = 0;
    }
    pthread_mutex_unlock(&g_mutex);
}