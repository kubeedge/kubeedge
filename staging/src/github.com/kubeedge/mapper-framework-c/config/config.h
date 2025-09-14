#ifndef COMMON_CONFIG_H
#define COMMON_CONFIG_H

#ifdef __cplusplus
extern "C" {
#endif

// Configuration for the gRPC server
typedef struct {
    char socket_path[256]; // Path to the gRPC server socket
} GRPCServerConfig;

// Common configuration for the framework
typedef struct {
    char name[64];         // Framework name
    char version[32];      // Framework version
    char api_version[32];  // API version
    char protocol[32];     // Communication protocol
    char address[128];     // Address of the framework
    char edgecore_sock[256]; // Path to the EdgeCore socket
    char http_port[16];    // HTTP server port
} CommonConfig;

// Configuration for MySQL database
typedef struct {
    int  enabled;          // Whether MySQL is enabled
    char addr[128];        // MySQL server address
    char database[64];     // Database name
    char username[64];     // Username for authentication
    int  port;             // MySQL server port
    char ssl_mode[16];     // SSL mode (DISABLED, PREFERRED, REQUIRED, etc.)
    char password[64];     // Password for authentication
} DatabaseMySQLConfig;

// Group of database configurations
typedef struct {
    DatabaseMySQLConfig mysql; // MySQL configuration
} DatabaseConfigGroup;

// Main configuration structure
typedef struct Config {
    GRPCServerConfig grpc_server; // gRPC server configuration
    CommonConfig common;          // Common framework configuration
    DatabaseConfigGroup database; // Database configuration group
} Config;

// Parses the configuration file
Config* config_parse(const char *filename);

// Frees the memory allocated for the configuration
void config_free(Config *cfg);

#ifdef __cplusplus
}
#endif

#endif