#include "httpserver/httpserver.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <cjson/cJSON.h>
#include "util/parse/grpc.h"
#include "common/datamodel.h"
#include "common/datamethod.h"
#include "device/device.h"
#include "device/dev_panel.h"
#include "log/log.h"
#define API_VERSION "v1"
#define API_BASE "/api/" API_VERSION
#define API_PING API_BASE "/ping"
#define API_DEVICE API_BASE "/device"
#define API_DEVICE_METHOD API_BASE "/devicemethod"
#define API_META API_BASE "/meta"
#define API_DATABASE API_BASE "/database"
#define CONTENT_TYPE "Content-Type"
#define CONTENT_TYPE_JSON "application/json"
#define CORRELATION_HEADER "X-Correlation-ID"

// Utility function: Get the current timestamp as a string
static void get_time_str(char *buf, size_t buflen)
{
    time_t now = time(NULL);
    struct tm *tm_info = localtime(&now);
    strftime(buf, buflen, "%Y-%m-%dT%H:%M:%SZ", tm_info);
}

// Utility function: Send a JSON response
static int send_json_response(struct MHD_Connection *connection, cJSON *resp, int status_code)
{
    char *json = cJSON_PrintUnformatted(resp);
    struct MHD_Response *response = MHD_create_response_from_buffer(strlen(json), json, MHD_RESPMEM_MUST_FREE);
    MHD_add_response_header(response, CONTENT_TYPE, CONTENT_TYPE_JSON);
    int ret = MHD_queue_response(connection, status_code, response);
    MHD_destroy_response(response);
    return ret;
}

// Handle the /ping endpoint
static int handle_ping(struct MHD_Connection *connection)
{
    cJSON *resp = cJSON_CreateObject();
    char timebuf[64];
    get_time_str(timebuf, sizeof(timebuf));
    cJSON_AddStringToObject(resp, "apiVersion", API_VERSION);
    cJSON_AddNumberToObject(resp, "statusCode", 200);
    cJSON_AddStringToObject(resp, "timeStamp", timebuf);
    cJSON_AddStringToObject(resp, "message", "This is v1 API, the server is running normally.");
    int ret = send_json_response(connection, resp, MHD_HTTP_OK);
    cJSON_Delete(resp);
    return ret;
}

// Handle reading device data
static int handle_device_read(RestServer *server, struct MHD_Connection *connection, const char *namespace, const char *name, const char *property)
{
    char deviceID[256];
    get_resource_id(namespace, name, deviceID, sizeof(deviceID));
    char *value = NULL, *datatype = NULL;
    int err = dev_panel_get_twin_result(server->dev_panel, deviceID, property, &value, &datatype);
    if (err != 0)
    {
        cJSON *resp = cJSON_CreateObject();
        char timebuf[64];
        get_time_str(timebuf, sizeof(timebuf));
        cJSON_AddStringToObject(resp, "apiVersion", API_VERSION);
        cJSON_AddNumberToObject(resp, "statusCode", 500);
        cJSON_AddStringToObject(resp, "timeStamp", timebuf);
        char msg[256];
        snprintf(msg, sizeof(msg), "Get device data error: %d", err);
        cJSON_AddStringToObject(resp, "message", msg);
        int ret = send_json_response(connection, resp, MHD_HTTP_INTERNAL_SERVER_ERROR);
        cJSON_Delete(resp);
        return ret;
    }
    cJSON *resp = cJSON_CreateObject();
    char timebuf[64];
    get_time_str(timebuf, sizeof(timebuf));
    cJSON_AddStringToObject(resp, "apiVersion", API_VERSION);
    cJSON_AddNumberToObject(resp, "statusCode", 200);
    cJSON_AddStringToObject(resp, "timeStamp", timebuf);

    cJSON *data = cJSON_CreateObject();
    cJSON_AddStringToObject(data, "deviceName", name);
    cJSON_AddStringToObject(data, "propertyName", property);
    cJSON_AddStringToObject(data, "deviceNamespace", namespace);
    cJSON_AddStringToObject(data, "value", value ? value : "");
    cJSON_AddStringToObject(data, "type", datatype ? datatype : "");
    cJSON_AddItemToObject(resp, "data", data);

    int ret = send_json_response(connection, resp, MHD_HTTP_OK);
    cJSON_Delete(resp);
    free(value);
    free(datatype);
    return ret;
}

// Handle writing device data
static int handle_device_write(RestServer *server, struct MHD_Connection *connection, const char *namespace, const char *name, const char *method, const char *property, const char *data)
{
    char deviceID[256];
    get_resource_id(namespace, name, deviceID, sizeof(deviceID));
    int err = dev_panel_write_device(server->dev_panel, method, deviceID, property, data);
    cJSON *resp = cJSON_CreateObject();
    char timebuf[64];
    get_time_str(timebuf, sizeof(timebuf));
    cJSON_AddStringToObject(resp, "apiVersion", API_VERSION);
    cJSON_AddNumberToObject(resp, "statusCode", err == 0 ? 200 : 500);
    cJSON_AddStringToObject(resp, "timeStamp", timebuf);
    if (err == 0)
    {
        char msg[512];
        snprintf(msg, sizeof(msg), "Write data %s to device %s successfully.", data, deviceID);
        cJSON_AddStringToObject(resp, "message", msg);
        int ret = send_json_response(connection, resp, MHD_HTTP_OK);
        cJSON_Delete(resp);
        return ret;
    }
    else
    {
        char msg[512];
        snprintf(msg, sizeof(msg), "Write device data error: %d", err);
        cJSON_AddStringToObject(resp, "message", msg);
        int ret = send_json_response(connection, resp, MHD_HTTP_INTERNAL_SERVER_ERROR);
        cJSON_Delete(resp);
        return ret;
    }
}

// Handle retrieving device methods
static int handle_get_device_method(RestServer *server, struct MHD_Connection *connection, const char *namespace, const char *name)
{
    char deviceID[256];
    get_resource_id(namespace, name, deviceID, sizeof(deviceID));
    char **method_map = NULL;
    int method_count = 0;
    char **property_map = NULL;
    int property_count = 0;
    int err = dev_panel_get_device_method(server->dev_panel, deviceID, &method_map, &method_count, &property_map, &property_count);
    if (err != 0)
    {
        char msg[512];
        snprintf(msg, sizeof(msg), "Get device method error: %d", err);
        struct MHD_Response *response = MHD_create_response_from_buffer(strlen(msg), (void *)msg, MHD_RESPMEM_PERSISTENT);
        int ret = MHD_queue_response(connection, MHD_HTTP_INTERNAL_SERVER_ERROR, response);
        MHD_destroy_response(response);
        return ret;
    }
    cJSON *resp = cJSON_CreateObject();
    char timebuf[64];
    get_time_str(timebuf, sizeof(timebuf));
    cJSON_AddStringToObject(resp, "apiVersion", API_VERSION);
    cJSON_AddNumberToObject(resp, "statusCode", 200);
    cJSON_AddStringToObject(resp, "timeStamp", timebuf);

    cJSON *data = cJSON_CreateObject();
    cJSON *methods = cJSON_CreateArray();
    for (int i = 0; i < method_count; ++i)
    {
        cJSON *method = cJSON_CreateObject();
        cJSON_AddStringToObject(method, "name", method_map[i]);
        char path[256];
        snprintf(path, sizeof(path), API_DEVICE_METHOD "/%s/%s/%s/{propertyName}/{data}", namespace, name, method_map[i]);
        cJSON_AddStringToObject(method, "path", path);
        cJSON *params = cJSON_CreateArray();
        if (i < property_count)
        {
            cJSON *param = cJSON_CreateObject();
            cJSON_AddStringToObject(param, "propertyName", property_map[i]);
            cJSON_AddStringToObject(param, "valueType", "string");
            cJSON_AddItemToArray(params, param);
        }
        cJSON_AddItemToObject(method, "parameters", params);
        cJSON_AddItemToArray(methods, method);
    }
    cJSON_AddItemToObject(data, "methods", methods);
    cJSON_AddItemToObject(resp, "data", data);

    int ret = send_json_response(connection, resp, MHD_HTTP_OK);
    cJSON_Delete(resp);
    for (int i = 0; i < method_count; ++i)
    {
        free(method_map[i]);
    }
    free(method_map);
    for (int i = 0; i < property_count; ++i)
    {
        free(property_map[i]);
    }
    free(property_map);
    return ret;
}

// Handle retrieving device model metadata
static int handle_meta_get_model(RestServer *server, struct MHD_Connection *connection, const char *namespace, const char *name)
{
    char deviceID[256];
    get_resource_id(namespace, name, deviceID, sizeof(deviceID));
    DeviceInstance instance = {0};
    int err = dev_panel_get_device(server->dev_panel, deviceID, &instance);
    if (err != 0)
    {
        char msg[512];
        snprintf(msg, sizeof(msg), "Get device error: %d", err);
        struct MHD_Response *response = MHD_create_response_from_buffer(strlen(msg), (void *)msg, MHD_RESPMEM_PERSISTENT);
        int ret = MHD_queue_response(connection, MHD_HTTP_INTERNAL_SERVER_ERROR, response);
        MHD_destroy_response(response);
        return ret;
    }
    char modelID[256];
    get_resource_id(instance.namespace_, instance.model, modelID, sizeof(modelID));
    DeviceModel model = {0};
    err = dev_panel_get_model(server->dev_panel, modelID, &model);
    if (err != 0)
    {
        char msg[512];
        snprintf(msg, sizeof(msg), "Get device model error: %d", err);
        struct MHD_Response *response = MHD_create_response_from_buffer(strlen(msg), (void *)msg, MHD_RESPMEM_PERSISTENT);
        int ret = MHD_queue_response(connection, MHD_HTTP_INTERNAL_SERVER_ERROR, response);
        MHD_destroy_response(response);
        return ret;
    }
    cJSON *resp = cJSON_CreateObject();
    char timebuf[64];
    get_time_str(timebuf, sizeof(timebuf));
    cJSON_AddStringToObject(resp, "apiVersion", API_VERSION);
    cJSON_AddNumberToObject(resp, "statusCode", 200);
    cJSON_AddStringToObject(resp, "timeStamp", timebuf);
    cJSON *data = cJSON_CreateObject();
    cJSON_AddStringToObject(data, "name", model.name ? model.name : "");
    cJSON_AddStringToObject(data, "namespace", model.namespace_ ? model.namespace_ : "");
    cJSON_AddStringToObject(data, "description", model.description ? model.description : "");
    cJSON_AddItemToObject(resp, "data", data);

    int ret = send_json_response(connection, resp, MHD_HTTP_OK);
    cJSON_Delete(resp);
    return ret;
}

// Handle retrieving database data (returns empty data)
static int handle_database_get_data(RestServer *server, struct MHD_Connection *connection)
{
    cJSON *resp = cJSON_CreateObject();
    char timebuf[64];
    get_time_str(timebuf, sizeof(timebuf));
    cJSON_AddStringToObject(resp, "apiVersion", API_VERSION);
    cJSON_AddNumberToObject(resp, "statusCode", 200);
    cJSON_AddStringToObject(resp, "timeStamp", timebuf);
    cJSON *data = cJSON_CreateArray();
    cJSON_AddItemToObject(resp, "data", data);
    int ret = send_json_response(connection, resp, MHD_HTTP_OK);
    cJSON_Delete(resp);
    return ret;
}

// Router callback for handling HTTP requests
static enum MHD_Result router_callback(void *cls, struct MHD_Connection *connection,
                                       const char *url, const char *method,
                                       const char *version, const char *upload_data,
                                       size_t *upload_data_size, void **con_cls)
{
    RestServer *server = (RestServer *)cls;
    if (strcmp(method, "GET") != 0)
    {
        struct MHD_Response *response = MHD_create_response_from_buffer(0, "", MHD_RESPMEM_PERSISTENT);
        int ret = MHD_queue_response(connection, MHD_HTTP_METHOD_NOT_ALLOWED, response);
        MHD_destroy_response(response);
        return ret;
    }
    if (strcmp(url, API_PING) == 0)
    {
        return handle_ping(connection);
    }
    if (strncmp(url, API_DEVICE, strlen(API_DEVICE)) == 0)
    {
        char namespace[64], name[64], property[64];
        if (sscanf(url, API_DEVICE "/%63[^/]/%63[^/]/%63[^/]", namespace, name, property) == 3)
        {
            return handle_device_read(server, connection, namespace, name, property);
        }
    }
    if (strncmp(url, API_DEVICE_METHOD, strlen(API_DEVICE_METHOD)) == 0)
    {
        char namespace[64], name[64], method[64], property[64], data[128];
        if (sscanf(url, API_DEVICE_METHOD "/%63[^/]/%63[^/]/%63[^/]/%63[^/]/%127[^/]", namespace, name, method, property, data) == 5)
        {
            return handle_device_write(server, connection, namespace, name, method, property, data);
        }
        if (sscanf(url, API_DEVICE_METHOD "/%63[^/]/%63[^/]", namespace, name) == 2)
        {
            return handle_get_device_method(server, connection, namespace, name);
        }
    }
    if (strncmp(url, API_META "/model", strlen(API_META "/model")) == 0)
    {
        char namespace[64], name[64];
        if (sscanf(url, API_META "/model/%63[^/]/%63[^/]", namespace, name) == 2)
        {
            return handle_meta_get_model(server, connection, namespace, name);
        }
    }
    if (strncmp(url, API_DATABASE, strlen(API_DATABASE)) == 0)
    {
        char namespace[64], name[64];
        if (sscanf(url, API_DATABASE "/%63[^/]/%63[^/]", namespace, name) == 2)
        {
            return handle_database_get_data(server, connection);
        }
    }
    struct MHD_Response *response = MHD_create_response_from_buffer(0, "", MHD_RESPMEM_PERSISTENT);
    int ret = MHD_queue_response(connection, MHD_HTTP_NOT_FOUND, response);
    MHD_destroy_response(response);
    return ret;
}

// Creates a new REST server instance
RestServer *rest_server_new(DeviceManager *panel, const char *port)
{
    RestServer *server = calloc(1, sizeof(RestServer));
    strcpy(server->ip, "0.0.0.0");
    strcpy(server->port, port ? port : "7777");
    server->dev_panel = panel;
    return server;
}

// Starts the REST server
void rest_server_start(RestServer *server)
{
    server->daemon = MHD_start_daemon(MHD_USE_SELECT_INTERNALLY, atoi(server->port),
                                      NULL, NULL, &router_callback, server, MHD_OPTION_END);
}

// Stops the REST server
void rest_server_stop(RestServer *server)
{
    if (server->daemon)
    {
        MHD_stop_daemon(server->daemon);
        server->daemon = NULL;
    }
}

// Frees the memory allocated for the REST server
void rest_server_free(RestServer *server)
{
    if (server)
        free(server);
}