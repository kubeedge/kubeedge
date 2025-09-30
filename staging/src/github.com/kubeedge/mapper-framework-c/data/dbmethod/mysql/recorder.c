#include "recorder.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <pthread.h>
#include <mysql/mysql.h>

static MySQLDataBaseConfig *g_mysql_db = NULL;
static pthread_mutex_t g_mysql_rec_mutex = PTHREAD_MUTEX_INITIALIZER;

// Set the global MySQL database configuration
void mysql_recorder_set_db(MySQLDataBaseConfig *db)
{
    g_mysql_db = db;
}

// Sanitize an identifier (e.g., namespace, device name, property name)
static void sanitize_id(const char *in, char *out, size_t outsz, const char *fallback)
{
    if (!in || !*in)
    {
        strlcpy(out, fallback, outsz);
        return;
    }
    size_t j = 0;
    for (size_t i = 0; in[i] && j + 1 < outsz; ++i)
    {
        unsigned char c = (unsigned char)in[i];
        if (c >= 'A' && c <= 'Z')
            c = (unsigned char)(c - 'A' + 'a');
        if ((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
            c == '-' || c == '_' || c == '/')
        {
            out[j++] = (char)c;
        }
        else
        {
            out[j++] = '_';
        }
    }
    out[j] = '\0';
    if (j == 0)
        strlcpy(out, fallback, outsz);
}

// Record a time-series data entry in the MySQL database
int mysql_recorder_record(const char *ns,
                          const char *deviceName,
                          const char *propertyName,
                          const char *value,
                          long long ts_ms)
{
    if (!deviceName || !propertyName || !value)
        return -1;

    pthread_mutex_lock(&g_mysql_rec_mutex);
    if (!g_mysql_db || !g_mysql_db->conn)
    {
        log_error("MySQL recorder: no DB or connection (detached or NULL)");
        pthread_mutex_unlock(&g_mysql_rec_mutex);
        return -1;
    }

    /* verify connection is alive; if dead, avoid using it and return error */
    if (mysql_ping((MYSQL *)g_mysql_db->conn) != 0)
    {
        log_error("MySQL recorder: connection lost (mysql_ping failed): %s", mysql_error((MYSQL *)g_mysql_db->conn));
        /* avoid further use until set_db is called again or reconnect logic added */
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
        pthread_mutex_unlock(&g_mysql_rec_mutex);
    }
    else
    {
        log_debug("MySQL record ok: %s/%s/%s=%s", ns_s, dev_s, prop_s, dm.value);
        pthread_mutex_unlock(&g_mysql_rec_mutex);
    }
    return rc;
}