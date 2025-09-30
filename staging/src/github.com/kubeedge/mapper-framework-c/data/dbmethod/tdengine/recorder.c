#include "recorder.h"
#include "tdengine_client.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <time.h>

static TDEngineDataBaseConfig *g_td_db = NULL;
static pthread_mutex_t g_td_mutex = PTHREAD_MUTEX_INITIALIZER;

int tdengine_recorder_set_db(TDEngineDataBaseConfig *db)
{
    if (!db)
        return -1;
    pthread_mutex_lock(&g_td_mutex);

    if (g_td_db && g_td_db->conn)
    {
        tdengine_close_client(g_td_db);
    }

    if (tdengine_init_client(db) != 0)
    {
        log_error("TDengine recorder: init client failed");
        pthread_mutex_unlock(&g_td_mutex);
        return -1;
    }

    g_td_db = db;
    log_info("TDengine recorder: attached and initialized");
    pthread_mutex_unlock(&g_td_mutex);
    return 0;
}

int tdengine_recorder_record(const char *ns, const char *device, const char *prop, const char *value, long long ts_ms)
{
    if (!device || !prop || !value)
        return -1;
    pthread_mutex_lock(&g_td_mutex);
    if (!g_td_db || !g_td_db->conn)
    {
        log_error("TDengine recorder: no DB/connection");
        pthread_mutex_unlock(&g_td_mutex);
        return -1;
    }

    DataModel dm = {0};
    dm.namespace_ = ns ? (char *)ns : (char *)"default";
    dm.deviceName = device ? (char *)device : (char *)"unknown";
    dm.propertyName = prop ? (char *)prop : (char *)"property";
    dm.value = (char *)value;
    dm.timeStamp = (time_t)(ts_ms / 1000);

    int rc = tdengine_add_data(g_td_db, &dm);
    if (rc != 0)
    {
        log_warn("TDengine recorder: add_data failed for %s/%s/%s val=%s", dm.namespace_, dm.deviceName, dm.propertyName, dm.value);
    }
    else
    {
        log_debug("TDengine recorder: add_data ok %s/%s/%s val=%s", dm.namespace_, dm.deviceName, dm.propertyName, dm.value);
    }

    pthread_mutex_unlock(&g_td_mutex);
    return rc;
}

void tdengine_recorder_close(void)
{
    pthread_mutex_lock(&g_td_mutex);
    if (g_td_db)
    {
        tdengine_close_client(g_td_db);
        g_td_db = NULL;
    }
    pthread_mutex_unlock(&g_td_mutex);
}