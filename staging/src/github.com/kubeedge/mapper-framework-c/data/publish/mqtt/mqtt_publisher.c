#include "mqtt_publisher.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <cjson/cJSON.h>
#include <stdio.h>

// Parse MQTT configuration
int mqtt_parse_config(const char *json, MqttPublishConfig *config)
{
    if (!json || !config)
        return -1;

    memset(config, 0, sizeof(MqttPublishConfig));

    cJSON *root = cJSON_Parse(json);
    if (!root)
    {
        log_error("Failed to parse MQTT config JSON");
        return -1;
    }

    cJSON *broker_url = cJSON_GetObjectItem(root, "brokerUrl");
    cJSON *port = cJSON_GetObjectItem(root, "port");
    cJSON *client_id = cJSON_GetObjectItem(root, "clientId");
    cJSON *username = cJSON_GetObjectItem(root, "username");
    cJSON *password = cJSON_GetObjectItem(root, "password");
    cJSON *topic_prefix = cJSON_GetObjectItem(root, "topicPrefix");
    cJSON *qos = cJSON_GetObjectItem(root, "qos");
    cJSON *keep_alive = cJSON_GetObjectItem(root, "keepAlive");

    config->broker_url = broker_url ? strdup(broker_url->valuestring) : strdup("localhost");
    config->port = port ? port->valueint : 1883;
    config->client_id = client_id ? strdup(client_id->valuestring) : strdup("mapper_client");
    config->username = username ? strdup(username->valuestring) : NULL;
    config->password = password ? strdup(password->valuestring) : NULL;
    config->topic_prefix = topic_prefix ? strdup(topic_prefix->valuestring) : strdup("kubeedge/device");
    config->qos = qos ? qos->valueint : 1;
    config->keep_alive = keep_alive ? keep_alive->valueint : 60;

    cJSON_Delete(root);
    return 0;
}

// Free MQTT configuration
void mqtt_free_config(MqttPublishConfig *config)
{
    if (!config)
        return;

    free(config->broker_url);
    free(config->client_id);
    free(config->username);
    free(config->password);
    free(config->topic_prefix);
    memset(config, 0, sizeof(MqttPublishConfig));
}

// MQTT connection callback
static void mqtt_connect_callback(struct mosquitto *mosq, void *userdata, int result)
{
    MqttPublisher *publisher = (MqttPublisher *)userdata;

    if (result == 0)
    {
        publisher->connected = 1;
        log_debug("MQTT connected successfully");
    }
    else
    {
        publisher->connected = 0;
        log_error("MQTT connection failed: %s", mosquitto_connack_string(result));
    }
}

// MQTT disconnect callback
static void mqtt_disconnect_callback(struct mosquitto *mosq, void *userdata, int rc)
{
    MqttPublisher *publisher = (MqttPublisher *)userdata;
    publisher->connected = 0;
    log_warn("MQTT disconnected: %s", rc ? "unexpected" : "clean");
}

// Create MQTT publisher
MqttPublisher *mqtt_publisher_new(const char *config_json)
{
    if (!config_json)
        return NULL;

    mosquitto_lib_init();

    MqttPublisher *publisher = calloc(1, sizeof(MqttPublisher));
    if (!publisher)
    {
        mosquitto_lib_cleanup();
        return NULL;
    }

    if (mqtt_parse_config(config_json, &publisher->config) != 0)
    {
        free(publisher);
        mosquitto_lib_cleanup();
        return NULL;
    }

    publisher->connected = 0;
    publisher->mosq = mosquitto_new(publisher->config.client_id, true, publisher);
    if (!publisher->mosq)
    {
        log_error("Failed to create MQTT client");
        mqtt_free_config(&publisher->config);
        free(publisher);
        mosquitto_lib_cleanup();
        return NULL;
    }

    mosquitto_connect_callback_set(publisher->mosq, mqtt_connect_callback);
    mosquitto_disconnect_callback_set(publisher->mosq, mqtt_disconnect_callback);

    if (publisher->config.username && publisher->config.password)
    {
        mosquitto_username_pw_set(publisher->mosq, publisher->config.username, publisher->config.password);
    }

    log_info("MQTT publisher created for broker: %s:%d",
             publisher->config.broker_url, publisher->config.port);
    return publisher;
}

// Free MQTT publisher
void mqtt_publisher_free(MqttPublisher *publisher)
{
    if (!publisher)
        return;

    if (publisher->mosq)
    {
        if (publisher->connected)
        {
            mosquitto_disconnect(publisher->mosq);
        }
        mosquitto_destroy(publisher->mosq);
    }

    mqtt_free_config(&publisher->config);
    free(publisher);
    mosquitto_lib_cleanup();
}

// Ensure MQTT connection
static int mqtt_ensure_connected(MqttPublisher *publisher)
{
    if (publisher->connected)
        return 0;

    int ret = mosquitto_connect(publisher->mosq, publisher->config.broker_url,
                                publisher->config.port, publisher->config.keep_alive);
    if (ret != MOSQ_ERR_SUCCESS)
    {
        log_error("Failed to connect to MQTT broker: %s", mosquitto_strerror(ret));
        return -1;
    }

    int wait_count = 0;
    while (!publisher->connected && wait_count < 50)
    {
        mosquitto_loop(publisher->mosq, 100, 1);
        wait_count++;
    }

    if (!publisher->connected)
    {
        log_error("MQTT connection timeout");
        return -1;
    }

    return 0;
}

// Publish data to MQTT (synchronous)
int mqtt_publisher_publish(MqttPublisher *publisher, const DataModel *data)
{
    if (!publisher || !data)
        return -1;

    if (mqtt_ensure_connected(publisher) != 0)
    {
        return -1;
    }

    cJSON *json = cJSON_CreateObject();
    cJSON_AddStringToObject(json, "deviceName", data->deviceName ? data->deviceName : "");
    cJSON_AddStringToObject(json, "namespace", data->namespace_ ? data->namespace_ : "");
    cJSON_AddStringToObject(json, "propertyName", data->propertyName ? data->propertyName : "");
    cJSON_AddStringToObject(json, "value", data->value ? data->value : "");
    cJSON_AddStringToObject(json, "type", data->type ? data->type : "string");
    cJSON_AddNumberToObject(json, "timestamp", data->timeStamp);

    char *json_string = cJSON_PrintUnformatted(json);
    cJSON_Delete(json);

    if (!json_string)
    {
        log_error("Failed to create JSON data");
        return -1;
    }

    char topic[256];
    snprintf(topic, sizeof(topic), "%s/%s/%s",
             publisher->config.topic_prefix,
             data->deviceName ? data->deviceName : "unknown",
             data->propertyName ? data->propertyName : "data");

    int ret = mosquitto_publish(publisher->mosq, NULL, topic, strlen(json_string),
                                json_string, publisher->config.qos, false);

    if (ret != MOSQ_ERR_SUCCESS)
    {
        log_error("Failed to publish MQTT message: %s", mosquitto_strerror(ret));
        free(json_string);
        return -1;
    }

    mosquitto_loop(publisher->mosq, 100, 1);
    free(json_string);
    log_debug("MQTT published data to topic: %s", topic);
    return 0;
}