#ifndef MYSQL_RECORDER_H
#define MYSQL_RECORDER_H

#include "data/dbmethod/mysql/mysql_client.h"

#ifdef __cplusplus
extern "C" {
#endif

// Inject a global MySQL connection (can be NULL, in which case record operations are no-op)
void mysql_recorder_set_db(MySQLDataBaseConfig *db);

// Record a time-series data entry; ts_ms is the timestamp in milliseconds (converted to seconds internally)
int mysql_recorder_record(const char *ns,
                          const char *deviceName,
                          const char *propertyName,
                          const char *value,
                          long long ts_ms);

// Initialize recorder by reading MySQL config from environment variables.
// No-op if MYSQL_ENABLED is "0" or "false".
int mysql_recorder_init_from_env(void);

// Shutdown recorder and free resources (if initialized).
void mysql_recorder_shutdown(void);

#ifdef __cplusplus
}
#endif

#endif