#ifndef REDIS_RECORDER_H
#define REDIS_RECORDER_H

#include "redis_client.h"

int redis_recorder_set_db(RedisDataBaseConfig *db);

int redis_recorder_record(const char *ns, const char *device, const char *prop, const char *value, long long ts_ms);

void redis_recorder_close(void);

#endif // REDIS_RECORDER_H