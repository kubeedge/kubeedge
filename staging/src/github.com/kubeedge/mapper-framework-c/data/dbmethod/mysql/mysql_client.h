#ifndef MYSQL_CLIENT_H
#define MYSQL_CLIENT_H

#include "common/datamodel.h"
#include "driver/driver.h"
#include <mysql.h>

// MySQL client configuration
typedef struct
{
    char *addr;
    char *database;
    char *userName;
    char *password;
    int port;
} MySQLClientConfig;

// MySQL database configuration
typedef struct
{
    MySQLClientConfig config;
    MYSQL *conn;
} MySQLDataBaseConfig;

// MySQL data handler arguments
typedef struct
{
    MySQLDataBaseConfig dbConfig;
    DataModel *dataModel;
    int reportCycleMs;
    CustomizedClient *customizedClient;
    VisitorConfig *visitorConfig;
    int running;
} MySQLDataHandlerArgs;

// MySQL client functions
int mysql_parse_client_config(const char *json, MySQLClientConfig *out);
int mysql_init_client(MySQLDataBaseConfig *db);
void mysql_close_client(MySQLDataBaseConfig *db);
int mysql_add_data(MySQLDataBaseConfig *db, const DataModel *data);

#endif