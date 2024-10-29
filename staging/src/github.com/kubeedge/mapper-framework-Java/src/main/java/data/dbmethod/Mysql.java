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
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;
import java.util.List;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;

import static data.DataConverter.convertToString;
import static model.Const.defaultReportCycle;
import static model.Const.env_PASSWORD;


@Slf4j
public class Mysql {
    public static void dataHandler(DeviceInstance.Twin twin, CustomizedClient client, VisitorConfig visitorConfig, DataModel dataModel, List<ScheduledFuture<?>> futures){
        DataBaseConfig dataBaseConfig = null;
        Connection dbClient;
        try {
            dataBaseConfig = newDataBaseClient(twin.getProperty().getPushMethod().getDbMethod().getDbConfig().getMysqlClientConfig());
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
                        log.error("Fail to close Mysql connection with error: {}",e.getMessage(),e);
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
        dataBaseConfig.setMysqlClientConfig(objectMapper.readValue(config, MysqlClientConfig.class));
        return dataBaseConfig;
    }
    @Getter @Setter
    public static class MysqlClientConfig {
        @JsonProperty("addr")
        private String addr;
        @JsonProperty("database")
        private String database;
        @JsonProperty("userName")
        private String userName;

        public MysqlClientConfig(String addr, String database, String userName) {
            this.addr = addr;
            this.database = database;
            this.userName = userName;
        }
        public MysqlClientConfig(){}
    }

    @Getter @Setter
    public static class DataBaseConfig{
        @JsonProperty("mysqlClientConfig")
        private MysqlClientConfig mysqlClientConfig;
        public Connection initDbClient(){
            Connection dbClient = null;
            String password = System.getenv(env_PASSWORD);
            String dataSourceName = String.format("jdbc:mysql://%s/%s", this.mysqlClientConfig.addr, this.mysqlClientConfig.database);
            try {
                dbClient = DriverManager.getConnection(dataSourceName, this.mysqlClientConfig.userName, password);
            } catch (SQLException e) {
                log.error("Connection to {} of MySQL failed with error: {}", this.mysqlClientConfig.database, e.getMessage(),e);
            }
            return dbClient;
        }
        public void closeSession(Connection dbClient) throws SQLException {
            dbClient.close();
        }
        public void addData(DataModel data, Connection dbClient){
            String tableName = data.getNameSpace()+"/"+data.getDeviceName()+"/"+data.getPropertyName();
            String dateTime = Instant.ofEpochMilli(data.getTimeStamp())
                    .atZone(ZoneId.systemDefault())
                    .format(DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss"));

            String createTableSQL = String.format("CREATE TABLE IF NOT EXISTS `%s` " +
                    "(id INT AUTO_INCREMENT PRIMARY KEY, ts DATETIME NOT NULL, field TEXT)", tableName);

            try (Statement stmt = dbClient.createStatement()) {
                stmt.execute(createTableSQL);
            } catch (SQLException e) {
                log.error("Create table in MySQL failed: {}" , e.getMessage(), e);
            }

            String insertSQL = String.format("INSERT INTO `%s` (ts, field) VALUES (?, ?)", tableName);
            try (PreparedStatement pstmt = dbClient.prepareStatement(insertSQL)) {
                pstmt.setString(1, dateTime);
                pstmt.setString(2, data.getValue());
                pstmt.executeUpdate();
            } catch (SQLException e) {
                log.error("Insert data into MySQL failed: {}", e.getMessage(), e);
            }
        }
    }
}

