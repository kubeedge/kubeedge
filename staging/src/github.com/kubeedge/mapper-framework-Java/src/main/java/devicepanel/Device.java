package devicepanel;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.google.protobuf.Any;
import com.google.protobuf.InvalidProtocolBufferException;
import com.hubspot.jackson.datatype.protobuf.ProtobufModule;
import data.DataConverter;
import data.TwinData;
import data.dbmethod.Influxdb2;
import data.dbmethod.Mysql;
import data.dbmethod.Redis;
import data.dbmethod.Tdengine;
import data.publish.DataPanel;
import data.publish.Http;
import data.publish.Mqtt;
import data.stream.StreamHandler;
import model.CustomizedDev;
import model.common.DataModel;
import model.common.DeviceInstance;
import model.common.DeviceModel;
import dmi.v1beta1.Api;
import driver.CustomizedClient;
import driver.VisitorConfig;
import lombok.extern.slf4j.Slf4j;

import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;

import static data.DataConverter.convertToString;
import static model.common.Const.*;

@Slf4j
public class Device {
    public static DeviceModel buildDeviceModelFromApi(Api.DeviceModel apiDeviceModel){
        // Build DeviceModel defined locally(model/common/DeviceModel.java) from the DeviceModel defined in dmi/v1beta1/api.proto
        String name = apiDeviceModel.getName();
        String nameSpace = apiDeviceModel.getNamespace();

        DeviceModel deviceModel = new DeviceModel();

        deviceModel.setId(nameSpace+"/"+name);
        deviceModel.setName(name);
        deviceModel.setNameSpace(nameSpace);

        if (apiDeviceModel.getSpec().getPropertiesList().isEmpty()){
            return deviceModel;
        }
        List<DeviceModel.ModelProperty> modelProperties = new ArrayList<>();

        for (Api.ModelProperty apiModelProperty : apiDeviceModel.getSpec().getPropertiesList()){
            modelProperties.add(new DeviceModel.ModelProperty(
                    apiModelProperty.getName(),
                    apiModelProperty.getDescription(),
                    apiModelProperty.getType(),
                    apiModelProperty.getAccessMode(),
                    apiModelProperty.getMinimum(),
                    apiModelProperty.getMaximum(),
                    apiModelProperty.getUnit())
            );
        }
        deviceModel.setProperties(modelProperties);
        return deviceModel;
    }
    public static DeviceInstance buildDeviceFromApi(Api.Device apiDevice, DeviceModel deviceModel) throws Exception {
        // Build DeviceInstance defined locally(model/common/DeviceInstance.java) from Api
        DeviceInstance deviceInstance = new DeviceInstance();
        deviceInstance.setId(apiDevice.getNamespace()+"/"+apiDevice.getName());
        deviceInstance.setName(apiDevice.getName());
        deviceInstance.setNameSpace(apiDevice.getNamespace());
        deviceInstance.setProtocolName(apiDevice.getSpec().getProtocol().getProtocolName());
        deviceInstance.setModel(apiDevice.getSpec().getDeviceModelReference());
        deviceInstance.setTwins(buildTwinsFromApi(apiDevice));
        deviceInstance.setProperties(buildPropertiesFromApi(apiDevice));


        Map<String, DeviceInstance.DeviceProperty> propertiesMap = new HashMap<>();
        if (deviceModel != null){
            // parse the content of the modelproperty field into DeviceInstance.DeviceProperty
            for (DeviceInstance.DeviceProperty deviceProperty : deviceInstance.getProperties()){
                for (DeviceModel.ModelProperty modelProperty: deviceModel.getProperties()){
                    if (modelProperty.getName().equals(deviceProperty.getPropertyName())){
                        deviceProperty.setModelProperty(modelProperty);
                        break;
                    }
                }
                propertiesMap.put(deviceProperty.getPropertyName(), deviceProperty);
            }
            // copy DeviceProperty to twin
            for (DeviceInstance.Twin twin : deviceInstance.getTwins()){
                twin.setProperty(propertiesMap.get(twin.getPropertyName()));
            }
        }
        return deviceInstance;
    }

    public static List<DeviceInstance.Twin> buildTwinsFromApi(Api.Device apiDevice){
        // build Twins defined in model/common/DeviceInstance.java from Api

        if (apiDevice.getSpec().getPropertiesList().isEmpty()){return null;}
        List<DeviceInstance.Twin> twins = new ArrayList<>();
        for (Api.Twin apiTwin :apiDevice.getStatus().getTwinsList()){

            DeviceInstance.TwinProperty.Metadata metadata = new DeviceInstance.TwinProperty.Metadata();
            metadata.setTimestamp(apiTwin.getObservedDesired().getMetadataMap().get("timestamp"));
            metadata.setType(apiTwin.getObservedDesired().getMetadataMap().get("type"));

            DeviceInstance.TwinProperty twinProperty = new DeviceInstance.TwinProperty();
            twinProperty.setValue(apiTwin.getObservedDesired().getValue());
            twinProperty.setMetadata(metadata);

            DeviceInstance.Twin twin = new DeviceInstance.Twin();
            twin.setPropertyName(apiTwin.getPropertyName());
            twin.setObservedDesired(twinProperty);// Desired value

            twins.add(twin);
        }
        return twins;
    }

    public static List<DeviceInstance.DeviceProperty> buildPropertiesFromApi(Api.Device apiDevice) throws Exception {
        if (apiDevice.getSpec().getPropertiesList().isEmpty()){return null;}
        List<DeviceInstance.DeviceProperty> deviceProperties = new ArrayList<>();

        for (Api.DeviceProperty apiDeviceProperty : apiDevice.getSpec().getPropertiesList()){
            // get visitorConfig field by grpc device instance
            byte[] visitorConfig = null;
            ObjectMapper objectMapper = new ObjectMapper();
            objectMapper.registerModule(new ProtobufModule());
            Map<String, Object> recvAdapter = new HashMap<>();

            for (Map.Entry<String, Any> entry: apiDeviceProperty.getVisitors().getConfigData().getDataMap().entrySet()){
                recvAdapter.put(entry.getKey(), DataConverter.decodeAnyValue(entry.getValue()));
            }

            Map<String,Object> customizedProtocol = new HashMap<>();
            customizedProtocol.put("protocolName", apiDeviceProperty.getVisitors().getProtocolName());
            customizedProtocol.put("configData",recvAdapter);

            try {
                visitorConfig = objectMapper.writeValueAsBytes(customizedProtocol);
            } catch (JsonProcessingException e) {
                log.error("Fail to serialize visitConfig to JSON with error: {}", e.getMessage());
            }

            // get the whole pushmethod field by grpc device instance
            String dbMethodName = "";
            DeviceInstance.DBConfig dbConfig = new DeviceInstance.DBConfig();
            byte[] pushMethod = null;
            String pushMethodName = "";

            if (apiDeviceProperty.getPushMethod()!=null
                    && apiDeviceProperty.getPushMethod().getDbMethod()!=null
                    && !apiDeviceProperty.getPushMethod().getDbMethod().equals("")
            ){
                //parse dbmethod field
                if (apiDeviceProperty.getPushMethod().getDbMethod().getInfluxdb2().getInfluxdb2ClientConfig().getUrl()!=""){
                    dbMethodName = "influx";
                    byte[] clientConfig = objectMapper.writeValueAsBytes(apiDeviceProperty.getPushMethod().getDbMethod().getInfluxdb2().getInfluxdb2ClientConfig());
                    byte[] dataConfig = objectMapper.writeValueAsBytes(apiDeviceProperty.getPushMethod().getDbMethod().getInfluxdb2().getInfluxdb2DataConfig());
                    dbConfig.setInfluxdb2ClientConfig(clientConfig);
                    dbConfig.setInfluxdb2DataConfig(dataConfig);

                }else if (apiDeviceProperty.getPushMethod().getDbMethod().getRedis().getRedisClientConfig().getAddr()!=""){
                    dbMethodName = "redis";
                    byte[] clientConfig = objectMapper.writeValueAsBytes(apiDeviceProperty.getPushMethod().getDbMethod().getRedis().getRedisClientConfig());
                    dbConfig.setRedisClientConfig(clientConfig);

                }else if (apiDeviceProperty.getPushMethod().getDbMethod().getTdengine().getTdEngineClientConfig().getAddr()!=""){
                    dbMethodName = "tdengine";
                    byte[] clientConfig = objectMapper.writeValueAsBytes(apiDeviceProperty.getPushMethod().getDbMethod().getTdengine().getTdEngineClientConfig());
                    dbConfig.setTdEngineClientConfig(clientConfig);

                }else if (apiDeviceProperty.getPushMethod().getDbMethod().getMysql().getMysqlClientConfig().getAddr()!=""){
                    dbMethodName = "mysql";
                    byte[] clientConfig = objectMapper.writeValueAsBytes(apiDeviceProperty.getPushMethod().getDbMethod().getMysql().getMysqlClientConfig());
                    dbConfig.setMysqlClientConfig(clientConfig);
                }else{
                    log.warn("Unsupported database type {} for Property {}",apiDeviceProperty.getPushMethod().getDbMethod() ,apiDeviceProperty.getName());
                }
            }

            // parse pushMethod field
            if (apiDeviceProperty.getPushMethod()!=null){
                if (apiDeviceProperty.getPushMethod().getHttp()!=null){
                    pushMethodName = pushMethodHttp;
                    pushMethod = objectMapper.writeValueAsBytes(apiDeviceProperty.getPushMethod().getHttp());

                }else if (apiDeviceProperty.getPushMethod().getMqtt()!=null){
                    pushMethodName = pushMethodMqtt;
                    pushMethod = objectMapper.writeValueAsBytes(apiDeviceProperty.getPushMethod().getMqtt());

                }else{
                    log.error("Get PushMethod err: Unsupported pushmethod type");
                }
            }
            // get the final properties
            DeviceInstance.DeviceProperty deviceProperty = new DeviceInstance.DeviceProperty();
            deviceProperty.setName(apiDevice.getName());
            deviceProperty.setPropertyName(apiDeviceProperty.getName());
            deviceProperty.setModelName(apiDevice.getSpec().getDeviceModelReference());
            deviceProperty.setReportToCloud(apiDeviceProperty.getReportToCloud());
            deviceProperty.setCollectCycle(apiDeviceProperty.getCollectCycle());
            deviceProperty.setReportCycle(apiDeviceProperty.getReportCycle());
            deviceProperty.setProtocol(apiDevice.getSpec().getProtocol().getProtocolName());
            deviceProperty.setVisitors(visitorConfig);

            DeviceInstance.DBMethodConfig dbMethodConfig = new DeviceInstance.DBMethodConfig();
            dbMethodConfig.setDbMethodName(dbMethodName);
            dbMethodConfig.setDbConfig(dbConfig);

            DeviceInstance.PushMethodConfig pushMethodConfig = new DeviceInstance.PushMethodConfig();
            pushMethodConfig.setMethodName(pushMethodName);
            pushMethodConfig.setMethodConfig(pushMethod);
            pushMethodConfig.setDbMethod(dbMethodConfig);

            deviceProperty.setPushMethod(pushMethodConfig);
            deviceProperties.add(deviceProperty);
        }
        return deviceProperties;
    }

    public static DeviceInstance.ProtocolConfig buildProtocolFromApi(Api.Device apiDevice) throws Exception {
        // Build the Protocol defined locally(src/main/java/model/common/DeviceInstance.ProtocolConfig) from Api
        String protocolName = apiDevice.getSpec().getProtocol().getProtocolName();
        if (protocolName == null){
            throw new IllegalArgumentException("ProtocolName is null, which caused this error.");
        }
        ObjectMapper objectMapper = new ObjectMapper();
        objectMapper.registerModule(new ProtobufModule());

        Map<String, Object> customizedProtocol = new HashMap<>();
        customizedProtocol.put("protocolName",protocolName);

        Map<String, Object> recvAdapter = new HashMap<>();
        for (Map.Entry<String, Any> entry : apiDevice.getSpec().getProtocol().getConfigData().getDataMap().entrySet()) {
            String key = entry.getKey();
            Any value = entry.getValue();
            try {
                Object decodedValue = DataConverter.decodeAnyValue(value);
                recvAdapter.put(key, decodedValue);
            } catch (InvalidProtocolBufferException e) {
                log.error("Error decoding value: {}",value);
            }
        }
        customizedProtocol.put("configData",recvAdapter);

        byte[] configData;

        try {
            configData = objectMapper.writeValueAsBytes(customizedProtocol);
        } catch (JsonProcessingException e) {
            throw new Exception("Serializing protocolConfig to JSON error", e);
        }
        return new DeviceInstance.ProtocolConfig(protocolName,configData);
    }
    public static List<Api.Twin> buildApiTwinsFromLocal(List<DeviceInstance.Twin> twins){
        List<Api.Twin> apiTwins = new ArrayList<>();
        for (DeviceInstance.Twin twin: twins){
            Api.TwinProperty observedDesired = Api.TwinProperty.newBuilder()
                    .setValue(twin.getObservedDesired().getValue())
                    .putMetadata("type", twin.getObservedDesired().getMetadata().getType())
                    .putMetadata("timestamp", twin.getObservedDesired().getMetadata().getTimestamp())
                    .build();

            Api.TwinProperty reported = Api.TwinProperty.newBuilder()
                    .setValue(twin.getReported().getValue())
                    .putMetadata("type", twin.getReported().getMetadata().getType())
                    .putMetadata("timestamp", twin.getReported().getMetadata().getTimestamp())
                    .build();

            Api.Twin apiTwin = Api.Twin.newBuilder().setPropertyName(twin.getPropertyName())
                    .setObservedDesired(observedDesired)
                    .setReported(reported)
                    .build();

            apiTwins.add(apiTwin);
        }
        return apiTwins;
    }

    public static List<ScheduledFuture<?>> dataHandler(CustomizedDev customizedDev){
        // handle data: 1) pushToEdgeCore, 2) save to database and 3) publish data to 3rd app through http or mqtt
        List<ScheduledFuture<?>> futures = new ArrayList<>();
        for (DeviceInstance.Twin twin : customizedDev.getDeviceInstance().getTwins()){
            if (twin.getProperty().getModelProperty().getDataType()==null){
                twin.getProperty().getModelProperty().setDataType("");
            }
            twin.getProperty().getModelProperty().setDataType(twin.getProperty().getModelProperty().getDataType().toLowerCase());

            ObjectMapper objectMapper = new ObjectMapper();
            objectMapper.registerModule(new ProtobufModule());
            VisitorConfig visitorConfig;
            try {
                visitorConfig = objectMapper.readValue(twin.getProperty().getVisitors(),VisitorConfig.class);
            } catch (Exception e) {
                log.error("Deserialize VisitorConfig or setVisitor error: {}", e.getMessage());
                continue;
            }

            try{
                setDeviceData(customizedDev,twin,visitorConfig);
            }catch (Exception e){
                log.error("Set visitor error: {}",e.getMessage(),e);
                continue;
            }

            // If the device property type is streaming, it will directly enter the streaming data processing function,
            // such as saving frames or saving videos, and will no longer push it to the user database and application.
            if (twin.getProperty().getModelProperty().getDataType().equals("stream")){
                StreamHandler.handler(twin, customizedDev.getCustomizedClient(),visitorConfig);
                continue;
            }

            // pushToEdgeCore
            if (twin.getProperty().isReportToCloud()){
                if (twin.getProperty().getCollectCycle()==0){
                    twin.getProperty().setCollectCycle(defaultCollectCycle.toSeconds());
                }

                TwinData twinData = new TwinData();
                twinData.setDeviceNameSpace(customizedDev.getDeviceInstance().getNameSpace());
                twinData.setDeviceName(customizedDev.getDeviceInstance().getName());
                twinData.setClient(customizedDev.getCustomizedClient());
                twinData.setName(twin.getPropertyName());
                twinData.setObservedDesired(twin.getObservedDesired());
                twinData.getObservedDesired().getMetadata().setTimestamp(twin.getObservedDesired().getMetadata().getTimestamp());
                twinData.getObservedDesired().getMetadata().setType(twin.getObservedDesired().getMetadata().getType());
                twinData.setVisitorConfig(visitorConfig);
                twinData.setCollectCycle(twin.getProperty().getCollectCycle());

                twinData.setReported(new DeviceInstance.TwinProperty());
                twinData.getReported().setMetadata(new DeviceInstance.TwinProperty.Metadata());
                twinData.getReported().getMetadata().setType(twinData.getObservedDesired().getMetadata().getType());

                ScheduledExecutorService scheduler_pushToEdgeCore = Executors.newSingleThreadScheduledExecutor();
                ScheduledFuture<?> future_pushToEdgecore = scheduler_pushToEdgeCore.scheduleAtFixedRate(()->{
                    twinData.pushToEdgeCore();
                    twin.setReported(twinData.getReported());
                    log.info("Device status update: {}/{}: {}", customizedDev.getDeviceInstance().getId(), twinData.getName(), twinData.getReported().getValue());
                },0, twinData.getCollectCycle(), TimeUnit.SECONDS);
                futures.add(future_pushToEdgecore);
            }

            // save to database
            if (twin.getProperty().getPushMethod().getMethodConfig() != null && twin.getProperty().getPushMethod().getMethodName()!=""){
                DataModel dataModel = new DataModel();
                dataModel.setDeviceName(customizedDev.getDeviceInstance().getName());
                dataModel.setPropertyName(twin.getProperty().getPropertyName());
                dataModel.setNameSpace(customizedDev.getDeviceInstance().getNameSpace());
                dataModel.setTimeStamp(System.currentTimeMillis());
                dataModel.setType(twin.getObservedDesired().getMetadata().getType());

                dataBaseHandler(twin, customizedDev.getCustomizedClient(), visitorConfig, dataModel, futures);
            }

            // push data to 3rd App
            if (twin.getProperty().getPushMethod().getMethodConfig() != null && twin.getProperty().getPushMethod().getMethodName()!=""){
                DataModel dataModel = new DataModel();
                dataModel.setDeviceName(customizedDev.getDeviceInstance().getName());
                dataModel.setPropertyName(twin.getProperty().getPropertyName());
                dataModel.setNameSpace(customizedDev.getDeviceInstance().getNameSpace());
                dataModel.setTimeStamp(System.currentTimeMillis());
                dataModel.setType(twin.getObservedDesired().getMetadata().getType());

                pushTo3rdAppHandler(twin, customizedDev.getCustomizedClient(),visitorConfig, dataModel,futures);
            }
        }
        return futures;
    }

    public static void pushTo3rdAppHandler(DeviceInstance.Twin twin, CustomizedClient client, VisitorConfig visitorConfig, DataModel dataModel, List<ScheduledFuture<?>> futures){
        // pushHandler start data panel work
        DataPanel dataPanel = new DataPanel();
        switch (twin.getProperty().getPushMethod().getMethodName()){
            case "mqtt":
                dataPanel = Mqtt.newDataPanel(twin.getProperty().getPushMethod().getMethodConfig());
                break;
            case "http":
                dataPanel = Http.newDataPanel(twin.getProperty().getPushMethod().getMethodConfig());
                break;
            default:
                log.error("Custom protocols are not currently supported when push data");
                break;
        }
        dataPanel.initPushMethod();
        Duration reportCycle = Duration.ofSeconds(twin.getProperty().getReportCycle());
        if(reportCycle.isZero()){
            reportCycle = defaultReportCycle;
        }
        ScheduledExecutorService scheduler_push = Executors.newSingleThreadScheduledExecutor();
        DataPanel finalDataPanel = dataPanel;
        ScheduledFuture<?> future_pushHandler = scheduler_push.scheduleAtFixedRate(()->{
            Object devicedata = client.getDeviceData(visitorConfig);
            String sdata = convertToString(devicedata);
            dataModel.setValue(sdata);
            dataModel.setTimeStamp(Instant.now().toEpochMilli());
            finalDataPanel.push(dataModel);
        },0, reportCycle.toSeconds(), TimeUnit.SECONDS);
        futures.add(future_pushHandler);
    }

    public static void dataBaseHandler(DeviceInstance.Twin twin, CustomizedClient client, VisitorConfig visitorConfig, DataModel dataModel , List<ScheduledFuture<?>> futures){
        // dbHandler start db client to save data
        if (twin.getProperty().getPushMethod().getDbMethod().getDbMethodName()!=null){
            switch (twin.getProperty().getPushMethod().getDbMethod().getDbMethodName()){
                case "influx":
                    Influxdb2.dataHandler(twin, client, visitorConfig, dataModel, futures);
                case "redis":
                    Redis.dataHandler(twin,client,visitorConfig,dataModel,futures);
                case "tdengine":
                    Tdengine.dataHandler(twin,client,visitorConfig,dataModel,futures);
                case "mysql":
                    Mysql.dataHandler(twin,client,visitorConfig,dataModel,futures);
            }
        }
    }

    public static void setDeviceData(CustomizedDev customizedDev, DeviceInstance.Twin twin, VisitorConfig visitorConfig){
        // check if visitor property is readonly, if not then set expected value to device.
        if (twin.getProperty().getModelProperty().getAccessMode().equals("ReadOnly")) return;
        Object value = null;
        try {
            value = DataConverter.convert(twin.getProperty().getModelProperty().getDataType(),twin.getObservedDesired().getValue());
        } catch (Exception e) {
            throw new RuntimeException(e);
        }
        if (value == null) return;
        if (customizedDev.getCustomizedClient()==null){
            CustomizedClient customizedClient = new CustomizedClient();
            customizedClient.setDeviceData(value,visitorConfig);
        }else{
            customizedDev.getCustomizedClient().setDeviceData(value,visitorConfig);
        }
    }
}
