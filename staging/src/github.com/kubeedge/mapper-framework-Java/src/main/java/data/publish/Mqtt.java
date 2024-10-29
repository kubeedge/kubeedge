package data.publish;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import model.common.DataModel;
import org.eclipse.paho.client.mqttv3.MqttClient;
import org.eclipse.paho.client.mqttv3.MqttConnectOptions;
import org.eclipse.paho.client.mqttv3.MqttException;
import org.eclipse.paho.client.mqttv3.MqttMessage;

import java.time.Instant;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;
@Slf4j
public class Mqtt {
    public static PushMethod newDataPanel(byte[] mqttConfig){
        ObjectMapper objectMapper = new ObjectMapper();
        MqttConfig config = null;
        try{
            config = objectMapper.readValue(mqttConfig, MqttConfig.class);
        }catch (Exception e){
            log.error("Fail to deserialize mqttConfig with error: {}", e.getMessage(),e);
        }
        PushMethod pushMethod = new PushMethod();
        pushMethod.setMqttpConfig(config);
        return pushMethod;
    }
    @Getter @Setter
    public static class PushMethod extends DataPanel {
        @JsonProperty("http")
        MqttConfig mqttpConfig;
        public void initPushMethod() {
            log.info("Init Mqtt");
        }
        public void push(DataModel data) {
            log.info("Publish {} to {} on topic: {}, Qos: {}, Retained: {}",
                    data.getValue(), this.mqttpConfig.address, this.mqttpConfig.topic, this.mqttpConfig.qos, this.mqttpConfig.retained);
            try{
                MqttClient client = new MqttClient(this.mqttpConfig.address, MqttClient.generateClientId());
                MqttConnectOptions options = new MqttConnectOptions();
                options.setCleanSession(true);
                client.connect(options);

                String formatTimeStr = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss")
                        .withZone(ZoneId.systemDefault())
                        .format(Instant.ofEpochMilli(data.getTimeStamp()));
                String str_time = "time is " + formatTimeStr + "  ";
                String str_publish = str_time + this.mqttpConfig.topic + ": " + data.getValue();
                MqttMessage message = new MqttMessage(str_publish.getBytes());
                message.setQos(this.mqttpConfig.qos);
                message.setRetained(this.mqttpConfig.retained);
                client.publish(this.mqttpConfig.topic, message);

                client.disconnect(250);
                log.info("###############  Message published.  ###############");
            } catch (MqttException e) {
                log.error("Publish device data by MQTT failed, err ={}", e.getMessage(),e);
            }
        }
    }

    @Getter @Setter
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class MqttConfig{
        @JsonProperty("address")
        private String address = "";
        @JsonProperty("topic")
        private String topic = "";
        @JsonProperty("qos")
        private int qos;
        @JsonProperty("retained")
        private boolean retained;
    }
}

