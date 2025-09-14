#ifndef DRIVER_DRIVER_H
#define DRIVER_DRIVER_H

#include "common/configmaptype.h"
#include <pthread.h>


// Visitor configuration structure (can be adjusted based on your VisitorConfig definition)
typedef struct {
    char *propertyName;   // Property name
    char *protocolName;   // Protocol name
    char *configData;     // Configuration data (recommended as a JSON string)
} VisitorConfig;

// Driver client structure
typedef struct {
    ProtocolConfig protocolConfig; // Protocol configuration
    pthread_mutex_t deviceMutex;   // Mutex for thread safety
    // Additional member variables can be added here
} CustomizedClient;

// Constructor and destructor
CustomizedClient *NewClient(const ProtocolConfig *protocol);
void FreeClient(CustomizedClient *client);

// Device operation interfaces
int InitDevice(CustomizedClient *client);
int GetDeviceData(CustomizedClient *client, const VisitorConfig *visitor, void **out_data);
int DeviceDataWrite(CustomizedClient *client, const VisitorConfig *visitor, const char *deviceMethodName, const char *propertyName, const void *data);
int SetDeviceData(CustomizedClient *client, const void *data, const VisitorConfig *visitor);
int StopDevice(CustomizedClient *client);
const char *GetDeviceStates(CustomizedClient *client);

#endif // DRIVER_DRIVER_H