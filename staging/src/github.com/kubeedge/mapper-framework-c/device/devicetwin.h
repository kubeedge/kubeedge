#ifndef DEVICE_DEVICETWIN_H
#define DEVICE_DEVICETWIN_H

#include "common/configmaptype.h"
#include "common/datamodel.h"
#include "driver/driver.h"
#include <pthread.h>

// Forward declaration of the Device structure
struct Device;
typedef struct Device Device;

// Represents the result of twin data processing
typedef struct {
    int success;                       // Indicates if the processing was successful
    char *value;                       // Processed value
    char *error;                       // Error message
    long long timestamp;               // Timestamp
} TwinResult;

// Represents a twin processor
typedef struct {
    char *propertyName;                // Property name
    char *dataType;                    // Data type
    char *accessMode;                  // Access mode (ReadOnly/ReadWrite)
    VisitorConfig *visitorConfig;      // Visitor configuration
    int reportCycle;                   // Reporting cycle in milliseconds
    pthread_t reportThread;            // Reporting thread
    int reportThreadRunning;           // Reporting thread running flag
} TwinProcessor;

// Represents a twin manager
typedef struct {
    TwinProcessor **processors;        // Array of processors
    int processorCount;                // Number of processors
    int capacity;                      // Capacity of the processor array
    pthread_mutex_t twinMutex;         // Mutex for thread safety
} TwinManager;

// Twin processing functions
int devicetwin_deal(Device *device, const Twin *twin);
int devicetwin_get(Device *device, const char *propertyName, TwinResult *result);
int devicetwin_set(Device *device, const char *propertyName, const char *value, TwinResult *result);

// Twin data processing
int devicetwin_process_data(Device *device, const Twin *twin, const void *data);
int devicetwin_validate_data(const Twin *twin, const char *value);
int devicetwin_convert_data(const Twin *twin, const char *rawValue, char **convertedValue);

// Twin reporting
int devicetwin_report_to_cloud(Device *device, const char *propertyName, const char *value);
int devicetwin_start_auto_report(Device *device, const Twin *twin);
int devicetwin_stop_auto_report(Device *device, const char *propertyName);

// Twin event handling
int devicetwin_handle_desired_change(Device *device, const Twin *twin, const char *newValue);
int devicetwin_handle_reported_update(Device *device, const Twin *twin, const char *newValue);

// Twin processor management
TwinProcessor *devicetwin_processor_new(const Twin *twin);
void devicetwin_processor_free(TwinProcessor *processor);

// Twin manager functions
TwinManager *devicetwin_manager_new(void);
void devicetwin_manager_free(TwinManager *manager);
int devicetwin_manager_add(TwinManager *manager, const Twin *twin);
int devicetwin_manager_remove(TwinManager *manager, const char *propertyName);
TwinProcessor *devicetwin_manager_get(TwinManager *manager, const char *propertyName);

// Utility functions
int devicetwin_parse_visitor_config(const char *configData, VisitorConfig *config);
char *devicetwin_build_report_data(const char *propertyName, const char *value, long long timestamp);

#endif // DEVICE_DEVICETWIN_H