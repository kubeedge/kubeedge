#ifndef HTTPSERVER_HTTPSERVER_H
#define HTTPSERVER_HTTPSERVER_H

#include <microhttpd.h>
#include "device/device.h"

// Represents the REST server structure
typedef struct {
    char ip[32];                  // IP address to bind the server
    char port[8];                 // Port to bind the server
    struct MHD_Daemon *daemon;    // MicroHTTPd daemon instance
    DeviceManager *dev_panel;     // Pointer to the device manager
    // Extendable: Add fields for TLS configuration, database clients, etc.
} RestServer;

// Creates a new REST server instance
RestServer *rest_server_new(DeviceManager *panel, const char *port);

// Starts the REST server
void rest_server_start(RestServer *server);

// Stops the REST server
void rest_server_stop(RestServer *server);

// Frees the memory allocated for the REST server
void rest_server_free(RestServer *server);

#endif // HTTPSERVER_HTTPSERVER_H