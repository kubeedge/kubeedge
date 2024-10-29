package grpc;

import com.google.protobuf.ByteString;
import config.ConfigParser;
import dmi.v1beta1.Api;
import dmi.v1beta1.DeviceManagerServiceGrpc;
import io.grpc.ManagedChannel;
import io.grpc.StatusRuntimeException;
import io.grpc.netty.NettyChannelBuilder;
import io.netty.channel.EventLoopGroup;
import io.netty.channel.epoll.EpollDomainSocketChannel;
import io.netty.channel.epoll.EpollEventLoopGroup;
import io.netty.channel.unix.DomainSocketAddress;
import lombok.extern.slf4j.Slf4j;
import model.common.Config;

import java.util.concurrent.TimeUnit;

import static model.common.Const.devStOK;

@Slf4j
public class GrpcClient {
    public static Api.MapperRegisterResponse registerMapper(Config cfg, boolean withData) throws Exception {
        // registerMapper register mapper to EdgeCore,then get device and model list from edgecore.
        // if withData is true, edgecore will send device and model list.
        ManagedChannel channel = null;
        Api.MapperRegisterResponse response = null;
        EventLoopGroup group = new EpollEventLoopGroup(1);
        try {
            channel = NettyChannelBuilder.forAddress(new DomainSocketAddress(cfg.getCommon().getEdgeCoreSock()))
                    .eventLoopGroup(group)
                    .channelType(EpollDomainSocketChannel.class)
                    .usePlaintext()
                    .build();

            DeviceManagerServiceGrpc.DeviceManagerServiceBlockingStub blockingStub = DeviceManagerServiceGrpc.newBlockingStub(channel);

            Api.MapperRegisterRequest request = Api.MapperRegisterRequest.newBuilder()
                    .setWithData(withData)
                    .setMapper(Api.MapperInfo.newBuilder()
                            .setName(cfg.getCommon().getName())
                            .setVersion(cfg.getCommon().getVersion())
                            .setApiVersion(cfg.getCommon().getApiVersion())
                            .setProtocol(cfg.getCommon().getProtocol())
                            .setAddress(ByteString.copyFrom(cfg.getCommon().getEdgeCoreSock().getBytes()))
                            .setState(devStOK)
                            .build())
                    .build();

            try {
                response = blockingStub.withDeadlineAfter(3, TimeUnit.SECONDS).mapperRegister(request);
            } catch (StatusRuntimeException e) {
                log.error("Fail to get GRPC response with error: {} while registering mapper to EdgeCore", e.getMessage());
            }
        }finally {
            if (channel != null) {
                channel.shutdown().awaitTermination(1, TimeUnit.SECONDS);
            }
            group.shutdownGracefully().sync();
        }
        return response;
    }

    public static void reportDeviceStatus(Api.ReportDeviceStatusRequest request) throws InterruptedException {
        // reportDeviceStatus report device status to EdgeCore
        Config cfg = ConfigParser.parse();
        ManagedChannel channel = null;
        EventLoopGroup group = new EpollEventLoopGroup(1);
        try {

            channel = NettyChannelBuilder.forAddress(new DomainSocketAddress(cfg.getCommon().getEdgeCoreSock()))
                    .eventLoopGroup(group)
                    .channelType(EpollDomainSocketChannel.class)
                    .usePlaintext()
                    .build();

            DeviceManagerServiceGrpc.DeviceManagerServiceBlockingStub blockingStub = DeviceManagerServiceGrpc.newBlockingStub(channel);
            blockingStub.withDeadlineAfter(1, TimeUnit.SECONDS)
                    .reportDeviceStatus(request);
        }finally {
            if (channel != null) {
                channel.shutdown().awaitTermination(1, TimeUnit.SECONDS);
            }
            group.shutdownGracefully().sync();
        }
    }
}
