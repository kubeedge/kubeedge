#include "mysql_client.h"
#include "log/log.h"

#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <time.h>
#include <cjson/cJSON.h>
#include <ctype.h>
#include <mysql.h>
#include <pthread.h>

#ifndef MYSQL_VERSION_ID
#define MYSQL_VERSION_ID 0
#endif

static const char *DEFAULT_MYSQL_HOST = "127.0.0.1";
static const char *DEFAULT_MYSQL_DB = "testdb";
static const char *DEFAULT_MYSQL_USER = "mapper";
static const char *DEFAULT_MYSQL_PASS = NULL;
static const int DEFAULT_MYSQL_PORT = 3306;

int mysql_parse_client_config(const char *json, MySQLClientConfig *out)
{
    if (!out)
        return -1;

    out->addr = NULL;
    out->database = NULL;
    out->userName = NULL;
    out->password = NULL;
    out->port = 0;

    if (json && *json)
    {
        cJSON *root = cJSON_Parse(json);
        if (root)
        {
            cJSON *jaddr = cJSON_GetObjectItem(root, "addr");
            cJSON *jdb = cJSON_GetObjectItem(root, "database");
            cJSON *juser = cJSON_GetObjectItem(root, "userName");
            cJSON *jpwd = cJSON_GetObjectItem(root, "password");
            cJSON *jport = cJSON_GetObjectItem(root, "port");

            if (jaddr && cJSON_IsString(jaddr)) out->addr = strdup(jaddr->valuestring);
            if (jdb && cJSON_IsString(jdb)) out->database = strdup(jdb->valuestring);
            if (juser && cJSON_IsString(juser)) out->userName = strdup(juser->valuestring);
            if (jpwd && cJSON_IsString(jpwd)) out->password = strdup(jpwd->valuestring);
            if (jport && cJSON_IsNumber(jport)) out->port = (int)jport->valuedouble;

            cJSON_Delete(root);
        }
        else
        {
            log_warn("mysql_parse_client_config: invalid JSON, using defaults");
        }
    }

    if (!out->addr) out->addr = strdup(DEFAULT_MYSQL_HOST);
    if (!out->database) out->database = strdup(DEFAULT_MYSQL_DB);
    if (!out->userName) out->userName = strdup(DEFAULT_MYSQL_USER);

    if (!out->password) {
        const char *envpwd = getenv("PASSWORD");
        if (!envpwd || !*envpwd) envpwd = getenv("MYSQL_PASSWORD");
        if (envpwd && *envpwd) out->password = strdup(envpwd);
        else if (DEFAULT_MYSQL_PASS) out->password = strdup(DEFAULT_MYSQL_PASS);
        else out->password = NULL;
    }
    if (out->port <= 0) out->port = DEFAULT_MYSQL_PORT;

    return 0;
}

int mysql_init_client(MySQLDataBaseConfig *db)
{
    if (!db)
        return -1;
    db->conn = mysql_init(NULL);
    if (!db->conn)
    {
        log_error("mysql_init failed");
        return -1;
    }

    unsigned int timeout = 10;
    mysql_options(db->conn, MYSQL_OPT_CONNECT_TIMEOUT, &timeout);

    const char *host = (db->config.addr && *db->config.addr) ? db->config.addr : DEFAULT_MYSQL_HOST;
    int port = (db->config.port > 0) ? db->config.port : DEFAULT_MYSQL_PORT;
    const char *user = db->config.userName ? db->config.userName : DEFAULT_MYSQL_USER;
    const char *pass = db->config.password ? db->config.password : DEFAULT_MYSQL_PASS;
    const char *dbname = db->config.database ? db->config.database : DEFAULT_MYSQL_DB;


    if (!mysql_real_connect(db->conn,
                            host,
                            user,
                            pass,
                            dbname,
                            port,
                            NULL,
                            0))
    {
        log_error("mysql_real_connect failed to %s:%u db=%s user=%s : %s", host, port, dbname, user, mysql_error(db->conn));
        mysql_close(db->conn);
        db->conn = NULL;
        return -1;
    }

    return 0;
}

void mysql_close_client(MySQLDataBaseConfig *db)
{
    if (db && db->conn)
    {
        mysql_close(db->conn);
        db->conn = NULL;
    }
}

int mysql_add_data(MySQLDataBaseConfig *db, const DataModel *data)
{
    if (!db || !db->conn || !data)
        return -1;

    char tableName[256];
    snprintf(tableName, sizeof(tableName), "%s_%s_%s",
             data->namespace_ ? data->namespace_ : "default",
             data->deviceName ? data->deviceName : "device",
             data->propertyName ? data->propertyName : "property");

    char createTable[1024];
    snprintf(createTable, sizeof(createTable),
             "CREATE TABLE IF NOT EXISTS `%s` ("
             "  id INT AUTO_INCREMENT PRIMARY KEY,"
             "  ts DATETIME NOT NULL,"
             "  field TEXT"
             ")",
             tableName);

    if (mysql_query(db->conn, createTable))
    {
        log_error("create table failed: %s", mysql_error(db->conn));
        return -1;
    }

    char insertSql[512];
    snprintf(insertSql, sizeof(insertSql),
             "INSERT INTO `%s` (ts, field) VALUES (?, ?)", tableName);

    MYSQL_STMT *stmt = mysql_stmt_init(db->conn);
    if (!stmt)
    {
        log_error("mysql_stmt_init failed: %s", mysql_error(db->conn));
        return -1;
    }
    if (mysql_stmt_prepare(stmt, insertSql, (unsigned long)strlen(insertSql)))
    {
        log_error("mysql_stmt_prepare failed: %s", mysql_stmt_error(stmt));
        mysql_stmt_close(stmt);
        return -1;
    }

    MYSQL_BIND bind[2];
    memset(bind, 0, sizeof(bind));

    char datetime[32];
    time_t ts = data->timeStamp;
    struct tm tm_info;
    localtime_r(&ts, &tm_info);
    strftime(datetime, sizeof(datetime), "%Y-%m-%d %H:%M:%S", &tm_info);

    unsigned long lengths[2];
    lengths[0] = (unsigned long)strlen(datetime);
    lengths[1] = (unsigned long)(data->value ? strlen(data->value) : 0);

    bind[0].buffer_type = MYSQL_TYPE_STRING;
    bind[0].buffer = (void *)datetime;
    bind[0].buffer_length = lengths[0];
    bind[0].length = &lengths[0];

    bind[1].buffer_type = MYSQL_TYPE_STRING;
    bind[1].buffer = (void *)(data->value ? data->value : "");
    bind[1].buffer_length = lengths[1];
    bind[1].length = &lengths[1];

    if (mysql_stmt_bind_param(stmt, bind))
    {
        log_error("mysql_stmt_bind_param failed: %s", mysql_stmt_error(stmt));
        mysql_stmt_close(stmt);
        return -1;
    }

    if (mysql_stmt_execute(stmt))
    {
        log_error("mysql_stmt_execute failed: %s", mysql_stmt_error(stmt));
        mysql_stmt_close(stmt);
        return -1;
    }

    mysql_stmt_close(stmt);
    return 0;
}

typedef struct MysqlCacheEntry {
    char *key;
    MySQLDataBaseConfig *db;
    int refcount;
    struct MysqlCacheEntry *next;
} MysqlCacheEntry;

static MysqlCacheEntry *g_mysql_cache = NULL;
static pthread_mutex_t g_mysql_cache_mu = PTHREAD_MUTEX_INITIALIZER;

static char *make_mysql_key(const MySQLClientConfig *cfg)
{
    if (!cfg) return NULL;
    const char *addr = cfg->addr ? cfg->addr : DEFAULT_MYSQL_HOST;
    const char *dbn = cfg->database ? cfg->database : DEFAULT_MYSQL_DB;
    const char *user = cfg->userName ? cfg->userName : DEFAULT_MYSQL_USER;
    int port = cfg->port > 0 ? cfg->port : DEFAULT_MYSQL_PORT;
    size_t n = strlen(addr) + strlen(dbn) + strlen(user) + 32;
    char *k = calloc(1, n);
    if (!k) return NULL;
    snprintf(k, n, "%s:%d/%s@%s", addr, port, dbn, user);
    return k;
}

MySQLDataBaseConfig *mysql_get_cached_db(const MySQLClientConfig *cfg)
{
    if (!cfg) return NULL;
    char *key = make_mysql_key(cfg);
    if (!key) return NULL;

    pthread_mutex_lock(&g_mysql_cache_mu);

    for (MysqlCacheEntry *e = g_mysql_cache; e; e = e->next) {
        if (e->key && strcmp(e->key, key) == 0) {
            e->refcount++;
            free(key);
            pthread_mutex_unlock(&g_mysql_cache_mu);
            return e->db;
        }
    }

    MysqlCacheEntry *ne = calloc(1, sizeof(*ne));
    if (!ne) {
        free(key);
        pthread_mutex_unlock(&g_mysql_cache_mu);
        return NULL;
    }
    ne->key = key;
    ne->db = calloc(1, sizeof(*ne->db));
    if (!ne->db) {
        free(ne->key);
        free(ne);
        pthread_mutex_unlock(&g_mysql_cache_mu);
        return NULL;
    }
    ne->db->config.addr = cfg->addr ? strdup(cfg->addr) : strdup(DEFAULT_MYSQL_HOST);
    ne->db->config.database = cfg->database ? strdup(cfg->database) : strdup(DEFAULT_MYSQL_DB);
    ne->db->config.userName = cfg->userName ? strdup(cfg->userName) : strdup(DEFAULT_MYSQL_USER);
    if (cfg->password && *cfg->password) {
        ne->db->config.password = strdup(cfg->password);
    } else {
        const char *envpwd = getenv("PASSWORD");
        if (!envpwd || !*envpwd) envpwd = getenv("MYSQL_PASSWORD");
        ne->db->config.password = envpwd && *envpwd ? strdup(envpwd) : (DEFAULT_MYSQL_PASS ? strdup(DEFAULT_MYSQL_PASS) : NULL);
    }
    ne->db->config.port = cfg->port > 0 ? cfg->port : DEFAULT_MYSQL_PORT;
    ne->db->conn = NULL;
    ne->refcount = 1;

    if (mysql_init_client(ne->db) != 0) {
        log_error("mysql_get_cached_db: init failed for key=%s", ne->key);
        free(ne->db->config.addr); free(ne->db->config.database); free(ne->db->config.userName);
        free(ne->db->config.password);
        free(ne->db);
        free(ne->key);
        free(ne);
        pthread_mutex_unlock(&g_mysql_cache_mu);
        return NULL;
    }

    ne->next = g_mysql_cache;
    g_mysql_cache = ne;

    pthread_mutex_unlock(&g_mysql_cache_mu);
    return ne->db;
}

void mysql_release_cached_db(MySQLDataBaseConfig *db)
{
    if (!db) return;
    pthread_mutex_lock(&g_mysql_cache_mu);
    MysqlCacheEntry *prev = NULL;
    for (MysqlCacheEntry *e = g_mysql_cache; e; prev = e, e = e->next) {
        if (e->db == db) {
            e->refcount--;
            if (e->refcount <= 0) {
                if (prev) prev->next = e->next;
                else g_mysql_cache = e->next;
                mysql_close_client(e->db);
                free(e->db->config.addr);
                free(e->db->config.database);
                free(e->db->config.userName);
                free(e->db->config.password);
                free(e->db);
                free(e->key);
                free(e);
            }
            break;
        }
    }
    pthread_mutex_unlock(&g_mysql_cache_mu);
}