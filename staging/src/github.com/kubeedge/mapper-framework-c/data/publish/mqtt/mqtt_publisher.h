#ifndef MQTT_PUBLISHER_H
#define MQTT_PUBLISHER_H

#include "common/datamodel.h"
#include <mosquitto.h>

// MQTT publish configuration
typedef struct
{
    char *broker_url;
    int port;
    char *client_id;
    char *username;
    char *password;
    char *topic_prefix;
    int qos;
    int keep_alive;
} MqttPublishConfig;

// MQTT publisher
typedef struct
{
    MqttPublishConfig config;
    struct mosquitto *mosq;
    int connected;
} MqttPublisher;

// Function declarations
int mqtt_parse_config(const char *json, MqttPublishConfig *config);
void mqtt_free_config(MqttPublishConfig *config);

MqttPublisher *mqtt_publisher_new(const char *config_json);
void mqtt_publisher_free(MqttPublisher *publisher);
int mqtt_publisher_publish(MqttPublisher *publisher, const DataModel *data);

#endif // MQTT_PUBLISHER_H