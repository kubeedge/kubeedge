#include "device/dev_panel.h"
#include "device/device.h"
#include "device/devicetwin.h"
#include "log/log.h"
#include <pthread.h>
#include <string.h>
#include <stdlib.h>
#include <stdio.h>

static DeviceManager *g_panel_mgr = NULL;
static pthread_t g_panel_start_thread = 0;

int panel_init(void)
{
    if (g_panel_mgr)
        return 0;
    g_panel_mgr = device_manager_new();
    if (!g_panel_mgr)
    {
        log_error("panel_init: device_manager_new failed");
        return -1;
    }
    return 0;
}

void panel_free(void)
{
    if (!g_panel_mgr)
        return;
    device_manager_stop_all(g_panel_mgr);
    if (g_panel_start_thread)
    {
        pthread_cancel(g_panel_start_thread);
        pthread_join(g_panel_start_thread, NULL);
        g_panel_start_thread = 0;
    }
    device_manager_free(g_panel_mgr);
    g_panel_mgr = NULL;
}

DeviceManager *panel_get_manager(void)
{
    return g_panel_mgr;
}

static DeviceModel *find_model_in_list(const DeviceModel *models, int modelCount, const char *name, const char *ns)
{
    if (!models || modelCount <= 0 || !name)
        return NULL;
    for (int i = 0; i < modelCount; i++)
    {
        const DeviceModel *m = &models[i];
        if (!m->name)
            continue;
        const char *mns = m->namespace_ ? m->namespace_ : "default";
        const char *qns = ns ? ns : "default";
        if (strcmp(m->name, name) == 0 && strcmp(mns, qns) == 0)
        {
            return (DeviceModel *)m;
        }
    }
    return NULL;
}

int panel_dev_init(DeviceInstance *deviceList, int deviceCount, DeviceModel *modelList, int modelCount)
{
    if (!g_panel_mgr)
    {
        if (panel_init() != 0)
            return -1;
    }
    for (int mi = 0; mi < modelCount; ++mi)
    {
        DeviceModel *m = &modelList[mi];
        if (!m)
            continue;
        if (dev_panel_update_model(g_panel_mgr, m) != 0)
        {
            log_warn("panel_dev_init: dev_panel_update_model failed for model %s", m->name ? m->name : "(nil)");
        }
    }

    for (int di = 0; di < deviceCount; ++di)
    {
        DeviceInstance *inst = &deviceList[di];
        if (!inst)
            continue;

        DeviceModel *m = NULL;
        if (inst->model)
        {
            m = find_model_in_list(modelList, modelCount, inst->model, inst->namespace_);
            if (!m)
            {
                log_warn("panel_dev_init: strict model lookup failed for device %s model=%s ns=%s, trying name-only match",
                         inst->name ? inst->name : "(nil)",
                         inst->model ? inst->model : "(nil)",
                         inst->namespace_ ? inst->namespace_ : "(nil)");
                for (int mi = 0; mi < modelCount && !m; ++mi)
                {
                    DeviceModel *cm = &modelList[mi];
                    if (cm && cm->name && strcmp(cm->name, inst->model) == 0)
                    {
                        m = cm;
                        log_info("panel_dev_init: name-only matched model ptr=%p ns='%s' name='%s'",
                                 (void *)m, m->namespace_ ? m->namespace_ : "(nil)", m->name ? m->name : "(nil)");
                        break;
                    }
                }
            }
        }
        if (!m)
        {
            log_warn("panel_dev_init: model not found for device %s (model=%s), device skipped",
                     inst->name ? inst->name : "(nil)", inst->model ? inst->model : "(nil)");
            continue;
        }

        if (dev_panel_update_dev(g_panel_mgr, m, inst) != 0)
        {
            log_error("panel_dev_init: failed to add device %s", inst->name ? inst->name : "(nil)");
        }
    }

    return 0;
}

static void *panel_start_thread(void *arg)
{
    DeviceManager *mgr = (DeviceManager *)arg;
    device_manager_start_all(mgr);
    return NULL;
}

int panel_dev_start(void)
{
    if (!g_panel_mgr)
    {
        log_error("panel_dev_start: panel not initialized");
        return -1;
    }
    if (g_panel_start_thread)
        return 0;
    if (pthread_create(&g_panel_start_thread, NULL, panel_start_thread, g_panel_mgr) != 0)
    {
        log_error("panel_dev_start: pthread_create failed");
        return -1;
    }
    return 0;
}

int panel_dev_stop(void)
{
    if (!g_panel_mgr)
        return -1;
    device_manager_stop_all(g_panel_mgr);
    return 0;
}

int dev_panel_get_twin_result(DeviceManager *manager, const char *deviceId,
                              const char *propertyName, char **value, char **datatype)
{
    if (!manager || !deviceId || !propertyName || !value || !datatype)
        return -1;
    Device *device = device_manager_get(manager, deviceId);
    if (!device)
    {
        log_warn("Device %s not found", deviceId);
        return -1;
    }

    for (int i = 0; i < device->instance.twinsCount; i++)
    {
        Twin *twin = &device->instance.twins[i];
        if (twin->propertyName && strcmp(twin->propertyName, propertyName) == 0)
        {
            *value = strdup(twin->reported.value ? twin->reported.value : "null");
            *datatype = strdup("string");
            return 0;
        }
    }

    log_warn("Property %s not found for device %s", propertyName, deviceId);
    return -1;
}

int dev_panel_write_device(DeviceManager *manager, const char *method,
                           const char *deviceId, const char *propertyName, const char *data)
{
    if (!manager || !deviceId || !propertyName || !data)
        return -1;

    Device *device = device_manager_get(manager, deviceId);
    if (!device)
    {
        log_warn("Device %s not found", deviceId);
        return -1;
    }

    TwinResult result = {0};
    if (devicetwin_set(device, propertyName, data, &result) != 0)
    {
        log_error("Failed to set twin property %s for device %s", propertyName, deviceId);
        free(result.value);
        free(result.error);
        return -1;
    }
    free(result.value);
    free(result.error);
    return 0;
}

int dev_panel_get_device_method(DeviceManager *manager, const char *deviceId,
                                char ***method_map, int *method_count,
                                char ***property_map, int *property_count)
{
    if (!manager || !deviceId || !method_map || !method_count || !property_map || !property_count)
        return -1;

    Device *device = device_manager_get(manager, deviceId);
    if (!device)
    {
        log_warn("Device %s not found", deviceId);
        *method_map = NULL;
        *method_count = 0;
        *property_map = NULL;
        *property_count = 0;
        return 0;
    }

    *method_count = device->instance.methodsCount;
    if (*method_count > 0)
    {
        *method_map = calloc((size_t)*method_count, sizeof(char *));
        for (int i = 0; i < *method_count; ++i)
        {
            DeviceMethod *method = &device->instance.methods[i];
            (*method_map)[i] = strdup(method->name ? method->name : "unknown");
        }
    }
    else
    {
        *method_map = NULL;
    }

    int total_props = 0;
    for (int i = 0; i < device->instance.methodsCount; ++i)
    {
        total_props += device->instance.methods[i].propertyNamesCount;
    }
    *property_count = total_props;
    if (total_props > 0)
    {
        *property_map = calloc((size_t)total_props, sizeof(char *));
        int k = 0;
        for (int i = 0; i < device->instance.methodsCount; ++i)
        {
            DeviceMethod *method = &device->instance.methods[i];
            for (int j = 0; j < method->propertyNamesCount; ++j)
            {
                (*property_map)[k++] = strdup(method->propertyNames[j] ? method->propertyNames[j] : "unknown");
            }
        }
    }
    else
    {
        *property_map = NULL;
    }

    return 0;
}

int dev_panel_get_device(DeviceManager *manager, const char *deviceId, DeviceInstance *instance)
{
    if (!manager || !deviceId || !instance)
        return -1;

    Device *device = device_manager_get(manager, deviceId);
    if (!device)
    {
        log_warn("Device %s not found", deviceId);
        return -1;
    }

    *instance = device->instance;
    return 0;
}

int dev_panel_get_model(DeviceManager *manager, const char *modelId, DeviceModel *model)
{
    if (!manager || !modelId || !model)
        return -1;

    pthread_mutex_lock(&manager->managerMutex);
    for (int i = 0; i < manager->deviceCount; i++)
    {
        Device *device = manager->devices[i];
        if (device && device->model.name)
        {
            char deviceModelId[256];
            snprintf(deviceModelId, sizeof(deviceModelId), "%s/%s",
                     device->model.namespace_ ? device->model.namespace_ : "default",
                     device->model.name);

            if (strcmp(deviceModelId, modelId) == 0)
            {
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

int dev_panel_has_device(DeviceManager *manager, const char *deviceId)
{
    if (!manager || !deviceId)
        return 0;
    Device *device = device_manager_get(manager, deviceId);
    return device != NULL ? 1 : 0;
}

int dev_panel_update_dev(DeviceManager *manager, const DeviceModel *model, const DeviceInstance *instance)
{
    if (!manager || !model || !instance)
    {
        log_error("dev_panel_update_dev: invalid args");
        return -1;
    }

    const char *deviceId = instance->id ? instance->id : instance->name;
    if (!deviceId)
    {
        log_error("dev_panel_update_dev: instance has no id/name");
        return -1;
    }

    Device *old = device_manager_get(manager, deviceId);
    if (old)
    {
        device_stop(old);
        device_manager_remove(manager, old->instance.name ? old->instance.name : deviceId);
    }

    Device *dev = device_new(instance, model);
    if (!dev)
    {
        log_error("dev_panel_update_dev: device_new failed for %s", deviceId);
        return -1;
    }

    if (device_manager_add(manager, dev) != 0)
    {
        log_error("dev_panel_update_dev: device_manager_add failed for %s", deviceId);
        device_free(dev);
        return -1;
    }

    if (device_start(dev) != 0)
    {
        log_error("dev_panel_update_dev: device_start failed for %s", deviceId);
        return -1;
    }

    return 0;
}

int dev_panel_update_model(DeviceManager *manager, const DeviceModel *model)
{
    if (!manager || !model)
        return -1;
    return 0;
}

int dev_panel_remove_model(DeviceManager *manager, const char *modelId)
{
    if (!manager || !modelId)
        return -1;
    return 0;
}