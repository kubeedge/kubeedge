package model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;

@Getter @Setter
public class VisitorConfig {
    @JsonProperty("protocolName")
    private String protocolName = "";

    @JsonProperty("configData")
    private VisitorConfigData visitorConfigData;

    @Getter @Setter
    public static class VisitorConfigData{
        // TODO: add your visitor config data
        // Example: Modbus
        @JsonProperty("offset")
        private int offset;
        @JsonProperty("register")
        private String register = "";
    }
}
