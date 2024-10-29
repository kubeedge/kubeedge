package model;

import java.time.Duration;

public class Const {
    public static final String devStOK = "OK";
    public static final String devStErr = "ERROR";
    public static final String devStDisconn = "DISCONNECTED";
    public static final String devStUnhealthy = "UNHEALTHY";
    public static final String devStUnknown = "UNKNOWN";
    public static final String protocolCustomized = "customized-protocol";
    public static final String pushMethodHttp = "http";
    public static final String pushMethodMqtt = "mqtt";
    public static final String devInitModelRegister = "register";
    public static final String devInitModelConfigmap = "configmap";
    public static final String saveFrame = "saveFrame";
    public static final String saveVideo = "saveVideo";

    public static final String topicTwinUpdateDelta = "$hw/events/device/%s/twin/update/delta";
    public static final String topicTwinUpdate = "$hw/events/device/%s/twin/update";
    public static final String topicStateUpdate = "$hw/events/device/%s/state/update";
    public static final String topicDataUpdate = "$ke/events/device/%s/data/update";

    public static final Duration defaultCollectCycle = Duration.ofSeconds(1);
    public static final Duration defaultReportCycle = Duration.ofSeconds(1);
    public static final String apiVersion = "v1";
    public static final String env_TOKEN = "TOKEN";
    public static final String env_PASSWORD = "PASSWORD";
    public static final String env_USERNAME = "USERNAME";
}
