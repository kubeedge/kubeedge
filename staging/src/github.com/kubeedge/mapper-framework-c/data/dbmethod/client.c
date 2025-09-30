#include "data/dbmethod/client.h"
#include "data/dbmethod/mysql/recorder.h"
#include "data/dbmethod/redis/recorder.h"
#include "data/dbmethod/influxdb2/recorder.h"
#include "data/dbmethod/tdengine/recorder.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>

MySQLDataBaseConfig *g_mysql = NULL;
RedisDataBaseConfig *g_redis = NULL;
Influxdb2Client *g_influxdb2 = NULL;
TDEngineDataBaseConfig *g_tdengine = NULL;

int dbmethod_global_init(void) {

    // MySQL
    {
        MySQLClientConfig clientCfg = {0};
        if (mysql_parse_client_config(NULL, &clientCfg) == 0) {
            g_mysql = (MySQLDataBaseConfig*)calloc(1, sizeof(MySQLDataBaseConfig));
            if (g_mysql) {
                g_mysql->config = clientCfg;
                if (mysql_init_client(g_mysql) == 0) {
                    mysql_recorder_set_db(g_mysql);
                } else {
                    mysql_close_client(g_mysql);
                    free(g_mysql);
                    g_mysql = NULL;
                }
            }
        }
    }
    // Redis
    {
        RedisClientConfig rcfg = {0};
        if (redis_parse_client_config(NULL, &rcfg) == 0) {
            g_redis = (RedisDataBaseConfig*)calloc(1, sizeof(RedisDataBaseConfig));
            if (g_redis) {
                g_redis->config = rcfg;
                if (redis_init_client(g_redis) == 0) {
                    redis_recorder_set_db(g_redis);
                } else {
                    redis_close_client(g_redis);
                    free(g_redis);
                    g_redis = NULL;
                }
            }
        }
    }
    // InfluxDB2
    {
        Influxdb2ClientConfig icfg = {0};
        Influxdb2DataConfig dcfg = {0};
        if (influxdb2_parse_client_config(NULL, &icfg) == 0) {
            Influxdb2DataBaseConfig dbcfg = {0};
            dbcfg.clientConfig = icfg;
            dbcfg.dataConfig = dcfg;
            influxdb2_recorder_set_db(&dbcfg);
            g_influxdb2 = (Influxdb2Client*)calloc(1, sizeof(Influxdb2Client));
            if (g_influxdb2) {
                if (influxdb2_init_client(&icfg, g_influxdb2) != 0) {
                    influxdb2_close_client(g_influxdb2);
                    free(g_influxdb2);
                    g_influxdb2 = NULL;
                }
            }
        }
    }
    // TDengine
    {
        TDEngineClientConfig tcfg = {0};
        if (tdengine_parse_client_config(NULL, &tcfg) == 0) {
            g_tdengine = (TDEngineDataBaseConfig*)calloc(1, sizeof(TDEngineDataBaseConfig));
            if (g_tdengine) {
                g_tdengine->config = tcfg;
                if (tdengine_init_client(g_tdengine) == 0) {
                    tdengine_recorder_set_db(g_tdengine);
                } else {
                    tdengine_close_client(g_tdengine);
                    free(g_tdengine);
                    g_tdengine = NULL;
                }
            }
        }
    }
    return 0;
}

void dbmethod_global_free(void) {
    if (g_mysql) {
        mysql_close_client(g_mysql);
        free(g_mysql->config.addr);
        free(g_mysql->config.database);
        free(g_mysql->config.userName);
        free(g_mysql->config.password);
        free(g_mysql);
        g_mysql = NULL;
    }
    if (g_redis) {
        redis_close_client(g_redis);
        free(g_redis->config.addr);
        free(g_redis->config.password);
        free(g_redis);
        g_redis = NULL;
    }
    if (g_influxdb2) {
        influxdb2_close_client(g_influxdb2);
        free(g_influxdb2);
        g_influxdb2 = NULL;
    }
    if (g_tdengine) {
        tdengine_close_client(g_tdengine);
        free(g_tdengine->config.addr);
        free(g_tdengine->config.dbName);
        free(g_tdengine->config.username);
        free(g_tdengine->config.password);
        free(g_tdengine);
        g_tdengine = NULL;
    }
}
