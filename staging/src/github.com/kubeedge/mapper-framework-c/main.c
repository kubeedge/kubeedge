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
#include "data/dbmethod/client.h"
#include "data/publish/publisher.h"
#include "device/dev_panel.h"

static volatile int running = 1;
static DeviceManager *g_deviceManager = NULL;
static GrpcServer *g_grpcServer = NULL;
static RestServer *g_httpServer = NULL;
Publisher *g_publisher = NULL;
static pthread_t g_grpcThread = 0;
static char g_grpcSockPath[PATH_MAX] = {0};
static pthread_t g_devStartThread = 0;

static void signal_handler(int sig)
{
    static int signal_received = 0;
    signal_received++;
    if (signal_received == 1)
    {
        running = 0;
        return;
    }
    log_flush();
    _exit(128 + sig);
}
static void setup_signal_handlers(void)
{
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);
    signal(SIGPIPE, SIG_IGN);
}

static void cleanup_resources(void)
{
    if (g_httpServer)
    {
        rest_server_stop(g_httpServer);
        rest_server_free(g_httpServer);
        g_httpServer = NULL;
    }
    if (g_deviceManager)
    {
        device_manager_stop_all(g_deviceManager);
    }
#ifdef __linux__
    if (g_devStartThread)
    {
        pthread_cancel(g_devStartThread);
        struct timespec ts;
        clock_gettime(CLOCK_REALTIME, &ts);
        ts.tv_sec += 3;
        if (pthread_timedjoin_np(g_devStartThread, NULL, &ts) != 0)
        {
            pthread_cancel(g_devStartThread);
            pthread_join(g_devStartThread, NULL);
        }
        g_devStartThread = 0;
    }
#endif
    if (g_deviceManager)
    {
        device_manager_free(g_deviceManager);
        g_deviceManager = NULL;
    }
    if (g_grpcServer)
    {
        grpcserver_stop(g_grpcServer);
        if (g_grpcThread)
        {
            pthread_join(g_grpcThread, NULL);
            g_grpcThread = 0;
        }
        grpcserver_free(g_grpcServer);
        g_grpcServer = NULL;
    }
    if (g_grpcSockPath[0])
    {
        unlink(g_grpcSockPath);
        g_grpcSockPath[0] = '\0';
    }
    dbmethod_global_free();
    if (g_publisher)
    {
        publisher_free(g_publisher);
        g_publisher = NULL;
    }
}

static void *grpc_server_thread(void *arg)
{
    GrpcServer *srv = (GrpcServer *)arg;
    grpcserver_start(srv);
    return NULL;
}
int main(int argc, char **argv)
{
    int ret = 0;
    Config *config = NULL;
    DeviceInstance *deviceList = NULL;
    DeviceModel *deviceModelList = NULL;
    int deviceCount = 0;
    int modelCount = 0;

    setup_signal_handlers();

    const char *configFile = "../config.yaml";
    if (argc > 1)
    {
        configFile = argv[1];
    }

    config = config_parse(configFile);
    if (!config)
    {
        ret = EXIT_FAILURE;
    }
    else
    {
        dbmethod_global_init();
        const char *pm = getenv("PUBLISH_METHOD");
        const char *pc = getenv("PUBLISH_CONFIG");
        if (pm && *pm && pc && *pc)
        {
            PublishMethodType t = publisher_get_type_from_string(pm);
            g_publisher = publisher_new(t, pc);
        }

        if (panel_init() != 0)
        {
            ret = EXIT_FAILURE;
        }
        else
        {
            g_deviceManager = panel_get_manager();
            if (!g_deviceManager)
            {
                ret = EXIT_FAILURE;
            }
            else
            {
                const char *grpc_sock = config->grpc_server.socket_path;
                unlink(grpc_sock);
                strncpy(g_grpcSockPath, grpc_sock, sizeof(g_grpcSockPath) - 1);

                ServerConfig *grpcConfig = server_config_new(grpc_sock, "customized");
                g_grpcServer = grpcserver_new(grpcConfig, g_deviceManager);
                if (!g_grpcServer)
                {
                    ret = EXIT_FAILURE;
                }
                else if (pthread_create(&g_grpcThread, NULL, grpc_server_thread, g_grpcServer) != 0)
                {
                    ret = EXIT_FAILURE;
                }
                server_config_free(grpcConfig);
                
                if (ret == 0)
                {
                    log_info("Mapper will register to edgecore");
                    ret = RegisterMapper(1, &deviceList, &deviceCount, &deviceModelList, &modelCount);
                    log_info("Mapper register finished (devices: %d, models: %d)", deviceCount, modelCount);
                    if (ret == 0)
                    {
                        panel_dev_init(deviceList, deviceCount, deviceModelList, modelCount);
                        panel_dev_start();

                        const char *httpPortStr = config->common.http_port;
                        if (httpPortStr && strlen(httpPortStr) > 0)
                        {
                            g_httpServer = rest_server_new(g_deviceManager, httpPortStr);
                            if (g_httpServer)
                            {
                                rest_server_start(g_httpServer);
                            }
                        }

                        while (running)
                        {
                            usleep(1000000);
                            if (g_deviceManager && g_deviceManager->deviceCount > 0)
                            {
                                static int health_check_counter = 0;
                                health_check_counter++;
                                if (health_check_counter >= 30)
                                {
                                    health_check_counter = 0;
                                    pthread_mutex_lock(&g_deviceManager->managerMutex);
                                    pthread_mutex_unlock(&g_deviceManager->managerMutex);
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    cleanup_resources();
    if (config)
    {
        config_free(config);
    }
    log_flush();
    return ret;
}
