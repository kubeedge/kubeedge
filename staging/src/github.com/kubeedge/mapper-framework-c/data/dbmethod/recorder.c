#include "data/dbmethod/recorder.h"
#include "common/configmaptype.h"
#include "data/dbmethod/mysql/recorder.h"
#include "data/dbmethod/redis/recorder.h"
#include "data/dbmethod/influxdb2/recorder.h"
#include "data/dbmethod/tdengine/recorder.h"
#include <string.h>
#include <stdio.h>
#include "log/log.h"
#include "common/datamodel.h"
#include "data/publish/publisher.h"
extern Publisher *g_publisher;

static DeviceProperty *find_property(Device *device, const char *propName)
{
    if (!device || !propName)
        return NULL;
    for (int i = 0; i < device->instance.propertiesCount; ++i)
    {
        DeviceProperty *p = &device->instance.properties[i];
        if (p->propertyName && strcmp(p->propertyName, propName) == 0)
            return p;
    }
    return NULL;
}

int dbmethod_recorder_record(Device *device, const char *propertyName, const char *value, long long timestamp)
{
    const char *ns = device && device->instance.namespace_ ? device->instance.namespace_ : "default";
    const char *dev = device && device->instance.name ? device->instance.name : "unknown";
    const char *prop = propertyName ? propertyName : "unknown";
    if (!device || !prop || !value) return -1;
    int rc = -1;
    DeviceProperty *p = find_property(device, propertyName);
    
    if (p && p->pushMethod) {
        if (p->pushMethod->dbMethod && p->pushMethod->dbMethod->dbConfig) {
            if (p->pushMethod->dbMethod->dbConfig->mysqlClientConfig) {
                rc = mysql_recorder_record(ns, dev, prop, value, timestamp);
            } else if (p->pushMethod->dbMethod->dbConfig->redisClientConfig) {
                rc = redis_recorder_record(ns, dev, prop, value, timestamp);
            } else if (p->pushMethod->dbMethod->dbConfig->influxdb2ClientConfig) {
                rc = influxdb2_recorder_record(ns, dev, prop, value, timestamp);
            } else if (p->pushMethod->dbMethod->dbConfig->tdengineClientConfig) {
                rc = tdengine_recorder_record(ns, dev, prop, value, timestamp);
            }
        }
        if (rc < 0 && p->pushMethod->methodName) {
            if (strcmp(p->pushMethod->methodName, "mysql") == 0) {
                log_info("dbmethod_recorder_record: backend=mysql (methodName)");
            } else if (strcmp(p->pushMethod->methodName, "redis") == 0) {
                log_info("dbmethod_recorder_record: backend=redis (methodName)");
            } else if (strcmp(p->pushMethod->methodName, "influxdb2") == 0) {
                log_info("dbmethod_recorder_record: backend=influxdb2 (methodName)");
            } else if (strcmp(p->pushMethod->methodName, "tdengine") == 0) {
                log_info("dbmethod_recorder_record: backend=tdengine (methodName)");
            }
        }
    }

    if (g_publisher) {
        DataModel dm = {0};
        dm.namespace_   = (char *)ns;
        dm.deviceName   = (char *)dev;
        dm.propertyName = (char *)prop;
        dm.type         = (char *)"string";
        dm.value        = (char *)(value ? value : "");
        dm.timeStamp    = (int64_t)timestamp;
        publisher_publish_data(g_publisher, &dm);
    }

    return rc;
}