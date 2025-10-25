#ifndef DRIVER_DEVICETYPE_H
#define DRIVER_DEVICETYPE_H

#include <pthread.h>
#include "common/configmaptype.h"

// CustomizedDev: Device configuration and client information
typedef struct
{
    DeviceInstance instance;
    struct CustomizedClient *client;
} CustomizedDev;

// Visitor configuration structure
typedef struct
{
    char *propertyName;
    char *protocolName;
    char *configData;
    char *dataType;
    int offset;
} VisitorConfig;

// CustomizedClient: Device driver client
typedef struct CustomizedClient
{
    pthread_mutex_t deviceMutex;
    ProtocolConfig protocolConfig;
} CustomizedClient;

#endif // DRIVER_DEVICETYPE_H