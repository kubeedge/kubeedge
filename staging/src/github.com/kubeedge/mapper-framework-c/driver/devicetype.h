#ifndef DRIVER_DEVICETYPE_H
#define DRIVER_DEVICETYPE_H

#include <pthread.h>
#include "common/configmaptype.h"

// CustomizedDev: Device configuration and client information
typedef struct {
    DeviceInstance instance;           // Device instance (defined in common/configmaptype.h)
    struct CustomizedClient *client;   // Pointer to the client
} CustomizedDev;

// Protocol configuration structure
typedef struct {
    char *protocolName;   // Protocol name
    char *configData;     // Protocol configuration (JSON string)
} ProtocolConfig;

// Visitor configuration structure
typedef struct {
    char *protocolName;   // Protocol name
    char *configData;     // Visitor configuration (JSON string)
    char *dataType;       // Data type
} VisitorConfig;

// CustomizedClient: Device driver client
typedef struct CustomizedClient {
    pthread_mutex_t deviceMutex;   // Mutex for thread safety
    ProtocolConfig protocolConfig; // Protocol configuration
    // Additional member variables can be added here
} CustomizedClient;

#endif // DRIVER_DEVICETYPE_H