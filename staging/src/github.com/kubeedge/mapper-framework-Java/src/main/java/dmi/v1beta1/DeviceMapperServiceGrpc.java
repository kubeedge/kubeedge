package dmi.v1beta1;

import java.io.IOException;

import static io.grpc.MethodDescriptor.generateFullMethodName;
import static io.grpc.stub.ClientCalls.asyncUnaryCall;
import static io.grpc.stub.ClientCalls.blockingUnaryCall;
import static io.grpc.stub.ClientCalls.futureUnaryCall;
import static io.grpc.stub.ServerCalls.asyncUnaryCall;
import static io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall;

/**
 * <pre>
 * DeviceMapperService defines the public APIS for remote device management.
 * The server is implemented by the device mapper
 * and the client is implemented by the module of device manager in edgecore for downstreaming.
 * The device manager can manage the device life cycle through these interfaces provided by DeviceMapperService.
 * When device manager gets a message of device management from cloudcore, it should call the corresponding grpc interface
 * to make the mapper maintain the list of device information.
 * </pre>
 */
@javax.annotation.Generated(
    value = "by gRPC proto compiler (version 1.26.0)",
    comments = "Source: v1beta1/api.proto")
public final class DeviceMapperServiceGrpc {

  private DeviceMapperServiceGrpc() {}

  public static final String SERVICE_NAME = "v1beta1.DeviceMapperService";

  // Static method descriptors that strictly reflect the proto.
  private static volatile io.grpc.MethodDescriptor<dmi.v1beta1.Api.RegisterDeviceRequest,
      dmi.v1beta1.Api.RegisterDeviceResponse> getRegisterDeviceMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "RegisterDevice",
      requestType = dmi.v1beta1.Api.RegisterDeviceRequest.class,
      responseType = dmi.v1beta1.Api.RegisterDeviceResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<dmi.v1beta1.Api.RegisterDeviceRequest,
      dmi.v1beta1.Api.RegisterDeviceResponse> getRegisterDeviceMethod() {
    io.grpc.MethodDescriptor<dmi.v1beta1.Api.RegisterDeviceRequest, dmi.v1beta1.Api.RegisterDeviceResponse> getRegisterDeviceMethod;
    if ((getRegisterDeviceMethod = DeviceMapperServiceGrpc.getRegisterDeviceMethod) == null) {
      synchronized (DeviceMapperServiceGrpc.class) {
        if ((getRegisterDeviceMethod = DeviceMapperServiceGrpc.getRegisterDeviceMethod) == null) {
          DeviceMapperServiceGrpc.getRegisterDeviceMethod = getRegisterDeviceMethod =
              io.grpc.MethodDescriptor.<dmi.v1beta1.Api.RegisterDeviceRequest, dmi.v1beta1.Api.RegisterDeviceResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "RegisterDevice"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.RegisterDeviceRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.RegisterDeviceResponse.getDefaultInstance()))
              .setSchemaDescriptor(new DeviceMapperServiceMethodDescriptorSupplier("RegisterDevice"))
              .build();
        }
      }
    }
    return getRegisterDeviceMethod;
  }

  private static volatile io.grpc.MethodDescriptor<dmi.v1beta1.Api.RemoveDeviceRequest,
      dmi.v1beta1.Api.RemoveDeviceResponse> getRemoveDeviceMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "RemoveDevice",
      requestType = dmi.v1beta1.Api.RemoveDeviceRequest.class,
      responseType = dmi.v1beta1.Api.RemoveDeviceResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<dmi.v1beta1.Api.RemoveDeviceRequest,
      dmi.v1beta1.Api.RemoveDeviceResponse> getRemoveDeviceMethod() {
    io.grpc.MethodDescriptor<dmi.v1beta1.Api.RemoveDeviceRequest, dmi.v1beta1.Api.RemoveDeviceResponse> getRemoveDeviceMethod;
    if ((getRemoveDeviceMethod = DeviceMapperServiceGrpc.getRemoveDeviceMethod) == null) {
      synchronized (DeviceMapperServiceGrpc.class) {
        if ((getRemoveDeviceMethod = DeviceMapperServiceGrpc.getRemoveDeviceMethod) == null) {
          DeviceMapperServiceGrpc.getRemoveDeviceMethod = getRemoveDeviceMethod =
              io.grpc.MethodDescriptor.<dmi.v1beta1.Api.RemoveDeviceRequest, dmi.v1beta1.Api.RemoveDeviceResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "RemoveDevice"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.RemoveDeviceRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.RemoveDeviceResponse.getDefaultInstance()))
              .setSchemaDescriptor(new DeviceMapperServiceMethodDescriptorSupplier("RemoveDevice"))
              .build();
        }
      }
    }
    return getRemoveDeviceMethod;
  }

  private static volatile io.grpc.MethodDescriptor<dmi.v1beta1.Api.UpdateDeviceRequest,
      dmi.v1beta1.Api.UpdateDeviceResponse> getUpdateDeviceMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "UpdateDevice",
      requestType = dmi.v1beta1.Api.UpdateDeviceRequest.class,
      responseType = dmi.v1beta1.Api.UpdateDeviceResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<dmi.v1beta1.Api.UpdateDeviceRequest,
      dmi.v1beta1.Api.UpdateDeviceResponse> getUpdateDeviceMethod() {
    io.grpc.MethodDescriptor<dmi.v1beta1.Api.UpdateDeviceRequest, dmi.v1beta1.Api.UpdateDeviceResponse> getUpdateDeviceMethod;
    if ((getUpdateDeviceMethod = DeviceMapperServiceGrpc.getUpdateDeviceMethod) == null) {
      synchronized (DeviceMapperServiceGrpc.class) {
        if ((getUpdateDeviceMethod = DeviceMapperServiceGrpc.getUpdateDeviceMethod) == null) {
          DeviceMapperServiceGrpc.getUpdateDeviceMethod = getUpdateDeviceMethod =
              io.grpc.MethodDescriptor.<dmi.v1beta1.Api.UpdateDeviceRequest, dmi.v1beta1.Api.UpdateDeviceResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "UpdateDevice"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.UpdateDeviceRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.UpdateDeviceResponse.getDefaultInstance()))
              .setSchemaDescriptor(new DeviceMapperServiceMethodDescriptorSupplier("UpdateDevice"))
              .build();
        }
      }
    }
    return getUpdateDeviceMethod;
  }

  private static volatile io.grpc.MethodDescriptor<dmi.v1beta1.Api.CreateDeviceModelRequest,
      dmi.v1beta1.Api.CreateDeviceModelResponse> getCreateDeviceModelMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "CreateDeviceModel",
      requestType = dmi.v1beta1.Api.CreateDeviceModelRequest.class,
      responseType = dmi.v1beta1.Api.CreateDeviceModelResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<dmi.v1beta1.Api.CreateDeviceModelRequest,
      dmi.v1beta1.Api.CreateDeviceModelResponse> getCreateDeviceModelMethod() {
    io.grpc.MethodDescriptor<dmi.v1beta1.Api.CreateDeviceModelRequest, dmi.v1beta1.Api.CreateDeviceModelResponse> getCreateDeviceModelMethod;
    if ((getCreateDeviceModelMethod = DeviceMapperServiceGrpc.getCreateDeviceModelMethod) == null) {
      synchronized (DeviceMapperServiceGrpc.class) {
        if ((getCreateDeviceModelMethod = DeviceMapperServiceGrpc.getCreateDeviceModelMethod) == null) {
          DeviceMapperServiceGrpc.getCreateDeviceModelMethod = getCreateDeviceModelMethod =
              io.grpc.MethodDescriptor.<dmi.v1beta1.Api.CreateDeviceModelRequest, dmi.v1beta1.Api.CreateDeviceModelResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "CreateDeviceModel"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.CreateDeviceModelRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.CreateDeviceModelResponse.getDefaultInstance()))
              .setSchemaDescriptor(new DeviceMapperServiceMethodDescriptorSupplier("CreateDeviceModel"))
              .build();
        }
      }
    }
    return getCreateDeviceModelMethod;
  }

  private static volatile io.grpc.MethodDescriptor<dmi.v1beta1.Api.RemoveDeviceModelRequest,
      dmi.v1beta1.Api.RemoveDeviceModelResponse> getRemoveDeviceModelMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "RemoveDeviceModel",
      requestType = dmi.v1beta1.Api.RemoveDeviceModelRequest.class,
      responseType = dmi.v1beta1.Api.RemoveDeviceModelResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<dmi.v1beta1.Api.RemoveDeviceModelRequest,
      dmi.v1beta1.Api.RemoveDeviceModelResponse> getRemoveDeviceModelMethod() {
    io.grpc.MethodDescriptor<dmi.v1beta1.Api.RemoveDeviceModelRequest, dmi.v1beta1.Api.RemoveDeviceModelResponse> getRemoveDeviceModelMethod;
    if ((getRemoveDeviceModelMethod = DeviceMapperServiceGrpc.getRemoveDeviceModelMethod) == null) {
      synchronized (DeviceMapperServiceGrpc.class) {
        if ((getRemoveDeviceModelMethod = DeviceMapperServiceGrpc.getRemoveDeviceModelMethod) == null) {
          DeviceMapperServiceGrpc.getRemoveDeviceModelMethod = getRemoveDeviceModelMethod =
              io.grpc.MethodDescriptor.<dmi.v1beta1.Api.RemoveDeviceModelRequest, dmi.v1beta1.Api.RemoveDeviceModelResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "RemoveDeviceModel"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.RemoveDeviceModelRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.RemoveDeviceModelResponse.getDefaultInstance()))
              .setSchemaDescriptor(new DeviceMapperServiceMethodDescriptorSupplier("RemoveDeviceModel"))
              .build();
        }
      }
    }
    return getRemoveDeviceModelMethod;
  }

  private static volatile io.grpc.MethodDescriptor<dmi.v1beta1.Api.UpdateDeviceModelRequest,
      dmi.v1beta1.Api.UpdateDeviceModelResponse> getUpdateDeviceModelMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "UpdateDeviceModel",
      requestType = dmi.v1beta1.Api.UpdateDeviceModelRequest.class,
      responseType = dmi.v1beta1.Api.UpdateDeviceModelResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<dmi.v1beta1.Api.UpdateDeviceModelRequest,
      dmi.v1beta1.Api.UpdateDeviceModelResponse> getUpdateDeviceModelMethod() {
    io.grpc.MethodDescriptor<dmi.v1beta1.Api.UpdateDeviceModelRequest, dmi.v1beta1.Api.UpdateDeviceModelResponse> getUpdateDeviceModelMethod;
    if ((getUpdateDeviceModelMethod = DeviceMapperServiceGrpc.getUpdateDeviceModelMethod) == null) {
      synchronized (DeviceMapperServiceGrpc.class) {
        if ((getUpdateDeviceModelMethod = DeviceMapperServiceGrpc.getUpdateDeviceModelMethod) == null) {
          DeviceMapperServiceGrpc.getUpdateDeviceModelMethod = getUpdateDeviceModelMethod =
              io.grpc.MethodDescriptor.<dmi.v1beta1.Api.UpdateDeviceModelRequest, dmi.v1beta1.Api.UpdateDeviceModelResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "UpdateDeviceModel"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.UpdateDeviceModelRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.UpdateDeviceModelResponse.getDefaultInstance()))
              .setSchemaDescriptor(new DeviceMapperServiceMethodDescriptorSupplier("UpdateDeviceModel"))
              .build();
        }
      }
    }
    return getUpdateDeviceModelMethod;
  }

  private static volatile io.grpc.MethodDescriptor<dmi.v1beta1.Api.GetDeviceRequest,
      dmi.v1beta1.Api.GetDeviceResponse> getGetDeviceMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetDevice",
      requestType = dmi.v1beta1.Api.GetDeviceRequest.class,
      responseType = dmi.v1beta1.Api.GetDeviceResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<dmi.v1beta1.Api.GetDeviceRequest,
      dmi.v1beta1.Api.GetDeviceResponse> getGetDeviceMethod() {
    io.grpc.MethodDescriptor<dmi.v1beta1.Api.GetDeviceRequest, dmi.v1beta1.Api.GetDeviceResponse> getGetDeviceMethod;
    if ((getGetDeviceMethod = DeviceMapperServiceGrpc.getGetDeviceMethod) == null) {
      synchronized (DeviceMapperServiceGrpc.class) {
        if ((getGetDeviceMethod = DeviceMapperServiceGrpc.getGetDeviceMethod) == null) {
          DeviceMapperServiceGrpc.getGetDeviceMethod = getGetDeviceMethod =
              io.grpc.MethodDescriptor.<dmi.v1beta1.Api.GetDeviceRequest, dmi.v1beta1.Api.GetDeviceResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetDevice"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.GetDeviceRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  dmi.v1beta1.Api.GetDeviceResponse.getDefaultInstance()))
              .setSchemaDescriptor(new DeviceMapperServiceMethodDescriptorSupplier("GetDevice"))
              .build();
        }
      }
    }
    return getGetDeviceMethod;
  }

  /**
   * Creates a new async stub that supports all call types for the service
   */
  public static DeviceMapperServiceStub newStub(io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<DeviceMapperServiceStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<DeviceMapperServiceStub>() {
        @java.lang.Override
        public DeviceMapperServiceStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new DeviceMapperServiceStub(channel, callOptions);
        }
      };
    return DeviceMapperServiceStub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports unary and streaming output calls on the service
   */
  public static DeviceMapperServiceBlockingStub newBlockingStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<DeviceMapperServiceBlockingStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<DeviceMapperServiceBlockingStub>() {
        @java.lang.Override
        public DeviceMapperServiceBlockingStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new DeviceMapperServiceBlockingStub(channel, callOptions);
        }
      };
    return DeviceMapperServiceBlockingStub.newStub(factory, channel);
  }

  /**
   * Creates a new ListenableFuture-style stub that supports unary calls on the service
   */
  public static DeviceMapperServiceFutureStub newFutureStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<DeviceMapperServiceFutureStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<DeviceMapperServiceFutureStub>() {
        @java.lang.Override
        public DeviceMapperServiceFutureStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new DeviceMapperServiceFutureStub(channel, callOptions);
        }
      };
    return DeviceMapperServiceFutureStub.newStub(factory, channel);
  }

  /**
   * <pre>
   * DeviceMapperService defines the public APIS for remote device management.
   * The server is implemented by the device mapper
   * and the client is implemented by the module of device manager in edgecore for downstreaming.
   * The device manager can manage the device life cycle through these interfaces provided by DeviceMapperService.
   * When device manager gets a message of device management from cloudcore, it should call the corresponding grpc interface
   * to make the mapper maintain the list of device information.
   * </pre>
   */
  public static abstract class DeviceMapperServiceImplBase implements io.grpc.BindableService {

    /**
     * <pre>
     * RegisterDevice registers a device to the device mapper.
     * Device manager registers a device instance with the information of device
     * to the mapper through the interface of RegisterDevice.
     * When the mapper gets the request of register with device information,
     * it should add the device to the device list and connect to the real physical device via the specific protocol.
     * </pre>
     */
    public void registerDevice(dmi.v1beta1.Api.RegisterDeviceRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.RegisterDeviceResponse> responseObserver) throws Exception {
      asyncUnimplementedUnaryCall(getRegisterDeviceMethod(), responseObserver);
    }

    /**
     * <pre>
     * RemoveDevice unregisters a device to the device mapper.
     * Device manager unregisters a device instance with the name of device
     * to the mapper through the interface of RemoveDevice.
     * When the mapper gets the request of unregister with device name,
     * it should remove the device from the device list and disconnect to the real physical device.
     * </pre>
     */
    public void removeDevice(dmi.v1beta1.Api.RemoveDeviceRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.RemoveDeviceResponse> responseObserver) {
      asyncUnimplementedUnaryCall(getRemoveDeviceMethod(), responseObserver);
    }

    /**
     * <pre>
     * UpdateDevice updates a device to the device mapper
     * Device manager updates the information of a device used by the mapper
     * through the interface of UpdateDevice.
     * The information of a device includes the meta data and the status data of a device.
     * When the mapper gets the request of updating with the information of a device,
     * it should update the device of the device list and connect to the real physical device via the updated information.
     * </pre>
     */
    public void updateDevice(dmi.v1beta1.Api.UpdateDeviceRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.UpdateDeviceResponse> responseObserver) throws Exception {
      asyncUnimplementedUnaryCall(getUpdateDeviceMethod(), responseObserver);
    }

    /**
     * <pre>
     * CreateDeviceModel creates a device model to the device mapper.
     * Device manager sends the information of device model to the mapper
     * through the interface of CreateDeviceModel.
     * When the mapper gets the request of creating with the information of device model,
     * it should create a new device model to the list of device models.
     * </pre>
     */
    public void createDeviceModel(dmi.v1beta1.Api.CreateDeviceModelRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.CreateDeviceModelResponse> responseObserver) {
      asyncUnimplementedUnaryCall(getCreateDeviceModelMethod(), responseObserver);
    }

    /**
     * <pre>
     * RemoveDeviceModel remove a device model to the device mapper.
     * Device manager sends the name of device model to the mapper
     * through the interface of RemoveDeviceModel.
     * When the mapper gets the request of removing with the name of device model,
     * it should remove the device model to the list of device models.
     * </pre>
     */
    public void removeDeviceModel(dmi.v1beta1.Api.RemoveDeviceModelRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.RemoveDeviceModelResponse> responseObserver) {
      asyncUnimplementedUnaryCall(getRemoveDeviceModelMethod(), responseObserver);
    }

    /**
     * <pre>
     * UpdateDeviceModel update a device model to the device mapper.
     * Device manager sends the information of device model to the mapper
     * through the interface of UpdateDeviceModel.
     * When the mapper gets the request of updating with the information of device model,
     * it should update the device model to the list of device models.
     * </pre>
     */
    public void updateDeviceModel(dmi.v1beta1.Api.UpdateDeviceModelRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.UpdateDeviceModelResponse> responseObserver) {
      asyncUnimplementedUnaryCall(getUpdateDeviceModelMethod(), responseObserver);
    }

    /**
     * <pre>
     * GetDevice get the information of a device from the device mapper.
     * Device sends the request of querying device information with the device name to the mapper
     * through the interface of GetDevice.
     * When the mapper gets the request of querying with the device name,
     * it should return the device information.
     * </pre>
     */
    public void getDevice(dmi.v1beta1.Api.GetDeviceRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.GetDeviceResponse> responseObserver) throws IOException {
      asyncUnimplementedUnaryCall(getGetDeviceMethod(), responseObserver);
    }

    @java.lang.Override public final io.grpc.ServerServiceDefinition bindService() {
      return io.grpc.ServerServiceDefinition.builder(getServiceDescriptor())
          .addMethod(
            getRegisterDeviceMethod(),
            asyncUnaryCall(
              new MethodHandlers<
                dmi.v1beta1.Api.RegisterDeviceRequest,
                dmi.v1beta1.Api.RegisterDeviceResponse>(
                  this, METHODID_REGISTER_DEVICE)))
          .addMethod(
            getRemoveDeviceMethod(),
            asyncUnaryCall(
              new MethodHandlers<
                dmi.v1beta1.Api.RemoveDeviceRequest,
                dmi.v1beta1.Api.RemoveDeviceResponse>(
                  this, METHODID_REMOVE_DEVICE)))
          .addMethod(
            getUpdateDeviceMethod(),
            asyncUnaryCall(
              new MethodHandlers<
                dmi.v1beta1.Api.UpdateDeviceRequest,
                dmi.v1beta1.Api.UpdateDeviceResponse>(
                  this, METHODID_UPDATE_DEVICE)))
          .addMethod(
            getCreateDeviceModelMethod(),
            asyncUnaryCall(
              new MethodHandlers<
                dmi.v1beta1.Api.CreateDeviceModelRequest,
                dmi.v1beta1.Api.CreateDeviceModelResponse>(
                  this, METHODID_CREATE_DEVICE_MODEL)))
          .addMethod(
            getRemoveDeviceModelMethod(),
            asyncUnaryCall(
              new MethodHandlers<
                dmi.v1beta1.Api.RemoveDeviceModelRequest,
                dmi.v1beta1.Api.RemoveDeviceModelResponse>(
                  this, METHODID_REMOVE_DEVICE_MODEL)))
          .addMethod(
            getUpdateDeviceModelMethod(),
            asyncUnaryCall(
              new MethodHandlers<
                dmi.v1beta1.Api.UpdateDeviceModelRequest,
                dmi.v1beta1.Api.UpdateDeviceModelResponse>(
                  this, METHODID_UPDATE_DEVICE_MODEL)))
          .addMethod(
            getGetDeviceMethod(),
            asyncUnaryCall(
              new MethodHandlers<
                dmi.v1beta1.Api.GetDeviceRequest,
                dmi.v1beta1.Api.GetDeviceResponse>(
                  this, METHODID_GET_DEVICE)))
          .build();
    }
  }

  /**
   * <pre>
   * DeviceMapperService defines the public APIS for remote device management.
   * The server is implemented by the device mapper
   * and the client is implemented by the module of device manager in edgecore for downstreaming.
   * The device manager can manage the device life cycle through these interfaces provided by DeviceMapperService.
   * When device manager gets a message of device management from cloudcore, it should call the corresponding grpc interface
   * to make the mapper maintain the list of device information.
   * </pre>
   */
  public static final class DeviceMapperServiceStub extends io.grpc.stub.AbstractAsyncStub<DeviceMapperServiceStub> {
    private DeviceMapperServiceStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected DeviceMapperServiceStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new DeviceMapperServiceStub(channel, callOptions);
    }

    /**
     * <pre>
     * RegisterDevice registers a device to the device mapper.
     * Device manager registers a device instance with the information of device
     * to the mapper through the interface of RegisterDevice.
     * When the mapper gets the request of register with device information,
     * it should add the device to the device list and connect to the real physical device via the specific protocol.
     * </pre>
     */
    public void registerDevice(dmi.v1beta1.Api.RegisterDeviceRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.RegisterDeviceResponse> responseObserver) {
      asyncUnaryCall(
          getChannel().newCall(getRegisterDeviceMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * RemoveDevice unregisters a device to the device mapper.
     * Device manager unregisters a device instance with the name of device
     * to the mapper through the interface of RemoveDevice.
     * When the mapper gets the request of unregister with device name,
     * it should remove the device from the device list and disconnect to the real physical device.
     * </pre>
     */
    public void removeDevice(dmi.v1beta1.Api.RemoveDeviceRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.RemoveDeviceResponse> responseObserver) {
      asyncUnaryCall(
          getChannel().newCall(getRemoveDeviceMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * UpdateDevice updates a device to the device mapper
     * Device manager updates the information of a device used by the mapper
     * through the interface of UpdateDevice.
     * The information of a device includes the meta data and the status data of a device.
     * When the mapper gets the request of updating with the information of a device,
     * it should update the device of the device list and connect to the real physical device via the updated information.
     * </pre>
     */
    public void updateDevice(dmi.v1beta1.Api.UpdateDeviceRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.UpdateDeviceResponse> responseObserver) {
      asyncUnaryCall(
          getChannel().newCall(getUpdateDeviceMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * CreateDeviceModel creates a device model to the device mapper.
     * Device manager sends the information of device model to the mapper
     * through the interface of CreateDeviceModel.
     * When the mapper gets the request of creating with the information of device model,
     * it should create a new device model to the list of device models.
     * </pre>
     */
    public void createDeviceModel(dmi.v1beta1.Api.CreateDeviceModelRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.CreateDeviceModelResponse> responseObserver) {
      asyncUnaryCall(
          getChannel().newCall(getCreateDeviceModelMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * RemoveDeviceModel remove a device model to the device mapper.
     * Device manager sends the name of device model to the mapper
     * through the interface of RemoveDeviceModel.
     * When the mapper gets the request of removing with the name of device model,
     * it should remove the device model to the list of device models.
     * </pre>
     */
    public void removeDeviceModel(dmi.v1beta1.Api.RemoveDeviceModelRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.RemoveDeviceModelResponse> responseObserver) {
      asyncUnaryCall(
          getChannel().newCall(getRemoveDeviceModelMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * UpdateDeviceModel update a device model to the device mapper.
     * Device manager sends the information of device model to the mapper
     * through the interface of UpdateDeviceModel.
     * When the mapper gets the request of updating with the information of device model,
     * it should update the device model to the list of device models.
     * </pre>
     */
    public void updateDeviceModel(dmi.v1beta1.Api.UpdateDeviceModelRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.UpdateDeviceModelResponse> responseObserver) {
      asyncUnaryCall(
          getChannel().newCall(getUpdateDeviceModelMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * GetDevice get the information of a device from the device mapper.
     * Device sends the request of querying device information with the device name to the mapper
     * through the interface of GetDevice.
     * When the mapper gets the request of querying with the device name,
     * it should return the device information.
     * </pre>
     */
    public void getDevice(dmi.v1beta1.Api.GetDeviceRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.GetDeviceResponse> responseObserver) {
      asyncUnaryCall(
          getChannel().newCall(getGetDeviceMethod(), getCallOptions()), request, responseObserver);
    }
  }

  /**
   * <pre>
   * DeviceMapperService defines the public APIS for remote device management.
   * The server is implemented by the device mapper
   * and the client is implemented by the module of device manager in edgecore for downstreaming.
   * The device manager can manage the device life cycle through these interfaces provided by DeviceMapperService.
   * When device manager gets a message of device management from cloudcore, it should call the corresponding grpc interface
   * to make the mapper maintain the list of device information.
   * </pre>
   */
  public static final class DeviceMapperServiceBlockingStub extends io.grpc.stub.AbstractBlockingStub<DeviceMapperServiceBlockingStub> {
    private DeviceMapperServiceBlockingStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected DeviceMapperServiceBlockingStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new DeviceMapperServiceBlockingStub(channel, callOptions);
    }

    /**
     * <pre>
     * RegisterDevice registers a device to the device mapper.
     * Device manager registers a device instance with the information of device
     * to the mapper through the interface of RegisterDevice.
     * When the mapper gets the request of register with device information,
     * it should add the device to the device list and connect to the real physical device via the specific protocol.
     * </pre>
     */
    public dmi.v1beta1.Api.RegisterDeviceResponse registerDevice(dmi.v1beta1.Api.RegisterDeviceRequest request) {
      return blockingUnaryCall(
          getChannel(), getRegisterDeviceMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * RemoveDevice unregisters a device to the device mapper.
     * Device manager unregisters a device instance with the name of device
     * to the mapper through the interface of RemoveDevice.
     * When the mapper gets the request of unregister with device name,
     * it should remove the device from the device list and disconnect to the real physical device.
     * </pre>
     */
    public dmi.v1beta1.Api.RemoveDeviceResponse removeDevice(dmi.v1beta1.Api.RemoveDeviceRequest request) {
      return blockingUnaryCall(
          getChannel(), getRemoveDeviceMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * UpdateDevice updates a device to the device mapper
     * Device manager updates the information of a device used by the mapper
     * through the interface of UpdateDevice.
     * The information of a device includes the meta data and the status data of a device.
     * When the mapper gets the request of updating with the information of a device,
     * it should update the device of the device list and connect to the real physical device via the updated information.
     * </pre>
     */
    public dmi.v1beta1.Api.UpdateDeviceResponse updateDevice(dmi.v1beta1.Api.UpdateDeviceRequest request) {
      return blockingUnaryCall(
          getChannel(), getUpdateDeviceMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * CreateDeviceModel creates a device model to the device mapper.
     * Device manager sends the information of device model to the mapper
     * through the interface of CreateDeviceModel.
     * When the mapper gets the request of creating with the information of device model,
     * it should create a new device model to the list of device models.
     * </pre>
     */
    public dmi.v1beta1.Api.CreateDeviceModelResponse createDeviceModel(dmi.v1beta1.Api.CreateDeviceModelRequest request) {
      return blockingUnaryCall(
          getChannel(), getCreateDeviceModelMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * RemoveDeviceModel remove a device model to the device mapper.
     * Device manager sends the name of device model to the mapper
     * through the interface of RemoveDeviceModel.
     * When the mapper gets the request of removing with the name of device model,
     * it should remove the device model to the list of device models.
     * </pre>
     */
    public dmi.v1beta1.Api.RemoveDeviceModelResponse removeDeviceModel(dmi.v1beta1.Api.RemoveDeviceModelRequest request) {
      return blockingUnaryCall(
          getChannel(), getRemoveDeviceModelMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * UpdateDeviceModel update a device model to the device mapper.
     * Device manager sends the information of device model to the mapper
     * through the interface of UpdateDeviceModel.
     * When the mapper gets the request of updating with the information of device model,
     * it should update the device model to the list of device models.
     * </pre>
     */
    public dmi.v1beta1.Api.UpdateDeviceModelResponse updateDeviceModel(dmi.v1beta1.Api.UpdateDeviceModelRequest request) {
      return blockingUnaryCall(
          getChannel(), getUpdateDeviceModelMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * GetDevice get the information of a device from the device mapper.
     * Device sends the request of querying device information with the device name to the mapper
     * through the interface of GetDevice.
     * When the mapper gets the request of querying with the device name,
     * it should return the device information.
     * </pre>
     */
    public dmi.v1beta1.Api.GetDeviceResponse getDevice(dmi.v1beta1.Api.GetDeviceRequest request) {
      return blockingUnaryCall(
          getChannel(), getGetDeviceMethod(), getCallOptions(), request);
    }
  }

  /**
   * <pre>
   * DeviceMapperService defines the public APIS for remote device management.
   * The server is implemented by the device mapper
   * and the client is implemented by the module of device manager in edgecore for downstreaming.
   * The device manager can manage the device life cycle through these interfaces provided by DeviceMapperService.
   * When device manager gets a message of device management from cloudcore, it should call the corresponding grpc interface
   * to make the mapper maintain the list of device information.
   * </pre>
   */
  public static final class DeviceMapperServiceFutureStub extends io.grpc.stub.AbstractFutureStub<DeviceMapperServiceFutureStub> {
    private DeviceMapperServiceFutureStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected DeviceMapperServiceFutureStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new DeviceMapperServiceFutureStub(channel, callOptions);
    }

    /**
     * <pre>
     * RegisterDevice registers a device to the device mapper.
     * Device manager registers a device instance with the information of device
     * to the mapper through the interface of RegisterDevice.
     * When the mapper gets the request of register with device information,
     * it should add the device to the device list and connect to the real physical device via the specific protocol.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<dmi.v1beta1.Api.RegisterDeviceResponse> registerDevice(
        dmi.v1beta1.Api.RegisterDeviceRequest request) {
      return futureUnaryCall(
          getChannel().newCall(getRegisterDeviceMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * RemoveDevice unregisters a device to the device mapper.
     * Device manager unregisters a device instance with the name of device
     * to the mapper through the interface of RemoveDevice.
     * When the mapper gets the request of unregister with device name,
     * it should remove the device from the device list and disconnect to the real physical device.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<dmi.v1beta1.Api.RemoveDeviceResponse> removeDevice(
        dmi.v1beta1.Api.RemoveDeviceRequest request) {
      return futureUnaryCall(
          getChannel().newCall(getRemoveDeviceMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * UpdateDevice updates a device to the device mapper
     * Device manager updates the information of a device used by the mapper
     * through the interface of UpdateDevice.
     * The information of a device includes the meta data and the status data of a device.
     * When the mapper gets the request of updating with the information of a device,
     * it should update the device of the device list and connect to the real physical device via the updated information.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<dmi.v1beta1.Api.UpdateDeviceResponse> updateDevice(
        dmi.v1beta1.Api.UpdateDeviceRequest request) {
      return futureUnaryCall(
          getChannel().newCall(getUpdateDeviceMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * CreateDeviceModel creates a device model to the device mapper.
     * Device manager sends the information of device model to the mapper
     * through the interface of CreateDeviceModel.
     * When the mapper gets the request of creating with the information of device model,
     * it should create a new device model to the list of device models.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<dmi.v1beta1.Api.CreateDeviceModelResponse> createDeviceModel(
        dmi.v1beta1.Api.CreateDeviceModelRequest request) {
      return futureUnaryCall(
          getChannel().newCall(getCreateDeviceModelMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * RemoveDeviceModel remove a device model to the device mapper.
     * Device manager sends the name of device model to the mapper
     * through the interface of RemoveDeviceModel.
     * When the mapper gets the request of removing with the name of device model,
     * it should remove the device model to the list of device models.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<dmi.v1beta1.Api.RemoveDeviceModelResponse> removeDeviceModel(
        dmi.v1beta1.Api.RemoveDeviceModelRequest request) {
      return futureUnaryCall(
          getChannel().newCall(getRemoveDeviceModelMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * UpdateDeviceModel update a device model to the device mapper.
     * Device manager sends the information of device model to the mapper
     * through the interface of UpdateDeviceModel.
     * When the mapper gets the request of updating with the information of device model,
     * it should update the device model to the list of device models.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<dmi.v1beta1.Api.UpdateDeviceModelResponse> updateDeviceModel(
        dmi.v1beta1.Api.UpdateDeviceModelRequest request) {
      return futureUnaryCall(
          getChannel().newCall(getUpdateDeviceModelMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * GetDevice get the information of a device from the device mapper.
     * Device sends the request of querying device information with the device name to the mapper
     * through the interface of GetDevice.
     * When the mapper gets the request of querying with the device name,
     * it should return the device information.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<dmi.v1beta1.Api.GetDeviceResponse> getDevice(
        dmi.v1beta1.Api.GetDeviceRequest request) {
      return futureUnaryCall(
          getChannel().newCall(getGetDeviceMethod(), getCallOptions()), request);
    }
  }

  private static final int METHODID_REGISTER_DEVICE = 0;
  private static final int METHODID_REMOVE_DEVICE = 1;
  private static final int METHODID_UPDATE_DEVICE = 2;
  private static final int METHODID_CREATE_DEVICE_MODEL = 3;
  private static final int METHODID_REMOVE_DEVICE_MODEL = 4;
  private static final int METHODID_UPDATE_DEVICE_MODEL = 5;
  private static final int METHODID_GET_DEVICE = 6;

  private static final class MethodHandlers<Req, Resp> implements
      io.grpc.stub.ServerCalls.UnaryMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.ServerStreamingMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.ClientStreamingMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.BidiStreamingMethod<Req, Resp> {
    private final DeviceMapperServiceImplBase serviceImpl;
    private final int methodId;

    MethodHandlers(DeviceMapperServiceImplBase serviceImpl, int methodId) {
      this.serviceImpl = serviceImpl;
      this.methodId = methodId;
    }

    @java.lang.Override
    @java.lang.SuppressWarnings("unchecked")
    public void invoke(Req request, io.grpc.stub.StreamObserver<Resp> responseObserver) {
      switch (methodId) {
        case METHODID_REGISTER_DEVICE:
            try {
                serviceImpl.registerDevice((Api.RegisterDeviceRequest) request,
                    (io.grpc.stub.StreamObserver<Api.RegisterDeviceResponse>) responseObserver);
            } catch (Exception e) {
                throw new RuntimeException(e);
            }
            break;
        case METHODID_REMOVE_DEVICE:
          serviceImpl.removeDevice((dmi.v1beta1.Api.RemoveDeviceRequest) request,
              (io.grpc.stub.StreamObserver<dmi.v1beta1.Api.RemoveDeviceResponse>) responseObserver);
          break;
        case METHODID_UPDATE_DEVICE:
            try {
                serviceImpl.updateDevice((Api.UpdateDeviceRequest) request,
                    (io.grpc.stub.StreamObserver<Api.UpdateDeviceResponse>) responseObserver);
            } catch (Exception e) {
                throw new RuntimeException(e);
            }
            break;
        case METHODID_CREATE_DEVICE_MODEL:
          serviceImpl.createDeviceModel((dmi.v1beta1.Api.CreateDeviceModelRequest) request,
              (io.grpc.stub.StreamObserver<dmi.v1beta1.Api.CreateDeviceModelResponse>) responseObserver);
          break;
        case METHODID_REMOVE_DEVICE_MODEL:
          serviceImpl.removeDeviceModel((dmi.v1beta1.Api.RemoveDeviceModelRequest) request,
              (io.grpc.stub.StreamObserver<dmi.v1beta1.Api.RemoveDeviceModelResponse>) responseObserver);
          break;
        case METHODID_UPDATE_DEVICE_MODEL:
          serviceImpl.updateDeviceModel((dmi.v1beta1.Api.UpdateDeviceModelRequest) request,
              (io.grpc.stub.StreamObserver<dmi.v1beta1.Api.UpdateDeviceModelResponse>) responseObserver);
          break;
        case METHODID_GET_DEVICE:
            try {
                serviceImpl.getDevice((Api.GetDeviceRequest) request,
                    (io.grpc.stub.StreamObserver<Api.GetDeviceResponse>) responseObserver);
            } catch (IOException e) {
                throw new RuntimeException(e);
            }
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

  private static abstract class DeviceMapperServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoFileDescriptorSupplier, io.grpc.protobuf.ProtoServiceDescriptorSupplier {
    DeviceMapperServiceBaseDescriptorSupplier() {}

    @java.lang.Override
    public com.google.protobuf.Descriptors.FileDescriptor getFileDescriptor() {
      return dmi.v1beta1.Api.getDescriptor();
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.ServiceDescriptor getServiceDescriptor() {
      return getFileDescriptor().findServiceByName("DeviceMapperService");
    }
  }

  private static final class DeviceMapperServiceFileDescriptorSupplier
      extends DeviceMapperServiceBaseDescriptorSupplier {
    DeviceMapperServiceFileDescriptorSupplier() {}
  }

  private static final class DeviceMapperServiceMethodDescriptorSupplier
      extends DeviceMapperServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoMethodDescriptorSupplier {
    private final String methodName;

    DeviceMapperServiceMethodDescriptorSupplier(String methodName) {
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
      synchronized (DeviceMapperServiceGrpc.class) {
        result = serviceDescriptor;
        if (result == null) {
          serviceDescriptor = result = io.grpc.ServiceDescriptor.newBuilder(SERVICE_NAME)
              .setSchemaDescriptor(new DeviceMapperServiceFileDescriptorSupplier())
              .addMethod(getRegisterDeviceMethod())
              .addMethod(getRemoveDeviceMethod())
              .addMethod(getUpdateDeviceMethod())
              .addMethod(getCreateDeviceModelMethod())
              .addMethod(getRemoveDeviceModelMethod())
              .addMethod(getUpdateDeviceModelMethod())
              .addMethod(getGetDeviceMethod())
              .build();
        }
      }
    }
    return result;
  }
}
