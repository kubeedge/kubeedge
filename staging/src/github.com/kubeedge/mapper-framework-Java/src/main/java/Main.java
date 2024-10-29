import com.fasterxml.jackson.databind.ObjectMapper;
import com.hubspot.jackson.datatype.protobuf.ProtobufModule;
import config.ConfigParser;
import devicepanel.DevPanel;
import dmi.v1beta1.Api;
import grpc.GrpcClient;
import grpc.GrpcServer;
import http.HTTPServer;
import lombok.extern.slf4j.Slf4j;
import model.CustomizedDev;
import model.Config;
import model.DeviceModel;

import java.io.IOException;
import java.util.List;

import static http.HTTPServer.newRestServer;

@Slf4j
public class Main {
    public static void main(String[] args) throws Exception {
        // Parse configuration file: src/main/resources/config.yaml
        Config cfg = ConfigParser.parse();
        ObjectMapper objectMapper = new ObjectMapper();
        objectMapper.registerModule(new ProtobufModule());
//        objectMapper.enable(SerializationFeature.INDENT_OUTPUT);

        log.info("Config: {}",objectMapper.writeValueAsString(cfg));
        log.info("Mapper will register to EdgeCore");

        // Register mapper to EdgeCore,then get device and model list from EdgeCore
        Api.MapperRegisterResponse mapperRegisterResponse = GrpcClient.registerMapper(cfg, true);

        List<Api.Device> deviceList = mapperRegisterResponse.getDeviceListList();
        List<Api.DeviceModel> deviceModelList = mapperRegisterResponse.getModelListList();
        log.info("Mapper register successfully, {} Api devices and {} Api models are received", deviceList.size(),deviceModelList.size());

        // List Devices and models received from EdgeCore
        log.info("Api Devices are listed as follows:");
        for (Api.Device device: deviceList){
            log.info("{}",objectMapper.writeValueAsString(device));
        }
        log.info("Api Models are listed as follows:");
        for (Api.DeviceModel model: deviceModelList){
            log.info("{}",objectMapper.writeValueAsString(model));
        }

        // Init and start the devPanel
        DevPanel devPanel = new DevPanel();
        devPanel.devInit(deviceList, deviceModelList);
        log.info("DevPanel initialized successfully, {} CustomizedDevices and {} DeviceModels are built locally"
                ,devPanel.getDevices().size(),devPanel.getModels().size());
        log.info("Local CustomizedDev are listed as follows:");
        for (CustomizedDev customizedDev: devPanel.getDevices().values()){
            log.info("{}",objectMapper.writeValueAsString(customizedDev));
        }
        log.info("Local DeviceModels are listed as follows:");
        for (DeviceModel model: devPanel.getModels().values()){
            log.info("{}",objectMapper.writeValueAsString(model));
        }
        new Thread(devPanel::devStart).start();
        log.info("Devices started successfully");

        // Create and Start grpcServer which implements DeviceMapperService defined in dmi/v1beta1/api.proto
        GrpcServer grpcServer = new GrpcServer(
                new GrpcServer.ConfigGrpcServer(cfg.getGrpcServer().getSocketPath(),"customized-protocol"),
                devPanel);

        new Thread(()-> {
            try {
                grpcServer.start();
            } catch (IOException e) {
                throw new RuntimeException(e);
            }
        }).start();

        HTTPServer.RestServer httpServer = newRestServer(devPanel, cfg.getCommon().getHttpPort());
        new Thread(httpServer::startServer).start();
    }
}
