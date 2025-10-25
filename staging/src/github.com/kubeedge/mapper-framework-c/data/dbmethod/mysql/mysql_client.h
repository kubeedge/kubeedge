#ifndef MYSQL_CLIENT_H
#define MYSQL_CLIENT_H

#include "common/datamodel.h"
#include "driver/driver.h"
#include <mysql.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct
{
    char *addr;
    char *database;
    char *userName;
    char *password;
    int port;
} MySQLClientConfig;

typedef struct
{
    MySQLClientConfig config;
    MYSQL *conn;
} MySQLDataBaseConfig;


int mysql_parse_client_config(const char *json, MySQLClientConfig *out);
int mysql_init_client(MySQLDataBaseConfig *db);
void mysql_close_client(MySQLDataBaseConfig *db);
int mysql_add_data(MySQLDataBaseConfig *db, const DataModel *data);

MySQLDataBaseConfig *mysql_get_cached_db(const MySQLClientConfig *cfg);
void mysql_release_cached_db(MySQLDataBaseConfig *db);

#ifdef __cplusplus
}
#endif

#endif