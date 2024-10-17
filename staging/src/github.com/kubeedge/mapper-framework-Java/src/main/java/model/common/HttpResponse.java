package model.common;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;

import java.util.List;

public class HttpResponse {

    @Getter @Setter
    public static class BaseResponse{
        @JsonProperty("apiVersion")
        private String apiVersion = "";
        @JsonProperty("statusCode")
        private int statusCode;
        @JsonProperty("timeStamp")
        private String timeStamp = "";
    }
    @Getter @Setter
    public static class PingResponse extends BaseResponse {
        private String message = "";
    }
    @Getter @Setter
    public static class DeviceReadResponse extends BaseResponse{
        private DataModel data;
    }

    @Getter @Setter
    public static class MetaGetModelResponse extends BaseResponse{
        private DeviceModel deviceModel;
    }

    @Getter @Setter
    public static class DataBaseResponse extends BaseResponse{
        private List<DataModel> dataList;
    }
}
