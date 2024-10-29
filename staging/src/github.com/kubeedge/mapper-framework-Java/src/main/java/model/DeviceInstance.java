package model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;

import java.util.List;


@Getter @Setter
public class DeviceInstance {
    @JsonProperty("id")
    private String id = ""; // nameSpace+"/"+name

    @JsonProperty("name")
    private String name = "";

    @JsonProperty("namespace")
    private String nameSpace = "";

    @JsonProperty("protocol")
    private String protocolName = "";

    private ProtocolConfig protocolConfig;

    @JsonProperty("model")
    private String model = "";

    @JsonProperty("twins")
    private List<Twin> twins;

    @JsonProperty("properties")
    private List<DeviceProperty> properties;


    @Getter @Setter
    public static class ProtocolConfig {
        @JsonProperty("protocolName")
        private String protocolName = "";

        @JsonProperty("configData")
        private byte[] configData;
        public ProtocolConfig(String protocolName, byte[] configData) {
            this.protocolName = protocolName;
            this.configData = configData;
        }

    }

    @Getter @Setter
    public static class Twin{
        @JsonProperty("propertyName")
        private String propertyName="";

        private DeviceProperty property;

        @JsonProperty("observedDesired")
        private DeviceInstance.TwinProperty observedDesired;

        @JsonProperty("reported")
        private DeviceInstance.TwinProperty reported;
    }

    @Getter @Setter
    public static class DeviceProperty{
        @JsonProperty("name")
        private String name = "";

        @JsonProperty("propertyName")
        private String propertyName = "";

        @JsonProperty("modelName")
        private String modelName = "";

        @JsonProperty("protocol")
        private String protocol = "";

        @JsonProperty("visitConfig")
        private byte[] visitors;

        @JsonProperty("reportToCloud")
        private boolean reportToCloud;

        @JsonProperty("collectCycle")
        private long collectCycle;

        @JsonProperty("reportCycle")
        private long reportCycle;

        @JsonProperty("pushMethod")
        private DeviceInstance.PushMethodConfig pushMethod;

        private DeviceModel.ModelProperty modelProperty;

    }

    @Getter @Setter
    public static class PushMethodConfig {
        @JsonProperty("MethodName")
        private String methodName = "";

        @JsonProperty("MethodConfig")
        private byte[] methodConfig;

        @JsonProperty("dbMethod")
        private DBMethodConfig dbMethod;
    }

    @Getter @Setter
    public static class DBMethodConfig{
        @JsonProperty("dbMethodName")
        private String dbMethodName = "";
        @JsonProperty("dbConfig")
        private DBConfig dbConfig;

    }

    @Getter @Setter
    public static class DBConfig {
        @JsonProperty("influxdb2ClientConfig")
        private byte[] influxdb2ClientConfig;
        @JsonProperty("influxdb2DataConfig")
        private byte[] influxdb2DataConfig;
        @JsonProperty("redisClientConfig")
        private byte[] redisClientConfig;
        @JsonProperty("TDEngineClientConfig")
        private byte[] tdEngineClientConfig;
        @JsonProperty("mysqlClientConfig")
        private byte[] mysqlClientConfig;
    }

    @Getter @Setter
    public static class TwinProperty {
        @JsonProperty("value")
        private String value = "";

        @JsonProperty("metadata")
        private Metadata metadata;
        @Getter @Setter
        public static class Metadata{
            @JsonProperty("timestamp")
            private String timestamp = "";

            @JsonProperty("type")
            private String type = "";
        }
    }
}
