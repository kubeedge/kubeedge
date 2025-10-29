#include "recorder.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <pthread.h>
#include "data/dbmethod/redis/redis_client.h"
#include "common/string_util.h"

static RedisDataBaseConfig *g_redis_db = NULL;
static int g_redis_owned = 0;
static pthread_mutex_t g_redis_mutex = PTHREAD_MUTEX_INITIALIZER;

int redis_recorder_set_db(RedisDataBaseConfig *db)
{
    pthread_mutex_lock(&g_redis_mutex);
    g_redis_db = db;
    g_redis_owned = 0;
    pthread_mutex_unlock(&g_redis_mutex);
    return 0;
}

static int ensure_redis_ready_locked(void)
{
    if (g_redis_db && g_redis_db->conn)
        return 0;

    RedisClientConfig cfg = {0};
    if (redis_parse_client_config(NULL, &cfg) != 0) {
        log_error("Redis recorder: parse client config failed");
        return -1;
    }

    RedisDataBaseConfig *db = calloc(1, sizeof(*db));
    if (!db) {
        free(cfg.addr); free(cfg.password);
        log_error("Redis recorder: alloc failed");
        return -1;
    }
    db->config = cfg;
    db->conn = NULL;

    if (redis_init_client(db) != 0) {
        free(db->config.addr); free(db->config.password);
        free(db);
        log_error("Redis recorder: init client failed");
        return -1;
    }

    g_redis_db = db;
    g_redis_owned = 1;
    return 0;
}

int redis_recorder_record(const char *ns, const char *device, const char *prop, const char *value, long long ts_ms)
{
    if (!device || !prop || !value)
        return -1;

    pthread_mutex_lock(&g_redis_mutex);

    if (!g_redis_db || !g_redis_db->conn) {
        if (ensure_redis_ready_locked() != 0) {
            log_error("Redis recorder: ensure_redis_ready failed");
            pthread_mutex_unlock(&g_redis_mutex);
            return -1;
        }
    }

    redisReply *r = redisCommand(g_redis_db->conn, "PING");
    if (!r || r->type != REDIS_REPLY_STATUS || strcmp(r->str, "PONG") != 0) {
        if (r)
            freeReplyObject(r);
        log_error("Redis recorder: PING failed");
        pthread_mutex_unlock(&g_redis_mutex);
        return -1;
    }
    freeReplyObject(r);

    char ns_s[128], dev_s[128], prop_s[128];
    sanitize_id(ns, ns_s, sizeof(ns_s), "default");
    sanitize_id(device, dev_s, sizeof(dev_s), "device");
    sanitize_id(prop, prop_s, sizeof(prop_s), "property");

    DataModel dm = {0};
    dm.namespace_ = ns_s;
    dm.deviceName = dev_s;
    dm.propertyName = prop_s;
    dm.type = "string";
    dm.value = (char *)value;
    dm.timeStamp = (time_t)(ts_ms / 1000);

    int rc = redis_add_data(g_redis_db, &dm);
    if (rc != 0)
        log_warn("Redis recorder: add_data failed %s/%s/%s val=%s", ns_s, dev_s, prop_s, dm.value);
    else
        log_debug("Redis recorder: add_data ok %s/%s/%s val=%s", ns_s, dev_s, prop_s, dm.value);

    pthread_mutex_unlock(&g_redis_mutex);
    return rc;
}

void redis_recorder_close(void)
{
    pthread_mutex_lock(&g_redis_mutex);
    if (g_redis_db)
    {
        if (g_redis_db->conn)
        redis_close_client(g_redis_db);
        if (g_redis_owned)
        {
            free(g_redis_db->config.addr);
            free(g_redis_db->config.password);
            free(g_redis_db);
        }
        g_redis_db = NULL;
        g_redis_owned = 0;
    }
    pthread_mutex_unlock(&g_redis_mutex);
}