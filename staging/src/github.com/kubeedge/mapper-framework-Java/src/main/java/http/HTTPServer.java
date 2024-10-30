package http;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;
import devicepanel.DevPanel;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import model.CustomizedDev;
import model.common.DataModel;
import model.common.HttpResponse;

import java.io.IOException;
import java.io.OutputStream;
import java.net.InetAddress;
import java.net.InetSocketAddress;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

import static model.common.Const.apiVersion;
@Slf4j
public class HTTPServer {

    public static class PingHandler implements HttpHandler {
        // handle /api/v1/ping request
        @Override
        public void handle(HttpExchange exchange) throws IOException {
            HttpResponse.PingResponse response = new HttpResponse.PingResponse();
            response.setApiVersion(apiVersion);
            response.setStatusCode(200);
            response.setTimeStamp(ZonedDateTime.now().format(DateTimeFormatter.ISO_OFFSET_DATE_TIME));
            response.setMessage(String.format("This is %s API, the server is running normally.", apiVersion));

            ObjectMapper objectMapper = new ObjectMapper();
            byte[] jsonResponse = objectMapper.writeValueAsBytes(response);
            exchange.getResponseHeaders().set("Content-Type", "application/json");
            exchange.sendResponseHeaders(response.getStatusCode(), jsonResponse.length);

            OutputStream os = exchange.getResponseBody();
            os.write(jsonResponse);
            os.close();
        }
    }
    public static class DeviceReadHandler implements HttpHandler{
        // handle /api/v1/device/{nameSpace}/{name}/{property} request
        private DevPanel devPanel;
        public DeviceReadHandler(DevPanel devPanel){
            this.devPanel = devPanel;
        }
        @Override
        public void handle(HttpExchange exchange) throws IOException {
            String path = exchange.getRequestURI().getPath();
            String[] segments = path.split("/");

            String nameSpace = segments[segments.length-3];
            String name = segments[segments.length-2];
            String property = segments[segments.length-1];
            String deviceId = nameSpace + "/" + name;

            String[] res = this.devPanel.getTwinResult(deviceId,property);
            String value = res[0];
            String type = res[1];
            String timeStamp = res[2];

            HttpResponse.DeviceReadResponse response = new HttpResponse.DeviceReadResponse();
            response.setApiVersion(apiVersion);
            response.setStatusCode(200);
            response.setTimeStamp(ZonedDateTime.now().format(DateTimeFormatter.ISO_OFFSET_DATE_TIME));

            DataModel dataModel = new DataModel();
            dataModel.setDeviceName(name);
            dataModel.setPropertyName(property);
            dataModel.setNameSpace(nameSpace);
            dataModel.setValue(value);
            dataModel.setType(type);
            dataModel.setTimeStamp(Long.parseLong(timeStamp));

            response.setData(dataModel);

            ObjectMapper objectMapper = new ObjectMapper();
            byte[] jsonResponse = objectMapper.writeValueAsBytes(response);
            exchange.getResponseHeaders().set("Content-Type", "application/json");
            exchange.sendResponseHeaders(response.getStatusCode(), jsonResponse.length);

            OutputStream os = exchange.getResponseBody();
            os.write(jsonResponse);
            os.close();
        }
    }
    public static class MetaGetModelHandler implements HttpHandler{
        // handle /api/v1/meta/model/{nameSpace}/{name} request
        private DevPanel devPanel;
        public MetaGetModelHandler(DevPanel devPanel){
            this.devPanel = devPanel;
        }
        @Override
        public void handle(HttpExchange exchange) throws IOException {
            String path = exchange.getRequestURI().getPath();
            String[] segments = path.split("/");

            String nameSpace = segments[segments.length-2];
            String name = segments[segments.length-1];
            String deviceId = nameSpace + "/" + name;

            CustomizedDev dev = this.devPanel.getDevice(deviceId);

            String modelID = dev.getDeviceInstance().getNameSpace() + "/" + dev.getDeviceInstance().getModel();

            HttpResponse.MetaGetModelResponse response = new HttpResponse.MetaGetModelResponse();
            response.setApiVersion(apiVersion);
            response.setStatusCode(200);
            response.setTimeStamp(ZonedDateTime.now().format(DateTimeFormatter.ISO_OFFSET_DATE_TIME));
            response.setDeviceModel(devPanel.getModel(modelID));

            ObjectMapper objectMapper = new ObjectMapper();
            byte[] jsonResponse = objectMapper.writeValueAsBytes(response);
            exchange.getResponseHeaders().set("Content-Type", "application/json");
            exchange.sendResponseHeaders(response.getStatusCode(), jsonResponse.length);

            OutputStream os = exchange.getResponseBody();
            os.write(jsonResponse);
            os.close();
        }
    }

    public static RestServer newRestServer(DevPanel devPanel, String httpPort){
        if (httpPort == null || httpPort.isEmpty()){
            httpPort = "7777";
        }
        RestServer restServer = new RestServer();
        restServer.setIp("0.0.0.0");
        restServer.setPort(httpPort);
        restServer.setDevPanel(devPanel);
        return restServer;
    }

    @Getter @Setter
    public static class RestServer{
        private String ip;
        private String port;
        private HttpServer server;
        private DevPanel devPanel;
        public void startServer() {
            try {
                this.server = HttpServer.create(new InetSocketAddress(InetAddress.getByName(this.ip), Integer.parseInt(this.port)),0);

                this.server.createContext("/api/v1/ping", new PingHandler());
                this.server.createContext("/api/v1/device/", new DeviceReadHandler(this.devPanel));
                this.server.createContext("/api/v1/meta/model/", new MetaGetModelHandler(this.devPanel));

                ExecutorService executorServices = Executors.newCachedThreadPool();
                this.server.setExecutor(executorServices);
                this.server.start();

                Runtime.getRuntime().addShutdownHook(new Thread(() -> this.server.stop(0)));
            } catch (IOException e) {
                log.error("Create Httpserver error: {}",e.getMessage(),e);
            }
        }
    }
}
