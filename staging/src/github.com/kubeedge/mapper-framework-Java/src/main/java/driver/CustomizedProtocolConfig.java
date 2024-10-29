package driver;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
@Getter @Setter
public class CustomizedProtocolConfig {
    @JsonProperty("protocolName")
    private String protocolName = "";

    @JsonProperty("configData")
    private ConfigData configData;
    @Getter @Setter
    public static class ConfigData{
        // TODO: add your protocol config data
        // Example: Modbus
        private String address = "";
    }
}
