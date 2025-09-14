#ifndef COMMON_CONST_H
#define COMMON_CONST_H

#define DEVICE_STATUS_OK        "ok"
#define DEVICE_STATUS_ONLINE    "online"
#define DEVICE_STATUS_OFFLINE   "offline"
#define DEVICE_STATUS_DISCONN   "disconnected"
#define DEVICE_STATUS_UNHEALTHY "unhealthy"
#define DEVICE_STATUS_UNKNOWN   "unknown"

#define PROTOCOL_CUSTOMIZED "customized-protocol"

#define PUSH_METHOD_HTTP "http"
#define PUSH_METHOD_MQTT "mqtt"
#define PUSH_METHOD_OTEL "otel"

#define DEFAULT_COLLECT_CYCLE 1
#define DEFAULT_REPORT_CYCLE  1

#define DEV_INIT_MODE_REGISTER  "register"
#define DEV_INIT_MODE_CONFIGMAP "configmap"

#define SAVE_FRAME "saveFrame"
#define SAVE_VIDEO "saveVideo"

#endif 