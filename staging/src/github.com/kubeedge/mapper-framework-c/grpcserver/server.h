#ifndef GRPC_SERVER_H
#define GRPC_SERVER_H

#ifdef __cplusplus
// C++ section
#include <string>
#include <memory>

// Device panel class declaration
class DevPanel;

// Configuration structure for the gRPC server
struct ServerConfig {
    std::string sockPath;  // Path to the Unix domain socket
    std::string protocol;  // Protocol name
    ServerConfig(const std::string& sock_path, const std::string& protocol);
};

// gRPC server wrapper
class GrpcServer {
public:
    GrpcServer(const ServerConfig& cfg, std::shared_ptr<DevPanel> devPanel);
    int Start();  // Starts the gRPC server
    void Stop();  // Stops the gRPC server
private:
    ServerConfig cfg_;  // Server configuration
    std::shared_ptr<DevPanel> devPanel_;  // Device panel instance
    std::unique_ptr<grpc::Server> server_;  // gRPC server instance
    bool stopped_ = false;  // Prevents multiple stops
};

#else
// C section - Provides interfaces for C code
typedef struct ServerConfig ServerConfig;
typedef struct GrpcServer GrpcServer;

// C interface functions
ServerConfig *server_config_new(const char *sock_path, const char *protocol);
void server_config_free(ServerConfig *config);

GrpcServer *grpcserver_new(ServerConfig *config, DeviceManager *device_manager);
int grpcserver_start(GrpcServer *server);
void grpcserver_stop(GrpcServer *server);
void grpcserver_free(GrpcServer *server);

#endif // __cplusplus

#endif // GRPC_SERVER_H