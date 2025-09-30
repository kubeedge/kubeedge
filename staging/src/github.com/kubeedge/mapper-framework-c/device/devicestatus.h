#ifndef DEVICE_DEVICESTATUS_H
#define DEVICE_DEVICESTATUS_H

#include "common/const.h"
#include "common/configmaptype.h"
#include <pthread.h>

#ifdef __cplusplus
extern "C"
{
#endif

    typedef struct Device Device;

    typedef struct DeviceStatusManager
    {
        int healthCheckRunning;
        int capacity;
        int statusCount;
        DeviceStatus **statusList;
        pthread_mutex_t statusMutex;
    } DeviceStatusManager;

    int device_set_status(Device *device, const char *newStatus);
    const char *device_get_status(Device *device);
    int device_status_update(Device *device, const char *newStatus);
    int device_status_check_change(Device *device, const char *currentStatus);
    const char *device_status_get_current(Device *device);
    long long device_status_get_last_update_time(Device *device);
    int device_status_health_check(Device *device);
    int device_status_start_health_monitor(Device *device);
    int device_status_stop_health_monitor(Device *device);
    int device_status_send_event(Device *device, const char *eventType, const char *message);
    int device_status_handle_offline(Device *device);
    int device_status_handle_online(Device *device);
#ifdef __cplusplus
}
#endif
#endif