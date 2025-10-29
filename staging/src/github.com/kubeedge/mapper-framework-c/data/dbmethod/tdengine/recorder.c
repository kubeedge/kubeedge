#include "recorder.h"
#include "tdengine_client.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <time.h>
#include "common/string_util.h"

static TDEngineDataBaseConfig *g_td_db = NULL;
static int g_td_owned = 0;
static pthread_mutex_t g_td_mutex = PTHREAD_MUTEX_INITIALIZER;

int tdengine_recorder_set_db(TDEngineDataBaseConfig *db)
{
    pthread_mutex_lock(&g_td_mutex);
    g_td_db = db;
    g_td_owned = 0;
    pthread_mutex_unlock(&g_td_mutex);
    return 0;
}

static int ensure_td_ready_locked(void)
{
    if (g_td_db && g_td_db->conn)
        return 0;

    TDEngineClientConfig cfg = {0};
    if (tdengine_parse_client_config(NULL, &cfg) != 0) {
        log_error("TDengine recorder: parse client config failed");
        return -1;
    }

    TDEngineDataBaseConfig *db = calloc(1, sizeof(*db));
    if (!db) {
        free(cfg.addr); free(cfg.dbName); free(cfg.username); free(cfg.password);
        log_error("TDengine recorder: alloc failed");
        return -1;
    }
    db->config = cfg;
    db->conn = NULL;

    if (tdengine_init_client(db) != 0) {
        free(db->config.addr); free(db->config.dbName); free(db->config.username); free(db->config.password);
        free(db);
        log_error("TDengine recorder: init client failed");
        return -1;
    }

    g_td_db = db;
    g_td_owned = 1;
    return 0;
}

int tdengine_recorder_record(const char *ns, const char *device, const char *prop, const char *value, long long ts_ms)
{
    if (!device || !prop || !value)
        return -1;

    pthread_mutex_lock(&g_td_mutex);

    if (!g_td_db || !g_td_db->conn) {
        if (ensure_td_ready_locked() != 0) {
            log_error("TDengine recorder: ensure_td_ready failed");
            pthread_mutex_unlock(&g_td_mutex);
            return -1;
        }
    }

    char ns_s[128], dev_s[128], prop_s[128];
    sanitize_id(ns, ns_s, sizeof(ns_s), "default");
    sanitize_id(device, dev_s, sizeof(dev_s), "unknown");
    sanitize_id(prop, prop_s, sizeof(prop_s), "property");

    DataModel dm = {0};
    dm.namespace_ = ns_s;
    dm.deviceName = dev_s;
    dm.propertyName = prop_s;
    dm.value = (char *)value;
    dm.type = "string";
    dm.timeStamp = (time_t)(ts_ms / 1000);

    int rc = tdengine_add_data(g_td_db, &dm);
    if (rc != 0)
        log_warn("TDengine recorder: add_data failed for %s/%s/%s val=%s", dm.namespace_, dm.deviceName, dm.propertyName, dm.value);
    else
        log_debug("TDengine recorder: add_data ok %s/%s/%s val=%s", dm.namespace_, dm.deviceName, dm.propertyName, dm.value);

    pthread_mutex_unlock(&g_td_mutex);
    return rc;
}

void tdengine_recorder_close(void)
{
    pthread_mutex_lock(&g_td_mutex);
    if (g_td_db)
    {
        tdengine_close_client(g_td_db);
        if (g_td_owned)
        {
            free(g_td_db->config.addr);
            free(g_td_db->config.dbName);
            free(g_td_db->config.username);
            free(g_td_db->config.password);
            free(g_td_db);
        }
        g_td_db = NULL;
        g_td_owned = 0;
    }
    pthread_mutex_unlock(&g_td_mutex);
}