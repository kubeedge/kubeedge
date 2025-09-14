#include "mysql_client.h"
#include "log/log.h"

#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <time.h>
#include <cjson/cJSON.h>
#include <ctype.h>
#include <mysql.h>
#ifndef MYSQL_VERSION_ID
#define MYSQL_VERSION_ID 0
#endif

// Retrieves an environment variable or returns a default value
static const char* getenv_def(const char *key, const char *default_value) {
    const char *value = getenv(key);
    return (value && *value) ? value : default_value;
}

// Logs MySQL connection errors
static void log_mysql_conn_error(MYSQL *conn, const char *phase) {
    if (!conn) return;
    log_error("%s: (%u) %s", phase, mysql_errno(conn), mysql_error(conn));
}

// Normalizes the SSL mode string
static const char* normalize_ssl_mode(const char *mode) {
    if (!mode || !*mode) return "DISABLED";
    if (!strcasecmp(mode, "DISABLED"))  return "DISABLED";
    if (!strcasecmp(mode, "PREFERRED")) return "PREFERRED";
    if (!strcasecmp(mode, "REQUIRED"))  return "REQUIRED";
    if (!strcasecmp(mode, "VERIFY_CA")) return "VERIFY_CA";
    if (!strcasecmp(mode, "VERIFY_IDENTITY")) return "VERIFY_IDENTITY";
    return "DISABLED";
}

// Parses the MySQL client configuration from JSON
int mysql_parse_client_config(const char *json, MySQLClientConfig *out) {
    if (!out) return -1;

    // Initialize default values from environment variables
    const char *env_addr = getenv_def("MYSQL_HOST", "127.0.0.1");
    const char *env_db   = getenv_def("MYSQL_DB",   "testdb");
    const char *env_user = getenv_def("MYSQL_USER", "mapper");
    const char *env_pwd  = getenv("MYSQL_PASSWORD");
    if (!env_pwd || !*env_pwd) env_pwd = getenv("PASSWORD");

    out->addr     = strdup(env_addr);
    out->database = strdup(env_db);
    out->userName = strdup(env_user);
    out->password = env_pwd ? strdup(env_pwd) : NULL;

    const char *env_port = getenv("MYSQL_PORT");
    int portVal = (env_port && *env_port) ? atoi(env_port) : 3306;

    // Override values with JSON configuration if provided
    if (json && *json) {
        cJSON *root = cJSON_Parse(json);
        if (root) {
            cJSON *jaddr = cJSON_GetObjectItem(root, "addr");
            cJSON *jdb   = cJSON_GetObjectItem(root, "database");
            cJSON *juser = cJSON_GetObjectItem(root, "userName");
            cJSON *jpwd  = cJSON_GetObjectItem(root, "password");
            cJSON *jport = cJSON_GetObjectItem(root, "port");
            cJSON *jssl  = cJSON_GetObjectItem(root, "ssl_mode");
            if (jaddr && cJSON_IsString(jaddr)) { free(out->addr); out->addr = strdup(jaddr->valuestring); }
            if (jdb && cJSON_IsString(jdb))     { free(out->database); out->database = strdup(jdb->valuestring); }
            if (juser && cJSON_IsString(juser)) { free(out->userName); out->userName = strdup(juser->valuestring); }
            if (jpwd && cJSON_IsString(jpwd)) {
                if (out->password) free(out->password);
                out->password = strdup(jpwd->valuestring);
            }
            if (jport && cJSON_IsNumber(jport)) out->port = (int)jport->valuedouble;
            if (jssl && cJSON_IsString(jssl)) {
                setenv("MYSQL_SSL_MODE", jssl->valuestring, 1);
            }
            cJSON_Delete(root);
        }
    }
    out->port = (portVal > 0 ? portVal : 3306);
    return 0;
}

// Initializes the MySQL client
int mysql_init_client(MySQLDataBaseConfig *db) {
    if (!db) return -1;
    db->conn = mysql_init(NULL);
    if (!db->conn) {
        log_error("mysql_init failed");
        return -1;
    }

    unsigned int timeout = 10;
    mysql_options(db->conn, MYSQL_OPT_CONNECT_TIMEOUT, &timeout);

    const char *host = (db->config.addr && *db->config.addr) ? db->config.addr : "127.0.0.1";
    unsigned int port = (db->config.port > 0) ? (unsigned int)db->config.port : 3306;
    const char *user = db->config.userName ? db->config.userName : "mapper";
    const char *pass = db->config.password;
    if (!pass || !*pass) {
        const char *envp = getenv("MYSQL_PASSWORD");
        if (envp && *envp) pass = envp;
    }
    if (!pass) pass = "";
    const char *dbname = db->config.database ? db->config.database : "testdb";
    const char *cfg_ssl = NULL;
    const char *env_ssl = getenv("MYSQL_SSL_MODE");
        if (env_ssl && *env_ssl) cfg_ssl = env_ssl;
    const char *ssl_mode = normalize_ssl_mode(cfg_ssl);

#if (MYSQL_VERSION_ID >= 50711)
    enum mysql_ssl_mode mode_enum = SSL_MODE_DISABLED;
    if (!strcasecmp(ssl_mode, "PREFERRED")) mode_enum = SSL_MODE_PREFERRED;
    else if (!strcasecmp(ssl_mode, "REQUIRED")) mode_enum = SSL_MODE_REQUIRED;
    else if (!strcasecmp(ssl_mode, "VERIFY_CA")) mode_enum = SSL_MODE_VERIFY_CA;
    else if (!strcasecmp(ssl_mode, "VERIFY_IDENTITY")) mode_enum = SSL_MODE_VERIFY_IDENTITY;
    else mode_enum = SSL_MODE_DISABLED;

    if (mysql_options(db->conn, MYSQL_OPT_SSL_MODE, &mode_enum) != 0) {
        log_warn("mysql_options(MYSQL_OPT_SSL_MODE=%s) failed, continuing", ssl_mode);
    } else {
        log_info("MySQL SSL mode set to %s", ssl_mode);
    }
#else
    log_info("MySQL client lacks MYSQL_OPT_SSL_MODE; forcing non-SSL attempt");
#endif

#ifdef MYSQL_OPT_PROTOCOL
    unsigned int proto = MYSQL_PROTOCOL_TCP;
    mysql_options(db->conn, MYSQL_OPT_PROTOCOL, &proto);
#endif

    const char *unix_sock = getenv("MYSQL_UNIX_SOCKET");
    if (!(unix_sock && *unix_sock)) unix_sock = NULL;

    log_info("MySQL connect try host=%s port=%u socket=%s db=%s user=%s ssl_mode=%s pw_len=%zu",
             host, port, unix_sock ? unix_sock : "(none)", dbname, user, ssl_mode, strlen(pass));

    if (!mysql_real_connect(db->conn,
                            host,
                            user,
                            pass,
                            dbname,
                            unix_sock ? 0 : port,
                            unix_sock,
                            0)) {
        log_mysql_conn_error(db->conn, "mysql_real_connect");   // 新增：实际调用，消除未使用告警
        log_error("mysql_real_connect failed (%u): %s",
                  mysql_errno(db->conn), mysql_error(db->conn));
        mysql_close(db->conn);
        db->conn = NULL;
        return -1;
    }

    log_info("MySQL connected proto=%s ssl=maybe(%s)", unix_sock ? "unix" : "tcp", ssl_mode);
    return 0;
}

// Closes the MySQL client
void mysql_close_client(MySQLDataBaseConfig *db) {
    if (db && db->conn) {
        mysql_close(db->conn);
        db->conn = NULL;
    }
}

int mysql_add_data(MySQLDataBaseConfig *db, const DataModel *data) {
    if (!db || !db->conn || !data) return -1;

    // 组合表名（包含 / 需要使用反引号）
    char tableName[256];
    snprintf(tableName, sizeof(tableName), "%s/%s/%s",
             data->namespace_ ? data->namespace_ : "default",
             data->deviceName ? data->deviceName : "device",
             data->propertyName ? data->propertyName : "property");

    // 建表
    char createTable[512];
    snprintf(createTable, sizeof(createTable),
        "CREATE TABLE IF NOT EXISTS `%s` ("
        "  id INT AUTO_INCREMENT PRIMARY KEY,"
        "  ts DATETIME NOT NULL,"
        "  field TEXT"
        ")", tableName);

    if (mysql_query(db->conn, createTable)) {
        log_error("create table failed: %s", mysql_error(db->conn));
        return -1;
    }

    // 插入语句（预处理）
    char insertSql[512];
    snprintf(insertSql, sizeof(insertSql),
             "INSERT INTO `%s` (ts, field) VALUES (?, ?)", tableName);

    MYSQL_STMT *stmt = mysql_stmt_init(db->conn);
    if (!stmt) {
        log_error("mysql_stmt_init failed: %s", mysql_error(db->conn));
        return -1;
    }
    if (mysql_stmt_prepare(stmt, insertSql, (unsigned long)strlen(insertSql))) {
        log_error("mysql_stmt_prepare failed: %s", mysql_stmt_error(stmt));
        mysql_stmt_close(stmt);
        return -1;
    }

    // 准备绑定参数
    MYSQL_BIND bind[2];
    memset(bind, 0, sizeof(bind));

    // 时间戳 -> 日期时间字符串
    char datetime[32];
    time_t ts = data->timeStamp;
    struct tm tm_info;
    localtime_r(&ts, &tm_info);
    strftime(datetime, sizeof(datetime), "%Y-%m-%d %H:%M:%S", &tm_info);

    unsigned long lengths[2];
    lengths[0] = (unsigned long)strlen(datetime);
    lengths[1] = (unsigned long)(data->value ? strlen(data->value) : 0);

    bind[0].buffer_type   = MYSQL_TYPE_STRING;
    bind[0].buffer        = (void*)datetime;
    bind[0].buffer_length = lengths[0];
    bind[0].length        = &lengths[0];

    bind[1].buffer_type   = MYSQL_TYPE_STRING;
    bind[1].buffer        = (void*)(data->value ? data->value : "");
    bind[1].buffer_length = lengths[1];
    bind[1].length        = &lengths[1];

    if (mysql_stmt_bind_param(stmt, bind)) {
        log_error("mysql_stmt_bind_param failed: %s", mysql_stmt_error(stmt));
        mysql_stmt_close(stmt);
        return -1;
    }

    if (mysql_stmt_execute(stmt)) {
        log_error("mysql_stmt_execute failed: %s", mysql_stmt_error(stmt));
        mysql_stmt_close(stmt);
        return -1;
    }

    mysql_stmt_close(stmt);
    return 0;
}