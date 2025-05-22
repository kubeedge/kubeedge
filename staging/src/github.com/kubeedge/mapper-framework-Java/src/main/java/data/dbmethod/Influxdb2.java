package data.dbmethod;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.hubspot.jackson.datatype.protobuf.ProtobufModule;
import com.influxdb.client.InfluxDBClient;
import com.influxdb.client.InfluxDBClientFactory;
import com.influxdb.client.WriteApiBlocking;
import com.influxdb.client.domain.WritePrecision;
import com.influxdb.client.write.Point;
import driver.CustomizedClient;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import driver.VisitorConfig;
import model.common.DataModel;
import model.common.DeviceInstance;

import java.io.IOException;
import java.time.Duration;
import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;

import static data.DataConverter.convertToString;
import static model.common.Const.defaultReportCycle;
import static model.common.Const.env_TOKEN;

@Slf4j
public class Influxdb2 {
    public static void dataHandler(DeviceInstance.Twin twin, CustomizedClient client, VisitorConfig visitorConfig, DataModel dataModel, List<ScheduledFuture<?>> futures){
        DataBaseConfig dataBaseConfig = null;
        InfluxDBClient dbClient;
        try {
            dataBaseConfig = newDataBaseClient(twin.getProperty().getPushMethod().getDbMethod().getDbConfig().getInfluxdb2ClientConfig(),twin.getProperty().getPushMethod().getDbMethod().getDbConfig().getInfluxdb2DataConfig());
        } catch (IOException e) {
            log.error("new database client error: {}",e.getMessage(),e);
        }
        if (dataBaseConfig != null) {
            dbClient = dataBaseConfig.initDbClient();
        } else {
            dbClient = null;
        }
        Duration reportCycle = Duration.ofSeconds(twin.getProperty().getReportCycle());
        if(reportCycle.isZero()){
            reportCycle = defaultReportCycle;
        }
        ScheduledExecutorService scheduler_db = Executors.newSingleThreadScheduledExecutor();
        DataBaseConfig finalDataBaseConfig = dataBaseConfig;
        ScheduledFuture<?> future_dbHandler = scheduler_db.scheduleAtFixedRate(()->{
            if (Thread.currentThread().isInterrupted()){
                if (finalDataBaseConfig != null) {
                    finalDataBaseConfig.closeSession(dbClient);
                }
                return;
            }
            Object deviceData = client.getDeviceData(visitorConfig);
            String sData = convertToString(deviceData);
            dataModel.setValue(sData);
            dataModel.setTimeStamp(Instant.now().toEpochMilli());
            finalDataBaseConfig.addData(dataModel,dbClient);
        },0, reportCycle.toSeconds(), TimeUnit.SECONDS);
        futures.add(future_dbHandler);
    }
    public static DataBaseConfig newDataBaseClient(byte[] clientConfig, byte[] dataConfig) throws IOException {
        ObjectMapper objectMapper = new ObjectMapper();
        objectMapper.registerModule(new ProtobufModule());
        DataBaseConfig dataBaseConfig = new DataBaseConfig();
        dataBaseConfig.setInfluxdb2ClientConfig(objectMapper.readValue(clientConfig, Influxdb2ClientConfig.class));
        dataBaseConfig.setInfluxdb2DataConfig(objectMapper.readValue(dataConfig, Influxdb2DataConfig.class));
        return dataBaseConfig;
    }
    @Getter @Setter
    public static class Influxdb2ClientConfig{
        @JsonProperty("url")
        private String url;
        @JsonProperty("org")
        private String org;
        @JsonProperty("bucket")
        private String bucket;
    }

    @Getter @Setter
    public  static class Influxdb2DataConfig{
        @JsonProperty("measurement")
        private String measurement;
        @JsonProperty("tag")
        private Map<String, String> tag;
        @JsonProperty("fieldKey")
        private String fieldKey;
    }
    @Getter @Setter
    public static class DataBaseConfig{
        @JsonProperty("influxdb2ClientConfig")
        private Influxdb2ClientConfig influxdb2ClientConfig;
        @JsonProperty("influxdb2DataConfig")
        private Influxdb2DataConfig influxdb2DataConfig;
        public InfluxDBClient initDbClient(){
            String usrtoken = System.getenv(env_TOKEN);
            InfluxDBClient client = InfluxDBClientFactory.create(influxdb2ClientConfig.getUrl(), usrtoken.toCharArray());
            return client;
        }
        public void closeSession(InfluxDBClient client){
            client.close();
        }
        public void addData(DataModel data, InfluxDBClient client){
            // Write device data to InfluxDB
            WriteApiBlocking writeApi = client.getWriteApiBlocking();
            // Create a point
            Point point = Point.measurement(influxdb2DataConfig.getMeasurement())
                    .addTags(influxdb2DataConfig.getTag())
                    .addField(influxdb2DataConfig.getFieldKey(), data.getValue())
                    .time(Instant.now(), WritePrecision.NS);
            // Write point immediately
            try {
                writeApi.writePoint(influxdb2ClientConfig.getBucket(), influxdb2ClientConfig.getOrg(), point);
            } catch (Exception e) {
                log.error("Exit AddData: {}", e.getMessage(),e);
            }
        }
    }

}
