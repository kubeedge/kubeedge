#ifndef COMMON_CONFIG_H
#define COMMON_CONFIG_H

#ifdef __cplusplus
extern "C" {
#endif

// Configuration for the gRPC server
typedef struct {
    char socket_path[256]; 
} GRPCServerConfig;

// Common configuration for the framework
typedef struct {
    char name[64];         
    char version[32];      
    char api_version[32]; 
    char protocol[32];     
    char address[128];     
    char edgecore_sock[256]; 
    char http_port[16];  
} CommonConfig;

typedef struct {
    int  enabled;
    char addr[128];
    char database[64];
    char username[64];
    int  port;
    char ssl_mode[16];
    char password[64];
} DatabaseMySQLConfig;

typedef struct {
    int enabled;
    char addr[128];
    int port;
    int db;
    int poolSize;
    int minIdleConns;
    char password[64];
} DatabaseRedisConfig;

typedef struct {
    int enabled;
    char url[256];
    char org[64];
    char bucket[64];
    char token[128];
} DatabaseInfluxdbConfig;

typedef struct {
    int enabled;
    char addr[128];
    char dbName[64];
    char username[64];
    char password[64];
    int port;
} DatabaseTDEngineConfig;

typedef struct {
    DatabaseMySQLConfig mysql;
    DatabaseRedisConfig redis;
    DatabaseInfluxdbConfig influxdb2;
    DatabaseTDEngineConfig tdengine;
} DatabaseConfigGroup;

// Main configuration structure
typedef struct Config {
    GRPCServerConfig grpc_server; 
    CommonConfig common;          
    DatabaseConfigGroup database; 
} Config;

// Parses the configuration file
Config* config_parse(const char *filename);

// Frees the memory allocated for the configuration
void config_free(Config *cfg);

#ifdef __cplusplus
}
#endif

#endif