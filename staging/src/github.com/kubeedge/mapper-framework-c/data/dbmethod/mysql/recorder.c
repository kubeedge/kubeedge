#include "recorder.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <pthread.h>
#include <mysql/mysql.h>
#include "data/dbmethod/mysql/mysql_client.h"
#include "common/string_util.h"

static MySQLDataBaseConfig *g_mysql_db = NULL;
static pthread_mutex_t g_mysql_rec_mutex = PTHREAD_MUTEX_INITIALIZER;

void mysql_recorder_set_db(MySQLDataBaseConfig *db)
{
    g_mysql_db = db;
}

static int ensure_mysql_ready_locked(void)
{
    if (g_mysql_db && g_mysql_db->conn)
        return 0;

    MySQLClientConfig cfg = {0};
    if (mysql_parse_client_config(NULL, &cfg) != 0) {
        log_error("MySQL recorder: parse client config failed");
        return -1;
    }

    MySQLDataBaseConfig *db = calloc(1, sizeof(*db));
    if (!db) {
        log_error("MySQL recorder: alloc db failed");
        free(cfg.addr); free(cfg.database); free(cfg.userName); free(cfg.password);
        return -1;
    }
    db->config = cfg;
    db->conn = NULL;

    if (mysql_init_client(db) != 0) {
        log_error("MySQL recorder: init client failed");
        free(db->config.addr); free(db->config.database); free(db->config.userName); free(db->config.password);
        free(db);
        return -1;
    }

    g_mysql_db = db;
    return 0;
}

int mysql_recorder_record(const char *ns,
                          const char *deviceName,
                          const char *propertyName,
                          const char *value,
                          long long ts_ms)
{
    if (!deviceName || !propertyName || !value)
        return -1;

    pthread_mutex_lock(&g_mysql_rec_mutex);

    if (!g_mysql_db || !g_mysql_db->conn) {
        if (ensure_mysql_ready_locked() != 0) {
            log_error("MySQL recorder: ensure_mysql_ready failed");
            pthread_mutex_unlock(&g_mysql_rec_mutex);
            return -1;
        }
    }

    if (mysql_ping((MYSQL *)g_mysql_db->conn) != 0)
    {
        log_error("MySQL recorder: connection lost (mysql_ping failed): %s", mysql_error((MYSQL *)g_mysql_db->conn));
        pthread_mutex_unlock(&g_mysql_rec_mutex);
        return -1;
    }

    char ns_s[128], dev_s[128], prop_s[128];
    sanitize_id(ns, ns_s, sizeof(ns_s), "default");
    sanitize_id(deviceName, dev_s, sizeof(dev_s), "device");
    sanitize_id(propertyName, prop_s, sizeof(prop_s), "property");

    DataModel dm = (DataModel){0};
    dm.namespace_ = ns_s;
    dm.deviceName = dev_s;
    dm.propertyName = prop_s;
    dm.type = "string";
    dm.value = (char *)value;
    dm.timeStamp = (time_t)(ts_ms / 1000);

    int rc = mysql_add_data(g_mysql_db, &dm);
    if (rc != 0)
    {
        log_warn("MySQL record failed: %s/%s/%s val=%s", ns_s, dev_s, prop_s, dm.value);
    }
    else
    {
        log_debug("MySQL record ok: %s/%s/%s=%s", ns_s, dev_s, prop_s, dm.value);
    }
    pthread_mutex_unlock(&g_mysql_rec_mutex);
    return rc;
}