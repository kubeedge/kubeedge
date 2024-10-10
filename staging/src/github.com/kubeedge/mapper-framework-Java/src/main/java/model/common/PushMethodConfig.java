package model.common;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;

@Getter @Setter
public class PushMethodConfig {
    @JsonProperty("MethodName")
    private String methodName = "";
    @JsonProperty("MethodConfig")
    private byte[] methodConfig;
    @JsonProperty("DBMethod")
    private DBMethodConfig dbMethod;

    @Getter @Setter
    public static class DBMethodConfig{
        @JsonProperty("dbMethodName")
        private String dbMethodName = "";
        @JsonProperty("dbConfig")
        private PushMethodConfig.DBConfig dbConfig;

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
}
