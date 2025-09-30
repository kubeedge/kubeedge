#include "data/dbmethod/recorder.h"
#include "common/configmaptype.h"
#include "data/dbmethod/mysql/recorder.h"
#include "data/dbmethod/redis/recorder.h"
#include "data/dbmethod/influxdb2/recorder.h"
#include "data/dbmethod/tdengine/recorder.h"
#include <string.h>
#include <stdio.h>

static DeviceProperty *find_property(Device *device, const char *propName)
{
    if (!device || !propName)
        return NULL;
    for (int i = 0; i < device->instance.propertiesCount; ++i)
    {
        DeviceProperty *p = &device->instance.properties[i];
        if (p->name && strcmp(p->name, propName) == 0)
            return p;
    }
    return NULL;
}

int dbmethod_recorder_record(Device *device, const char *propertyName, const char *value, long long timestamp)
{
    const char *ns = device && device->instance.namespace_ ? device->instance.namespace_ : "default";
    const char *dev = device && device->instance.name ? device->instance.name : "unknown";
    const char *prop = propertyName ? propertyName : "unknown";

    DeviceProperty *p = find_property(device, propertyName);
    if (p && p->pushMethod)
    {
        if (p->pushMethod->dbMethod && p->pushMethod->dbMethod->dbConfig)
        {
            if (p->pushMethod->dbMethod->dbConfig->mysqlClientConfig)
            {
                return mysql_recorder_record(ns, dev, prop, value, timestamp);
            }
            if (p->pushMethod->dbMethod->dbConfig->redisClientConfig)
            {
                return redis_recorder_record(ns, dev, prop, value, timestamp);
            }
            if (p->pushMethod->dbMethod->dbConfig->influxdb2ClientConfig)
            {
                return influxdb2_recorder_record(ns, dev, prop, value, timestamp);
            }
            if (p->pushMethod->dbMethod->dbConfig->tdengineClientConfig)
            {
                return tdengine_recorder_record(ns, dev, prop, value, timestamp);
            }
        }
        if (p->pushMethod->methodName)
        {
            if (strcmp(p->pushMethod->methodName, "mysql") == 0)
            {
                return mysql_recorder_record(ns, dev, prop, value, timestamp);
            }
            else if (strcmp(p->pushMethod->methodName, "redis") == 0)
            {
                return redis_recorder_record(ns, dev, prop, value, timestamp);
            }
            else if (strcmp(p->pushMethod->methodName, "influxdb2") == 0)
            {
                return influxdb2_recorder_record(ns, dev, prop, value, timestamp);
            }
            else if (strcmp(p->pushMethod->methodName, "tdengine") == 0)
            {
                return tdengine_recorder_record(ns, dev, prop, value, timestamp);
            }
        }
    }

    return mysql_recorder_record(ns, dev, prop, value, timestamp);
}