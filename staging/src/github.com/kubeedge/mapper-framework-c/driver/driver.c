#include "driver/driver.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include "common/const.h"
#include <time.h>
#include <math.h>
#include <strings.h>

#define MAX_SIM_CLIENTS 64
static void *sim_clients[MAX_SIM_CLIENTS];
static double sim_baseline[MAX_SIM_CLIENTS];
static double sim_threshold[MAX_SIM_CLIENTS];
static int sim_threshold_offset[MAX_SIM_CLIENTS];

static int sim_index_of(void *client)
{
    if (!client)
        return -1;
    for (int i = 0; i < MAX_SIM_CLIENTS; ++i)
    {
        if (sim_clients[i] == client)
            return i;
    }
    return -1;
}

static void sim_register_client(void *client)
{
    if (!client)
        return;
    for (int i = 0; i < MAX_SIM_CLIENTS; ++i)
    {
        if (sim_clients[i] == NULL)
        {
            sim_clients[i] = client;
            sim_baseline[i] = 25.0;
            sim_threshold[i] = 50.0;
            sim_threshold_offset[i] = 2; // offsets are 1-based in device layer: temperature=1, threshold=2
            return;
        }
    }
}

static void sim_unregister_client(void *client)
{
    if (!client)
        return;
    for (int i = 0; i < MAX_SIM_CLIENTS; ++i)
    {
        if (sim_clients[i] == client)
        {
            sim_clients[i] = NULL;
            sim_baseline[i] = 0.0;
            sim_threshold[i] = 0.0;
            sim_threshold_offset[i] = -1;
            return;
        }
    }
}

static double *sim_find_baseline_ptr(void *client)
{
    int idx = sim_index_of(client);
    if (idx < 0)
        return NULL;
    return &sim_baseline[idx];
}

static double *sim_find_threshold_ptr(void *client)
{
    int idx = sim_index_of(client);
    if (idx < 0)
        return NULL;
    return &sim_threshold[idx];
}

static int *sim_find_threshold_offset_ptr(void *client)
{
    int idx = sim_index_of(client);
    if (idx < 0)
        return NULL;
    return &sim_threshold_offset[idx];
}

static int parse_number_field(const char *json, const char *key, double *out_val)
{
    if (!json || !key || !out_val)
        return 0;
    const char *p = json;
    size_t klen = strlen(key);
    while ((p = strstr(p, key)) != NULL)
    {
        const char *q = p - 1;
        if (q >= json && (*q == '\"' || *q == '\'' || *q == ' ' || *q == '{' || *q == ','))
        {
            const char *colon = strchr(p + klen, ':');
            if (!colon)
            {
                p += klen;
                continue;
            }
            const char *v = colon + 1;
            while (*v && (*v == ' ' || *v == '\t' || *v == '\"' || *v == '\''))
            {
                if (*v == '\"' || *v == '\'')
                {
                    v++;
                    break;
                }
                v++;
            }
            char tmp[64] = {0};
            int i = 0;
            while (*v && i < (int)sizeof(tmp) - 1 && (*v == '+' || *v == '-' || (*v >= '0' && *v <= '9') || *v == '.'))
            {
                tmp[i++] = *v++;
            }
            if (i == 0)
                return 0;
            tmp[i] = '\0';
            *out_val = atof(tmp);
            return 1;
        }
        p += klen;
    }
    return 0;
}

CustomizedClient *NewClient(const ProtocolConfig *protocol)
{
    CustomizedClient *c = calloc(1, sizeof(*c));
    if (!c)
    {
        log_error("driver: NewClient calloc failed");
        return NULL;
    }
    if (protocol)
    {
        c->protocolConfig.protocolName = protocol->protocolName ? strdup(protocol->protocolName) : NULL;
        c->protocolConfig.configData = protocol->configData ? strdup(protocol->configData) : NULL;
    }
    pthread_mutex_init(&c->deviceMutex, NULL);
    sim_register_client((void *)c);
    srand((unsigned int)(time(NULL) ^ (uintptr_t)c));

    if (protocol && protocol->configData)
    {
        double v = 0;
        if (parse_number_field(protocol->configData, "threshold", &v))
        {
            double *tp = sim_find_threshold_ptr((void *)c);
            if (tp)
                *tp = v;
        }
        if (parse_number_field(protocol->configData, "threshold_offset", &v))
        {
            int *toff = sim_find_threshold_offset_ptr((void *)c);
            if (toff)
                *toff = (int)v;
        }
    }
    return c;
}

void FreeClient(CustomizedClient *client)
{
    if (!client)
        return;
    free(client->protocolConfig.protocolName);
    free(client->protocolConfig.configData);
    pthread_mutex_destroy(&client->deviceMutex);
    sim_unregister_client((void *)client);
    free(client);
}

int InitDevice(CustomizedClient *client)
{
    if (!client)
        return -1;
    pthread_mutex_lock(&client->deviceMutex);
    pthread_mutex_unlock(&client->deviceMutex);
    return 0;
}

static inline int is_threshold_name(const char *s) {
    return s && strcasecmp(s, "threshold") == 0;
}

int GetDeviceData(CustomizedClient *client, const VisitorConfig *visitor, void **out_data)
{
    if (!client || !visitor || !out_data)
        return -1;

    pthread_mutex_lock(&client->deviceMutex);

    const char *pname = visitor->propertyName ? visitor->propertyName : "";
    double thr_val = 50.0;
    double *tp = sim_find_threshold_ptr((void *)client);
    if (tp) thr_val = *tp;

    if (strcasecmp(pname, "threshold") == 0) {
        char buf[32];
        snprintf(buf, sizeof(buf), "%.2f", thr_val);
        *out_data = strdup(buf);
        pthread_mutex_unlock(&client->deviceMutex);
        return 0;
    }

    double baseline = 25.0;
    double *bp = sim_find_baseline_ptr((void *)client);
    if (bp) baseline = *bp;

    time_t now = time(NULL);
    double slow = sin((double)now / 60.0) * 0.5;
    double jitter = ((double)(rand() % 100) - 50.0) / 200.0;
    double value = baseline + slow + jitter;

    char buf[32];
    snprintf(buf, sizeof(buf), "%.2f", value);
    *out_data = strdup(buf);

    pthread_mutex_unlock(&client->deviceMutex);
    return 0;
}


int DeviceDataWrite(CustomizedClient *client, const VisitorConfig *visitor, const char *deviceMethodName, const char *propertyName, const void *data)
{
    if (!client || !visitor)
        return -1;
    pthread_mutex_lock(&client->deviceMutex);

    const char *pname = propertyName ? propertyName : "";
    int *toff_ptr = sim_find_threshold_offset_ptr((void *)client);
    int cfg_off = toff_ptr ? *toff_ptr : -1;
    log_info("driver:DeviceDataWrite name='%s' offset=%d cfg_off=%d data='%s'",
             pname, visitor->offset, cfg_off, (const char*)data);

    if (data) {
        char *endptr = NULL;
        double val = strtod((const char *)data, &endptr);
        if (endptr != (const char *)data) {
            if (is_threshold_name(pname)) {
                log_info("driver:DeviceDataWrite THRESHOLD read-only, ignore");
            } else {
                double *bp = sim_find_baseline_ptr((void *)client);
                if (bp) *bp = val;
                log_info("driver:DeviceDataWrite TEMPERATURE baseline->%.2f", val);
            }
        } else {
            log_info("driver:DeviceDataWrite received non-numeric data, ignored");
        }
    }
    pthread_mutex_unlock(&client->deviceMutex);
    return 0;
}

int SetDeviceData(CustomizedClient *client, const void *data, const VisitorConfig *visitor)
{
    if (!client || !visitor)
        return -1;
    pthread_mutex_lock(&client->deviceMutex);

    pthread_mutex_unlock(&client->deviceMutex);
    return 0;
}

int StopDevice(CustomizedClient *client)
{
    if (!client)
        return -1;
    pthread_mutex_lock(&client->deviceMutex);
    pthread_mutex_unlock(&client->deviceMutex);
    return 0;
}

const char *GetDeviceStates(CustomizedClient *client)
{
    if (!client)
        return DEVICE_STATUS_UNKNOWN;
    pthread_mutex_lock(&client->deviceMutex);
    pthread_mutex_unlock(&client->deviceMutex);
    return DEVICE_STATUS_OK;
}