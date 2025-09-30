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

int devicetwin_deal(Device *device, const Twin *twin);
int devicetwin_get(Device *device, const char *propertyName, TwinResult *result);
int devicetwin_set(Device *device, const char *propertyName, const char *value, TwinResult *result);

int devicetwin_process_data(Device *device, const Twin *twin, const void *data);
int devicetwin_validate_data(const Twin *twin, const char *value);
int devicetwin_convert_data(const Twin *twin, const char *rawValue, char **convertedValue);

int devicetwin_report_to_cloud(Device *device, const char *propertyName, const char *value);
int devicetwin_start_auto_report(Device *device, const Twin *twin);
int devicetwin_stop_auto_report(Device *device, const char *propertyName);

int devicetwin_handle_desired_change(Device *device, const Twin *twin, const char *newValue);
int devicetwin_handle_reported_update(Device *device, const Twin *twin, const char *newValue);

TwinProcessor *devicetwin_processor_new(const Twin *twin);
void devicetwin_processor_free(TwinProcessor *processor);

TwinManager *devicetwin_manager_new(void);
void devicetwin_manager_free(TwinManager *manager);
int devicetwin_manager_add(TwinManager *manager, const Twin *twin);
int devicetwin_manager_remove(TwinManager *manager, const char *propertyName);
TwinProcessor *devicetwin_manager_get(TwinManager *manager, const char *propertyName);

int devicetwin_parse_visitor_config(const char *configData, VisitorConfig *config);
char *devicetwin_build_report_data(const char *propertyName, const char *value, long long timestamp);

#endif // DEVICE_DEVICETWIN_H