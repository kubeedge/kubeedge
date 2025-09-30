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
extern "C"
{
#include "data/publish/publisher.h"
}

extern Publisher *g_publisher;

using namespace std;

static inline std::string uds_with_scheme(const char *p)
{
    if (!p || !*p)
        return "unix:///tmp/mapper_dmi.sock";
    std::string s(p);
    if (s.rfind("unix://", 0) == 0)
        return s;
    if (!s.empty() && s[0] == '/')
        return "unix://" + s;
    return "unix:///" + s;
}

static int RegisterMapperCpp(
    bool withData,
    std::vector<v1beta1::Device> &deviceList,
    std::vector<v1beta1::DeviceModel> &modelList)
{
    const char *env_sock = getenv("EDGECORE_SOCK");
    std::string sock_path;
    Config *cfg = nullptr;

    if (env_sock && *env_sock)
    {
        sock_path = env_sock;
    }
    else
    {
        cfg = config_parse("config.yaml");
        if (!cfg)
            cfg = config_parse("../config.yaml");
        if (!cfg)
        {
            log_error("RegisterMapper: config.yaml not found (tried ./ and ../)");
            return -1;
        }
        if (cfg->common.edgecore_sock[0])
        {
            sock_path = cfg->common.edgecore_sock;
        }
        else
        {
            sock_path = "/var/lib/kubeedge/kubeedge.sock";
        }
    }

    std::string sock_addr = uds_with_scheme(sock_path.c_str());
    auto channel = grpc::CreateChannel(sock_addr, grpc::InsecureChannelCredentials());
    auto stub = v1beta1::DeviceManagerService::NewStub(channel);

    v1beta1::MapperInfo mapper;
    mapper.set_name(cfg->common.name);
    mapper.set_version(cfg->common.version);
    mapper.set_api_version(cfg->common.api_version);
    const char *proto = cfg->common.protocol[0] ? cfg->common.protocol : "modbus-tcp";
    mapper.set_protocol(proto);
    mapper.set_address(cfg->grpc_server.socket_path);

    mapper.set_state(DEVICE_STATUS_OK);

    v1beta1::MapperRegisterRequest req;
    req.set_withdata(withData);
    *req.mutable_mapper() = mapper;

    v1beta1::MapperRegisterResponse resp;
    grpc::ClientContext ctx;
    ctx.set_deadline(std::chrono::system_clock::now() + std::chrono::seconds(5)); // Set a 5-second timeout
    grpc::Status status = stub->MapperRegister(&ctx, req, &resp);

    if (!status.ok())
    {
        log_error("MapperRegister RPC failed: code=%d msg=%s",
                  (int)status.error_code(), status.error_message().c_str());
        if (cfg)
            config_free(cfg);
        return -1;
    }
    deviceList.assign(resp.devicelist().begin(), resp.devicelist().end());
    modelList.assign(resp.modellist().begin(), resp.modellist().end());
    if (cfg)
        config_free(cfg);
    return 0;
}

extern "C" int RegisterMapper(
    int withData,
    DeviceInstance **outDeviceList, int *outDeviceCount,
    DeviceModel **outModelList, int *outModelCount)
{
    std::vector<v1beta1::Device> devList;
    std::vector<v1beta1::DeviceModel> mdlList;
    int ret = RegisterMapperCpp(withData != 0, devList, mdlList);
    if (ret != 0)
        return ret;

    if (outDeviceList && outDeviceCount)
    {
        *outDeviceCount = devList.size();
        *outDeviceList = (DeviceInstance *)calloc(*outDeviceCount, sizeof(DeviceInstance));
        for (int i = 0; i < *outDeviceCount; ++i)
        {
            std::string buf;
            if (!devList[i].SerializeToString(&buf))
            {
                log_error("Serialize device[%d] failed", i);
                continue;
            }
            V1beta1__Device *pb_dev =
                v1beta1__device__unpack(NULL, buf.size(), (const uint8_t *)buf.data());
            if (!pb_dev)
            {
                log_error("Unpack device[%d] failed", i);
                continue;
            }
            get_device_from_grpc(pb_dev, NULL, &((*outDeviceList)[i]));
            v1beta1__device__free_unpacked(pb_dev, NULL);
        }
    }

    if (outModelList && outModelCount)
    {
        *outModelCount = mdlList.size();
        *outModelList = (DeviceModel *)calloc(*outModelCount, sizeof(DeviceModel));
        for (int i = 0; i < *outModelCount; ++i)
        {
            std::string buf;
            if (!mdlList[i].SerializeToString(&buf))
            {
                log_error("Serialize model[%d] failed", i);
                continue;
            }
            V1beta1__DeviceModel *pb_model =
                v1beta1__device_model__unpack(NULL, buf.size(), (const uint8_t *)buf.data());
            if (!pb_model)
            {
                log_error("Unpack model[%d] failed", i);
                continue;
            }
            get_device_model_from_grpc(pb_model, &((*outModelList)[i]));
            v1beta1__device_model__free_unpacked(pb_model, NULL);
        }
    }
    return 0;
}

int ReportDeviceStatus(const char *namespace_, const char *deviceName, const char *status)
{
    const char *ns = (namespace_ && *namespace_) ? namespace_ : "default";
    const char *dn = (deviceName && *deviceName) ? deviceName : "unknown";
    const char *st = (status && *status) ? status : "unknown";

    if (!g_publisher)
    {
        return -1;
    }

    DataModel dm{};
    dm.namespace_ = (char *)ns;
    dm.deviceName = (char *)dn;
    dm.propertyName = (char *)"status";
    dm.type = (char *)"string";
    dm.value = (char *)st;
    dm.timeStamp = (int64_t)time(NULL) * 1000;

    int rc = publisher_publish_data(g_publisher, &dm);
    if (rc != 0)
    {
        log_warn("ReportDeviceStatus publish failed: ns=%s device=%s status=%s rc=%d", ns, dn, st, rc);
    }
    else
    {
        log_info("ReportDeviceStatus ok: ns=%s device=%s status=%s", ns, dn, st);
    }
    return rc;
}