#include "redis_client.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <cjson/cJSON.h>

int redis_parse_client_config(const char *json, RedisClientConfig *out)
{
    if (!out)
        return -1;

    if (json && *json)
    {
        cJSON *root = cJSON_Parse(json);
        if (root)
        {
            cJSON *jaddr = cJSON_GetObjectItem(root, "addr");
            cJSON *jdb = cJSON_GetObjectItem(root, "db");
            cJSON *jpool = cJSON_GetObjectItem(root, "poolSize");
            cJSON *jminidle = cJSON_GetObjectItem(root, "minIdleConns");
            cJSON *jpwd = cJSON_GetObjectItem(root, "password");

            out->addr = jaddr && cJSON_IsString(jaddr) ? strdup(jaddr->valuestring) : NULL;
            out->db = jdb && cJSON_IsNumber(jdb) ? jdb->valueint : 0;
            out->poolSize = jpool && cJSON_IsNumber(jpool) ? jpool->valueint : 10;
            out->minIdleConns = jminidle && cJSON_IsNumber(jminidle) ? jminidle->valueint : 3;
            out->password = jpwd && cJSON_IsString(jpwd) ? strdup(jpwd->valuestring) : NULL;

            cJSON_Delete(root);

            if (!out->addr)
            {
                const char *addr = getenv("REDIS_ADDR");
                if (addr)
                    out->addr = strdup(addr);
                else
                {
                    const char *host = getenv("REDIS_HOST");
                    const char *port = getenv("REDIS_PORT");
                    if (host && port)
                    {
                        size_t len = strlen(host) + 1 + strlen(port) + 1;
                        char *buf = malloc(len);
                        if (buf)
                        {
                            snprintf(buf, len, "%s:%s", host, port);
                            out->addr = buf;
                        }
                        else
                        {
                            out->addr = strdup("localhost:6379");
                        }
                    }
                    else
                    {
                        out->addr = strdup("localhost:6379");
                    }
                }
            }
            if (!out->password)
            {
                const char *pwd = getenv("REDIS_PASSWORD");
                if (!pwd)
                    pwd = getenv("PASSWORD");
                out->password = pwd ? strdup(pwd) : NULL;
            }
            return 0;
        }
        else
        {
            log_warn("redis_parse_client_config: invalid JSON, falling back to env");
        }
    }

    const char *addr = getenv("REDIS_ADDR");
    if (!addr)
    {
        const char *host = getenv("REDIS_HOST");
        const char *port = getenv("REDIS_PORT");
        if (host && port)
        {
            size_t len = strlen(host) + 1 + strlen(port) + 1;
            char *buf = malloc(len);
            if (buf)
            {
                snprintf(buf, len, "%s:%s", host, port);
                out->addr = buf;
            }
            else
            {
                out->addr = strdup("localhost:6379");
            }
        }
        else
        {
            out->addr = strdup("localhost:6379");
        }
    }
    else
    {
        out->addr = strdup(addr);
    }

    const char *dbenv = getenv("REDIS_DB");
    out->db = dbenv ? atoi(dbenv) : 0;

    const char *poolEnv = getenv("REDIS_POOLSIZE");
    out->poolSize = poolEnv ? atoi(poolEnv) : 10;
    const char *minIdleEnv = getenv("REDIS_MINIDLE");
    out->minIdleConns = minIdleEnv ? atoi(minIdleEnv) : 3;

    const char *pwd = getenv("REDIS_PASSWORD");
    if (!pwd)
        pwd = getenv("PASSWORD");
    out->password = pwd ? strdup(pwd) : NULL;

    return 0;
}

int redis_init_client(RedisDataBaseConfig *db)
{
    if (!db)
        return -1;

    char *addr_copy = strdup(db->config.addr);
    char *host = strtok(addr_copy, ":");
    char *port_str = strtok(NULL, ":");
    int port = port_str ? atoi(port_str) : 6379;

    db->conn = redisConnect(host, port);
    free(addr_copy);

    if (db->conn == NULL || db->conn->err)
    {
        if (db->conn)
        {
            redisFree(db->conn);
            db->conn = NULL;
        }
        else
        {
            log_error("Connection error: can't allocate redis context");
        }
        return -1;
    }

    if (db->config.password)
    {
        redisReply *reply = redisCommand(db->conn, "AUTH %s", db->config.password);
        if (reply == NULL)
        {
            log_error("AUTH command failed");
            redisFree(db->conn);
            db->conn = NULL;
            return -1;
        }
        if (reply->type == REDIS_REPLY_ERROR)
        {
            log_error("AUTH failed: %s", reply->str);
            freeReplyObject(reply);
            redisFree(db->conn);
            db->conn = NULL;
            return -1;
        }
        freeReplyObject(reply);
    }

    if (db->config.db != 0)
    {
        redisReply *reply = redisCommand(db->conn, "SELECT %d", db->config.db);
        if (reply == NULL || reply->type == REDIS_REPLY_ERROR)
        {
            log_error("SELECT DB failed");
            if (reply)
                freeReplyObject(reply);
            redisFree(db->conn);
            db->conn = NULL;
            return -1;
        }
        freeReplyObject(reply);
    }

    redisReply *reply = redisCommand(db->conn, "PING");
    if (reply == NULL || reply->type != REDIS_REPLY_STATUS || strcmp(reply->str, "PONG") != 0)
    {
        log_error("PING failed");
        if (reply)
            freeReplyObject(reply);
        redisFree(db->conn);
        db->conn = NULL;
        return -1;
    }
    freeReplyObject(reply);

    log_info("Redis client initialized successfully");
    return 0;
}

void redis_close_client(RedisDataBaseConfig *db)
{
    if (db && db->conn)
    {
        redisFree(db->conn);
        db->conn = NULL;
    }
}

int redis_add_data(RedisDataBaseConfig *db, const DataModel *data)
{
    if (!db || !db->conn || !data)
        return -1;

    char deviceData[1024];
    snprintf(deviceData, sizeof(deviceData),
             "TimeStamp: %ld PropertyName: %s data: %s",
             data->timeStamp,
             data->propertyName ? data->propertyName : "",
             data->value ? data->value : "");

    redisReply *reply = redisCommand(db->conn, "ZADD %s %ld %s",
                                     data->deviceName ? data->deviceName : "unknown_device",
                                     data->timeStamp,
                                     deviceData);

    if (reply == NULL)
    {
        log_error("ZADD command failed");
        return -1;
    }

    if (reply->type == REDIS_REPLY_ERROR)
    {
        log_error("ZADD failed: %s", reply->str);
        freeReplyObject(reply);
        return -1;
    }

    freeReplyObject(reply);
    return 0;
}

int redis_get_data_by_device_id(RedisDataBaseConfig *db, const char *deviceID, DataModel ***dataModels, int *count)
{
    if (!db || !db->conn || !deviceID || !dataModels || !count)
        return -1;

    redisReply *reply = redisCommand(db->conn, "ZREVRANGE %s 0 -1", deviceID);

    if (reply == NULL)
    {
        log_error("ZREVRANGE command failed");
        return -1;
    }

    if (reply->type == REDIS_REPLY_ERROR)
    {
        log_error("ZREVRANGE failed: %s", reply->str);
        freeReplyObject(reply);
        return -1;
    }

    if (reply->type != REDIS_REPLY_ARRAY)
    {
        log_error("Unexpected reply type");
        freeReplyObject(reply);
        return -1;
    }

    *count = reply->elements;
    if (*count == 0)
    {
        *dataModels = NULL;
        freeReplyObject(reply);
        return 0;
    }

    *dataModels = calloc(*count, sizeof(DataModel *));
    if (!*dataModels)
    {
        log_error("Failed to allocate memory for data models");
        freeReplyObject(reply);
        return -1;
    }

    for (int i = 0; i < *count; i++)
    {
        (*dataModels)[i] = calloc(1, sizeof(DataModel));
        if ((*dataModels)[i])
        {
            (*dataModels)[i]->deviceName = strdup(deviceID);
            (*dataModels)[i]->value = strdup(reply->element[i]->str);
        }
    }

    freeReplyObject(reply);
    return 0;
}