#include "data/dbmethod/recorder.h"
#include "common/configmaptype.h"
#include "data/dbmethod/mysql/recorder.h"
#include "data/dbmethod/redis/recorder.h"
#include "data/dbmethod/influxdb2/recorder.h"
#include "data/dbmethod/tdengine/recorder.h"
#include "grpcclient/register.h"
#include <string.h>
#include <stdio.h>
#include "log/log.h"
#include "common/datamodel.h"

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
    }
    return rc;
}