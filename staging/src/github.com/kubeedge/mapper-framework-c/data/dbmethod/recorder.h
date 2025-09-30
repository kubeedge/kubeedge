#ifndef DBMETHOD_RECORDER_H
#define DBMETHOD_RECORDER_H

#include "device/device.h"

int dbmethod_recorder_record(Device *device, const char *propertyName, const char *value, long long timestamp);

#endif // DBMETHOD_RECORDER_H