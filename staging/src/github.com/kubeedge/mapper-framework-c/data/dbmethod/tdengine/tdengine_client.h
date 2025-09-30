#ifndef TDENGINE_CLIENT_H
#define TDENGINE_CLIENT_H

#include "common/datamodel.h"
#include "driver/driver.h"
#include <taos.h>

typedef struct
{
    char *addr;
    char *dbName;
    char *username;
    char *password;
} TDEngineClientConfig;

typedef struct
{
    TDEngineClientConfig config;
    TAOS *conn;
} TDEngineDataBaseConfig;

typedef struct
{
    TDEngineDataBaseConfig dbConfig;
    DataModel *dataModel;
    int reportCycleMs;
    CustomizedClient *customizedClient;
    VisitorConfig *visitorConfig;
    int running;
} TDEngineDataHandlerArgs;

int tdengine_parse_client_config(const char *json, TDEngineClientConfig *out);
int tdengine_init_client(TDEngineDataBaseConfig *db);
void tdengine_close_client(TDEngineDataBaseConfig *db);
int tdengine_add_data(TDEngineDataBaseConfig *db, const DataModel *data);
int tdengine_get_data_by_device_id(TDEngineDataBaseConfig *db, const char *deviceID, DataModel ***dataModels, int *count);
int tdengine_get_data_by_time_range(TDEngineDataBaseConfig *db, const char *deviceID, int64_t start, int64_t end, DataModel ***dataModels, int *count);

#endif