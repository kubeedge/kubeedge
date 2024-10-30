package data.publish;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import model.common.DataModel;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Instant;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;

@Slf4j
public class Http {
    public static PushMethod newDataPanel(byte[] httpConfig){
        ObjectMapper objectMapper = new ObjectMapper();
        HttpConfig config = null;
        try{
            config = objectMapper.readValue(httpConfig,HttpConfig.class);
        }catch (Exception e){
            log.error("Fail to deserialize httpConfig with error: {}", e.getMessage(),e);
        }
        PushMethod pushMethod = new PushMethod();
        pushMethod.setHttpConfig(config);
        return pushMethod;
    }

    @Getter @Setter
    public static class PushMethod extends DataPanel {
        @JsonProperty("http")
        HttpConfig httpConfig;
        public void initPushMethod() {
        }

        public void push(DataModel data) {
            log.info("Publish {} by HTTP", data.getNameSpace()+"/"+data.getDeviceName()+"/"+data.getPropertyName());
            String targetUrl = this.httpConfig.hostName + ":" + this.httpConfig.port + this.httpConfig.requestPath;
            String payload = data.getPropertyName() + "=" + data.getValue();
            Instant instant = Instant.ofEpochMilli(data.getTimeStamp());

            DateTimeFormatter formatter = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss")
                    .withZone(ZoneId.systemDefault());

            String formatTimeStr = formatter.format(instant);


            String currentTime = "&time=" + formatTimeStr;

            payload += currentTime;

            log.info("Publish {} to {}",payload,targetUrl);
            try{
                HttpClient client = HttpClient.newHttpClient();

                HttpRequest request = HttpRequest.newBuilder()
                        .uri(URI.create(targetUrl))
                        .header("Content-Type", "application/x-www-form-urlencoded")
                        .POST(HttpRequest.BodyPublishers.ofString(payload))
                        .build();

                HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());

                if (response.statusCode() != 200) {
                    log.info("Error: " + response.statusCode());
                    return;
                }

                String body = response.body();
                log.info("###############  Message published.  ###############");
                log.info("HTTP received: {}", body);
            } catch (IOException | InterruptedException e) {
                log.error("Publish device data by HTTP failed, err ={}", e.getMessage(),e);
            }
        }
    }

    @Getter @Setter
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class HttpConfig{
        @JsonProperty("hostName")
        private String hostName = "";
        @JsonProperty("port")
        private int port;
        @JsonProperty("requestPath")
        private String requestPath = "";
        @JsonProperty("timeout")
        private int timeOut;
    }
}
