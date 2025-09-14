#include "grpcclient/register.h"
#include <grpcpp/grpcpp.h>
#include <sys/un.h>
#include <unistd.h>
#include <memory>
#include <string>
#include <vector>
#include <cstring>
#include <chrono>

#include "config/config.h"
#include "common/const.h"
#include "dmi/v1beta1/api.grpc.pb.h"
#include "dmi/v1beta1/api.pb-c.h"
#include "util/parse/grpc.h"
#include "common/datamodel.h"
#include "log/log.h"

using namespace std;

// Constructs a Unix Domain Socket (UDS) address with the appropriate scheme
static inline std::string uds_with_scheme(const char* p) {
    if (!p || !*p) return "unix:///tmp/mapper_dmi.sock";
    std::string s(p);
    if (s.rfind("unix://", 0) == 0) return s; // Already has the scheme
    if (!s.empty() && s[0] == '/') return "unix://" + s; // Absolute path
    return "unix:///" + s; // Relative path
}

// Registers the mapper with the edgecore service using gRPC
static int RegisterMapperCpp(
    bool withData,
    std::vector<v1beta1::Device> &deviceList,
    std::vector<v1beta1::DeviceModel> &modelList
) {
    const char* env_sock = getenv("EDGECORE_SOCK");
    std::string sock_path;
    Config *cfg = nullptr;

    if (env_sock && *env_sock) {
        sock_path = env_sock;
    } else {
        // Attempt to load configuration from config.yaml or ../config.yaml
        cfg = config_parse("config.yaml");
        if (!cfg) cfg = config_parse("../config.yaml");
        if (!cfg) {
            log_error("RegisterMapper: config.yaml not found (tried ./ and ../)");
            return -1;
        }
        if (cfg->common.edgecore_sock[0]) {
            sock_path = cfg->common.edgecore_sock;
        } else {
            sock_path = "/var/lib/kubeedge/kubeedge.sock";
        }
    }

    // Create a gRPC channel to the edgecore service
    std::string sock_addr = uds_with_scheme(sock_path.c_str());
    auto channel = grpc::CreateChannel(sock_addr, grpc::InsecureChannelCredentials());
    auto stub = v1beta1::DeviceManagerService::NewStub(channel);

    // Prepare the mapper information
    v1beta1::MapperInfo mapper;
    if (!cfg) {
        mapper.set_name("arduino-mapper");
        mapper.set_version("v1.13.0");
        mapper.set_api_version("v1.0.0");
        mapper.set_protocol("modbus-tcp");
        mapper.set_address("/tmp/mapper_dmi.sock");
    } else {
        mapper.set_name(cfg->common.name);
        mapper.set_version(cfg->common.version);
        mapper.set_api_version(cfg->common.api_version);
        const char* proto = cfg->common.protocol[0] ? cfg->common.protocol : "modbus-tcp";
        mapper.set_protocol(proto);
        mapper.set_address(cfg->grpc_server.socket_path);
    }

    log_info("RegisterMapper: edgecore=%s, mapper_addr=%s, protocol=%s, name=%s",
             sock_addr.c_str(), mapper.address().c_str(), mapper.protocol().c_str(), mapper.name().c_str());

    mapper.set_state(DEVICE_STATUS_OK);

    // Prepare the registration request
    v1beta1::MapperRegisterRequest req;
    req.set_withdata(withData);
    *req.mutable_mapper() = mapper;

    v1beta1::MapperRegisterResponse resp;
    grpc::ClientContext ctx;
    ctx.set_deadline(std::chrono::system_clock::now() + std::chrono::seconds(5)); // Set a 5-second timeout
    grpc::Status status = stub->MapperRegister(&ctx, req, &resp);

    if (!status.ok()) {
        log_error("MapperRegister RPC failed: code=%d msg=%s",
                  (int)status.error_code(), status.error_message().c_str());
        if (cfg) config_free(cfg);
        return -1;
    }

    log_info("MapperRegister ok: devices=%d, models=%d",
             resp.devicelist_size(), resp.modellist_size());

    deviceList.assign(resp.devicelist().begin(), resp.devicelist().end());
    modelList.assign(resp.modellist().begin(), resp.modellist().end());
    if (cfg) config_free(cfg);
    return 0;
}

// C-style wrapper for RegisterMapperCpp
// Converts the C++ vectors into C-style arrays
extern "C" int RegisterMapper(
    int withData,
    DeviceInstance **outDeviceList, int *outDeviceCount,
    DeviceModel **outModelList, int *outModelCount
) {
    std::vector<v1beta1::Device> devList;
    std::vector<v1beta1::DeviceModel> mdlList;
    int ret = RegisterMapperCpp(withData != 0, devList, mdlList);
    if (ret != 0) return ret;

    // Convert device list to C-style array
    if (outDeviceList && outDeviceCount) {
        *outDeviceCount = devList.size();
        *outDeviceList = (DeviceInstance*)calloc(*outDeviceCount, sizeof(DeviceInstance));
        for (int i = 0; i < *outDeviceCount; ++i) {
            std::string buf;
            if (!devList[i].SerializeToString(&buf)) {
                log_error("Serialize device[%d] failed", i);
                continue;
            }
            V1beta1__Device* pb_dev =
                v1beta1__device__unpack(NULL, buf.size(), (const uint8_t*)buf.data());
            if (!pb_dev) {
                log_error("Unpack device[%d] failed", i);
                continue;
            }
            get_device_from_grpc(pb_dev, NULL, &((*outDeviceList)[i]));
            v1beta1__device__free_unpacked(pb_dev, NULL);
        }
    }

    // Convert model list to C-style array
    if (outModelList && outModelCount) {
        *outModelCount = mdlList.size();
        *outModelList = (DeviceModel*)calloc(*outModelCount, sizeof(DeviceModel));
        for (int i = 0; i < *outModelCount; ++i) {
            std::string buf;
            if (!mdlList[i].SerializeToString(&buf)) {
                log_error("Serialize model[%d] failed", i);
                continue;
            }
            V1beta1__DeviceModel* pb_model =
                v1beta1__device_model__unpack(NULL, buf.size(), (const uint8_t*)buf.data());
            if (!pb_model) {
                log_error("Unpack model[%d] failed", i);
                continue;
            }
            get_device_model_from_grpc(pb_model, &((*outModelList)[i]));
            v1beta1__device_model__free_unpacked(pb_model, NULL);
        }
    }
    return 0;
}