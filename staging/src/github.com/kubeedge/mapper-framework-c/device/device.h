#ifndef DEVICE_DEVICE_H
#define DEVICE_DEVICE_H

#include "common/configmaptype.h"
#include "common/eventtype.h"
#include "driver/driver.h"
#include <pthread.h>

#ifdef __cplusplus
extern "C"
{
#endif

    typedef struct Device
    {
        DeviceInstance instance;
        DeviceModel model;
        CustomizedClient *client;
        char *status;
        pthread_mutex_t mutex;
        int stopChan;
        pthread_t dataThread;
        int dataThreadRunning;
        int started;
    } Device;

    typedef struct
    {
        Device **devices;
        int deviceCount;
        int capacity;
        pthread_mutex_t managerMutex;
        int stopped;
    } DeviceManager;

    Device *device_new(const DeviceInstance *instance, const DeviceModel *model);
    void device_free(Device *device);
    int device_start(Device *device);
    int device_stop(Device *device);
    int device_restart(Device *device);
    int device_deal_twin(Device *device, const Twin *twin);
    const char *device_get_status(Device *device);
    int device_set_status(Device *device, const char *status);
    DeviceManager *device_manager_new(void);
    void device_manager_free(DeviceManager *manager);
    int device_manager_add(DeviceManager *manager, Device *device);
    int device_manager_remove(DeviceManager *manager, const char *deviceId);
    Device *device_manager_get(DeviceManager *manager, const char *deviceId);
    int device_manager_start_all(DeviceManager *manager);
    int device_manager_stop_all(DeviceManager *manager);
    int device_resolve_offset(struct Device *device, const char *propName);

#ifdef __cplusplus
}
#endif

#endif