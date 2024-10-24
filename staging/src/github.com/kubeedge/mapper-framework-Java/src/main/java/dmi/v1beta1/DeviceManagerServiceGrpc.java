package dmi.v1beta1;

import static io.grpc.MethodDescriptor.generateFullMethodName;
import static io.grpc.stub.ClientCalls.asyncBidiStreamingCall;
import static io.grpc.stub.ClientCalls.asyncClientStreamingCall;
import static io.grpc.stub.ClientCalls.asyncServerStreamingCall;
import static io.grpc.stub.ClientCalls.asyncUnaryCall;
import static io.grpc.stub.ClientCalls.blockingServerStreamingCall;
import static io.grpc.stub.ClientCalls.blockingUnaryCall;
import static io.grpc.stub.ClientCalls.futureUnaryCall;
import static io.grpc.stub.ServerCalls.asyncBidiStreamingCall;
import static io.grpc.stub.ServerCalls.asyncClientStreamingCall;
import static io.grpc.stub.ServerCalls.asyncServerStreamingCall;
import static io.grpc.stub.ServerCalls.asyncUnaryCall;
import static io.grpc.stub.ServerCalls.asyncUnimplementedStreamingCall;
import static io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall;

/**
 * <pre>
 * DeviceManagerService defines the public APIS for remote device management.
 * The server is implemented by the module of device manager in edgecore
 * and the client is implemented by the device mapper for upstreaming.
 * The mapper should register itself to the device manager when it is online
 * to get the list of devices. And then the mapper can report the device status to the device manager.
 * </pre>
 */
@javax.annotation.Generated(
    value = "by gRPC proto compiler (version 1.26.0)",
    comments = "Source: v1beta1/api.proto")
public final class DeviceManagerServiceGrpc {

  private DeviceManagerServiceGrpc() {}

  public static final String SERVICE_NAME = "v1beta1.DeviceManagerService";

  // Static method descriptors that strictly reflect the proto.
  private static volatile io.grpc.MethodDescriptor<dmi.v1beta1.Api.MapperRegisterRequest,
      dmi.v1beta1.Api.MapperRegisterResponse> getMapperRegisterMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "MapperRegister",
      requestType = dmi.v1beta1.Api.MapperRegisterRequest.class,
      responseType = dmi.v1beta1.Api.MapperRegisterResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<dmi.v1beta1.Api.MapperRegisterRequest,
      dmi.v1beta1.Api.MapperRegisterResponse> getMapperRegisterMethod() {
    io.grpc.MethodDescriptor<dmi.v1beta1.Api.MapperRegisterRequest, dmi.v1beta1.Api.MapperRegisterResponse> getMapperRegisterMethod;
    if ((getMapperRegisterMethod = DeviceManagerServiceGrpc.getMapperRegisterMethod) == null) {
      synchronized (DeviceManagerServiceGrpc.class) {
        if ((getMapperRegisterMethod = DeviceManagerServiceGrpc.getMapperRegisterMethod) == null) {
          DeviceManagerServiceGrpc.getMapperRegisterMethod = getMapperRegisterMethod =
              io.grpc.MethodDescriptor.<dmi.v1beta1.Api.MapperRegisterRequest, dmi.v1beta1.Api.MapperRegisterResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "MapperRegister"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.MapperRegisterRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.MapperRegisterResponse.getDefaultInstance()))
              .setSchemaDescriptor(new DeviceManagerServiceMethodDescriptorSupplier("MapperRegister"))
              .build();
        }
      }
    }
    return getMapperRegisterMethod;
  }

  private static volatile io.grpc.MethodDescriptor<dmi.v1beta1.Api.ReportDeviceStatusRequest,
      dmi.v1beta1.Api.ReportDeviceStatusResponse> getReportDeviceStatusMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ReportDeviceStatus",
      requestType = dmi.v1beta1.Api.ReportDeviceStatusRequest.class,
      responseType = dmi.v1beta1.Api.ReportDeviceStatusResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<dmi.v1beta1.Api.ReportDeviceStatusRequest,
      dmi.v1beta1.Api.ReportDeviceStatusResponse> getReportDeviceStatusMethod() {
    io.grpc.MethodDescriptor<dmi.v1beta1.Api.ReportDeviceStatusRequest, dmi.v1beta1.Api.ReportDeviceStatusResponse> getReportDeviceStatusMethod;
    if ((getReportDeviceStatusMethod = DeviceManagerServiceGrpc.getReportDeviceStatusMethod) == null) {
      synchronized (DeviceManagerServiceGrpc.class) {
        if ((getReportDeviceStatusMethod = DeviceManagerServiceGrpc.getReportDeviceStatusMethod) == null) {
          DeviceManagerServiceGrpc.getReportDeviceStatusMethod = getReportDeviceStatusMethod =
              io.grpc.MethodDescriptor.<dmi.v1beta1.Api.ReportDeviceStatusRequest, dmi.v1beta1.Api.ReportDeviceStatusResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ReportDeviceStatus"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.ReportDeviceStatusRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.ReportDeviceStatusResponse.getDefaultInstance()))
              .setSchemaDescriptor(new DeviceManagerServiceMethodDescriptorSupplier("ReportDeviceStatus"))
              .build();
        }
      }
    }
    return getReportDeviceStatusMethod;
  }

  private static volatile io.grpc.MethodDescriptor<dmi.v1beta1.Api.ReportDeviceStatesRequest,
      dmi.v1beta1.Api.ReportDeviceStatesResponse> getReportDeviceStatesMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ReportDeviceStates",
      requestType = dmi.v1beta1.Api.ReportDeviceStatesRequest.class,
      responseType = dmi.v1beta1.Api.ReportDeviceStatesResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<dmi.v1beta1.Api.ReportDeviceStatesRequest,
      dmi.v1beta1.Api.ReportDeviceStatesResponse> getReportDeviceStatesMethod() {
    io.grpc.MethodDescriptor<dmi.v1beta1.Api.ReportDeviceStatesRequest, dmi.v1beta1.Api.ReportDeviceStatesResponse> getReportDeviceStatesMethod;
    if ((getReportDeviceStatesMethod = DeviceManagerServiceGrpc.getReportDeviceStatesMethod) == null) {
      synchronized (DeviceManagerServiceGrpc.class) {
        if ((getReportDeviceStatesMethod = DeviceManagerServiceGrpc.getReportDeviceStatesMethod) == null) {
          DeviceManagerServiceGrpc.getReportDeviceStatesMethod = getReportDeviceStatesMethod =
              io.grpc.MethodDescriptor.<dmi.v1beta1.Api.ReportDeviceStatesRequest, dmi.v1beta1.Api.ReportDeviceStatesResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ReportDeviceStates"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.ReportDeviceStatesRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.ReportDeviceStatesResponse.getDefaultInstance()))
              .setSchemaDescriptor(new DeviceManagerServiceMethodDescriptorSupplier("ReportDeviceStates"))
              .build();
        }
      }
    }
    return getReportDeviceStatesMethod;
  }

  /**
   * Creates a new async stub that supports all call types for the service
   */
  public static DeviceManagerServiceStub newStub(io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<DeviceManagerServiceStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<DeviceManagerServiceStub>() {
        @java.lang.Override
        public DeviceManagerServiceStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new DeviceManagerServiceStub(channel, callOptions);
        }
      };
    return DeviceManagerServiceStub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports unary and streaming output calls on the service
   */
  public static DeviceManagerServiceBlockingStub newBlockingStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<DeviceManagerServiceBlockingStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<DeviceManagerServiceBlockingStub>() {
        @java.lang.Override
        public DeviceManagerServiceBlockingStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new DeviceManagerServiceBlockingStub(channel, callOptions);
        }
      };
    return DeviceManagerServiceBlockingStub.newStub(factory, channel);
  }

  /**
   * Creates a new ListenableFuture-style stub that supports unary calls on the service
   */
  public static DeviceManagerServiceFutureStub newFutureStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<DeviceManagerServiceFutureStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<DeviceManagerServiceFutureStub>() {
        @java.lang.Override
        public DeviceManagerServiceFutureStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new DeviceManagerServiceFutureStub(channel, callOptions);
        }
      };
    return DeviceManagerServiceFutureStub.newStub(factory, channel);
  }

  /**
   * <pre>
   * DeviceManagerService defines the public APIS for remote device management.
   * The server is implemented by the module of device manager in edgecore
   * and the client is implemented by the device mapper for upstreaming.
   * The mapper should register itself to the device manager when it is online
   * to get the list of devices. And then the mapper can report the device status to the device manager.
   * </pre>
   */
  public static abstract class DeviceManagerServiceImplBase implements io.grpc.BindableService {

    /**
     * <pre>
     * MapperRegister registers the information of the mapper to device manager
     * when the mapper is online. Device manager returns the list of devices and device models which
     * this mapper should manage.
     * </pre>
     */
    public void mapperRegister(dmi.v1beta1.Api.MapperRegisterRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.MapperRegisterResponse> responseObserver) {
      asyncUnimplementedUnaryCall(getMapperRegisterMethod(), responseObserver);
    }

    /**
     * <pre>
     * ReportDeviceStatus reports the status of devices to device manager.
     * When the mapper collects some properties of a device, it can make them a map of device twins
     * and report it to the device manager through the interface of ReportDeviceStatus.
     * </pre>
     */
    public void reportDeviceStatus(dmi.v1beta1.Api.ReportDeviceStatusRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.ReportDeviceStatusResponse> responseObserver) {
      asyncUnimplementedUnaryCall(getReportDeviceStatusMethod(), responseObserver);
    }

    /**
     * <pre>
     * TODO Rename ReportDeviceStatus to ReportDeviceTwins
     * ReportDeviceStates reports the state of devices to device manager.
     * </pre>
     */
    public void reportDeviceStates(dmi.v1beta1.Api.ReportDeviceStatesRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.ReportDeviceStatesResponse> responseObserver) {
      asyncUnimplementedUnaryCall(getReportDeviceStatesMethod(), responseObserver);
    }

    @java.lang.Override public final io.grpc.ServerServiceDefinition bindService() {
      return io.grpc.ServerServiceDefinition.builder(getServiceDescriptor())
          .addMethod(
            getMapperRegisterMethod(),
            asyncUnaryCall(
              new MethodHandlers<
                dmi.v1beta1.Api.MapperRegisterRequest,
                dmi.v1beta1.Api.MapperRegisterResponse>(
                  this, METHODID_MAPPER_REGISTER)))
          .addMethod(
            getReportDeviceStatusMethod(),
            asyncUnaryCall(
              new MethodHandlers<
                dmi.v1beta1.Api.ReportDeviceStatusRequest,
                dmi.v1beta1.Api.ReportDeviceStatusResponse>(
                  this, METHODID_REPORT_DEVICE_STATUS)))
          .addMethod(
            getReportDeviceStatesMethod(),
            asyncUnaryCall(
              new MethodHandlers<
                dmi.v1beta1.Api.ReportDeviceStatesRequest,
                dmi.v1beta1.Api.ReportDeviceStatesResponse>(
                  this, METHODID_REPORT_DEVICE_STATES)))
          .build();
    }
  }

  /**
   * <pre>
   * DeviceManagerService defines the public APIS for remote device management.
   * The server is implemented by the module of device manager in edgecore
   * and the client is implemented by the device mapper for upstreaming.
   * The mapper should register itself to the device manager when it is online
   * to get the list of devices. And then the mapper can report the device status to the device manager.
   * </pre>
   */
  public static final class DeviceManagerServiceStub extends io.grpc.stub.AbstractAsyncStub<DeviceManagerServiceStub> {
    private DeviceManagerServiceStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected DeviceManagerServiceStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new DeviceManagerServiceStub(channel, callOptions);
    }

    /**
     * <pre>
     * MapperRegister registers the information of the mapper to device manager
     * when the mapper is online. Device manager returns the list of devices and device models which
     * this mapper should manage.
     * </pre>
     */
    public void mapperRegister(dmi.v1beta1.Api.MapperRegisterRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.MapperRegisterResponse> responseObserver) {
      asyncUnaryCall(
          getChannel().newCall(getMapperRegisterMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * ReportDeviceStatus reports the status of devices to device manager.
     * When the mapper collects some properties of a device, it can make them a map of device twins
     * and report it to the device manager through the interface of ReportDeviceStatus.
     * </pre>
     */
    public void reportDeviceStatus(dmi.v1beta1.Api.ReportDeviceStatusRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.ReportDeviceStatusResponse> responseObserver) {
      asyncUnaryCall(
          getChannel().newCall(getReportDeviceStatusMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * TODO Rename ReportDeviceStatus to ReportDeviceTwins
     * ReportDeviceStates reports the state of devices to device manager.
     * </pre>
     */
    public void reportDeviceStates(dmi.v1beta1.Api.ReportDeviceStatesRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.ReportDeviceStatesResponse> responseObserver) {
      asyncUnaryCall(
          getChannel().newCall(getReportDeviceStatesMethod(), getCallOptions()), request, responseObserver);
    }
  }

  /**
   * <pre>
   * DeviceManagerService defines the public APIS for remote device management.
   * The server is implemented by the module of device manager in edgecore
   * and the client is implemented by the device mapper for upstreaming.
   * The mapper should register itself to the device manager when it is online
   * to get the list of devices. And then the mapper can report the device status to the device manager.
   * </pre>
   */
  public static final class DeviceManagerServiceBlockingStub extends io.grpc.stub.AbstractBlockingStub<DeviceManagerServiceBlockingStub> {
    private DeviceManagerServiceBlockingStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected DeviceManagerServiceBlockingStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new DeviceManagerServiceBlockingStub(channel, callOptions);
    }

    /**
     * <pre>
     * MapperRegister registers the information of the mapper to device manager
     * when the mapper is online. Device manager returns the list of devices and device models which
     * this mapper should manage.
     * </pre>
     */
    public dmi.v1beta1.Api.MapperRegisterResponse mapperRegister(dmi.v1beta1.Api.MapperRegisterRequest request) {
      return blockingUnaryCall(
          getChannel(), getMapperRegisterMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * ReportDeviceStatus reports the status of devices to device manager.
     * When the mapper collects some properties of a device, it can make them a map of device twins
     * and report it to the device manager through the interface of ReportDeviceStatus.
     * </pre>
     */
    public dmi.v1beta1.Api.ReportDeviceStatusResponse reportDeviceStatus(dmi.v1beta1.Api.ReportDeviceStatusRequest request) {
      return blockingUnaryCall(
          getChannel(), getReportDeviceStatusMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * TODO Rename ReportDeviceStatus to ReportDeviceTwins
     * ReportDeviceStates reports the state of devices to device manager.
     * </pre>
     */
    public dmi.v1beta1.Api.ReportDeviceStatesResponse reportDeviceStates(dmi.v1beta1.Api.ReportDeviceStatesRequest request) {
      return blockingUnaryCall(
          getChannel(), getReportDeviceStatesMethod(), getCallOptions(), request);
    }
  }

  /**
   * <pre>
   * DeviceManagerService defines the public APIS for remote device management.
   * The server is implemented by the module of device manager in edgecore
   * and the client is implemented by the device mapper for upstreaming.
   * The mapper should register itself to the device manager when it is online
   * to get the list of devices. And then the mapper can report the device status to the device manager.
   * </pre>
   */
  public static final class DeviceManagerServiceFutureStub extends io.grpc.stub.AbstractFutureStub<DeviceManagerServiceFutureStub> {
    private DeviceManagerServiceFutureStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected DeviceManagerServiceFutureStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new DeviceManagerServiceFutureStub(channel, callOptions);
    }

    /**
     * <pre>
     * MapperRegister registers the information of the mapper to device manager
     * when the mapper is online. Device manager returns the list of devices and device models which
     * this mapper should manage.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<dmi.v1beta1.Api.MapperRegisterResponse> mapperRegister(
        dmi.v1beta1.Api.MapperRegisterRequest request) {
      return futureUnaryCall(
          getChannel().newCall(getMapperRegisterMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * ReportDeviceStatus reports the status of devices to device manager.
     * When the mapper collects some properties of a device, it can make them a map of device twins
     * and report it to the device manager through the interface of ReportDeviceStatus.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<dmi.v1beta1.Api.ReportDeviceStatusResponse> reportDeviceStatus(
        dmi.v1beta1.Api.ReportDeviceStatusRequest request) {
      return futureUnaryCall(
          getChannel().newCall(getReportDeviceStatusMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * TODO Rename ReportDeviceStatus to ReportDeviceTwins
     * ReportDeviceStates reports the state of devices to device manager.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<dmi.v1beta1.Api.ReportDeviceStatesResponse> reportDeviceStates(
        dmi.v1beta1.Api.ReportDeviceStatesRequest request) {
      return futureUnaryCall(
          getChannel().newCall(getReportDeviceStatesMethod(), getCallOptions()), request);
    }
  }

  private static final int METHODID_MAPPER_REGISTER = 0;
  private static final int METHODID_REPORT_DEVICE_STATUS = 1;
  private static final int METHODID_REPORT_DEVICE_STATES = 2;

  private static final class MethodHandlers<Req, Resp> implements
      io.grpc.stub.ServerCalls.UnaryMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.ServerStreamingMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.ClientStreamingMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.BidiStreamingMethod<Req, Resp> {
    private final DeviceManagerServiceImplBase serviceImpl;
    private final int methodId;

    MethodHandlers(DeviceManagerServiceImplBase serviceImpl, int methodId) {
      this.serviceImpl = serviceImpl;
      this.methodId = methodId;
    }

    @java.lang.Override
    @java.lang.SuppressWarnings("unchecked")
    public void invoke(Req request, io.grpc.stub.StreamObserver<Resp> responseObserver) {
      switch (methodId) {
        case METHODID_MAPPER_REGISTER:
          serviceImpl.mapperRegister((dmi.v1beta1.Api.MapperRegisterRequest) request,
              (io.grpc.stub.StreamObserver<dmi.v1beta1.Api.MapperRegisterResponse>) responseObserver);
          break;
        case METHODID_REPORT_DEVICE_STATUS:
          serviceImpl.reportDeviceStatus((dmi.v1beta1.Api.ReportDeviceStatusRequest) request,
              (io.grpc.stub.StreamObserver<dmi.v1beta1.Api.ReportDeviceStatusResponse>) responseObserver);
          break;
        case METHODID_REPORT_DEVICE_STATES:
          serviceImpl.reportDeviceStates((dmi.v1beta1.Api.ReportDeviceStatesRequest) request,
              (io.grpc.stub.StreamObserver<dmi.v1beta1.Api.ReportDeviceStatesResponse>) responseObserver);
          break;
        default:
          throw new AssertionError();
      }
    }

    @java.lang.Override
    @java.lang.SuppressWarnings("unchecked")
    public io.grpc.stub.StreamObserver<Req> invoke(
        io.grpc.stub.StreamObserver<Resp> responseObserver) {
      switch (methodId) {
        default:
          throw new AssertionError();
      }
    }
  }

  private static abstract class DeviceManagerServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoFileDescriptorSupplier, io.grpc.protobuf.ProtoServiceDescriptorSupplier {
    DeviceManagerServiceBaseDescriptorSupplier() {}

    @java.lang.Override
    public com.google.protobuf.Descriptors.FileDescriptor getFileDescriptor() {
      return dmi.v1beta1.Api.getDescriptor();
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.ServiceDescriptor getServiceDescriptor() {
      return getFileDescriptor().findServiceByName("DeviceManagerService");
    }
  }

  private static final class DeviceManagerServiceFileDescriptorSupplier
      extends DeviceManagerServiceBaseDescriptorSupplier {
    DeviceManagerServiceFileDescriptorSupplier() {}
  }

  private static final class DeviceManagerServiceMethodDescriptorSupplier
      extends DeviceManagerServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoMethodDescriptorSupplier {
    private final String methodName;

    DeviceManagerServiceMethodDescriptorSupplier(String methodName) {
      this.methodName = methodName;
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.MethodDescriptor getMethodDescriptor() {
      return getServiceDescriptor().findMethodByName(methodName);
    }
  }

  private static volatile io.grpc.ServiceDescriptor serviceDescriptor;

  public static io.grpc.ServiceDescriptor getServiceDescriptor() {
    io.grpc.ServiceDescriptor result = serviceDescriptor;
    if (result == null) {
      synchronized (DeviceManagerServiceGrpc.class) {
        result = serviceDescriptor;
        if (result == null) {
          serviceDescriptor = result = io.grpc.ServiceDescriptor.newBuilder(SERVICE_NAME)
              .setSchemaDescriptor(new DeviceManagerServiceFileDescriptorSupplier())
              .addMethod(getMapperRegisterMethod())
              .addMethod(getReportDeviceStatusMethod())
              .addMethod(getReportDeviceStatesMethod())
              .build();
        }
      }
    }
    return result;
  }
}
