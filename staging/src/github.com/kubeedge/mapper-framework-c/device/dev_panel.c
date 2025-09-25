#include "dev_panel.h"
#include "device.h"
#include "devicetwin.h"
#include "log/log.h"
#include <string.h>
#include <stdlib.h>
#include <stdio.h>
#include <cjson/cJSON.h>

// Retrieve the twin result of a device
int dev_panel_get_twin_result(DeviceManager *manager, const char *deviceId, 
                              const char *propertyName, char **value, char **datatype) {
    Device *device = device_manager_get(manager, deviceId);
    if (!device) {
        log_warn("Device %s not found", deviceId);
        return -1;
    }

    for (int i = 0; i < device->instance.twinsCount; i++) {
        Twin *twin = &device->instance.twins[i];
        if (strcmp(twin->propertyName, propertyName) == 0) {
            *value = strdup(twin->reported.value ? twin->reported.value : "null");
            *datatype = strdup("string");
            return 0;
        }
    }

    log_warn("Property %s not found for device %s", propertyName, deviceId);
    return -1;
}

// Write data to a device
int dev_panel_write_device(DeviceManager *manager, const char *method, 
                           const char *deviceId, const char *propertyName, const char *data) {
    if (!manager || !deviceId || !propertyName || !data) return -1;
    
    Device *device = device_manager_get(manager, deviceId);
    if (!device) {
        log_warn("Device %s not found", deviceId);
        return -1;
    }
    
    TwinResult result = {0};
    if (devicetwin_set(device, propertyName, data, &result) != 0) {
        log_error("Failed to set twin property %s for device %s", propertyName, deviceId);
        free(result.value);
        free(result.error);
        return -1;
    }
    
    log_info("Successfully wrote property %s=%s to device %s", propertyName, data, deviceId);
    free(result.value);
    free(result.error);
    return 0;
}

// Retrieve the methods of a device
int dev_panel_get_device_method(DeviceManager *manager, const char *deviceId,
                                char ***method_map, int *method_count,
                                char ***property_map, int *property_count) {
    if (!manager || !deviceId || !method_map || !method_count ||
        !property_map || !property_count) return -1;

    Device *device = device_manager_get(manager, deviceId);
    if (!device) {
        log_warn("Device %s not found", deviceId);
        *method_map = NULL; *method_count = 0;
        *property_map = NULL; *property_count = 0;
        return 0;
    }

    *method_count = device->instance.methodsCount;
    if (*method_count > 0) {
        *method_map = calloc((size_t)*method_count, sizeof(char*));
        for (int i = 0; i < *method_count; ++i) {
            DeviceMethod *method = &device->instance.methods[i];
            (*method_map)[i] = strdup(method->name ? method->name : "unknown");
        }
    } else {
        *method_map = NULL;
    }

    int total_props = 0;
    for (int i = 0; i < device->instance.methodsCount; ++i) {
        total_props += device->instance.methods[i].propertyNamesCount;
    }
    *property_count = total_props;
    if (total_props > 0) {
        *property_map = calloc((size_t)total_props, sizeof(char*));
        int k = 0;
        for (int i = 0; i < device->instance.methodsCount; ++i) {
            DeviceMethod *method = &device->instance.methods[i];
            for (int j = 0; j < method->propertyNamesCount; ++j) {
                (*property_map)[k++] = strdup(method->propertyNames[j] ?
                                              method->propertyNames[j] : "unknown");
            }
        }
    } else {
        *property_map = NULL;
    }

    return 0;
}

// Retrieve device information
int dev_panel_get_device(DeviceManager *manager, const char *deviceId, DeviceInstance *instance) {
    if (!manager || !deviceId || !instance) return -1;
    
    Device *device = device_manager_get(manager, deviceId);
    if (!device) {
        log_warn("Device %s not found", deviceId);
        return -1;
    }
    
    *instance = device->instance;
    return 0;
}

// Retrieve the model of a device
int dev_panel_get_model(DeviceManager *manager, const char *modelId, DeviceModel *model) {
    if (!manager || !modelId || !model) return -1;
    
    pthread_mutex_lock(&manager->managerMutex);
    for (int i = 0; i < manager->deviceCount; i++) {
        Device *device = manager->devices[i];
        if (device && device->model.name) {
            char deviceModelId[256];
            snprintf(deviceModelId, sizeof(deviceModelId), "%s/%s", 
                     device->model.namespace_ ? device->model.namespace_ : "default",
                     device->model.name);
            
            if (strcmp(deviceModelId, modelId) == 0) {
                *model = device->model;
                pthread_mutex_unlock(&manager->managerMutex);
                return 0;
            }
        }
    }
    pthread_mutex_unlock(&manager->managerMutex);
    log_warn("Model %s not found", modelId);
    return -1;
}

// Check if a device exists
int dev_panel_has_device(DeviceManager *manager, const char *deviceId) {
    if (!manager || !deviceId) return 0;
    Device *device = device_manager_get(manager, deviceId);
    return device != NULL ? 1 : 0;
}

// Update a device (placeholder implementation)
int dev_panel_update_dev(DeviceManager *manager, const DeviceModel *model, const DeviceInstance *instance) {
    if (!manager || !model || !instance) return -1;
    log_info("Updating device: %s", instance->name ? instance->name : "unknown");
    return 0;
}

// Update a model (placeholder implementation)
int dev_panel_update_model(DeviceManager *manager, const DeviceModel *model) {
    if (!manager || !model) return -1;
    log_info("Updating model: %s", model->name ? model->name : "unknown");
    return 0;
}

// Remove a model (placeholder implementation)
int dev_panel_remove_model(DeviceManager *manager, const char *modelId) {
    if (!manager || !modelId) return -1;
    log_info("Removing model: %s", modelId);
    return 0;
}