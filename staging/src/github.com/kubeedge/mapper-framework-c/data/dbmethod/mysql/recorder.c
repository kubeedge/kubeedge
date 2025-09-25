#include "recorder.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <strings.h>

// Global MySQL database configuration
static MySQLDataBaseConfig *g_mysql_db = NULL;

// Set the global MySQL database configuration
void mysql_recorder_set_db(MySQLDataBaseConfig *db) {
    g_mysql_db = db;
    if (g_mysql_db) {
        log_info("MySQL recorder attached");
    } else {
        log_warn("MySQL recorder detached (NULL)");
    }
}

// Sanitize an identifier (e.g., namespace, device name, property name)
static void sanitize_id(const char *in, char *out, size_t outsz, const char *fallback) {
    if (!in || !*in) {
        strlcpy(out, fallback, outsz);
        return;
    }
    size_t j = 0;
    for (size_t i = 0; in[i] && j + 1 < outsz; ++i) {
        unsigned char c = (unsigned char)in[i];
        if (c >= 'A' && c <= 'Z') c = (unsigned char)(c - 'A' + 'a');
        if ((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
            c == '-' || c == '_' || c == '/') {
            out[j++] = (char)c;
        } else {
            out[j++] = '_';
        }
    }
    out[j] = '\0';
    if (j == 0) strlcpy(out, fallback, outsz);
}

// Record a time-series data entry in the MySQL database
int mysql_recorder_record(const char *ns,
                          const char *deviceName,
                          const char *propertyName,
                          const char *value,
                          long long ts_ms) {
    if (!g_mysql_db || !g_mysql_db->conn) return -1;
    if (!deviceName || !propertyName || !value) return -1;

    char ns_s[128], dev_s[128], prop_s[128];
    sanitize_id(ns, ns_s, sizeof(ns_s), "default");
    sanitize_id(deviceName, dev_s, sizeof(dev_s), "device");
    sanitize_id(propertyName, prop_s, sizeof(prop_s), "property");

    DataModel dm = (DataModel){0};
    dm.namespace_   = ns_s;
    dm.deviceName   = dev_s;
    dm.propertyName = prop_s;
    dm.type         = "string";
    dm.value        = (char*)value;
    dm.timeStamp    = (time_t)(ts_ms / 1000);

    int rc = mysql_add_data(g_mysql_db, &dm);
    if (rc != 0) {
        log_warn("MySQL record failed: %s/%s/%s val=%s", ns_s, dev_s, prop_s, dm.value);
    } else {
        log_debug("MySQL record ok: %s/%s/%s=%s", ns_s, dev_s, prop_s, dm.value);
    }
    return rc;
}

int mysql_recorder_init_from_env(void) {
    const char *env_mysql = getenv("MYSQL_ENABLED");
    if (env_mysql && *env_mysql) {
        if (*env_mysql=='0' || strcasecmp(env_mysql,"false")==0) {
            log_info("MySQL recorder disabled by MYSQL_ENABLED");
            g_mysql_db = NULL;
            mysql_recorder_set_db(NULL);
            return 0;
        }
    }

    MySQLClientConfig clientCfg = (MySQLClientConfig){0};
    if (mysql_parse_client_config(NULL, &clientCfg) != 0) {
        log_error("MySQL client config parse from env failed");
        return -1;
    }

    MySQLDataBaseConfig *db = (MySQLDataBaseConfig*)calloc(1, sizeof(MySQLDataBaseConfig));
    if (!db) {
        free(clientCfg.addr); free(clientCfg.database); free(clientCfg.userName); free(clientCfg.password);
        return -1;
    }
    db->config = clientCfg;

    if (mysql_init_client(db) != 0) {
        log_error("MySQL init failed (host=%s db=%s user=%s)",
                  clientCfg.addr ? clientCfg.addr : "(nil)",
                  clientCfg.database ? clientCfg.database : "(nil)",
                  clientCfg.userName ? clientCfg.userName : "(nil)");
        free(db->config.addr); free(db->config.database); free(db->config.userName); free(db->config.password);
        free(db);
        return -1;
    }

    g_mysql_db = db;
    mysql_recorder_set_db(g_mysql_db);
    log_info("MySQL recorder initialized");
    return 0;
}

void mysql_recorder_shutdown(void) {
    if (g_mysql_db) {
        mysql_close_client(g_mysql_db);
        free(g_mysql_db->config.addr);
        free(g_mysql_db->config.database);
        free(g_mysql_db->config.userName);
        free(g_mysql_db->config.password);
        free(g_mysql_db);
        g_mysql_db = NULL;
        mysql_recorder_set_db(NULL);
        log_info("MySQL recorder shutdown complete");
    }
}