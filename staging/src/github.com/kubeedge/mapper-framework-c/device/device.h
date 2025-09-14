#ifndef DEVICE_DEVICE_H
#define DEVICE_DEVICE_H

#include "common/configmaptype.h"
#include "common/eventtype.h"
#include "driver/driver.h"
#include <pthread.h>

/* Only declare interfaces, avoid including specific database/streaming implementations in public headers */

#ifdef __cplusplus
extern "C" {
#endif

#ifndef DEVICE_TYPE_DEFINED
#define DEVICE_TYPE_DEFINED
// Represents a device structure
typedef struct Device {
    DeviceInstance instance;       // Device instance information
    DeviceModel model;             // Device model information
    CustomizedClient *client;      // Customized client for the device
    char *status;                  // Device status
    pthread_mutex_t mutex;         // Mutex for thread safety
    int stopChan;                  // Stop channel flag
    pthread_t dataThread;          // Data processing thread
    int dataThreadRunning;         // Flag indicating if the data thread is running
} Device;
#endif

// Represents a device manager structure
typedef struct {
    Device **devices;              // Array of devices
    int deviceCount;               // Number of devices
    int capacity;                  // Capacity of the device array
    pthread_mutex_t managerMutex;  // Mutex for thread safety
    int stopped;                   // Flag indicating if all devices are stopped
} DeviceManager;

/* Interface declarations */
Device *device_new(const DeviceInstance *instance, const DeviceModel *model);
void device_free(Device *device);
int device_start(Device *device);
int device_stop(Device *device);
int device_restart(Device *device);
int device_deal_twin(Device *device, const Twin *twin);
int device_data_process(Device *device, const char *method, const char *config,
                        const char *propertyName, const void *data);
const char *device_get_status(Device *device);
int device_set_status(Device *device, const char *status);
DeviceManager *device_manager_new(void);
void device_manager_free(DeviceManager *manager);
int device_manager_add(DeviceManager *manager, Device *device);
int device_manager_remove(DeviceManager *manager, const char *deviceId);
Device *device_manager_get(DeviceManager *manager, const char *deviceId);
int device_manager_start_all(DeviceManager *manager);
int device_manager_stop_all(DeviceManager *manager);
int device_init_from_config(Device *device, const char *configPath);
int device_register_to_edge(Device *device);

#ifdef __cplusplus
}
#endif
#endif