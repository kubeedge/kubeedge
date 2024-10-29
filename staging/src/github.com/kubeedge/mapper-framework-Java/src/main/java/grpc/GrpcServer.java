package grpc;

import devicepanel.DevPanel;
import devicepanel.Device;
import dmi.v1beta1.Api;
import dmi.v1beta1.DeviceManagerServiceGrpc;
import dmi.v1beta1.DeviceMapperServiceGrpc;
import io.grpc.Server;
import io.grpc.netty.NettyServerBuilder;
import io.grpc.stub.StreamObserver;
import io.netty.channel.epoll.EpollEventLoopGroup;
import io.netty.channel.epoll.EpollServerDomainSocketChannel;
import io.netty.channel.unix.DomainSocketAddress;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import model.CustomizedDev;
import model.common.DeviceInstance;
import model.common.DeviceModel;

import java.io.IOException;
import java.util.List;

import static devicepanel.Device.buildApiTwinsFromLocal;

@Slf4j
@Getter @Setter
public class GrpcServer extends DeviceManagerServiceGrpc.DeviceManagerServiceImplBase {
    private Server grpcServer;
    private ConfigGrpcServer cfg;
    private DevPanel devPanel;

    @Getter @Setter
    public static class ConfigGrpcServer{
        private String socketPath;
        private String protocol;

        public ConfigGrpcServer(String socketPath, String protocol) {
            this.socketPath = socketPath;
            this.protocol = protocol;
        }
    }

    public GrpcServer(ConfigGrpcServer cfg, DevPanel devPanel){
        this.cfg = cfg;
        this.devPanel = devPanel;
    }

    public void start() throws IOException {

        grpcServer = NettyServerBuilder.forAddress(new DomainSocketAddress(cfg.getSocketPath()))
                .addService(new DeviceMapperServiceImpl(devPanel))
                .channelType(EpollServerDomainSocketChannel.class)
                .bossEventLoopGroup(new EpollEventLoopGroup(1))
                .workerEventLoopGroup(new EpollEventLoopGroup(1))
                .build()
                .start();

        log.info("Grpc Server start successfully");

        Runtime.getRuntime().addShutdownHook(new Thread(this::stop));

        try {
            grpcServer.awaitTermination();
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
    }

    public void stop(){
        if (grpcServer != null) {
            grpcServer.shutdown();
        }
    }


    @Getter @Setter
    public static class DeviceMapperServiceImpl extends DeviceMapperServiceGrpc.DeviceMapperServiceImplBase{
        private DevPanel devPanel;

        public DeviceMapperServiceImpl(DevPanel devPanel) {
            this.devPanel = devPanel;
        }

        @Override
        public void registerDevice(Api.RegisterDeviceRequest request, StreamObserver<Api.RegisterDeviceResponse> responseObserver) throws Exception {
            // RegisterDevice registers a device to the mapper.
            Api.Device apiDevice = request.getDevice();
            String deviceId = apiDevice.getNamespace()+"/"+apiDevice.getName();

            if (!this.devPanel.getDevices().containsKey(deviceId)){
                String modelId = apiDevice.getNamespace() + "/" + apiDevice.getSpec().getDeviceModelReference();
                DeviceModel deviceModel = devPanel.getModel(modelId);
                DeviceInstance.ProtocolConfig protocolConfig = Device.buildProtocolFromApi(apiDevice);

                DeviceInstance deviceInstance = Device.buildDeviceFromApi(apiDevice,deviceModel);
                deviceInstance.setProtocolConfig(protocolConfig);

                this.devPanel.updateDev(deviceInstance, deviceModel);
            }

            Api.RegisterDeviceResponse registerDeviceResponse = Api.RegisterDeviceResponse.newBuilder()
                    .setDeviceNamespace(apiDevice.getNamespace())
                    .setDeviceName(apiDevice.getName())
                    .build();
            responseObserver.onNext(registerDeviceResponse);
            responseObserver.onCompleted();
        }

        @Override
        public void removeDevice(Api.RemoveDeviceRequest request, StreamObserver<Api.RemoveDeviceResponse> responseObserver) {
            // RemoveDevice unregisters a device to the device mapper.
            String deviceID = request.getDeviceNamespace() + "/" + request.getDeviceName();
            log.info("deviceID is "+deviceID);
            this.devPanel.removeDevice(deviceID);

            Api.RemoveDeviceResponse removeDeviceResponse = Api.RemoveDeviceResponse.newBuilder().build();
            responseObserver.onNext(removeDeviceResponse);
            responseObserver.onCompleted();
        }

        @Override
        public void updateDevice(Api.UpdateDeviceRequest request, StreamObserver<Api.UpdateDeviceResponse> responseObserver) throws Exception {
            // UpdateDevice updates a device to the device mapper
            Api.Device apiDevice = request.getDevice();
            String modelId = apiDevice.getNamespace() + "/" + apiDevice.getSpec().getDeviceModelReference();
            DeviceModel deviceModel = this.devPanel.getModel(modelId);

            DeviceInstance.ProtocolConfig protocolConfig = Device.buildProtocolFromApi(apiDevice);
            DeviceInstance deviceInstance = Device.buildDeviceFromApi(apiDevice,deviceModel);
            deviceInstance.setProtocolConfig(protocolConfig);
            this.devPanel.updateDev(deviceInstance,deviceModel);

            Api.UpdateDeviceResponse updateDeviceResponse = Api.UpdateDeviceResponse.newBuilder().build();
            responseObserver.onNext(updateDeviceResponse);
            responseObserver.onCompleted();
        }

        @Override
        public void createDeviceModel(Api.CreateDeviceModelRequest request, StreamObserver<Api.CreateDeviceModelResponse> responseObserver) {
            // CreateDeviceModel creates a device model to the device mapper.
            Api.DeviceModel apiDeviceModel = request.getModel();
            DeviceModel deviceModel = Device.buildDeviceModelFromApi(apiDeviceModel);
            this.devPanel.updateModel(deviceModel);

            Api.CreateDeviceModelResponse createDeviceModelResponse = Api.CreateDeviceModelResponse.newBuilder()
                    .setDeviceModelNamespace(deviceModel.getNameSpace())
                    .setDeviceModelName(deviceModel.getName())
                    .build();
            responseObserver.onNext(createDeviceModelResponse);
            responseObserver.onCompleted();
        }

        @Override
        public void removeDeviceModel(Api.RemoveDeviceModelRequest request, StreamObserver<Api.RemoveDeviceModelResponse> responseObserver) {
            // RemoveDeviceModel remove a device model to the device mapper.
            String modelId = request.getModelNamespace()+"/"+ request.getModelName();
            this.devPanel.removeModel(modelId);
            Api.RemoveDeviceModelResponse removeDeviceResponse = Api.RemoveDeviceModelResponse.newBuilder().build();

            responseObserver.onNext(removeDeviceResponse);
            responseObserver.onCompleted();
        }

        @Override
        public void updateDeviceModel(Api.UpdateDeviceModelRequest request, StreamObserver<Api.UpdateDeviceModelResponse> responseObserver) {
            // UpdateDeviceModel update a device model to the device mapper.
            Api.DeviceModel apiDeviceModel = request.getModel();
            String modelId = apiDeviceModel.getNamespace()+"/"+apiDeviceModel.getName();
            if (!this.devPanel.getModels().containsKey(modelId)){
                DeviceModel deviceModel = Device.buildDeviceModelFromApi(apiDeviceModel);
                this.devPanel.updateModel(deviceModel);
            }

            Api.UpdateDeviceModelResponse updateDeviceModelResponse = Api.UpdateDeviceModelResponse.newBuilder().build();
            responseObserver.onNext(updateDeviceModelResponse);
            responseObserver.onCompleted();
        }

        @Override
        public void getDevice(Api.GetDeviceRequest request, StreamObserver<Api.GetDeviceResponse> responseObserver) throws IOException {
            // GetDevice get the information of a device from the device mapper.
            String deviceId = request.getDeviceNamespace()+"/"+request.getDeviceName();
            if (this.devPanel.getDevices().containsKey(deviceId)){
                CustomizedDev customizedDev = this.devPanel.getDevice(deviceId);
                List<DeviceInstance.Twin> twins = customizedDev.getDeviceInstance().getTwins();
                List<Api.Twin> apiTwins = buildApiTwinsFromLocal(twins);

                Api.DeviceStatus deviceStatus = Api.DeviceStatus.newBuilder().addAllTwins(apiTwins).build();
                Api.Device apiDevice = Api.Device.newBuilder().setStatus(deviceStatus).build();

                Api.GetDeviceResponse getDeviceResponse = Api.GetDeviceResponse.newBuilder()
                        .setDevice(apiDevice).build();
                responseObserver.onNext(getDeviceResponse);
                responseObserver.onCompleted();
            }
        }
    }
}
