#ifndef DEVICE_DEVICETWIN_H
#define DEVICE_DEVICETWIN_H

#include "common/configmaptype.h"
#include "common/datamodel.h"
#include "driver/driver.h"
#include <pthread.h>

struct Device;
typedef struct Device Device;

typedef struct
{
    int success;
    char *value;
    char *error;
    long long timestamp;
} TwinResult;

typedef struct
{
    char *propertyName;
    char *dataType;
    char *accessMode;
    VisitorConfig *visitorConfig;
    int reportCycle;
    pthread_t reportThread;
    int reportThreadRunning;
} TwinProcessor;

typedef struct
{
    TwinProcessor **processors;
    int processorCount;
    int capacity;
    pthread_mutex_t twinMutex;
} TwinManager;

int devicetwin_get(Device *device, const char *propertyName, TwinResult *result);
int devicetwin_set(Device *device, const char *propertyName, const char *value, TwinResult *result);

int devicetwin_process_data(Device *device, const Twin *twin, const void *data);
int devicetwin_validate_data(const Twin *twin, const char *value);

int devicetwin_report_to_cloud(Device *device, const char *propertyName, const char *value);

TwinProcessor *devicetwin_processor_new(const Twin *twin);
void devicetwin_processor_free(TwinProcessor *processor);

int devicetwin_manager_remove(TwinManager *manager, const char *propertyName);
TwinProcessor *devicetwin_manager_get(TwinManager *manager, const char *propertyName);

char *devicetwin_build_report_data(const char *propertyName, const char *value, long long timestamp);

#endif // DEVICE_DEVICETWIN_H