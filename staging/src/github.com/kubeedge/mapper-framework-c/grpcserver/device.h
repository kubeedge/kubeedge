#ifndef GRPCSERVER_DEVICE_H
#define GRPCSERVER_DEVICE_H

#include "dmi/v1beta1/api.pb-c.h"

// Forward declaration of the device panel structure
typedef struct DevPanel DevPanel;

// Device service structure
typedef struct {
    DevPanel *dev_panel; // Pointer to the device panel
} DeviceService;

// Creates a new device service
DeviceService *device_service_new(DevPanel *panel);

// Frees the memory allocated for a device service
void device_service_free(DeviceService *svc);

// gRPC interface functions

// Registers a device
int device_register(DeviceService *svc, const V1beta1__RegisterDeviceRequest *req, V1beta1__RegisterDeviceResponse *resp);

// Removes a device
int device_remove(DeviceService *svc, const V1beta1__RemoveDeviceRequest *req, V1beta1__RemoveDeviceResponse *resp);

// Updates a device
int device_update(DeviceService *svc, const V1beta1__UpdateDeviceRequest *req, V1beta1__UpdateDeviceResponse *resp);

// Creates a new device model
int device_create_model(DeviceService *svc, const V1beta1__CreateDeviceModelRequest *req, V1beta1__CreateDeviceModelResponse *resp);

// Updates an existing device model
int device_update_model(DeviceService *svc, const V1beta1__UpdateDeviceModelRequest *req, V1beta1__UpdateDeviceModelResponse *resp);

// Removes a device model
int device_remove_model(DeviceService *svc, const V1beta1__RemoveDeviceModelRequest *req, V1beta1__RemoveDeviceModelResponse *resp);

// Retrieves a device
int device_get(DeviceService *svc, const V1beta1__GetDeviceRequest *req, V1beta1__GetDeviceResponse *resp);

#endif // GRPCSERVER_DEVICE_H