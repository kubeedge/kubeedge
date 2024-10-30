package model.common;

import lombok.Getter;
import lombok.Setter;
@Getter @Setter
public class DataModel {
    private String deviceName = "";
    private String propertyName = "";
    private String nameSpace = "";
    private String value = "";
    private String type = "";
    private long timeStamp;
}
