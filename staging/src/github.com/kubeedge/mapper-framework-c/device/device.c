#include "device.h"
#include "log/log.h"
#include "driver/driver.h"
#include "common/const.h"
#include "common/json_util.h"
#include "common/string_util.h"
#include "data/publish/publisher.h"
#include "grpcclient/register.h"
#include "device/devicestatus.h"
#include "data/dbmethod/recorder.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <pthread.h>
#include <unistd.h>
#include <time.h>
#include <ctype.h>


static const char *normalize_status(const char *s)
{
    if (!s || !*s)
        return DEVICE_STATUS_OFFLINE;
    if (strcasecmp(s, "OK") == 0 || strcasecmp(s, "ONLINE") == 0)
        return DEVICE_STATUS_OK;
    if (strcasecmp(s, "OFFLINE") == 0 || strcasecmp(s, "DOWN") == 0)
        return DEVICE_STATUS_OFFLINE;
    return s;
}

static int device_update_status_from_driver(Device *device, int force_report, const char **out_new_status)
{
    if (!device || !device->client)
        return 0;
    const char *drv = GetDeviceStates(device->client);
    const char *newst = normalize_status(drv);
    int need_report = force_report || device_status_check_change(device, newst);
    if (need_report)
        device_set_status(device, newst);
    if (out_new_status)
        *out_new_status = newst;
    return need_report;
}

static void *device_data_thread(void *arg)
{
    Device *device = (Device *)arg;
    while (device->dataThreadRunning)
    {
        int need_report_status = 0;
        const char *status_to_report = NULL;
        const char *ns = NULL;
        const char *name = NULL;

        pthread_mutex_lock(&device->mutex);

        need_report_status = device_update_status_from_driver(device, 0, &status_to_report);
        ns = device->instance.namespace_ ? device->instance.namespace_ : "default";
        name = device->instance.name ? device->instance.name : "unknown";

        if (strcmp(device->status, DEVICE_STATUS_OK) != 0)
        {
            pthread_mutex_unlock(&device->mutex);
            if (need_report_status && status_to_report)
            {
                ReportDeviceStates(ns, name, status_to_report);
                ReportTwinKV(ns, name, "status", status_to_report, "string");
            }
            usleep(1000000);
            continue;
        }

        for (int i = 0; i < device->instance.twinsCount; i++)
        {
            Twin *twin = &device->instance.twins[i];
            if (!twin || !twin->propertyName)
                continue;
            if (device->client)
            {
                void *drv_out = NULL;
                VisitorConfig vis = (VisitorConfig){0};
                int off = device_resolve_offset(device, twin->propertyName);
                if (off > 0)
                    vis.offset = off;
                vis.propertyName = twin->propertyName;

                int drv_rc = GetDeviceData(device->client, &vis, &drv_out);
                const char *ns2 = device->instance.namespace_ ? device->instance.namespace_ : "default";
                const char *name2 = device->instance.name ? device->instance.name : "unknown";
                ReportDeviceStates(ns2, name2, device->status);
                ReportTwinKV(ns2, name2, "status", device->status ? device->status : "unknown", "string");

                if (drv_rc == 0 && drv_out)
                {
                    free(twin->reported.value);
                    twin->reported.value = strdup((char *)drv_out);
                    free(drv_out);
                    log_info("device=%s prop=%s reported='%s'",
                             device->instance.name ? device->instance.name : "(nil)",
                             twin->propertyName ? twin->propertyName : "(nil)",
                             twin->reported.value ? twin->reported.value : "(nil)");
                    dbmethod_recorder_record(device,
                                                         twin->propertyName ? twin->propertyName : "unknown",
                                                         twin->reported.value ? twin->reported.value : "(nil)",
                                                         (long long)time(NULL) * 1000);
                    
                    publisher_publish_from_device(device,
                                                              twin->propertyName ? twin->propertyName : "unknown",
                                                              twin->reported.value ? twin->reported.value : "(nil)",
                                                              (long long)time(NULL) * 1000);
                    
                    ReportTwinKV(ns, name,
                                 twin->propertyName ? twin->propertyName : "unknown",
                                 twin->reported.value ? twin->reported.value : "(nil)",
                                 "string");
                }
            }
            device_deal_twin(device, twin);
        }

        pthread_mutex_unlock(&device->mutex);

        usleep(1000000);
    }

    return NULL;
}

Device *device_new(const DeviceInstance *instance, const DeviceModel *model)
{
    Device *d = calloc(1, sizeof(Device));
    if (!d)
        return NULL;
    memset(&d->instance, 0, sizeof(DeviceInstance));

    if (instance->id)
        d->instance.id = strdup(instance->id);
    if (instance->name)
        d->instance.name = strdup(instance->name);
    if (instance->namespace_)
        d->instance.namespace_ = strdup(instance->namespace_);
    if (!d->instance.namespace_ || !*d->instance.namespace_)
    {
        if (d->instance.namespace_)
            free(d->instance.namespace_);
        d->instance.namespace_ = strdup("default");
    }
    if (instance->model)
        d->instance.model = strdup(instance->model);
    if (instance->protocolName)
        d->instance.protocolName = strdup(instance->protocolName);

    if (instance->pProtocol.protocolName)
    {
        d->instance.pProtocol.protocolName = strdup(instance->pProtocol.protocolName);
    }
    if (instance->pProtocol.configData)
    {
        d->instance.pProtocol.configData = strdup(instance->pProtocol.configData);
    }

    if (instance->twins && instance->twinsCount > 0)
    {
        d->instance.twinsCount = instance->twinsCount;
        d->instance.twins = calloc(instance->twinsCount, sizeof(Twin));

        for (int i = 0; i < instance->twinsCount; i++)
        {
            Twin *srcTwin = &instance->twins[i];
            Twin *dstTwin = &d->instance.twins[i];

            if (srcTwin->propertyName)
            {
                dstTwin->propertyName = strdup(srcTwin->propertyName);
            }
            if (srcTwin->observedDesired.value)
            {
                dstTwin->observedDesired.value = strdup(srcTwin->observedDesired.value);
            }
            if (srcTwin->observedDesired.metadata.timestamp)
            {
                dstTwin->observedDesired.metadata.timestamp = strdup(srcTwin->observedDesired.metadata.timestamp);
            }
            if (srcTwin->observedDesired.metadata.type)
            {
                dstTwin->observedDesired.metadata.type = strdup(srcTwin->observedDesired.metadata.type);
            }
            if (srcTwin->reported.value)
            {
                dstTwin->reported.value = strdup(srcTwin->reported.value);
            }
            if (srcTwin->reported.metadata.timestamp)
            {
                dstTwin->reported.metadata.timestamp = strdup(srcTwin->reported.metadata.timestamp);
            }
            if (srcTwin->reported.metadata.type)
            {
                dstTwin->reported.metadata.type = strdup(srcTwin->reported.metadata.type);
            }

            dstTwin->property = NULL;
        }
    }

    if (instance->properties && instance->propertiesCount > 0)
    {
        d->instance.propertiesCount = instance->propertiesCount;
        d->instance.properties = calloc(instance->propertiesCount, sizeof(DeviceProperty));

        for (int i = 0; i < instance->propertiesCount; i++)
        {
            DeviceProperty *srcProp = &instance->properties[i];
            DeviceProperty *dstProp = &d->instance.properties[i];

            if (srcProp->name)
                dstProp->name = strdup(srcProp->name);
            if (srcProp->propertyName)
                dstProp->propertyName = strdup(srcProp->propertyName);
            if (srcProp->modelName)
                dstProp->modelName = strdup(srcProp->modelName);
            if (srcProp->protocol)
                dstProp->protocol = strdup(srcProp->protocol);
            if (srcProp->pushMethod)
            {
                dstProp->pushMethod = calloc(1, sizeof(PushMethodConfig));
                if (srcProp->pushMethod->methodName)
                    dstProp->pushMethod->methodName = strdup(srcProp->pushMethod->methodName);
                if (srcProp->pushMethod->methodConfig)
                    dstProp->pushMethod->methodConfig = strdup(srcProp->pushMethod->methodConfig);
                if (srcProp->pushMethod->dbMethod)
                {
                    dstProp->pushMethod->dbMethod = calloc(1, sizeof(DBMethodConfig));
                    if (srcProp->pushMethod->dbMethod->dbMethodName)
                        dstProp->pushMethod->dbMethod->dbMethodName = strdup(srcProp->pushMethod->dbMethod->dbMethodName);
                    if (srcProp->pushMethod->dbMethod->dbConfig)
                    {
                        dstProp->pushMethod->dbMethod->dbConfig = calloc(1, sizeof(DBConfig));
                        if (srcProp->pushMethod->dbMethod->dbConfig->mysqlClientConfig)
                            dstProp->pushMethod->dbMethod->dbConfig->mysqlClientConfig = strdup(srcProp->pushMethod->dbMethod->dbConfig->mysqlClientConfig);
                        if (srcProp->pushMethod->dbMethod->dbConfig->redisClientConfig)
                            dstProp->pushMethod->dbMethod->dbConfig->redisClientConfig = strdup(srcProp->pushMethod->dbMethod->dbConfig->redisClientConfig);
                        if (srcProp->pushMethod->dbMethod->dbConfig->influxdb2ClientConfig)
                            dstProp->pushMethod->dbMethod->dbConfig->influxdb2ClientConfig = strdup(srcProp->pushMethod->dbMethod->dbConfig->influxdb2ClientConfig);
                        if (srcProp->pushMethod->dbMethod->dbConfig->tdengineClientConfig)
                            dstProp->pushMethod->dbMethod->dbConfig->tdengineClientConfig = strdup(srcProp->pushMethod->dbMethod->dbConfig->tdengineClientConfig);
                    }
                }
            }

            dstProp->collectCycle = srcProp->collectCycle;
            dstProp->reportCycle = srcProp->reportCycle;
            dstProp->reportToCloud = srcProp->reportToCloud;
        }

        for (int t = 0; t < d->instance.twinsCount; ++t)
        {
            Twin *tw = &d->instance.twins[t];
            if (!tw->propertyName)
                continue;
            for (int j = 0; j < d->instance.propertiesCount; ++j)
            {
                DeviceProperty *p = &d->instance.properties[j];
                const char *pname = p->name ? p->name : p->propertyName;
                if (pname && strcmp(pname, tw->propertyName) == 0)
                {
                    tw->property = p;
                    break;
                }
            }
        }
    }

    if (instance->methods && instance->methodsCount > 0)
    {
        d->instance.methodsCount = instance->methodsCount;
        d->instance.methods = calloc(instance->methodsCount, sizeof(DeviceMethod));

        for (int i = 0; i < instance->methodsCount; i++)
        {
            DeviceMethod *srcMethod = &instance->methods[i];
            DeviceMethod *dstMethod = &d->instance.methods[i];

            if (srcMethod->name)
                dstMethod->name = strdup(srcMethod->name);
            if (srcMethod->description)
                dstMethod->description = strdup(srcMethod->description);

            if (srcMethod->propertyNames && srcMethod->propertyNamesCount > 0)
            {
                dstMethod->propertyNamesCount = srcMethod->propertyNamesCount;
                dstMethod->propertyNames = calloc(srcMethod->propertyNamesCount, sizeof(char *));

                for (int j = 0; j < srcMethod->propertyNamesCount; j++)
                {
                    if (srcMethod->propertyNames[j])
                    {
                        dstMethod->propertyNames[j] = strdup(srcMethod->propertyNames[j]);
                    }
                }
            }
        }
    }

    memset(&d->model, 0, sizeof(DeviceModel));
    if (model->id)
        d->model.id = strdup(model->id);
    if (model->name)
        d->model.name = strdup(model->name);
    if (model->namespace_)
        d->model.namespace_ = strdup(model->namespace_);
    if (!d->model.namespace_ || !*d->model.namespace_)
    {
        if (d->model.namespace_)
            free(d->model.namespace_);
        d->model.namespace_ = strdup("default");
    }
    if (model->description)
        d->model.description = strdup(model->description);

    if (model->properties && model->propertiesCount > 0)
    {
        d->model.propertiesCount = model->propertiesCount;
        d->model.properties = calloc(model->propertiesCount, sizeof(ModelProperty));

        for (int i = 0; i < model->propertiesCount; i++)
        {
            ModelProperty *srcProp = &model->properties[i];
            ModelProperty *dstProp = &d->model.properties[i];

            if (srcProp->name)
                dstProp->name = strdup(srcProp->name);
            if (srcProp->dataType)
                dstProp->dataType = strdup(srcProp->dataType);
            if (srcProp->description)
                dstProp->description = strdup(srcProp->description);
            if (srcProp->accessMode)
                dstProp->accessMode = strdup(srcProp->accessMode);
            if (srcProp->minimum)
                dstProp->minimum = strdup(srcProp->minimum);
            if (srcProp->maximum)
                dstProp->maximum = strdup(srcProp->maximum);
            if (srcProp->unit)
                dstProp->unit = strdup(srcProp->unit);
        }
    }

    d->status = strdup(DEVICE_STATUS_UNKNOWN);
    d->stopChan = 0;
    d->dataThreadRunning = 0;
    d->removing = 0;
    pthread_mutex_init(&d->mutex, NULL);

    if (d->instance.pProtocol.protocolName)
    {
        d->client = NewClient(&d->instance.pProtocol);
        if (!d->client)
        {
            log_error("Failed to create device client");
            device_free(d);
            return NULL;
        }
    }

    if (d->instance.twinsCount == 0 && d->instance.propertiesCount > 0)
    {
        d->instance.twinsCount = d->instance.propertiesCount;
        d->instance.twins = calloc(d->instance.twinsCount, sizeof(Twin));
        for (int i = 0; i < d->instance.twinsCount; ++i)
        {
            DeviceProperty *p = &d->instance.properties[i];
            Twin *tw = &d->instance.twins[i];
            tw->propertyName = p->name ? strdup(p->name) : strdup("unknown");
            tw->property = p;
            tw->observedDesired.value = NULL;
            tw->reported.value = NULL;
        }
    }

    if (d->instance.methodsCount == 0 && d->instance.propertiesCount > 0)
    {
        d->instance.methodsCount = 1;
        d->instance.methods = calloc(1, sizeof(DeviceMethod));
        DeviceMethod *m = &d->instance.methods[0];
        m->name = strdup("SetProperty");
        m->propertyNamesCount = d->instance.propertiesCount;
        m->propertyNames = calloc(m->propertyNamesCount, sizeof(char *));
        for (int i = 0; i < m->propertyNamesCount; ++i)
        {
            m->propertyNames[i] = d->instance.properties[i].name
                                      ? strdup(d->instance.properties[i].name)
                                      : strdup("unknown");
        }
    }

    return d;
}

void device_free(Device *device)
{
    if (!device)
        return;

    if (device->dataThreadRunning || device->dataThread)
    {
        device_stop(device);
    }

    if (device->client)
    {
        FreeClient(device->client);
    }
    free(device->instance.name);
    free(device->instance.namespace_);
    free(device->instance.model);
    free(device->instance.protocolName);
    free(device->instance.pProtocol.protocolName);
    free(device->instance.pProtocol.configData);

    if (device->instance.twins)
    {
        for (int i = 0; i < device->instance.twinsCount; i++)
        {
            Twin *twin = &device->instance.twins[i];
            free(twin->propertyName);
            free(twin->observedDesired.value);
            free(twin->observedDesired.metadata.timestamp);
            free(twin->observedDesired.metadata.type);
            free(twin->reported.value);
            free(twin->reported.metadata.timestamp);
            free(twin->reported.metadata.type);

            if (twin->property)
            {

                int embedded = 0;
                if (device->instance.properties &&
                    twin->property >= device->instance.properties &&
                    twin->property < device->instance.properties + device->instance.propertiesCount)
                {
                    embedded = 1;
                }
                if (!embedded)
                {
                    free(twin->property->name);
                    free(twin->property);
                }
            }
        }
        free(device->instance.twins);
    }

    if (device->instance.properties)
    {
        for (int i = 0; i < device->instance.propertiesCount; i++)
        {
            DeviceProperty *prop = &device->instance.properties[i];
            free(prop->name);
            free(prop->propertyName);
            free(prop->modelName);
            free(prop->protocol);
        }
        free(device->instance.properties);
    }

    if (device->instance.methods)
    {
        for (int i = 0; i < device->instance.methodsCount; i++)
        {
            DeviceMethod *method = &device->instance.methods[i];
            free(method->name);
            free(method->description);
            if (method->propertyNames)
            {
                for (int j = 0; j < method->propertyNamesCount; j++)
                {
                    free(method->propertyNames[j]);
                }
                free(method->propertyNames);
            }
        }
        free(device->instance.methods);
    }

    free(device->model.name);
    free(device->model.namespace_);
    free(device->model.description);
    if (device->model.properties)
    {
        for (int i = 0; i < device->model.propertiesCount; i++)
        {
            ModelProperty *prop = &device->model.properties[i];
            free(prop->name);
            free(prop->dataType);
            free(prop->description);
            free(prop->accessMode);
            free(prop->minimum);
            free(prop->maximum);
            free(prop->unit);
        }
        free(device->model.properties);
    }

    free(device->status);
    pthread_mutex_destroy(&device->mutex);
    free(device);
}

static void device_runtime_rebuild(Device *device)
{
    if (!device)
        return;
    int rebuild = 0;
    if (device->instance.propertiesCount > 0 &&
        (device->instance.twinsCount == 0 || !device->instance.twins))
    {
        rebuild = 1;
    }
    else
    {

        for (int i = 0; i < device->instance.twinsCount; ++i)
        {
            if (device->instance.twins[i].property == NULL)
            {
                rebuild = 1;
                break;
            }
        }
    }
    if (rebuild)
    {
        free(device->instance.twins);
        device->instance.twinsCount = device->instance.propertiesCount;
        device->instance.twins = calloc(device->instance.twinsCount, sizeof(Twin));
        for (int i = 0; i < device->instance.twinsCount; ++i)
        {
            DeviceProperty *p = &device->instance.properties[i];
            Twin *tw = &device->instance.twins[i];
            tw->propertyName = p->name ? strdup(p->name) : strdup("unknown");
            tw->property = p;
        }
    }
    if (device->instance.methodsCount == 0 && device->instance.propertiesCount > 0)
    {
        device->instance.methodsCount = 1;
        device->instance.methods = calloc(1, sizeof(DeviceMethod));
        DeviceMethod *m = &device->instance.methods[0];
        m->name = strdup("SetProperty");
        m->propertyNamesCount = device->instance.propertiesCount;
        m->propertyNames = calloc(m->propertyNamesCount, sizeof(char *));
        for (int i = 0; i < m->propertyNamesCount; ++i)
        {
            m->propertyNames[i] = device->instance.properties[i].name ? strdup(device->instance.properties[i].name) : strdup("unknown");
        }
    }
}

int device_start(Device *device)
{
    if (!device)
        return -1;
    pthread_mutex_lock(&device->mutex);
    device_runtime_rebuild(device);
    if (device->dataThreadRunning)
    {
        pthread_mutex_unlock(&device->mutex);
        return 0;
    }
    if (device->client)
    {
        if (InitDevice(device->client) != 0)
        {
            log_error("device_start: InitDevice failed for device %s", device->instance.name);
            device_set_status(device, DEVICE_STATUS_OFFLINE);
            ReportDeviceStates(device->instance.namespace_, device->instance.name, DEVICE_STATUS_OFFLINE);
            ReportTwinKV(device->instance.namespace_ ? device->instance.namespace_ : "default",
                         device->instance.name ? device->instance.name : "unknown",
                         "status", DEVICE_STATUS_OFFLINE, "string");
            pthread_mutex_unlock(&device->mutex);
            return -1;
        }
    }
    else
    {
        log_warn("device_start: no client to Init for device %s", device->instance.name);
    }

    {
        const char *init_st = NULL;
        int need = device_update_status_from_driver(device, 1, &init_st);
        const char *ns = device->instance.namespace_ ? device->instance.namespace_ : "default";
        const char *name = device->instance.name ? device->instance.name : "unknown";
        if (need && init_st)
        {
            ReportDeviceStates(ns, name, init_st);
            ReportTwinKV(ns, name, "status", init_st, "string");
        }
    }

    device->dataThreadRunning = 1;
    if (pthread_create(&device->dataThread, NULL, device_data_thread, device) != 0)
    {
        log_error("Failed to create data thread for device %s", device->instance.name);
        device->dataThreadRunning = 0;
        device_set_status(device, DEVICE_STATUS_OFFLINE);
        ReportDeviceStates(device->instance.namespace_, device->instance.name, DEVICE_STATUS_OFFLINE);
        ReportTwinKV(device->instance.namespace_ ? device->instance.namespace_ : "default",
                     device->instance.name ? device->instance.name : "unknown",
                     "status", DEVICE_STATUS_OFFLINE, "string");
        pthread_mutex_unlock(&device->mutex);
        return -1;
    }
    pthread_mutex_unlock(&device->mutex);
    return 0;
}

int device_stop(Device *device)
{
    if (!device)
        return -1;
    pthread_mutex_lock(&device->mutex);
    device->stopChan = 1;
    device->dataThreadRunning = 0;

    if (device->client)
    {
        StopDevice(device->client);
    }

    device_set_status(device, DEVICE_STATUS_OFFLINE);
    ReportDeviceStatus(device->instance.namespace_ ? device->instance.namespace_ : "default",
                       device->instance.name ? device->instance.name : "unknown",
                       DEVICE_STATUS_OFFLINE);

    pthread_mutex_unlock(&device->mutex);

    if (device->dataThread)
    {
        for (int i = 0; i < 10; ++i)
        {
            if (!device->dataThreadRunning)
                break;
            usleep(50000);
        }
        pthread_cancel(device->dataThread);
        pthread_join(device->dataThread, NULL);
        device->dataThread = 0;
    }
    return 0;
}

int device_deal_twin(Device *device, const Twin *twin_in)
{
    if (!device->client)
    {
        log_warn("device_deal_twin: no client to write for device=%s", device->instance.name);
        return -1;
    }

    const char *desired = twin_in->observedDesired.value;
    const char *reported = twin_in->reported.value;

    if (!desired || !*desired)
    {
        log_debug("Twin %s no desired, skip", twin_in->propertyName);
        return 0;
    }
    if (reported && strcmp(reported, desired) == 0)
    {
        log_debug("Twin %s desired == reported (%s), skip", twin_in->propertyName, desired);
        return 0;
    }

    VisitorConfig vis = (VisitorConfig){0};
    vis.propertyName = twin_in->propertyName;
    int offset = device_resolve_offset(device, twin_in->propertyName);
    if (offset > 0)
        vis.offset = offset;
    int rc = SetDeviceData(device->client, desired, &vis);
    if (rc != 0)
    {
        log_warn("SetDeviceData failed for device=%s prop=%s rc=%d", device->instance.name, twin_in->propertyName, rc);
        return -1;
    }

    char *new_reported = NULL;
    void *drv_out = NULL;
    int get_rc = GetDeviceData(device->client, &vis, &drv_out);
    if (get_rc == 0 && drv_out)
    {
        new_reported = strdup((char *)drv_out);
        free(drv_out);
    }
    else
    {
        new_reported = strdup(desired);
    }

    free(((Twin *)twin_in)->reported.value);
    ((Twin *)twin_in)->reported.value = new_reported;

    return 0;
}

DeviceManager *device_manager_new(void)
{
    DeviceManager *manager = calloc(1, sizeof(DeviceManager));
    if (!manager)
        return NULL;

    manager->capacity = 10;
    manager->devices = calloc(manager->capacity, sizeof(Device *));
    if (!manager->devices)
    {
        free(manager);
        return NULL;
    }

    if (pthread_mutex_init(&manager->managerMutex, NULL) != 0)
    {
        free(manager->devices);
        free(manager);
        return NULL;
    }

    manager->stopped = 0;
    return manager;
}

void device_manager_free(DeviceManager *manager)
{
    if (!manager)
        return;
    if (!manager->stopped)
    {
        device_manager_stop_all(manager);
    }
    pthread_mutex_lock(&manager->managerMutex);
    for (int i = 0; i < manager->deviceCount; i++)
    {
        device_free(manager->devices[i]);
    }
    free(manager->devices);
    pthread_mutex_unlock(&manager->managerMutex);
    pthread_mutex_destroy(&manager->managerMutex);
    free(manager);
}

int device_manager_add(DeviceManager *manager, Device *device)
{
    if (!manager || !device)
        return -1;

    pthread_mutex_lock(&manager->managerMutex);

    if (manager->deviceCount >= manager->capacity)
    {
        manager->capacity *= 2;
        Device **newDevices = realloc(manager->devices,
                                      manager->capacity * sizeof(Device *));
        if (!newDevices)
        {
            pthread_mutex_unlock(&manager->managerMutex);
            return -1;
        }
        manager->devices = newDevices;
    }

    manager->devices[manager->deviceCount++] = device;

    pthread_mutex_unlock(&manager->managerMutex);
    return 0;
}

Device *device_manager_get(DeviceManager *manager, const char *deviceId)
{
    if (!manager || !deviceId)
        return NULL;

    pthread_mutex_lock(&manager->managerMutex);

    for (int i = 0; i < manager->deviceCount; i++)
    {
        if (manager->devices[i] && manager->devices[i]->instance.name &&
            strcmp(manager->devices[i]->instance.name, deviceId) == 0)
        {
            Device *device = manager->devices[i];
            pthread_mutex_unlock(&manager->managerMutex);
            return device;
        }
    }

    const char *sep = strrchr(deviceId, '.');
    if (!sep)
        sep = strrchr(deviceId, '/');
    if (sep && *(sep + 1))
    {
        const char *shortId = sep + 1;
        for (int i = 0; i < manager->deviceCount; i++)
        {
            if (manager->devices[i] && manager->devices[i]->instance.name &&
                strcmp(manager->devices[i]->instance.name, shortId) == 0)
            {
                Device *device = manager->devices[i];
                pthread_mutex_unlock(&manager->managerMutex);
                return device;
            }
        }
    }

    pthread_mutex_unlock(&manager->managerMutex);
    return NULL;
}

int device_manager_start_all(DeviceManager *manager)
{
    if (!manager)
        return -1;

    pthread_mutex_lock(&manager->managerMutex);

    int success = 0;
    for (int i = 0; i < manager->deviceCount; i++)
    {
        if (device_start(manager->devices[i]) == 0)
        {
            success++;
        }
    }

    pthread_mutex_unlock(&manager->managerMutex);

    log_info("Started %d/%d devices", success, manager->deviceCount);
    return success == manager->deviceCount ? 0 : -1;
}

int device_manager_stop_all(DeviceManager *manager)
{
    if (!manager)
        return -1;
    if (manager->stopped)
    {
        log_debug("device_manager_stop_all: already stopped");
        return 0;
    }
    pthread_mutex_lock(&manager->managerMutex);
    for (int i = 0; i < manager->deviceCount; i++)
    {
        device_stop(manager->devices[i]);
    }
    pthread_mutex_unlock(&manager->managerMutex);
    manager->stopped = 1;
    log_info("Stopped all devices");
    return 0;
}

int device_resolve_offset(Device *device, const char *propName)
{
    if (!propName)
        return -1;

    if (device && device->instance.pProtocol.configData)
    {
        int v = -1;
        if (json_get_int(device->instance.pProtocol.configData, propName, &v) == 0)
        {
            if (v > 0)
            {
                return v;
            }
        }
        const char *cfg = NULL;
        cfg = strcasestr(device->instance.pProtocol.configData, "\"configData\"");
        if (!cfg)
            cfg = strcasestr(device->instance.pProtocol.configData, "configData");
        if (cfg)
        {
            const char *p = strchr(cfg, ':');
            if (p)
            {
                p++;
                while (*p && *p != '{')
                    p++;
                if (*p == '{')
                {
                    if (json_get_int(p, propName, &v) == 0 && v > 0)
                    {
                        return v;
                    }
                }
            }
        }
    }

    if (device && device->instance.properties && device->instance.propertiesCount > 0)
    {
        for (int i = 0; i < device->instance.propertiesCount; ++i)
        {
            DeviceProperty *p = &device->instance.properties[i];
            if (p && p->name && strcmp(p->name, propName) == 0)
            {
                int base_offset = 1;
                int resolved = base_offset + i;
                return resolved;
            }
        }
    }
    return -1;
}
