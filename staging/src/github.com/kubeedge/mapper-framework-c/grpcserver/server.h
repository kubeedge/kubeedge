#ifndef GRPC_SERVER_H
#define GRPC_SERVER_H

#ifdef __cplusplus
#include <string>
#include <memory>

// Device panel class declaration
class DevPanel;

// Configuration structure for the gRPC server
struct ServerConfig
{
    std::string sockPath;
    std::string protocol;
    ServerConfig(const std::string &sock_path, const std::string &protocol);
};

// gRPC server wrapper
class GrpcServer
{
public:
    GrpcServer(const ServerConfig &cfg, std::shared_ptr<DevPanel> devPanel);
    int Start();
    void Stop();

private:
    ServerConfig cfg_;
    std::shared_ptr<DevPanel> devPanel_;
    std::unique_ptr<grpc::Server> server_;
    bool stopped_ = false;
};

#else
typedef struct ServerConfig ServerConfig;
typedef struct GrpcServer GrpcServer;

ServerConfig *server_config_new(const char *sock_path, const char *protocol);
void server_config_free(ServerConfig *config);

GrpcServer *grpcserver_new(ServerConfig *config, DeviceManager *device_manager);
int grpcserver_start(GrpcServer *server);
void grpcserver_stop(GrpcServer *server);
void grpcserver_free(GrpcServer *server);

#endif // __cplusplus

#endif // GRPC_SERVER_H