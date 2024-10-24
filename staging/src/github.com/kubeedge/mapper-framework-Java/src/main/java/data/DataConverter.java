package data;

import com.google.gson.Gson;
import com.google.protobuf.*;
import lombok.extern.slf4j.Slf4j;

@Slf4j
public class DataConverter {

    public static Object decodeAnyValue(Any any) throws InvalidProtocolBufferException {
        // Decode any value to Object
        String typeUrl = any.getTypeUrl();
        String messageTypeName = getMessageTypeName(typeUrl);
        if (messageTypeName.isEmpty()){
            log.error("Can not get message type: {}",typeUrl);
        }
        if (messageTypeName.contains("google.protobuf.")){
            switch (messageTypeName){
                case "google.protobuf.Int32Value":
//                    return Int32Value.getDefaultInstance().getParserForType().parseFrom(any.getValue());
                    return Int32Value.parseFrom(any.getValue()).getValue();
                case "google.protobuf.StringValue":
                    return StringValue.parseFrom(any.getValue()).getValue();
                case "google.protobuf.FloatValue":
                    return FloatValue.parseFrom(any.getValue()).getValue();
                case "google.protobuf.BoolValue":
                    return BoolValue.parseFrom(any.getValue()).getValue();
                case "google.protobuf.Int64Value":
                    return Int64Value.parseFrom(any.getValue()).getValue();
                default:
                    log.error("Unknown google.protobuf type: {}",messageTypeName);
                    return null;
            }
        }else{
            log.error("Can not get messageType: {}",messageTypeName);
        }

        return null;
    }

    public static String getMessageTypeName(String typeUrl){// typeUrl例如 type.googleapis.com/google.protobuf.Int32Value
        int lastSlash = typeUrl.lastIndexOf('/');
        if (lastSlash != -1 && lastSlash < typeUrl.length() - 1) {
            return typeUrl.substring(lastSlash + 1);
        }
        return "";
    }

    public static Object convert(String valueType, String value) throws Exception {
        switch (valueType){
            case "int": return Long.parseLong(value);
            case "float": return Float.parseFloat(value);
            case "double": return Double.parseDouble(value);
            case "boolean": return Boolean.parseBoolean(value);
            case "string": return value;
            default:
                log.error("Failed to convert value as {}",valueType);
                throw new Exception("Failed to convert value as " + valueType);
        }
    }

    public static String convertToString(Object value){
        if (value == null){
            return "";
        }else if(value instanceof Number || value instanceof Boolean || value instanceof Character){
            return String.valueOf(value);
        }else if (value instanceof String) {
            return  (String) value;
        } else if (value instanceof byte[]) {
            return new String((byte[]) value);
        } else {
            log.warn("Unknown value Type: {}",value.getClass());
            Gson gson = new Gson();
            try {
                return gson.toJson(value);
            } catch (Exception e) {
                log.error("Failed to convert {} value as string", value.getClass(),e);
            }
        }
        return null;
    }
}
