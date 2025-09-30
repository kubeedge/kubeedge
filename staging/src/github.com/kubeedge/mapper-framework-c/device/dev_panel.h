#ifndef DEVICE_DEV_PANEL_H
#define DEVICE_DEV_PANEL_H

#include "common/configmaptype.h"
#include "device/device.h"

#ifdef __cplusplus
extern "C"
{
#endif

    int panel_init(void);
    void panel_free(void);
    DeviceManager *panel_get_manager(void);

    int panel_dev_init(DeviceInstance *deviceList, int deviceCount, DeviceModel *modelList, int modelCount);

    int panel_dev_start(void);
    int panel_dev_stop(void);

    int dev_panel_get_twin_result(DeviceManager *manager, const char *deviceId,
                                  const char *propertyName, char **value, char **datatype);

    int dev_panel_write_device(DeviceManager *manager, const char *method,
                               const char *deviceId, const char *propertyName, const char *data);

    int dev_panel_get_device_method(DeviceManager *manager, const char *deviceId,
                                    char ***method_map, int *method_count,
                                    char ***property_map, int *property_count);

    int dev_panel_get_device(DeviceManager *manager, const char *deviceId, DeviceInstance *instance);

    int dev_panel_get_model(DeviceManager *manager, const char *modelId, DeviceModel *model);

    int dev_panel_has_device(DeviceManager *manager, const char *deviceId);

    int dev_panel_update_dev(DeviceManager *manager, const DeviceModel *model, const DeviceInstance *instance);

    int dev_panel_update_model(DeviceManager *manager, const DeviceModel *model);

    int dev_panel_remove_model(DeviceManager *manager, const char *modelId);

#ifdef __cplusplus
}
#endif

#endif // DEVICE_DEVPANEL_H