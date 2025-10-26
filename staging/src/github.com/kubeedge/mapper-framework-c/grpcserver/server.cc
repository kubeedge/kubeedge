#include <grpcpp/grpcpp.h>
#include <sys/stat.h>
#include <unistd.h>
#include <iostream>
#include <memory>
#include <future>
#include <chrono>
#include <cstdlib>
#include <cstring>
#include <grpcpp/ext/proto_server_reflection_plugin.h>
#include "grpcserver/server.h"
#include "dmi/v1beta1/api.grpc.pb.h"
#include "log/log.h"
#include <google/protobuf/util/json_util.h>
extern "C"
{
#include "device/device.h"
#include "device/dev_panel.h"
}

extern "C"
{
    int dev_panel_remove_dev(DeviceManager *manager, const char *ns, const char *name);
}

static DeviceManager *g_device_manager = nullptr;

class DevPanel
{
public:
    DevPanel() {}
    ~DevPanel() {}

    void processDevice(const std::string &deviceId)
    {
    }
};

static void build_model_min(const v1beta1::Device &src, DeviceModel *out)
{
    memset(out, 0, sizeof(*out));
    if (src.has_spec())
    {
        const auto &sp = src.spec();
        if (!sp.devicemodelreference().empty())
            out->name = strdup(sp.devicemodelreference().c_str());
    }
    if (!src.namespace_().empty())
        out->namespace_ = strdup(src.namespace_().c_str());
}

static void build_instance_min(const v1beta1::Device &src, DeviceInstance *out)
{
    memset(out, 0, sizeof(*out));
    if (!src.name().empty())
        out->name = strdup(src.name().c_str());
    if (!src.namespace_().empty())
        out->namespace_ = strdup(src.namespace_().c_str());
    if (src.has_spec())
    {
        const auto &sp = src.spec();
        if (!sp.devicemodelreference().empty())
            out->model = strdup(sp.devicemodelreference().c_str());
        if (sp.has_protocol())
        {
            const auto &pr = sp.protocol();
            if (!pr.protocolname().empty())
                out->pProtocol.protocolName = strdup(pr.protocolname().c_str());
            if (pr.has_configdata()) {
                std::string tmp;
                auto s = google::protobuf::util::MessageToJsonString(pr.configdata(), &tmp);
                out->pProtocol.configData = strdup(s.ok() ? tmp.c_str() : "{}");
            } else {
                out->pProtocol.configData = NULL;
            }
        }
        if (sp.properties_size() > 0)
        {
            out->twinsCount = sp.properties_size();
            out->twins = (Twin *)calloc(out->twinsCount, sizeof(Twin));
            for (int t = 0; t < out->twinsCount; ++t) {
                const auto &pp = sp.properties(t);
                if (!pp.name().empty())
                    out->twins[t].propertyName = strdup(pp.name().c_str());
                if (pp.has_desired() && !pp.desired().value().empty()) {
                    out->twins[t].observedDesired.value = strdup(pp.desired().value().c_str());
                    // optional: set metadata timestamp now
                    char tsbuf[64];
                    time_t now = time(NULL);
                    snprintf(tsbuf, sizeof(tsbuf), "%lld", (long long)now * 1000);
                    out->twins[t].observedDesired.metadata.timestamp = strdup(tsbuf);
                    out->twins[t].observedDesired.metadata.type = strdup("string");
                }
                out->twins[t].reported.value = NULL;
            }
            out->propertiesCount = sp.properties_size();
            out->properties = (DeviceProperty *)calloc(out->propertiesCount, sizeof(DeviceProperty));
            for (int i = 0; i < out->propertiesCount; ++i)
            {
                const auto &p = sp.properties(i);
                if (!p.name().empty()) {
                    out->properties[i].name = strdup(p.name().c_str());
                    out->properties[i].propertyName = strdup(p.name().c_str());
                }
                if (p.has_pushmethod()) {
                    out->properties[i].pushMethod = (PushMethodConfig *)calloc(1, sizeof(PushMethodConfig));
                    const auto &pm = p.pushmethod();

                    std::string pm_json;
                    if (pm.has_mqtt()) {
                        const auto &m = pm.mqtt();
                        std::string raw_addr = m.address().empty() ? "127.0.0.1" : m.address();
                        std::string addr = raw_addr;
                        auto scheme_pos = raw_addr.find("://");
                        if (scheme_pos != std::string::npos && scheme_pos + 3 < raw_addr.size()) {
                            addr = raw_addr.substr(scheme_pos + 3);
                        }
                        int port = 1883;
                        auto pos = addr.rfind(':');
                        if (pos != std::string::npos && pos + 1 < addr.size()) {
                            std::string portstr = addr.substr(pos + 1);
                            bool nums = true;
                            for (char c : portstr) if (!isdigit((unsigned char)c)) { nums = false; break; }
                            if (nums) {
                                try { port = std::stoi(portstr); } catch(...) { port = 1883; }
                                addr = addr.substr(0, pos);
                            }
                        }
                        char buf[512];
                        snprintf(buf, sizeof(buf),
                                 "{\"brokerUrl\":\"%s\",\"port\":%d,\"topicPrefix\":\"%s\",\"qos\":%d,\"keepAlive\":%d}",
                                 addr.c_str(), port,
                                 m.topic().empty() ? "kubeedge/device" : m.topic().c_str(),
                                 (int)m.qos(), 60);
                        pm_json = buf;
                        out->properties[i].pushMethod->methodName = strdup("mqtt");
                    } else if (pm.has_http()) {
                        const auto &h = pm.http();
                        char buf[512];
                        const char *host = h.hostname().empty() ? "127.0.0.1" : h.hostname().c_str();
                        int port = h.port() ? (int)h.port() : 80;
                        const char *path = h.requestpath().empty() ? "/" : h.requestpath().c_str();
                        int timeout = h.timeout() ? (int)h.timeout() : 3000;
                        snprintf(buf, sizeof(buf),
                                 "{\"endpoint\":\"http://%s:%d%s\",\"method\":\"POST\",\"timeout\":%d}",
                                 host, port, path, timeout);
                        pm_json = buf;
                        out->properties[i].pushMethod->methodName = strdup("http");
                    } else if (pm.has_otel()) {
                        const auto &o = pm.otel();
                        char buf[256];
                        const char *ep = o.endpointurl().empty() ? "http://localhost:4318/v1/metrics" : o.endpointurl().c_str();
                        snprintf(buf, sizeof(buf), "{\"endpoint\":\"%s\"}", ep);
                        pm_json = buf;
                        out->properties[i].pushMethod->methodName = strdup("otel");
                    } else {
                        std::string tmp;
                        auto s = google::protobuf::util::MessageToJsonString(pm, &tmp);
                        pm_json = s.ok() ? tmp : std::string("{}");
                        out->properties[i].pushMethod->methodName = strdup("unknown");
                    }

                    out->properties[i].pushMethod->methodConfig = strdup(pm_json.c_str());

                    if (pm.has_dbmethod()) {
                        out->properties[i].pushMethod->dbMethod = (DBMethodConfig *)calloc(1, sizeof(DBMethodConfig));
                        const auto &db = pm.dbmethod();
                        if (db.has_mysql()) out->properties[i].pushMethod->dbMethod->dbMethodName = strdup("mysql");
                        else if (db.has_redis()) out->properties[i].pushMethod->dbMethod->dbMethodName = strdup("redis");
                        else if (db.has_influxdb2()) out->properties[i].pushMethod->dbMethod->dbMethodName = strdup("influxdb2");
                        else if (db.has_tdengine()) out->properties[i].pushMethod->dbMethod->dbMethodName = strdup("tdengine");
                        else out->properties[i].pushMethod->dbMethod->dbMethodName = strdup("unknown");

                        out->properties[i].pushMethod->dbMethod->dbConfig = (DBConfig *)calloc(1, sizeof(DBConfig));
                        std::string tmp;
                        if (db.has_mysql()) {
                            auto st = google::protobuf::util::MessageToJsonString(db.mysql(), &tmp);
                            const char *val = st.ok() ? tmp.c_str() : "{}";
                            out->properties[i].pushMethod->dbMethod->dbConfig->mysqlClientConfig = strdup(val);
                        } else if (db.has_redis()) {
                            auto st = google::protobuf::util::MessageToJsonString(db.redis(), &tmp);
                            const char *val = st.ok() ? tmp.c_str() : "{}";
                            out->properties[i].pushMethod->dbMethod->dbConfig->redisClientConfig = strdup(val);
                        } else if (db.has_influxdb2()) {
                            auto st = google::protobuf::util::MessageToJsonString(db.influxdb2(), &tmp);
                            const char *val = st.ok() ? tmp.c_str() : "{}";
                            out->properties[i].pushMethod->dbMethod->dbConfig->influxdb2ClientConfig = strdup(val);
                        } else if (db.has_tdengine()) {
                            auto st = google::protobuf::util::MessageToJsonString(db.tdengine(), &tmp);
                            const char *val = st.ok() ? tmp.c_str() : "{}";
                            out->properties[i].pushMethod->dbMethod->dbConfig->tdengineClientConfig = strdup(val);
                        }
                    }
                
                }
            }
         }
     }
 }

static void free_model_min(DeviceModel *m)
{
    if (!m)
        return;
    free(m->name);
    free(m->namespace_);
    free(m->description);
    if (m->properties)
        free(m->properties);
    memset(m, 0, sizeof(*m));
}

static void free_instance_min(DeviceInstance *d)
{
    if (!d)
        return;
    free(d->id);
    free(d->name);
    free(d->namespace_);
    free(d->model);
    free(d->protocolName);
    free(d->pProtocol.protocolName);
    free(d->pProtocol.configData);
    if (d->properties)
    {
        for (int i = 0; i < d->propertiesCount; ++i) {
            free(d->properties[i].name);
            if (d->properties[i].pushMethod) {
                PushMethodConfig *pm = d->properties[i].pushMethod;
                free(pm->methodName);
                free(pm->methodConfig);
                if (pm->dbMethod) {
                    DBMethodConfig *dbm = pm->dbMethod;
                    free(dbm->dbMethodName);
                    if (dbm->dbConfig) {
                        free(dbm->dbConfig->mysqlClientConfig);
                        free(dbm->dbConfig->redisClientConfig);
                        free(dbm->dbConfig->influxdb2ClientConfig);
                        free(dbm->dbConfig->tdengineClientConfig);
                        free(dbm->dbConfig);
                    }
                    free(dbm);
                }
                free(pm);
            }
        }
        free(d->properties);
    }
    if (d->twins) {
        for (int i = 0; i < d->twinsCount; ++i) {
            Twin *t = &d->twins[i];
            free(t->propertyName);
            free(t->observedDesired.value);
            free(t->observedDesired.metadata.timestamp);
            free(t->observedDesired.metadata.type);
            free(t->reported.value);
            free(t->reported.metadata.timestamp);
            free(t->reported.metadata.type);
        }
        free(d->twins);
    }
    memset(d, 0, sizeof(*d));
}

class DeviceMapperServiceImpl final : public v1beta1::DeviceMapperService::Service
{
public:
    explicit DeviceMapperServiceImpl(std::shared_ptr<DevPanel> devPanel) : devPanel_(devPanel) {}

    ::grpc::Status RegisterDevice(::grpc::ServerContext *context,
                                  const ::v1beta1::RegisterDeviceRequest *request,
                                  ::v1beta1::RegisterDeviceResponse *response) override
    {
        if (!request || !request->has_device())
            return ::grpc::Status(::grpc::StatusCode::INVALID_ARGUMENT, "empty device");
        const auto &dev = request->device();
        log_info("RegisterDevice: ns=%s name=%s", dev.namespace_().c_str(), dev.name().c_str());
        DeviceModel mdl;
        DeviceInstance inst;
        build_model_min(dev, &mdl);
        build_instance_min(dev, &inst);
        int rc = dev_panel_update_dev(panel_get_manager(), &mdl, &inst);
        free_model_min(&mdl);
        free_instance_min(&inst);
        if (response)
        {
            response->set_devicename(dev.name());
            response->set_devicenamespace(dev.namespace_());
        }
        return rc == 0 ? ::grpc::Status::OK
                       : ::grpc::Status(::grpc::StatusCode::INTERNAL, "register failed");
    }

    ::grpc::Status RemoveDevice(::grpc::ServerContext *context,
                                const ::v1beta1::RemoveDeviceRequest *request,
                                ::v1beta1::RemoveDeviceResponse *response) override
    {
        if (!request)
            return ::grpc::Status(::grpc::StatusCode::INVALID_ARGUMENT, "empty request");
        const std::string ns = request->devicenamespace();
        const std::string name = request->devicename();
        if (name.empty())
            return ::grpc::Status(::grpc::StatusCode::INVALID_ARGUMENT, "empty name");
        int rc = dev_panel_remove_dev(panel_get_manager(), ns.c_str(), name.c_str());
        log_info("RemoveDevice: ns=%s name=%s rc=%d", ns.c_str(), name.c_str(), rc);
        return ::grpc::Status::OK;
    }

    ::grpc::Status UpdateDevice(::grpc::ServerContext *context,
                                const ::v1beta1::UpdateDeviceRequest *request,
                                ::v1beta1::UpdateDeviceResponse *response) override
    {
        log_info("grpc UpdateDevice: name=%s", request->device().name().c_str());
        if (!request || !request->has_device())
            return ::grpc::Status(::grpc::StatusCode::INVALID_ARGUMENT, "empty request");
        const auto &dev = request->device();
        log_info("UpdateDevice called: name=%s ns=%s has_spec=%d props=%d",
                 dev.name().c_str(), dev.namespace_().c_str(),
                 dev.has_spec(), dev.has_spec() ? dev.spec().properties_size() : 0);
        {
            DeviceModel mdl;
            DeviceInstance inst;
            build_model_min(dev, &mdl);
            build_instance_min(dev, &inst);
            // submit updated instance (including desired copied above) to dev_panel
            (void)dev_panel_update_dev(panel_get_manager(), &mdl, &inst);
            free_model_min(&mdl);
            free_instance_min(&inst);
        }

        // desired values are applied by device layer when processing twins (device_deal_twin -> SetDeviceData)
        return ::grpc::Status::OK;
    }

    ::grpc::Status CreateDeviceModel(::grpc::ServerContext *context,
                                     const ::v1beta1::CreateDeviceModelRequest *request,
                                     ::v1beta1::CreateDeviceModelResponse *response) override
    {
        if (!request || !request->has_model())
            return ::grpc::Status(::grpc::StatusCode::INVALID_ARGUMENT, "empty model");
        const auto &m = request->model();
        log_info("CreateDeviceModel: ns=%s name=%s", m.namespace_().c_str(), m.name().c_str());

        DeviceModel mdl;
        memset(&mdl, 0, sizeof(mdl));
        if (!m.name().empty())
            mdl.name = strdup(m.name().c_str());
        if (!m.namespace_().empty())
            mdl.namespace_ = strdup(m.namespace_().c_str());
        int rc = dev_panel_update_model(panel_get_manager(), &mdl);

        free(mdl.name);
        free(mdl.namespace_);
        free(mdl.description);

        if (rc == 0)
        {
            if (response)
            {
                response->set_devicemodelname(m.name());
                response->set_devicemodelnamespace(m.namespace_());
            }
            return ::grpc::Status::OK;
        }
        log_error("CreateDeviceModel failed for %s/%s rc=%d", m.namespace_().c_str(), m.name().c_str(), rc);
        return ::grpc::Status(::grpc::StatusCode::INTERNAL, "create model failed");
    }

    ::grpc::Status RemoveDeviceModel(::grpc::ServerContext *context,
                                     const ::v1beta1::RemoveDeviceModelRequest *request,
                                     ::v1beta1::RemoveDeviceModelResponse *response) override
    {
        if (!request)
            return ::grpc::Status(::grpc::StatusCode::INVALID_ARGUMENT, "empty request");
        const std::string ns = request->modelnamespace();
        const std::string name = request->modelname();
        if (name.empty())
            return ::grpc::Status(::grpc::StatusCode::INVALID_ARGUMENT, "empty name");
        std::string id = ns.empty() ? name : (ns + "/" + name);
        log_info("RemoveDeviceModel: ns=%s name=%s id=%s", ns.c_str(), name.c_str(), id.c_str());
        int rc = dev_panel_remove_model(panel_get_manager(), id.c_str());
        if (rc == 0)
            return ::grpc::Status::OK;
        return ::grpc::Status(::grpc::StatusCode::NOT_FOUND, "model not found");
    }

    ::grpc::Status UpdateDeviceModel(::grpc::ServerContext *context,
                                     const ::v1beta1::UpdateDeviceModelRequest *request,
                                     ::v1beta1::UpdateDeviceModelResponse *response) override
    {
        if (!request || !request->has_model())
            return ::grpc::Status(::grpc::StatusCode::INVALID_ARGUMENT, "empty model");
        const auto &m = request->model();
        log_info("UpdateDeviceModel: ns=%s name=%s", m.namespace_().c_str(), m.name().c_str());

        DeviceModel mdl;
        memset(&mdl, 0, sizeof(mdl));
        if (!m.name().empty())
            mdl.name = strdup(m.name().c_str());
        if (!m.namespace_().empty())
            mdl.namespace_ = strdup(m.namespace_().c_str());

        int rc = dev_panel_update_model(panel_get_manager(), &mdl);
        free(mdl.name);
        free(mdl.namespace_);
        free(mdl.description);
        if (rc == 0)
            return ::grpc::Status::OK;
        log_error("UpdateDeviceModel failed for %s/%s rc=%d", m.namespace_().c_str(), m.name().c_str(), rc);
        return ::grpc::Status(::grpc::StatusCode::INTERNAL, "update model failed");
    }

    ::grpc::Status GetDevice(::grpc::ServerContext *context,
                             const ::v1beta1::GetDeviceRequest *request,
                             ::v1beta1::GetDeviceResponse *response) override
    {
        log_info("GetDevice called");
        return ::grpc::Status::OK;
    }

private:
    std::shared_ptr<DevPanel> devPanel_;
};

ServerConfig::ServerConfig(const std::string &sock_path, const std::string &protocol)
    : sockPath(sock_path), protocol(protocol) {}

GrpcServer::GrpcServer(const ServerConfig &cfg, std::shared_ptr<DevPanel> devPanel)
    : cfg_(cfg), devPanel_(devPanel) {}

int GrpcServer::Start()
{

    struct stat st;
    if (stat(cfg_.sockPath.c_str(), &st) == 0)
    {
        if (unlink(cfg_.sockPath.c_str()) != 0)
        {
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
    if (!server_)
    {
        log_error("failed to start grpc server");
        return -1;
    }

    server_->Wait();
    return 0;
}


void GrpcServer::Stop()
{
    if (stopped_)
        return;
    stopped_ = true;
    if (server_)
    {
        server_->Shutdown();
    }
}

extern "C"
{

    ServerConfig *server_config_new(const char *sock_path, const char *protocol)
    {
        if (!sock_path || !protocol)
        {
            log_error("Invalid parameters for server config creation");
            return nullptr;
        }

        try
        {
            return new ServerConfig(std::string(sock_path), std::string(protocol));
        }
        catch (const std::exception &e)
        {
            log_error("Failed to create server config: %s", e.what());
            return nullptr;
        }
    }

    void server_config_free(ServerConfig *config)
    {
        if (config)
        {
            delete config;
        }
    }

    GrpcServer *grpcserver_new(ServerConfig *config, DeviceManager *device_manager)
    {
        if (!config || !device_manager)
        {
            log_error("Invalid parameters for gRPC server creation");
            return nullptr;
        }
        g_device_manager = device_manager;

        try
        {
            auto devPanel = std::make_shared<DevPanel>();

            return new GrpcServer(*config, devPanel);
        }
        catch (const std::exception &e)
        {
            log_error("Failed to create gRPC server: %s", e.what());
            return nullptr;
        }
    }

    int grpcserver_start(GrpcServer *server)
    {
        if (!server)
        {
            log_error("Invalid gRPC server pointer");
            return -1;
        }

        try
        {
            return server->Start();
        }
        catch (const std::exception &e)
        {
            log_error("Failed to start gRPC server: %s", e.what());
            return -1;
        }
    }

    void grpcserver_stop(GrpcServer *server)
    {
        if (!server)
        {
            log_warn("Trying to stop NULL gRPC server");
            return;
        }
        try
        {
            server->Stop();
        }
        catch (const std::exception &e)
        {
            log_error("Failed to stop gRPC server: %s", e.what());
        }
    }

    void grpcserver_free(GrpcServer *server)
    {
        if (server)
        {
            delete server;
        }
    }

} // extern "C"