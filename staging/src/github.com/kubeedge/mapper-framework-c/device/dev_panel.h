#ifndef DEVICE_DEV_PANEL_H
#define DEVICE_DEV_PANEL_H

#include "common/configmaptype.h"
#include "device/device.h"

#ifdef __cplusplus
extern "C" {
#endif

// Device panel interface functions

// Retrieve the twin result of a device
int dev_panel_get_twin_result(DeviceManager *manager, const char *deviceId, 
                              const char *propertyName, char **value, char **datatype);

// Write data to a device
int dev_panel_write_device(DeviceManager *manager, const char *method, 
                           const char *deviceId, const char *propertyName, const char *data);

// Retrieve the methods of a device
int dev_panel_get_device_method(DeviceManager *manager, const char *deviceId,
                                char ***method_map, int *method_count,
                                char ***property_map, int *property_count);

// Retrieve device information
int dev_panel_get_device(DeviceManager *manager, const char *deviceId, DeviceInstance *instance);

// Retrieve the model of a device
int dev_panel_get_model(DeviceManager *manager, const char *modelId, DeviceModel *model);

// Retrieve all twins of devices
int dev_panel_get_all_twins(DeviceManager *manager, char **response);

// Check if a device exists
int dev_panel_has_device(DeviceManager *manager, const char *deviceId);

// Update a device
int dev_panel_update_dev(DeviceManager *manager, const DeviceModel *model, const DeviceInstance *instance);

// Update a model
int dev_panel_update_model(DeviceManager *manager, const DeviceModel *model);

// Remove a model
int dev_panel_remove_model(DeviceManager *manager, const char *modelId);

#ifdef __cplusplus
}
#endif

#endif // DEVICE_DEV_PANEL_H