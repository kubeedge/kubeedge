#include "recorder.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <pthread.h>

static RedisDataBaseConfig *g_redis_db = NULL;
static pthread_mutex_t g_redis_mutex = PTHREAD_MUTEX_INITIALIZER;

int redis_recorder_set_db(RedisDataBaseConfig *db)
{
    if (!db)
        return -1;
    pthread_mutex_lock(&g_redis_mutex);

    if (g_redis_db && g_redis_db->conn)
    {
        redis_close_client(g_redis_db);
    }

    if (redis_init_client(db) != 0)
    {
        log_error("Redis recorder: init client failed");
        pthread_mutex_unlock(&g_redis_mutex);
        return -1;
    }

    g_redis_db = db;
    log_info("Redis recorder: attached DB config and initialized");
    pthread_mutex_unlock(&g_redis_mutex);
    return 0;
}

int redis_recorder_record(const char *ns, const char *device, const char *prop, const char *value, long long ts_ms)
{
    if (!device || !prop || !value)
        return -1;

    pthread_mutex_lock(&g_redis_mutex);
    if (!g_redis_db || !g_redis_db->conn)
    {
        log_error("Redis recorder: no DB/connection");
        pthread_mutex_unlock(&g_redis_mutex);
        return -1;
    }

    DataModel dm = {0};
    dm.namespace_ = ns ? (char *)ns : (char *)"default";
    dm.deviceName = device ? (char *)device : (char *)"unknown_device";
    dm.propertyName = prop ? (char *)prop : (char *)"property";
    dm.value = (char *)value;
    dm.timeStamp = (time_t)(ts_ms / 1000);

    int rc = redis_add_data(g_redis_db, &dm);
    if (rc != 0)
    {
        log_warn("Redis recorder: add_data failed for %s/%s/%s val=%s", dm.namespace_, dm.deviceName, dm.propertyName, dm.value);
    }
    else
    {
        log_debug("Redis recorder: add_data ok %s/%s/%s val=%s", dm.namespace_, dm.deviceName, dm.propertyName, dm.value);
    }

    pthread_mutex_unlock(&g_redis_mutex);
    return rc;
}

void redis_recorder_close(void)
{
    pthread_mutex_lock(&g_redis_mutex);
    if (g_redis_db)
    {
        redis_close_client(g_redis_db);
        g_redis_db = NULL;
    }
    pthread_mutex_unlock(&g_redis_mutex);
}