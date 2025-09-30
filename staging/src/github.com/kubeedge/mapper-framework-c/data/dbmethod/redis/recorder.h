#ifndef REDIS_RECORDER_H
#define REDIS_RECORDER_H

#include "redis_client.h"

/* Attach a Redis DB config (will init client). db must remain valid or be managed by caller. */
int redis_recorder_set_db(RedisDataBaseConfig *db);

/* Record one data point: ns/device/prop -> stored in Redis (uses redis_add_data) */
int redis_recorder_record(const char *ns, const char *device, const char *prop, const char *value, long long ts_ms);

/* Cleanup */
void redis_recorder_close(void);

#endif // REDIS_RECORDER_H