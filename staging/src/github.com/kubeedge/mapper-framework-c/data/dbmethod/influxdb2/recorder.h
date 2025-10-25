#ifndef INFLUXDB2_RECORDER_H
#define INFLUXDB2_RECORDER_H

#include "influxdb2_client.h"

int influxdb2_recorder_set_db(const Influxdb2DataBaseConfig *cfg);

int influxdb2_recorder_record(const char *ns, const char *device, const char *prop, const char *value, long long ts_ms);

void influxdb2_recorder_close(void);

#endif // INFLUXDB2_RECORDER_H