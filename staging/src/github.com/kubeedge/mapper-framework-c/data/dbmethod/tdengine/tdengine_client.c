#include "tdengine_client.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <time.h>
#include <cjson/cJSON.h>

static char *replace_char(const char *str, char old_char, char new_char)
{
    if (!str)
        return NULL;
    char *result = strdup(str);
    for (int i = 0; result && result[i]; i++)
    {
        if (result[i] == old_char)
        {
            result[i] = new_char;
        }
    }
    return result;
}

int tdengine_parse_client_config(const char *json, TDEngineClientConfig *out)
{
    if (!out)
        return -1;

    if (json && *json)
    {
        cJSON *root = cJSON_Parse(json);
        if (root)
        {
            cJSON *jaddr = cJSON_GetObjectItem(root, "addr");
            cJSON *jdb = cJSON_GetObjectItem(root, "dbName");
            cJSON *juser = cJSON_GetObjectItem(root, "username");
            cJSON *jpwd = cJSON_GetObjectItem(root, "password");

            out->addr = jaddr && cJSON_IsString(jaddr) ? strdup(jaddr->valuestring) : NULL;
            out->dbName = jdb && cJSON_IsString(jdb) ? strdup(jdb->valuestring) : NULL;
            out->username = juser && cJSON_IsString(juser) ? strdup(juser->valuestring) : NULL;
            out->password = jpwd && cJSON_IsString(jpwd) ? strdup(jpwd->valuestring) : NULL;

            cJSON_Delete(root);

            if (!out->addr)
                out->addr = getenv("TDENGINE_ADDR") ? strdup(getenv("TDENGINE_ADDR")) : strdup("localhost:6041");
            if (!out->dbName)
                out->dbName = getenv("TDENGINE_DBNAME") ? strdup(getenv("TDENGINE_DBNAME")) : strdup("test");
            if (!out->username)
                out->username = getenv("TDENGINE_USER") ? strdup(getenv("TDENGINE_USER")) : getenv("USERNAME") ? strdup(getenv("USERNAME"))
                                                                                                               : strdup("root");
            if (!out->password)
                out->password = getenv("TDENGINE_PASSWORD") ? strdup(getenv("TDENGINE_PASSWORD")) : getenv("PASSWORD") ? strdup(getenv("PASSWORD"))
                                                                                                                       : strdup("taosdata");
            return 0;
        }
        else
        {
            log_warn("tdengine_parse_client_config: invalid JSON, falling back to env");
        }
    }

    const char *addr = getenv("TDENGINE_ADDR");
    const char *dbName = getenv("TDENGINE_DBNAME");
    const char *user = getenv("TDENGINE_USER");
    const char *password = getenv("TDENGINE_PASSWORD");

    if (!user)
        user = getenv("USERNAME");
    if (!password)
        password = getenv("PASSWORD");

    out->addr = addr ? strdup(addr) : strdup("localhost:6041");
    out->dbName = dbName ? strdup(dbName) : strdup("test");
    out->username = user ? strdup(user) : strdup("root");
    out->password = password ? strdup(password) : strdup("taosdata");
    return 0;
}

int tdengine_init_client(TDEngineDataBaseConfig *db)
{
    setenv("LANG", "C.UTF-8", 1);
    setenv("LC_ALL", "C.UTF-8", 1);
    taos_options(TSDB_OPTION_LOCALE, "C.UTF-8");
    taos_options(TSDB_OPTION_CHARSET, "UTF-8");
    if (!db)
        return -1;

    if (taos_init() != 0)
    {
        return -1;
    }

    db->conn = taos_connect(db->config.addr, db->config.username, db->config.password, db->config.dbName, 0);
    if (db->conn == NULL)
    {
        taos_cleanup();
        return -1;
    }

    char sql[256];
    snprintf(sql, sizeof(sql), "CREATE DATABASE IF NOT EXISTS %s", db->config.dbName);
    TAOS_RES *result = taos_query(db->conn, sql);
    if (taos_errno(result) != 0)
    {
        taos_free_result(result);
        taos_close(db->conn);
        taos_cleanup();
        return -1;
    }
    taos_free_result(result);

    snprintf(sql, sizeof(sql), "USE %s", db->config.dbName);
    result = taos_query(db->conn, sql);
    if (taos_errno(result) != 0)
    {
        log_error("Failed to use database: %s", taos_errstr(result));
        taos_free_result(result);
        taos_close(db->conn);
        taos_cleanup();
        return -1;
    }
    taos_free_result(result);

    return 0;
}

void tdengine_close_client(TDEngineDataBaseConfig *db)
{
    if (db && db->conn)
    {
        taos_close(db->conn);
        db->conn = NULL;
        taos_cleanup();
    }
}

int tdengine_add_data(TDEngineDataBaseConfig *db, const DataModel *data)
{
    if (!db || !db->conn || !data)
        return -1;

    char tableName[256];
    snprintf(tableName, sizeof(tableName), "%s/%s",
             data->namespace_ ? data->namespace_ : "default",
             data->deviceName ? data->deviceName : "unknown");

    char *legalTable = replace_char(tableName, '-', '_');
    char *legalTag = replace_char(data->propertyName ? data->propertyName : "property", '-', '_');

    char showStableSQL[512];
    snprintf(showStableSQL, sizeof(showStableSQL), "SHOW STABLES LIKE '%s'", legalTable);

    TAOS_RES *result = taos_query(db->conn, showStableSQL);
    if (taos_errno(result) != 0)
    {
        log_error("Failed to show stables: %s", taos_errstr(result));
        free(legalTable);
        free(legalTag);
        taos_free_result(result);
        return -1;
    }

    TAOS_ROW row = taos_fetch_row(result);
    int stable_exists = (row != NULL);
    taos_free_result(result);

    if (!stable_exists)
    {
        char createStableSQL[1024];
        snprintf(createStableSQL, sizeof(createStableSQL),
                 "CREATE STABLE %s (ts timestamp, deviceid binary(64), propertyname binary(64), data binary(64), type binary(64)) TAGS (location binary(64))",
                 legalTable);

        result = taos_query(db->conn, createStableSQL);
        if (taos_errno(result) != 0)
        {
            log_error("Failed to create stable: %s", taos_errstr(result));
            free(legalTable);
            free(legalTag);
            taos_free_result(result);
            return -1;
        }
        taos_free_result(result);
    }

    time_t timestamp = data->timeStamp / 1000;
    struct tm *tm_info = localtime(&timestamp);
    char datetime[64];
    strftime(datetime, sizeof(datetime), "%Y-%m-%d %H:%M:%S", tm_info);

    char insertSQL[2048];
    snprintf(insertSQL, sizeof(insertSQL),
             "INSERT INTO %s USING %s TAGS ('%s') VALUES('%s','%s', '%s', '%s', '%s')",
             legalTag, legalTable, legalTag, datetime, tableName,
             data->propertyName ? data->propertyName : "",
             data->value ? data->value : "",
             data->type ? data->type : "string");

    result = taos_query(db->conn, insertSQL);
    if (taos_errno(result) != 0)
    {
        log_error("Failed to insert data: %s", taos_errstr(result));
        free(legalTable);
        free(legalTag);
        taos_free_result(result);
        return -1;
    }

    taos_free_result(result);
    free(legalTable);
    free(legalTag);

    return 0;
}

int tdengine_get_data_by_device_id(TDEngineDataBaseConfig *db, const char *deviceID, DataModel ***dataModels, int *count)
{
    if (!db || !db->conn || !deviceID || !dataModels || !count)
        return -1;

    char *legalTable = replace_char(deviceID, '-', '_');
    char querySQL[512];
    snprintf(querySQL, sizeof(querySQL), "SELECT ts, deviceid, propertyname, data, type FROM %s", legalTable);

    TAOS_RES *result = taos_query(db->conn, querySQL);
    if (taos_errno(result) != 0)
    {
        log_error("Failed to query data: %s", taos_errstr(result));
        free(legalTable);
        taos_free_result(result);
        return -1;
    }

    *count = 0;
    TAOS_ROW row;

    while ((row = taos_fetch_row(result)) != NULL)
    {
        (*count)++;
    }

    if (*count == 0)
    {
        *dataModels = NULL;
        free(legalTable);
        taos_free_result(result);
        return 0;
    }

    taos_free_result(result);
    result = taos_query(db->conn, querySQL);
    if (taos_errno(result) != 0)
    {
        log_error("Failed to re-query data: %s", taos_errstr(result));
        free(legalTable);
        taos_free_result(result);
        return -1;
    }

    *dataModels = calloc(*count, sizeof(DataModel *));
    if (!*dataModels)
    {
        log_error("Failed to allocate memory for data models");
        free(legalTable);
        taos_free_result(result);
        return -1;
    }

    int i = 0;
    while ((row = taos_fetch_row(result)) != NULL && i < *count)
    {
        (*dataModels)[i] = calloc(1, sizeof(DataModel));
        if ((*dataModels)[i])
        {
            if (row[0])
            {
                (*dataModels)[i]->timeStamp = *(int64_t *)row[0] * 1000;
            }
            if (row[1])
                (*dataModels)[i]->deviceName = strdup((char *)row[1]);
            if (row[2])
                (*dataModels)[i]->propertyName = strdup((char *)row[2]);
            if (row[3])
                (*dataModels)[i]->value = strdup((char *)row[3]);
            if (row[4])
                (*dataModels)[i]->type = strdup((char *)row[4]);
        }
        i++;
    }

    free(legalTable);
    taos_free_result(result);
    return 0;
}

int tdengine_get_data_by_time_range(TDEngineDataBaseConfig *db, const char *deviceID, int64_t start, int64_t end, DataModel ***dataModels, int *count)
{
    if (!db || !db->conn || !deviceID || !dataModels || !count)
        return -1;

    char *legalTable = replace_char(deviceID, '-', '_');

    time_t start_time = start;
    time_t end_time = end;
    struct tm *start_tm = gmtime(&start_time);
    struct tm *end_tm = gmtime(&end_time);

    char start_str[64], end_str[64];
    strftime(start_str, sizeof(start_str), "%Y-%m-%d %H:%M:%S", start_tm);
    strftime(end_str, sizeof(end_str), "%Y-%m-%d %H:%M:%S", end_tm);

    char querySQL[1024];
    snprintf(querySQL, sizeof(querySQL),
             "SELECT ts, deviceid, propertyname, data, type FROM %s WHERE ts >= '%s' AND ts <= '%s'",
             legalTable, start_str, end_str);

    log_info("Query SQL: %s", querySQL);

    TAOS_RES *result = taos_query(db->conn, querySQL);
    if (taos_errno(result) != 0)
    {
        log_error("Failed to query data by time range: %s", taos_errstr(result));
        free(legalTable);
        taos_free_result(result);
        return -1;
    }

    *count = 0;
    TAOS_ROW row;
    while ((row = taos_fetch_row(result)) != NULL)
    {
        (*count)++;
    }

    if (*count == 0)
    {
        *dataModels = NULL;
        free(legalTable);
        taos_free_result(result);
        return 0;
    }

    free(legalTable);
    taos_free_result(result);
    return 0;
}