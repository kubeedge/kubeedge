// Enable GNU extensions before all includes (for pthread_timedjoin_np)
#define _GNU_SOURCE

#include <stdio.h>
#include <stdlib.h>
#include <signal.h>
#include <unistd.h>
#include <string.h>
#include <time.h>
#include <pthread.h>
#include <sys/stat.h>
#include <limits.h>
#include <cjson/cJSON.h>
#include <errno.h>
#include "log/log.h"
#include "config/config.h"
#include "device/device.h"
#include "grpcclient/register.h"
#include "grpcserver/server.h"
#include "httpserver/httpserver.h"
#include "common/configmaptype.h"
#include "common/const.h"
#include "data/dbmethod/mysql/mysql_client.h"
#include "data/dbmethod/mysql/recorder.h"
#include "data/publish/publisher.h"

static volatile int running = 1;
static DeviceManager *g_deviceManager = NULL;
static GrpcServer *g_grpcServer = NULL;
static RestServer *g_httpServer = NULL;
static MySQLDataBaseConfig *g_mysql = NULL;
// Global publisher (matches extern in device.c)
Publisher *g_publisher = NULL;
static pthread_t g_grpcThread = 0;
static char g_grpcSockPath[PATH_MAX] = {0};
// Device start thread handle
static pthread_t g_devStartThread = 0;

static void cleanup_resources(void);
static void signal_handler(int sig) {
    static int signal_received = 0;
    signal_received++;
    if (signal_received == 1) {
        log_info("Received signal %d, shutting down gracefully...", sig);
        running = 0;
        return;
    }
    // On the second Ctrl+C, force exit to avoid hanging
    log_warn("Received signal %d again, force exiting now.", sig);
    log_flush();
    _exit(128 + sig);
}
static void setup_signal_handlers(void) {
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);
    signal(SIGPIPE, SIG_IGN);
}

static void cleanup_resources(void) {
    log_info("Cleaning up resources...");

    // 1) HTTP server
    log_info("[cleanup] stopping HTTP...");
    if (g_httpServer) {
        rest_server_stop(g_httpServer);
        rest_server_free(g_httpServer);
        g_httpServer = NULL;
    }
    log_info("[cleanup] HTTP done");

    // 2) Devices: send stop first
    log_info("[cleanup] stopping devices...");
    if (g_deviceManager) {
        device_manager_stop_all(g_deviceManager);
    }
    log_info("[cleanup] devices stop_all issued");

    // 2.1) Wait with timeout for the device start thread to exit; try cancel first, then timed join
#ifdef __linux__
    if (g_devStartThread) {
        log_info("[cleanup] joining device_start_thread...");
        // Request cancellation first (if start_all blocks internally, join may not return)
        pthread_cancel(g_devStartThread);
        struct timespec ts;
        clock_gettime(CLOCK_REALTIME, &ts);
        ts.tv_sec += 3; // wait up to 3 seconds
        if (pthread_timedjoin_np(g_devStartThread, NULL, &ts) != 0) {
            log_warn("device_start_thread timed out, force cancel+join");
            pthread_cancel(g_devStartThread);
            pthread_join(g_devStartThread, NULL);
        }
        g_devStartThread = 0;
    }
#endif
    log_info("[cleanup] device_start_thread done");

    // 2.2) Free device manager
    log_info("[cleanup] freeing device manager...");
    if (g_deviceManager) {
        device_manager_free(g_deviceManager);
        g_deviceManager = NULL;
    }
    log_info("[cleanup] device manager freed");

    // 3) gRPC
    log_info("[cleanup] stopping gRPC...");
    if (g_grpcServer) {
        grpcserver_stop(g_grpcServer);
        if (g_grpcThread) {
            pthread_join(g_grpcThread, NULL);
            g_grpcThread = 0;
        }
        grpcserver_free(g_grpcServer);
        g_grpcServer = NULL;
    }
    if (g_grpcSockPath[0]) {
        if (unlink(g_grpcSockPath) != 0) {
            if (errno == ENOENT) {
                log_info("uds socket already gone: %s", g_grpcSockPath);
            } else {
                log_warn("unlink(%s) failed: errno=%d (%s)",
                         g_grpcSockPath, errno, strerror(errno));
            }
        } else {
            log_info("uds socket removed: %s", g_grpcSockPath);
        }
        g_grpcSockPath[0] = '\0';
    }
    log_info("[cleanup] gRPC done");

    // 4) MySQL
    log_info("[cleanup] closing MySQL...");
    if (g_mysql) {
        mysql_close_client(g_mysql);
        free(g_mysql->config.addr);
        free(g_mysql->config.database);
        free(g_mysql->config.userName);
        free(g_mysql->config.password);
        free(g_mysql);
        g_mysql = NULL;
    }
    log_info("[cleanup] MySQL done");

    // 5) Publisher
    if (g_publisher) {
        publisher_free(g_publisher);
        g_publisher = NULL;
        log_info("[cleanup] publisher freed");
    }

    log_info("Cleanup completed");
}

static void* grpc_server_thread(void *arg) {
    GrpcServer *srv = (GrpcServer*)arg;
    // Run in a background thread to avoid blocking the main thread
    grpcserver_start(srv);
    return NULL;
}

// Device start thread function (avoid blocking the main thread in start_all)
static void* device_start_thread(void *arg) {
    pthread_setcancelstate(PTHREAD_CANCEL_ENABLE, NULL);
    pthread_setcanceltype(PTHREAD_CANCEL_DEFERRED, NULL);
    DeviceManager *mgr = (DeviceManager*)arg;
    device_manager_start_all(mgr);
    return NULL;
}

static int wait_uds_ready(const char *path, int timeout_ms) {
    struct stat st;
    int waited = 0;
    while (waited < timeout_ms) {
        if (stat(path, &st) == 0) return 0;
        usleep(100 * 1000);
        waited += 100;
    }
    return -1;
}

int main(int argc, char **argv) {
    int ret = 0;
    Config *config = NULL;
    DeviceInstance *deviceList = NULL;
    DeviceModel *deviceModelList = NULL;
    int deviceCount = 0;
    int modelCount = 0;

    log_init();

    log_info("=== KubeEdge Mapper Framework C Version Starting ===");

    setup_signal_handlers();

    const char *configFile = "../config.yaml";
    if (argc > 1) {
        configFile = argv[1];
    }

    config = config_parse(configFile);
    if (!config) {
        log_error("Failed to parse configuration: %s", configFile);
        ret = EXIT_FAILURE;
        goto cleanup;
    }

    log_info("Configuration loaded successfully");
    log_info("MySQL cfg parsed: enabled=%d addr=%s db=%s user=%s",
             config->database.mysql.enabled,
             config->database.mysql.addr[0] ? config->database.mysql.addr : "(empty)",
             config->database.mysql.database[0] ? config->database.mysql.database : "(empty)",
             config->database.mysql.username[0] ? config->database.mysql.username : "(empty)");
    log_info("MySQL ssl_mode=%s", config->database.mysql.ssl_mode[0] ? config->database.mysql.ssl_mode : "DISABLED");
    setenv("MYSQL_SSL_MODE",
       config->database.mysql.ssl_mode[0] ? config->database.mysql.ssl_mode : "DISABLED",
       1);

    // Environment override: 1/true to enable, 0/false to disable
    const char *env_mysql = getenv("MYSQL_ENABLED");
    if (env_mysql && *env_mysql) {
        if (*env_mysql=='0' || strcasecmp(env_mysql,"false")==0) {
            log_warn("MYSQL_ENABLED env overrides config: disabling MySQL");
            config->database.mysql.enabled = 0;
        } else if (*env_mysql=='1' || strcasecmp(env_mysql,"true")==0) {
            log_warn("MYSQL_ENABLED env overrides config: enabling MySQL");
            config->database.mysql.enabled = 1;
        }
    }

    // Initialize MySQL (self-test)
    if (config->database.mysql.enabled) {
        // Increase buffer to avoid snprintf warnings
        char json[512];
        snprintf(json, sizeof(json),
                 "{\"addr\":\"%s\",\"database\":\"%s\",\"userName\":\"%s\",\"password\":\"%s\",\"port\":%d,\"ssl_mode\":\"%s\"}",
                 config->database.mysql.addr[0] ? config->database.mysql.addr : "127.0.0.1",
                 config->database.mysql.database[0] ? config->database.mysql.database : "testdb",
                 config->database.mysql.username[0] ? config->database.mysql.username : "mapper",
                 config->database.mysql.password[0] ? config->database.mysql.password : "",
                 config->database.mysql.port > 0 ? config->database.mysql.port : 3306,
                 config->database.mysql.ssl_mode[0] ? config->database.mysql.ssl_mode : "DISABLED");
        // Also export to env (if mysql_parse_client_config reads env or prioritizes it)
        if (config->database.mysql.password[0]) {
            setenv("MYSQL_PASSWORD", config->database.mysql.password, 1);
        }

        MySQLClientConfig clientCfg = (MySQLClientConfig){0};
        if (mysql_parse_client_config(json, &clientCfg) != 0) {
            log_error("MySQL client config parse failed");
        } else {
            g_mysql = (MySQLDataBaseConfig*)calloc(1, sizeof(MySQLDataBaseConfig));
            g_mysql->config = clientCfg;
            if (mysql_init_client(g_mysql) != 0) {
                log_error("MySQL init failed (host=%s db=%s user=%s). Set MYSQL_PASSWORD and MYSQL_PORT if needed.",
                          clientCfg.addr, clientCfg.database, clientCfg.userName);
            } else {
                log_info("MySQL connected (host=%s db=%s user=%s pw_len=%zu)",
                         clientCfg.addr, clientCfg.database, clientCfg.userName,
                         clientCfg.password ? strlen(clientCfg.password) : 0);
                DataModel dm = (DataModel){0};
                dm.namespace_   = "default";
                dm.deviceName   = "mysql-selftest";
                dm.propertyName = "ping";
                dm.type         = "string";
                dm.value        = "ok";
                dm.timeStamp    = time(NULL);
                if (mysql_add_data(g_mysql, &dm) == 0) {
                    log_info("MySQL self-test OK -> `%s/%s/%s`", dm.namespace_, dm.deviceName, dm.propertyName);
                } else {
                    log_error("MySQL self-test insert failed");
                }
                mysql_recorder_set_db(g_mysql);
            }
        }
    } else {
        log_info("MySQL disabled in config");
    }

    // Initialize Publisher (via environment variables)
    const char *pm = getenv("PUBLISH_METHOD");     // http | mqtt | otel
    const char *pc = getenv("PUBLISH_CONFIG");     // channel-specific JSON
    if (pm && *pm && pc && *pc) {
        PublishMethodType t = publisher_get_type_from_string(pm);
        g_publisher = publisher_new(t, pc);
        if (g_publisher) {
            log_info("Publish channel ready: %s", pm);
        } else {
            log_warn("Failed to init publish channel: %s", pm);
        }
    } else {
        log_info("Publish channel disabled (set PUBLISH_METHOD and PUBLISH_CONFIG to enable)");
    }

    // Create DeviceManager first (used by gRPC callbacks)
    g_deviceManager = device_manager_new();
    if (!g_deviceManager) {
        log_error("Failed to create device manager");
        ret = EXIT_FAILURE;
        goto cleanup;
    }

    // Start the local gRPC server in a background thread so EdgeCore can connect
    const char *grpc_sock = (config->grpc_server.socket_path[0]
                             ? config->grpc_server.socket_path
                             : "/tmp/mapper_dmi.sock");
    unlink(grpc_sock); // Remove stale socket before starting
    // Save path for unlink during cleanup
    strncpy(g_grpcSockPath, grpc_sock, sizeof(g_grpcSockPath)-1);

    log_info("Starting GRPC server on socket: %s", grpc_sock);
    ServerConfig *grpcConfig = server_config_new(grpc_sock, "customized");
    g_grpcServer = grpcserver_new(grpcConfig, g_deviceManager);
    if (!g_grpcServer) {
        log_error("Failed to create GRPC server");
        server_config_free(grpcConfig);
        ret = EXIT_FAILURE;
        goto cleanup;
    }
    if (pthread_create(&g_grpcThread, NULL, grpc_server_thread, g_grpcServer) != 0) {
        log_error("Failed to create GRPC server thread");
        ret = EXIT_FAILURE;
        goto cleanup;
    }
    server_config_free(grpcConfig);

    // Wait for the UDS file to be ready (up to 3 seconds)
    if (wait_uds_ready(grpc_sock, 3000) != 0) {
        log_warn("GRPC UDS not ready yet: %s", grpc_sock);
    } else {
        chmod(grpc_sock, 0666); // Relax UDS permissions to allow EdgeCore to connect
        log_info("GRPC server started successfully (pre-register)");
    }

    // Register to EdgeCore
    log_info("Mapper will register to edgecore");
    ret = RegisterMapper(1, &deviceList, &deviceCount, &deviceModelList, &modelCount);
    if (ret != 0) {
        log_error("Failed to register mapper to edgecore");
        ret = EXIT_FAILURE;
        goto cleanup;
    }
    log_info("Mapper register finished (devices: %d, models: %d)", deviceCount, modelCount);

    log_info("Initializing devices...");
    for (int i = 0; i < deviceCount; i++) {
        DeviceModel *model = NULL;
        for (int j = 0; j < modelCount; j++) {
            if (deviceModelList[j].name && deviceList[i].model &&
                strcmp(deviceModelList[j].name, deviceList[i].model) == 0) {
                model = &deviceModelList[j];
                break;
            }
        }

        if (!model) {
            log_warn("No model found for device %s", deviceList[i].name);
            continue;
        }

        Device *device = device_new(&deviceList[i], model);
        if (!device) {
            log_error("Failed to create device %s", deviceList[i].name);
            continue;
        }

        if (device_manager_add(g_deviceManager, device) != 0) {
            log_error("Failed to add device %s to manager", deviceList[i].name);
            device_free(device);
            continue;
        }

        log_info("Device %s initialized successfully", deviceList[i].name);
    }

    if (g_deviceManager->deviceCount == 0) {
        log_warn("No devices initialized - mapper will run with empty device list");
    } else {
        log_info("Device initialization finished (%d devices)",
                 g_deviceManager->deviceCount);
    }

    log_info("Starting all devices...");
    // Start devices in a separate thread to avoid blocking the main thread (Ctrl+C responsive)
    if (pthread_create(&g_devStartThread, NULL, device_start_thread, g_deviceManager) != 0) {
        log_error("Failed to create device_start_thread");
    }

    const char *httpPortStr = config->common.http_port;
    if (httpPortStr && strlen(httpPortStr) > 0) {
        log_info("Starting HTTP server on port %s", httpPortStr);
        g_httpServer = rest_server_new(g_deviceManager, httpPortStr);
        if (!g_httpServer) {
            log_error("Failed to create HTTP server");
        } else {
            rest_server_start(g_httpServer);
            log_info("HTTP server started successfully");
        }
    } else {
        log_info("HTTP server disabled (no port configured)");
    }

    log_info("=== Mapper startup completed, running... ===");

    while (running) {
        usleep(1000000);

        if (g_deviceManager && g_deviceManager->deviceCount > 0) {
            static int health_check_counter = 0;
            health_check_counter++;

            if (health_check_counter >= 30) {
                health_check_counter = 0;

                pthread_mutex_lock(&g_deviceManager->managerMutex);
                for (int i = 0; i < g_deviceManager->deviceCount; i++) {
                    Device *device = g_deviceManager->devices[i];
                    if (device) {
                        const char *status = device_get_status(device);
                        if (strcmp(status, DEVICE_STATUS_OK) != 0) {
                            log_warn("Device %s status: %s",
                                   device->instance.name, status);
                        }
                    }
                }
                pthread_mutex_unlock(&g_deviceManager->managerMutex);
            }
        }
    }
    log_info("Main loop exited, shutting down...");
cleanup:
    cleanup_resources();

    if (config) {
        config_free(config);
    }

    log_flush();

    log_info("=== Mapper shutdown completed ===");

    return 0;
}