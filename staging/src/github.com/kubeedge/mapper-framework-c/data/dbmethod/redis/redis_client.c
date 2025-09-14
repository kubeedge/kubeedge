#include "redis_client.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <cjson/cJSON.h>

int redis_parse_client_config(const char *json, RedisClientConfig *out) {
    if (!json || !out) return -1;
    
    cJSON *root = cJSON_Parse(json);
    if (!root) {
        log_error("Failed to parse JSON config");
        return -1;
    }
    
    cJSON *addr = cJSON_GetObjectItem(root, "addr");
    cJSON *db = cJSON_GetObjectItem(root, "db");
    cJSON *poolSize = cJSON_GetObjectItem(root, "poolSize");
    cJSON *minIdleConns = cJSON_GetObjectItem(root, "minIdleConns");
    
    out->addr = addr ? strdup(addr->valuestring) : strdup("localhost:6379");
    out->db = db ? db->valueint : 0;
    out->poolSize = poolSize ? poolSize->valueint : 10;
    out->minIdleConns = minIdleConns ? minIdleConns->valueint : 3;
    
    // 从环境变量读取密码
    char *pwd = getenv("PASSWORD");
    out->password = pwd ? strdup(pwd) : NULL;
    
    cJSON_Delete(root);
    return 0;
}

int redis_init_client(RedisDataBaseConfig *db) {
    if (!db) return -1;
    
    // 解析地址和端口
    char *addr_copy = strdup(db->config.addr);
    char *host = strtok(addr_copy, ":");
    char *port_str = strtok(NULL, ":");
    int port = port_str ? atoi(port_str) : 6379;
    
    // 连接 Redis
    db->conn = redisConnect(host, port);
    free(addr_copy);
    
    if (db->conn == NULL || db->conn->err) {
        if (db->conn) {
            log_error("Connection error: %s", db->conn->errstr);
            redisFree(db->conn);
            db->conn = NULL;
        } else {
            log_error("Connection error: can't allocate redis context");
        }
        return -1;
    }
    
    // 认证（如果有密码）
    if (db->config.password) {
        redisReply *reply = redisCommand(db->conn, "AUTH %s", db->config.password);
        if (reply == NULL) {
            log_error("AUTH command failed");
            redisFree(db->conn);
            db->conn = NULL;
            return -1;
        }
        if (reply->type == REDIS_REPLY_ERROR) {
            log_error("AUTH failed: %s", reply->str);
            freeReplyObject(reply);
            redisFree(db->conn);
            db->conn = NULL;
            return -1;
        }
        freeReplyObject(reply);
    }
    
    // 选择数据库
    if (db->config.db != 0) {
        redisReply *reply = redisCommand(db->conn, "SELECT %d", db->config.db);
        if (reply == NULL || reply->type == REDIS_REPLY_ERROR) {
            log_error("SELECT DB failed");
            if (reply) freeReplyObject(reply);
            redisFree(db->conn);
            db->conn = NULL;
            return -1;
        }
        freeReplyObject(reply);
    }
    
    // 测试连接
    redisReply *reply = redisCommand(db->conn, "PING");
    if (reply == NULL || reply->type != REDIS_REPLY_STATUS || strcmp(reply->str, "PONG") != 0) {
        log_error("PING failed");
        if (reply) freeReplyObject(reply);
        redisFree(db->conn);
        db->conn = NULL;
        return -1;
    }
    freeReplyObject(reply);
    
    log_info("Redis client initialized successfully");
    return 0;
}

void redis_close_client(RedisDataBaseConfig *db) {
    if (db && db->conn) {
        redisFree(db->conn);
        db->conn = NULL;
    }
}

int redis_add_data(RedisDataBaseConfig *db, const DataModel *data) {
    if (!db || !db->conn || !data) return -1;
    
    // 构造设备数据字符串
    char deviceData[1024];
    snprintf(deviceData, sizeof(deviceData), 
             "TimeStamp: %ld PropertyName: %s data: %s", 
             data->timeStamp, 
             data->propertyName ? data->propertyName : "", 
             data->value ? data->value : "");
    
    // 使用 ZADD 命令添加到有序集合
    redisReply *reply = redisCommand(db->conn, "ZADD %s %ld %s", 
                                   data->deviceName ? data->deviceName : "unknown_device",
                                   data->timeStamp,
                                   deviceData);
    
    if (reply == NULL) {
        log_error("ZADD command failed");
        return -1;
    }
    
    if (reply->type == REDIS_REPLY_ERROR) {
        log_error("ZADD failed: %s", reply->str);
        freeReplyObject(reply);
        return -1;
    }
    
    freeReplyObject(reply);
    return 0;
}

int redis_get_data_by_device_id(RedisDataBaseConfig *db, const char *deviceID, DataModel ***dataModels, int *count) {
    if (!db || !db->conn || !deviceID || !dataModels || !count) return -1;
    
    // 使用 ZREVRANGE 命令获取数据（按时间戳倒序）
    redisReply *reply = redisCommand(db->conn, "ZREVRANGE %s 0 -1", deviceID);
    
    if (reply == NULL) {
        log_error("ZREVRANGE command failed");
        return -1;
    }
    
    if (reply->type == REDIS_REPLY_ERROR) {
        log_error("ZREVRANGE failed: %s", reply->str);
        freeReplyObject(reply);
        return -1;
    }
    
    if (reply->type != REDIS_REPLY_ARRAY) {
        log_error("Unexpected reply type");
        freeReplyObject(reply);
        return -1;
    }
    
    *count = reply->elements;
    if (*count == 0) {
        *dataModels = NULL;
        freeReplyObject(reply);
        return 0;
    }
    
    // 分配内存存储数据模型
    *dataModels = calloc(*count, sizeof(DataModel*));
    if (!*dataModels) {
        log_error("Failed to allocate memory for data models");
        freeReplyObject(reply);
        return -1;
    }
    
    // 解析返回的数据（这里简化处理，实际应该解析 deviceData 字符串）
    for (int i = 0; i < *count; i++) {
        (*dataModels)[i] = calloc(1, sizeof(DataModel));
        if ((*dataModels)[i]) {
            (*dataModels)[i]->deviceName = strdup(deviceID);
            (*dataModels)[i]->value = strdup(reply->element[i]->str);
            // 可以进一步解析 TimeStamp 和 PropertyName
        }
    }
    
    freeReplyObject(reply);
    return 0;
}