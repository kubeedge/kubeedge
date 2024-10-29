package data.dbmethod;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.hubspot.jackson.datatype.protobuf.ProtobufModule;
import driver.CustomizedClient;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import driver.VisitorConfig;
import model.DataModel;
import model.DeviceInstance;

import java.io.IOException;
import java.sql.*;
import java.time.Duration;
import java.time.Instant;
import java.util.List;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;

import static data.DataConverter.convertToString;
import static model.Const.*;

@Slf4j
public class Tdengine {
    public static void dataHandler(DeviceInstance.Twin twin, CustomizedClient client, VisitorConfig visitorConfig, DataModel dataModel, List<ScheduledFuture<?>> futures){
        DataBaseConfig dataBaseConfig = null;
        Connection dbClient;
        try {
            dataBaseConfig = newDataBaseClient(twin.getProperty().getPushMethod().getDbMethod().getDbConfig().getTdEngineClientConfig());
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
                    try {
                        finalDataBaseConfig.closeSession(dbClient);
                    } catch (SQLException e) {
                        log.error("Fail to close TDEngine connection with error: {}",e.getMessage(),e);
                    }
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
    public static DataBaseConfig newDataBaseClient(byte[] config) throws IOException {
        ObjectMapper objectMapper = new ObjectMapper();
        objectMapper.registerModule(new ProtobufModule());
        DataBaseConfig dataBaseConfig = new DataBaseConfig();
        dataBaseConfig.setTdClientConfig(objectMapper.readValue(config,TDEngineClientConfig.class));
        return dataBaseConfig;
    }
    @Getter
    @Setter
    public static class TDEngineClientConfig {
        @JsonProperty("addr")
        private String addr;
        @JsonProperty("dbName")
        private String dbName;
    }

    @Getter @Setter
    public static class DataBaseConfig{
        @JsonProperty("config")
        private TDEngineClientConfig tdClientConfig;
        public Connection initDbClient(){
            Connection dbClient = null;
            String userName = System.getenv(env_USERNAME);
            String password = System.getenv(env_PASSWORD);
            String url = String.format("jdbc:TAOS-RS://%s/%s", this.tdClientConfig.addr, this.tdClientConfig.dbName);
            try {
                dbClient = DriverManager.getConnection(url, userName, password);
                log.info("init TDEngine database successfully");
            } catch (SQLException e) {
                log.error("init TDEngine db failed, error: {}", e.getMessage(),e);
            }
            return dbClient;
        }
        public void closeSession(Connection dbClient) throws SQLException {
            dbClient.close();
        }
        public void addData(DataModel data, Connection dbClient){
            String legalTable = data.getDeviceName().replace("-", "_");
            String legalTag = data.getPropertyName().replace("-", "_");
            String stableName = String.format("SHOW STABLES LIKE '%s'", legalTable);
            String createStable = String.format(
                    "CREATE STABLE %s (ts timestamp, devicename binary(64), propertyname binary(64), data binary(64), type binary(64)) " +
                            "TAGS (location binary(64));", legalTable);
            String dateTime = Instant.ofEpochMilli(data.getTimeStamp()).toString();
            String insertSQL = String.format(
                    "INSERT INTO %s USING %s TAGS ('%s') VALUES('%s','%s', '%s', '%s', '%s');",
                    legalTag, legalTable, legalTag, dateTime, data.getDeviceName(), data.getPropertyName(), data.getValue(), data.getType());

            try (Statement statement = dbClient.createStatement()) {
                ResultSet rows = statement.executeQuery(stableName);

                if (!rows.next()) {
                    statement.execute(createStable);
                    log.info("Created stable: {}", legalTable);
                }

                statement.execute(insertSQL);
                log.info("Inserted data into TDEngine: {}", insertSQL);
            } catch (SQLException e) {
                log.error("Error while interacting with TDEngine: {}", e.getMessage(), e);
            }
        }
    }
}
