package model.common;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;

import java.util.List;

@Getter @Setter
public class DeviceModel {
    @JsonProperty("id")
    private String id = "";// nameSpace+"/"+name

    @JsonProperty("name")
    private String name = "";

    @JsonProperty("namespace")
    private String nameSpace = "";

    @JsonProperty("description")
    private String description = "";

    @JsonProperty("properties")
    private List<ModelProperty> properties;

    @Getter @Setter
    public static class ModelProperty{
        @JsonProperty("name")
        private String name = "";

        @JsonProperty("description")
        private String description = "";

        @JsonProperty("dataType")
        private String dataType = "";

        @JsonProperty("accessMode")
        private String accessMode = "";

        @JsonProperty("minimum")
        private String minimum = "";

        @JsonProperty("maximum")
        private String maximum = "";

        @JsonProperty("unit")
        private String unit = "";

        public ModelProperty(String name, String description, String dataType, String accessMode, String minimum, String maximum, String unit) {
            this.name = name;
            this.dataType = dataType;
            this.description = description;
            this.accessMode = accessMode;
            this.minimum = minimum;
            this.maximum = maximum;
            this.unit = unit;
        }
    }
}
