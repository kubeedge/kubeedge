#ifndef CLIENT_H
#define CLIENT_H

#include "data/dbmethod/mysql/mysql_client.h"
#include "data/dbmethod/redis/redis_client.h"
#include "data/dbmethod/influxdb2/influxdb2_client.h"
#include "data/dbmethod/tdengine/tdengine_client.h"

extern MySQLDataBaseConfig *g_mysql;
extern RedisDataBaseConfig *g_redis;
extern Influxdb2Client *g_influxdb2;
extern TDEngineDataBaseConfig *g_tdengine;

int dbmethod_global_init(void);
void dbmethod_global_free(void);
#endif