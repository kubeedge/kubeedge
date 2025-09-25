#ifndef DEVICE_DEVICESTATUS_H
#define DEVICE_DEVICESTATUS_H

#include "common/const.h"
#include "common/configmaptype.h"
#include <pthread.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct Device Device;

// Represents the device status manager
typedef struct DeviceStatusManager {
    int healthCheckRunning;           // Indicates if health check is running
    int capacity;                     // Capacity of the status list
    int statusCount;                  // Number of statuses
    DeviceStatus **statusList;        // List of device statuses
    pthread_mutex_t statusMutex;      // Mutex for thread safety
} DeviceStatusManager;

/* 接口声明保持不变 */
int device_set_status(Device *device, const char *newStatus);
const char *device_get_status(Device *device);
DeviceStatus *device_status_new(const char *initialStatus);
void device_status_free(DeviceStatus *status);
int device_status_update(Device *device, const char *newStatus);
int device_status_check_change(Device *device, const char *currentStatus);
const char *device_status_get_current(Device *device);
const char *device_status_get_last(Device *device);
long long device_status_get_last_update_time(Device *device);
int device_status_health_check(Device *device);
int device_status_start_health_monitor(Device *device);
int device_status_stop_health_monitor(Device *device);
int device_status_send_event(Device *device, const char *eventType, const char *message);
int device_status_handle_offline(Device *device);
int device_status_handle_online(Device *device);
DeviceStatusManager *device_status_manager_new(void);
void device_status_manager_free(DeviceStatusManager *manager);
int device_status_manager_add(DeviceStatusManager *manager, Device *device);
int device_status_manager_remove(DeviceStatusManager *manager, const char *deviceId);
int device_status_manager_update_all(DeviceStatusManager *manager);

#ifdef __cplusplus
}
#endif
#endif