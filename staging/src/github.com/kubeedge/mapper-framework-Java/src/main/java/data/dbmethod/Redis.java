package data.dbmethod;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.hubspot.jackson.datatype.protobuf.ProtobufModule;
import driver.CustomizedClient;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import driver.VisitorConfig;
import model.common.DataModel;
import model.common.DeviceInstance;
import redis.clients.jedis.Jedis;
import redis.clients.jedis.JedisPool;
import redis.clients.jedis.JedisPoolConfig;

import java.io.IOException;
import java.sql.SQLException;
import java.time.Duration;
import java.time.Instant;
import java.util.List;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;

import static data.DataConverter.convertToString;
import static model.common.Const.defaultReportCycle;
import static model.common.Const.env_PASSWORD;

@Slf4j
public class Redis {
    public static void dataHandler(DeviceInstance.Twin twin, CustomizedClient client, VisitorConfig visitorConfig, DataModel dataModel, List<ScheduledFuture<?>> futures){
        DataBaseConfig dataBaseConfig = null;
        JedisPool dbClient;
        try {
            dataBaseConfig = newDataBaseClient(twin.getProperty().getPushMethod().getDbMethod().getDbConfig().getRedisClientConfig());
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
                        log.error("Fail to close Redis connection with error: {}",e.getMessage(),e);
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
        dataBaseConfig.setRedisClientConfig(objectMapper.readValue(config, RedisClientConfig.class));
        return dataBaseConfig;
    }
    @Getter
    @Setter
    public static class RedisClientConfig {
        @JsonProperty("addr")
        private String addr;
        @JsonProperty("db")
        private int db;
        @JsonProperty("poolSize")
        private int poolSize;
        @JsonProperty("mineIdleConns")
        private int mineIdleConns;
    }

    @Getter @Setter
    public static class DataBaseConfig{
        @JsonProperty("redisClientConfig")
        private RedisClientConfig redisClientConfig;
        public JedisPool initDbClient(){
            String password = System.getenv(env_PASSWORD);
            JedisPoolConfig poolConfig = new JedisPoolConfig();
            poolConfig.setMaxTotal(this.redisClientConfig.poolSize);
            poolConfig.setMinIdle(this.redisClientConfig.mineIdleConns);
            JedisPool jedisPool = new JedisPool(poolConfig, this.redisClientConfig.addr, this.redisClientConfig.db, 2000, password);
            try (Jedis jedis = jedisPool.getResource()) {
                String pong = jedis.ping();
                log.info("Init Redis database successfully, with return cmd {}", pong);
            } catch (Exception e) {
                log.error("Init Redis database failed with error: {}", e.getMessage(),e);
            }
            return jedisPool;
        }
        public void closeSession(JedisPool jedisPool) throws SQLException {
            jedisPool.close();
        }
        public void addData(DataModel data, JedisPool jedisPool){
            log.info("device name: {}",data.getDeviceName());
            String deviceData = String.format("TimeStamp: %d PropertyName: %s data: %s",
                    data.getTimeStamp(),
                    data.getPropertyName(),
                    data.getValue());
            try(Jedis jedis = jedisPool.getResource()){
                jedis.zadd(data.getDeviceName(), data.getTimeStamp(), deviceData);
            } catch (Exception e) {
                log.error("Exit AddData with error: {}",e.getMessage(),e);
            }
        }
    }
}
