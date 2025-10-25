#include "devicetwin.h"
#include "device.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <time.h>
#include <unistd.h>
#include <cjson/cJSON.h>

static long long get_current_time_ms(void)
{
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    return ts.tv_sec * 1000LL + ts.tv_nsec / 1000000LL;
}



int devicetwin_get(Device *device, const char *propertyName, TwinResult *result)
{
    if (!device || !propertyName || !result)
        return -1;

    memset(result, 0, sizeof(TwinResult));
    result->timestamp = get_current_time_ms();

    Twin *twin = NULL;
    for (int i = 0; i < device->instance.twinsCount; i++)
    {
        if (device->instance.twins[i].propertyName &&
            strcmp(device->instance.twins[i].propertyName, propertyName) == 0)
        {
            twin = &device->instance.twins[i];
            break;
        }
    }

    if (!twin)
    {
        result->error = strdup("Property not found");
        return -1;
    }

    if (twin->reported.value)
    {
        result->value = strdup(twin->reported.value);
        result->success = 1;
        return 0;
    }

    VisitorConfig visitorConfig = (VisitorConfig){0};
    visitorConfig.propertyName = (char *)propertyName;
    visitorConfig.protocolName = device->instance.protocolName;
    if (twin->property && twin->property->visitors)
    {
        visitorConfig.configData = twin->property->visitors;
    }

    void *deviceData = NULL;
    int ret = GetDeviceData(device->client, &visitorConfig, &deviceData);
    if (ret != 0 || !deviceData)
    {
        result->error = strdup("Failed to read device data");
        return -1;
    }

    result->value = strdup((char *)deviceData);
    free(deviceData);
    result->success = 1;
    return 0;
}

int devicetwin_set(Device *device, const char *propertyName, const char *value, TwinResult *result)
{
    if (!device || !propertyName || !value || !result)
        return -1;

    memset(result, 0, sizeof(TwinResult));
    result->timestamp = get_current_time_ms();

    log_debug("Setting twin property %s for device %s to value: %s",
              propertyName, device->instance.name, value);

    Twin *twin = NULL;
    for (int i = 0; i < device->instance.twinsCount; i++)
    {
        if (device->instance.twins[i].propertyName &&
            strcmp(device->instance.twins[i].propertyName, propertyName) == 0)
        {
            twin = &device->instance.twins[i];
            break;
        }
    }

    if (!twin || !twin->property)
    {
        result->error = strdup("Property not found or not configured");
        return -1;
    }

    if (devicetwin_validate_data(twin, value) != 0)
    {
        result->error = strdup("Invalid data value");
        return -1;
    }

    VisitorConfig visitorConfig = {0};
    visitorConfig.propertyName = (char *)propertyName;
    visitorConfig.protocolName = device->instance.protocolName;

    if (twin->property->visitors)
    {
        visitorConfig.configData = twin->property->visitors;
    }

    int ret = DeviceDataWrite(device->client, &visitorConfig, "SetProperty", propertyName, value);
    if (ret != 0)
    {
        result->error = strdup("Failed to write device data");
        return -1;
    }

    void *deviceData = NULL;
    ret = GetDeviceData(device->client, &visitorConfig, &deviceData);
    if (ret == 0 && deviceData)
    {
        result->value = strdup((char *)deviceData);
        result->success = 1;
        free(deviceData);
    }
    else
    {
        result->value = strdup(value);
        result->success = 1;
    }

    log_debug("Set twin property %s to value: %s", propertyName, result->value);
    return 0;
}

int devicetwin_process_data(Device *device, const Twin *twin, const void *data)
{
    if (!device || !twin || !data)
        return -1;

    log_debug("Processing twin data for property: %s", twin->propertyName);

    return 0;
}

int devicetwin_validate_data(const Twin *twin, const char *value)
{
    if (!twin || !twin->property || !value)
        return -1;

    if (strlen(value) == 0)
    {
        return -1;
    }

    return 0;
}


char *devicetwin_build_report_data(const char *propertyName, const char *value, long long timestamp)
{
    if (!propertyName || !value)
        return NULL;

    cJSON *root = cJSON_CreateObject();
    cJSON *twin = cJSON_CreateObject();
    cJSON *reported = cJSON_CreateObject();

    cJSON_AddStringToObject(reported, propertyName, value);
    cJSON_AddNumberToObject(reported, "timestamp", timestamp);

    cJSON_AddItemToObject(twin, "reported", reported);
    cJSON_AddItemToObject(root, "twin", twin);

    char *jsonString = cJSON_PrintUnformatted(root);
    cJSON_Delete(root);

    return jsonString;
}

int devicetwin_report_to_cloud(Device *device, const char *propertyName, const char *value)
{
    if (!device || !propertyName || !value)
        return -1;

    log_debug("Reporting twin property %s=%s for device %s",
              propertyName, value, device->instance.name);

    char *reportData = devicetwin_build_report_data(propertyName, value, get_current_time_ms());
    if (!reportData)
        return -1;

    free(reportData);
    return 0;
}

TwinProcessor *devicetwin_processor_new(const Twin *twin)
{
    if (!twin)
        return NULL;

    TwinProcessor *processor = calloc(1, sizeof(TwinProcessor));
    if (!processor)
        return NULL;

    processor->propertyName = twin->propertyName ? strdup(twin->propertyName) : NULL;

    processor->dataType = strdup("string");
    processor->accessMode = strdup("ReadWrite");

    processor->reportCycle = 10000;
    processor->reportThreadRunning = 0;

    return processor;
}

void devicetwin_processor_free(TwinProcessor *processor)
{
    if (!processor)
        return;

    processor->reportThreadRunning = 0;

    free(processor->propertyName);
    free(processor->dataType);
    free(processor->accessMode);
    free(processor);
}

