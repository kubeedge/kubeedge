#include <grpcpp/grpcpp.h>
#include <sys/stat.h>
#include <unistd.h>
#include <iostream>
#include <memory>
#include "dmi/v1beta1/api.grpc.pb.h"
#include "log/log.h"
#include "common/configmaptype.h"
#include "server.h"
// Wrap this line with extern "C" to avoid C/C++ symbol mismatch
extern "C" {
#include "device/device.h"
}
#include <grpcpp/health_check_service_interface.h>
#include <grpcpp/ext/proto_server_reflection_plugin.h>
#include <fcntl.h>
#include <string.h>  // For strdup/free
#include <cstdlib>  // For system, getenv
#include <future>
#include <chrono>

// Predefined/declared for internal use
static DeviceManager *g_device_manager = nullptr;
static int apply_desired_to_device(DeviceManager *mgr, const v1beta1::Device &dev);
// Direct write function prototype (defined below)
static int write_modbus_direct(const std::string &prop, const std::string &val);

// Retrieves the timeout for applying desired properties (default 1500ms, can be overridden by MAPPER_APPLY_TIMEOUT_MS)
static int get_apply_timeout_ms() {
    const char *v = std::getenv("MAPPER_APPLY_TIMEOUT_MS");
    if (!v || !*v) return 1500;
    int n = std::atoi(v);
    return n > 0 ? n : 1500;
}

// Applies desired properties to a device with a timeout
static int apply_desired_to_device_with_timeout(DeviceManager *mgr, const v1beta1::Device &dev, int timeout_ms) {
    using namespace std::chrono;
    auto fut = std::async(std::launch::async, [mgr, &dev]() {
        return apply_desired_to_device(mgr, dev);
    });
    if (fut.wait_for(milliseconds(timeout_ms)) == std::future_status::ready) {
        return fut.get();
    }
    log_warn("apply_desired_to_device timeout after %d ms", timeout_ms);
    return -2;
}

// Device panel class
class DevPanel {
public:
    DevPanel() {}
    ~DevPanel() {}
    
    void processDevice(const std::string& deviceId) {
        // Process a device by its ID
    }
};

// Implementation of the DeviceMapperService
class DeviceMapperServiceImpl final : public v1beta1::DeviceMapperService::Service {
public:
    explicit DeviceMapperServiceImpl(std::shared_ptr<DevPanel> devPanel) : devPanel_(devPanel) {}

    // Handles the RegisterDevice gRPC call
    ::grpc::Status RegisterDevice(::grpc::ServerContext* context,
                                  const ::v1beta1::RegisterDeviceRequest* request,
                                  ::v1beta1::RegisterDeviceResponse* response) override {
        log_info("RegisterDevice called");
        return ::grpc::Status::OK;
    }
    
    // Handles the RemoveDevice gRPC call
    ::grpc::Status RemoveDevice(::grpc::ServerContext* context,
                                const ::v1beta1::RemoveDeviceRequest* request,
                                ::v1beta1::RemoveDeviceResponse* response) override {
        log_info("RemoveDevice called");
        return ::grpc::Status::OK;
    }
    
    // Handles the UpdateDevice gRPC call
    ::grpc::Status UpdateDevice(::grpc::ServerContext* context,
                                const ::v1beta1::UpdateDeviceRequest* request,
                                ::v1beta1::UpdateDeviceResponse* response) override {
        if (!request || !request->has_device()) {
            return ::grpc::Status(::grpc::StatusCode::INVALID_ARGUMENT, "empty request");
        }
        const auto &dev = request->device();
        log_info("UpdateDevice called: name=%s ns=%s has_spec=%d props=%d",
                 dev.name().c_str(),
                 dev.namespace_().c_str(),
                 dev.has_spec(),
                 dev.has_spec() ? dev.spec().properties_size() : 0);

        if (dev.has_spec()) {
            const auto &spec = dev.spec();
            for (int i = 0; i < spec.properties_size(); ++i) {
                const auto &p = spec.properties(i);
                const std::string pn = p.name();
                const bool hd = p.has_desired();
                const std::string dv = hd ? p.desired().value() : std::string();
                log_info("CloudProp[%d]: name=%s hasDesired=%d desired=%s", i, pn.c_str(), hd, dv.c_str());
            }
        }

        const char *force = std::getenv("MAPPER_FORCE_FALLBACK");
        if (force && std::strcmp(force, "1") == 0 && dev.has_spec()) {
            int ok = 0;
            const auto &spec = dev.spec();
            for (int i = 0; i < spec.properties_size(); ++i) {
                const auto &p = spec.properties(i);
                if (!p.has_desired()) continue;
                const std::string propName = p.name();
                const std::string desired  = p.desired().value();
                if (desired.empty()) continue;
                int wrc = write_modbus_direct(propName, desired);
                log_info("Force DirectWrite prop=%s val=%s rc=%d", propName.c_str(), desired.c_str(), wrc);
                if (wrc == 0) {
                    ok++;
                    Device *local = device_manager_get(g_device_manager, dev.name().c_str());
                    if (!local && !dev.namespace_().empty()) {
                        std::string key = dev.namespace_() + "/" + dev.name();
                        local = device_manager_get(g_device_manager, key.c_str());
                    }
                    if (local && local->instance.twins) {
                        for (int t = 0; t < local->instance.twinsCount; ++t) {
                            Twin *tw = &local->instance.twins[t];
                            if (tw && tw->propertyName &&
                                propName == tw->propertyName) {
                                free(tw->observedDesired.value);
                                tw->observedDesired.value = strdup(desired.c_str());
                                free(tw->reported.value);
                                tw->reported.value = strdup(desired.c_str());
                                char ts[32]; time_t tt=time(NULL); struct tm tm; gmtime_r(&tt,&tm);
                                strftime(ts,32,"%Y-%m-%dT%H:%M:%SZ",&tm);
                                free(tw->reported.metadata.timestamp);
                                tw->reported.metadata.timestamp = strdup(ts);
                                break;
                            }
                        }
                    }
                }
            }
            if (ok > 0) return ::grpc::Status::OK;
        }

        int arc = apply_desired_to_device_with_timeout(g_device_manager, dev, get_apply_timeout_ms());
        log_info("apply_desired_to_device rc=%d", arc);

        if (arc != 0 && dev.has_spec()) {
            int ok = 0;
            const auto &spec = dev.spec();
            for (int i = 0; i < spec.properties_size(); ++i) {
                const auto &p = spec.properties(i);
                if (!p.has_desired()) continue;
                const std::string propName = p.name();
                const std::string desired  = p.desired().value();
                if (desired.empty()) continue;
                int wrc = write_modbus_direct(propName, desired);
                log_info("DirectWrite prop=%s val=%s rc=%d", propName.c_str(), desired.c_str(), wrc);
                if (wrc == 0) ok++;
            }
            if (ok > 0) return ::grpc::Status::OK;
            return ::grpc::Status(::grpc::StatusCode::NOT_FOUND, "device or properties not applied");
        }
        return ::grpc::Status::OK;
    }
    
    // Handles the CreateDeviceModel gRPC call
    ::grpc::Status CreateDeviceModel(::grpc::ServerContext* context,
                                     const ::v1beta1::CreateDeviceModelRequest* request,
                                     ::v1beta1::CreateDeviceModelResponse* response) override {
        log_info("CreateDeviceModel called");
        return ::grpc::Status::OK;
    }
    
    // Handles the RemoveDeviceModel gRPC call
    ::grpc::Status RemoveDeviceModel(::grpc::ServerContext* context,
                                     const ::v1beta1::RemoveDeviceModelRequest* request,
                                     ::v1beta1::RemoveDeviceModelResponse* response) override {
        log_info("RemoveDeviceModel called");
        return ::grpc::Status::OK;
    }
    
    // Handles the UpdateDeviceModel gRPC call
    ::grpc::Status UpdateDeviceModel(::grpc::ServerContext* context,
                                     const ::v1beta1::UpdateDeviceModelRequest* request,
                                     ::v1beta1::UpdateDeviceModelResponse* response) override {
        log_info("UpdateDeviceModel called");
        return ::grpc::Status::OK;
    }
    
    // Handles the GetDevice gRPC call
    ::grpc::Status GetDevice(::grpc::ServerContext* context,
                             const ::v1beta1::GetDeviceRequest* request,
                             ::v1beta1::GetDeviceResponse* response) override {
        log_info("GetDevice called");
        return ::grpc::Status::OK;
    }

private:
    std::shared_ptr<DevPanel> devPanel_;  // Shared pointer to the device panel
};

// Implementation of the ServerConfig constructor
ServerConfig::ServerConfig(const std::string& sock_path, const std::string& protocol)
    : sockPath(sock_path), protocol(protocol) {}

// Implementation of the GrpcServer class
GrpcServer::GrpcServer(const ServerConfig& cfg, std::shared_ptr<DevPanel> devPanel)
    : cfg_(cfg), devPanel_(devPanel) {}

// Starts the gRPC server
int GrpcServer::Start() {
    log_info("uds socket path: %s", cfg_.sockPath.c_str());

    struct stat st;
    if (stat(cfg_.sockPath.c_str(), &st) == 0) {
        if (unlink(cfg_.sockPath.c_str()) != 0) {
            log_error("Failed to remove uds socket: %s", cfg_.sockPath.c_str());
            return -1;
        }
    }
    grpc::EnableDefaultHealthCheckService(true);
    grpc::reflection::InitProtoReflectionServerBuilderPlugin();

    std::string server_address = "unix://" + cfg_.sockPath;
    DeviceMapperServiceImpl service(devPanel_);

    grpc::ServerBuilder builder;
    builder.AddListeningPort(server_address, grpc::InsecureServerCredentials());
    builder.RegisterService(&service);

    server_ = builder.BuildAndStart();
    if (!server_) {
        log_error("failed to start grpc server");
        return -1;
    }

    log_info("start grpc server on %s", server_address.c_str());
    server_->Wait();  // Waits until Stop() is called
    return 0;
}

// Stops the gRPC server
void GrpcServer::Stop() {
    if (stopped_) return;
    stopped_ = true;
    log_info("Stopping gRPC server...");
    if (server_) {
        server_->Shutdown();  // Wakes up Wait()
    }
    log_info("gRPC server stopped");
}

// 获取直写目标（可通过环境变量覆盖）
static void get_modbus_target(std::string &host, int &port) {
    const char *h = std::getenv("MAPPER_MODBUS_ADDR");
    const char *p = std::getenv("MAPPER_MODBUS_PORT");
    host = (h && *h) ? h : "127.0.0.1";
    port = (p && *p) ? std::atoi(p) : 1502;
    if (port <= 0) port = 1502;
}

// 简单映射：属性名 -> Modbus 寄存器偏移（按你的demo）
static int map_offset_by_prop(const std::string &name) {
    if (name == "threshold") return 2;    // 40002
    if (name == "temperature") return 1;  // 40001
    return -1;
}

// 读取超时（毫秒），默认 1000，可用 MAPPER_DEVICE_TIMEOUT_MS 覆盖
static int get_device_timeout_ms() {
    const char *v = std::getenv("MAPPER_DEVICE_TIMEOUT_MS");
    if (!v || !*v) return 1000;
    int n = std::atoi(v);
    return n > 0 ? n : 1000;
}

// 带超时调用 device_deal_twin，超时返回 -2
static int device_deal_twin_with_timeout(Device* device, Twin* twin, int timeout_ms) {
    using namespace std::chrono;
    auto fut = std::async(std::launch::async, [device, twin]() {
        return device_deal_twin(device, twin);
    });
    if (fut.wait_for(milliseconds(timeout_ms)) == std::future_status::ready) {
        return fut.get();
    }
    log_warn("device_deal_twin timeout after %d ms", timeout_ms);
    return -2;
}

// 兜底直写：不依赖 DeviceManager（打印命令与返回码）
static int write_modbus_direct(const std::string &prop, const std::string &val) {
    int offset = map_offset_by_prop(prop);
    if (offset < 0) {
        log_warn("Fallback skip: unknown prop %s", prop.c_str());
        return -1;
    }
    std::string host; int port;
    get_modbus_target(host, port);

    int iv = std::atoi(val.c_str());
    char cmd[512];
    std::snprintf(cmd, sizeof(cmd),
                  "/usr/bin/mbpoll -1 -m tcp %s -p %d -a 1 -t 4 -r %d %d",
                  host.c_str(), port, offset, iv);
    log_info("DirectWrite exec: %s", cmd);
    int rc = std::system(cmd);
    log_info("DirectWrite rc=%d (prop=%s, val=%s)", rc, prop.c_str(), val.c_str());
    if (rc != 0) {
        log_error("DirectWrite failed rc=%d to %s:%d HR[%d]=%d", rc, host.c_str(), port, offset, iv);
        return -1;
    }
    log_info("DirectWrite OK to %s:%d HR[%d]=%d (prop=%s)",
             host.c_str(), port, offset, iv, prop.c_str());
    return 0;
}

// 新增：把云端 desired 写入本地 twin，并触发一次下发
static int apply_desired_to_device(DeviceManager *mgr, const v1beta1::Device &dev) {
    if (!mgr) {
        log_warn("apply_desired_to_device: mgr is null, fallback to direct writes");
        int ok = 0;
        if (dev.has_spec()) {
            const auto &spec = dev.spec();
            for (int i = 0; i < spec.properties_size(); ++i) {
                const auto &p = spec.properties(i);
                if (!p.has_desired()) continue;
                const std::string propName = p.name();
                const std::string desired = p.desired().value();
                if (desired.empty()) continue;
                ok += (write_modbus_direct(propName, desired) == 0);
            }
        }
        return ok > 0 ? 0 : -1;
    }

    const std::string ns = dev.namespace_();
    const std::string devName = dev.name();
    if (devName.empty()) {
        log_warn("UpdateDevice: empty device name");
        return -1;
    }

    Device *local = device_manager_get(mgr, devName.c_str());
    if (!local && !ns.empty()) {
        std::string key = ns + "/" + devName;
        local = device_manager_get(mgr, key.c_str());
    }
    if (!local) {
        log_warn("UpdateDevice: device %s (ns=%s) not found locally, use fallback", devName.c_str(), ns.c_str());
        int ok = 0;
        if (dev.has_spec()) {
            const auto &spec = dev.spec();
            for (int i = 0; i < spec.properties_size(); ++i) {
                const auto &p = spec.properties(i);
                if (!p.has_desired()) continue;
                const std::string propName = p.name();
                const std::string desired = p.desired().value();
                if (desired.empty()) continue;
                ok += (write_modbus_direct(propName, desired) == 0);
            }
        }
        return ok > 0 ? 0 : -1;
    }

    const int twinsCount = local->instance.twinsCount;
    const int cloudProps = dev.has_spec() ? dev.spec().properties_size() : 0;
    log_info("UpdateDevice target %s/%s: localTwins=%d cloudProps=%d",
             ns.c_str(), devName.c_str(), twinsCount, cloudProps);

    for (int t = 0; t < twinsCount; ++t) {
        Twin *tw = &local->instance.twins[t];
        log_info("LocalTwin[%d] name=%s desired=%s reported=%s",
                 t,
                 tw && tw->propertyName ? tw->propertyName : "(null)",
                 tw && tw->observedDesired.value ? tw->observedDesired.value : "(nil)",
                 tw && tw->reported.value ? tw->reported.value : "(nil)");
    }

    int updated = 0, fallback_ok = 0;
    if (dev.has_spec()) {
        const auto &spec = dev.spec();
        for (int i = 0; i < spec.properties_size(); ++i) {
            const auto &p = spec.properties(i);
            const std::string propName = p.name();
            const bool hasDesired = p.has_desired();
            const std::string desired = hasDesired ? p.desired().value() : std::string();
            log_info("CloudProp[%d] name=%s hasDesired=%d desired=%s",
                     i, propName.c_str(), hasDesired, desired.c_str());
            if (!hasDesired || desired.empty()) continue;

            // 名称匹配
            int matchIdx = -1;
            for (int t = 0; t < twinsCount; ++t) {
                Twin *tw = &local->instance.twins[t];
                if (tw && tw->propertyName && propName == tw->propertyName) {
                    matchIdx = t; break;
                }
            }
            // 索引兜底
            if (matchIdx < 0 && i < twinsCount) {
                log_warn("No twin matched by name '%s', fallback to index %d", propName.c_str(), i);
                matchIdx = i;
            }

            if (matchIdx >= 0) {
                Twin *tw = &local->instance.twins[matchIdx];
                free(tw->observedDesired.value);
                tw->observedDesired.value = strdup(desired.c_str());
                log_info("Apply desired -> twin[%d](%s) = %s",
                         matchIdx,
                         tw->propertyName ? tw->propertyName : "(null)",
                         desired.c_str());

                int timeout_ms = get_device_timeout_ms();
                int drc = device_deal_twin_with_timeout(local, tw, timeout_ms);
                log_info("device_deal_twin rc=%d for prop=%s", drc, propName.c_str());
                if (drc == 0) {
                    updated++;
                } else {
                    int wrc = write_modbus_direct(propName, desired);
                    log_info("Fallback DirectWrite after device_deal_twin rc=%d: prop=%s val=%s rc=%d",
                             drc, propName.c_str(), desired.c_str(), wrc);
                    if (wrc == 0) fallback_ok++;
                }
            } else {
                // 最后兜底：直写
                int wrc = write_modbus_direct(propName, desired);
                log_info("Fallback DirectWrite (no twin matched): prop=%s val=%s rc=%d",
                         propName.c_str(), desired.c_str(), wrc);
                if (wrc == 0) fallback_ok++;
            }
        }
    }
    return (updated > 0 || fallback_ok > 0) ? 0 : -1;
}

extern "C" {

ServerConfig *server_config_new(const char *sock_path, const char *protocol) {
    if (!sock_path || !protocol) {
        log_error("Invalid parameters for server config creation");
        return nullptr;
    }
    
    try {
        return new ServerConfig(std::string(sock_path), std::string(protocol));
    } catch (const std::exception& e) {
        log_error("Failed to create server config: %s", e.what());
        return nullptr;
    }
}

void server_config_free(ServerConfig *config) {
    if (config) {
        delete config;
    }
}

GrpcServer *grpcserver_new(ServerConfig *config, DeviceManager *device_manager) {
    if (!config || !device_manager) {
        log_error("Invalid parameters for gRPC server creation");
        return nullptr;
    }
    // 绑定全局 DeviceManager 指针
    g_device_manager = device_manager;
    
    try {
        auto devPanel = std::make_shared<DevPanel>();
        
        return new GrpcServer(*config, devPanel);
    } catch (const std::exception& e) {
        log_error("Failed to create gRPC server: %s", e.what());
        return nullptr;
    }
}

int grpcserver_start(GrpcServer *server) {
    if (!server) {
        log_error("Invalid gRPC server pointer");
        return -1;
    }
    
    try {
        return server->Start();
    } catch (const std::exception& e) {
        log_error("Failed to start gRPC server: %s", e.what());
        return -1;
    }
}

void grpcserver_stop(GrpcServer *server) {
    if (!server) {
        log_warn("Trying to stop NULL gRPC server");
        return;
    }
    try {
        server->Stop();
    } catch (const std::exception& e) {
        log_error("Failed to stop gRPC server: %s", e.what());
    }
}

void grpcserver_free(GrpcServer *server) {
    if (server) {
        // 此处不再强制 Stop，调用方已在 cleanup 中显式停止并 unlink
        delete server;
    }
}

} // extern "C"