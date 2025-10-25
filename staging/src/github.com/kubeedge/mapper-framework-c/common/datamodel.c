#include "common/datamodel.h"
#include <stdlib.h>
#include <string.h>
#include <time.h>

static int64_t get_timestamp() {
    return (int64_t)time(NULL);
}

DataModel *datamodel_new(const char *deviceName, const char *propertyName, const char *namespace_) {
    DataModel *dm = (DataModel *)calloc(1, sizeof(DataModel));
    if (!dm) return NULL;
    dm->deviceName = deviceName ? strdup(deviceName) : NULL;
    dm->propertyName = propertyName ? strdup(propertyName) : NULL;
    dm->namespace_ = namespace_ ? strdup(namespace_) : NULL;
    dm->value = NULL;
    dm->type = NULL;
    dm->timeStamp = get_timestamp();
    return dm;
}

void datamodel_set_type(DataModel *dm, const char *type) {
    if (!dm) return;
    if (dm->type) free(dm->type);
    dm->type = type ? strdup(type) : NULL;
}

void datamodel_set_value(DataModel *dm, const char *value) {
    if (!dm) return;
    if (dm->value) free(dm->value);
    dm->value = value ? strdup(value) : NULL;
}

void datamodel_set_timestamp(DataModel *dm) {
    if (!dm) return;
    dm->timeStamp = get_timestamp();
}

void datamodel_set_timestamp_value(DataModel *dm, int64_t timestamp) {
    if (!dm) return;
    dm->timeStamp = timestamp;
}

void datamodel_free(DataModel *dm) {
    if (!dm) return;
    free(dm->deviceName);
    free(dm->propertyName);
    free(dm->namespace_);
    free(dm->value);
    free(dm->type);
    free(dm);
}