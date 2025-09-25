#ifndef COMMON_DATAMODEL_H
#define COMMON_DATAMODEL_H

#include <stdint.h>

typedef struct DataModel {
    char *deviceName;
    char *propertyName;
    char *namespace_;
    char *value;
    char *type;
    int64_t timeStamp;
} DataModel;

// Create a new DataModel
DataModel *datamodel_new(const char *deviceName, const char *propertyName, const char *namespace_);

// Set type
void datamodel_set_type(DataModel *dm, const char *type);

// Set value
void datamodel_set_value(DataModel *dm, const char *value);

// Set timestamp to current time
void datamodel_set_timestamp(DataModel *dm);

// Set timestamp to a specific value
void datamodel_set_timestamp_value(DataModel *dm, int64_t timestamp);

// Free a DataModel
void datamodel_free(DataModel *dm);

#endif // DATAMODEL_H