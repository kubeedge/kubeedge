#ifndef TDENGINE_RECORDER_H
#define TDENGINE_RECORDER_H

#include "tdengine_client.h"

int tdengine_recorder_set_db(TDEngineDataBaseConfig *db);

int tdengine_recorder_record(const char *ns, const char *device, const char *prop, const char *value, long long ts_ms);

void tdengine_recorder_close(void);

#endif // TDENGINE_RECORDER_H