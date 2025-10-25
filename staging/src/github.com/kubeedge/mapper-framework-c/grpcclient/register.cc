#include "grpcclient/register.h"
#include <grpcpp/grpcpp.h>
#include "dmi/v1beta1/api.grpc.pb.h"
#include "log/log.h"
#include <vector>
#include <string>
#include <cstdlib>
#include <memory>
#include <chrono>
#include <map>
#include <thread>
#include <mutex>
#include <condition_variable>
#include <atomic>
#include <cerrno>
#include <algorithm>

#if defined(__GNUC__)
#define UNUSED __attribute__((unused))
#else
#define UNUSED
#endif

static void rl_acquire_token();

extern "C" {
#include "config/config.h"
#include "util/parse/grpc.h"
#include "dmi/v1beta1/api.pb-c.h"
}

static std::string g_dmi_sock_override;
static std::shared_ptr<v1beta1::DeviceManagerService::Stub> g_stub;
static std::mutex g_twin_mu;
struct TwinRate { std::string key; long long last_ms = 0; };
static TwinRate g_twin_rates[128];
static std::mutex g_twin_val_mu;
struct TwinLast { std::string key; std::string val; };
static TwinLast g_twin_last[128];

struct TwinBatchItem { std::string val; std::string typ; };
struct TwinBatch { std::string devkey; long long last_ms = 0; std::map<std::string, TwinBatchItem> kv; };
static std::mutex g_batch_mu;
static TwinBatch g_batches[32];
static std::condition_variable g_batch_cv;
static std::mutex g_batch_cv_mu;
static std::atomic<bool> g_batch_loop_started{false};
static std::atomic<bool> g_batch_stop{false};
static std::thread g_batch_thread;
static std::once_flag g_atexit_once;
static int g_twin_start_delay_ms = 1500;
static const long long g_batch_start_ms = []{
    return std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::steady_clock::now().time_since_epoch()).count();
}();
static bool g_disable_twin = false;
static int g_twin_min_interval_ms = 1000;
static int g_twin_jitter_ms = 30;
static int g_twin_max_failures = 5;
static std::atomic<int> g_twin_failures{0};

static inline std::string uds_with_scheme(const char *p);
static void batch_flush_loop();

static void batch_flush_stop() {
    g_batch_stop.store(true);
    g_batch_cv.notify_all();
    if (g_batch_thread.joinable()) g_batch_thread.join();
    g_stub.reset();
}

static void on_exit_cleanup() { batch_flush_stop(); }

extern "C" void register_set_dmi_sock(const char *sock_path) {
    if (sock_path && *sock_path) {
        g_dmi_sock_override = sock_path;
        g_stub.reset();
    }
    if (!g_batch_loop_started.exchange(true)) {
        g_batch_thread = std::thread(batch_flush_loop);
        std::call_once(g_atexit_once, [](){ atexit(on_exit_cleanup); });
    }
}

static std::shared_ptr<v1beta1::DeviceManagerService::Stub> DeviceManagerServiceClient() {
    if (g_stub) return g_stub;
    std::string path = g_dmi_sock_override;
    if (path.empty()) {
        const char *env = getenv("MAPPER_DMI_SOCK");
        if (env && *env) path = env;
    }
    if (path.empty()) {
        log_error("DMI sock not set. Set common.edgecore_sock in config.yaml or env MAPPER_DMI_SOCK");
        return nullptr;
    }
    std::string addr = uds_with_scheme(path.c_str());
    if (addr.empty()) {
        log_error("Invalid DMI sock path");
        return nullptr;
    }
    grpc::ChannelArguments args;
    args.SetInt(GRPC_ARG_KEEPALIVE_TIME_MS, 30000);
    args.SetInt(GRPC_ARG_KEEPALIVE_TIMEOUT_MS, 10000);
    args.SetInt(GRPC_ARG_HTTP2_MIN_RECV_PING_INTERVAL_WITHOUT_DATA_MS, 30000);
    args.SetInt(GRPC_ARG_HTTP2_MAX_PINGS_WITHOUT_DATA, 0);
    args.SetInt(GRPC_ARG_MAX_RECONNECT_BACKOFF_MS, 2000);
    auto channel = grpc::CreateCustomChannel(addr, grpc::InsecureChannelCredentials(), args);
    g_stub.reset(v1beta1::DeviceManagerService::NewStub(channel).release());
    return g_stub;
}

static UNUSED std::unique_ptr<v1beta1::DeviceManagerService::Stub> NewStatusClientOnce() {
    std::string path = g_dmi_sock_override;
    if (path.empty()) {
        const char *env = getenv("MAPPER_DMI_SOCK");
        if (env && *env) path = env;
    }
    if (path.empty()) return nullptr;
    std::string addr = uds_with_scheme(path.c_str());
    if (addr.empty()) return nullptr;
    grpc::ChannelArguments args;
    args.SetInt(GRPC_ARG_MAX_RECONNECT_BACKOFF_MS, 1000);
    auto channel = grpc::CreateCustomChannel(addr, grpc::InsecureChannelCredentials(), args);
    return v1beta1::DeviceManagerService::NewStub(channel);
}

static inline std::string uds_with_scheme(const char *p)
{
    if (!p || !*p)
        return std::string();
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
        if (cfg->common.edgecore_sock[0]) {
            sock_path = cfg->common.edgecore_sock;
        } else {
            log_error("RegisterMapper: common.edgecore_sock not set");
            config_free(cfg);
            return -1;
        }
    }

    std::string sock_addr = uds_with_scheme(sock_path.c_str());
    if (sock_addr.empty()) {
        log_error("RegisterMapper: invalid EDGECORE_SOCK/common.edgecore_sock");
        if (cfg) config_free(cfg);
        return -1;
    }
    auto channel = grpc::CreateChannel(sock_addr, grpc::InsecureChannelCredentials());
    auto stub = v1beta1::DeviceManagerService::NewStub(channel);

    v1beta1::MapperInfo mapper;
    mapper.set_name(cfg->common.name);
    mapper.set_version(cfg->common.version);
    mapper.set_api_version(cfg->common.api_version);
    const char *proto = cfg->common.protocol[0] ? cfg->common.protocol : "modbus-tcp";
    mapper.set_protocol(proto);
    mapper.set_address(cfg->grpc_server.socket_path);

    mapper.set_state("OK");

    v1beta1::MapperRegisterRequest req;
    req.set_withdata(withData);
    *req.mutable_mapper() = mapper;

    v1beta1::MapperRegisterResponse resp;
    grpc::ClientContext ctx;
    ctx.set_deadline(std::chrono::system_clock::now() + std::chrono::seconds(5));
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
    if (!namespace_ || !deviceName || !status)
        return -1;
    v1beta1::ReportDeviceStatesRequest req;
    req.set_devicenamespace(namespace_);
    req.set_devicename(deviceName);
    req.set_state(status);
    v1beta1::ReportDeviceStatesResponse rsp;
    auto stub = DeviceManagerServiceClient();
    if (!stub) {
        log_error("ReportDeviceStatus: DMI client not initialized (no sock)");
        return -1;
    }
    int attempts = 0;
    bool ok = false;
    int last_code = 0;
    std::string last_msg;
    while (attempts < 2 && !ok) {
        attempts++;
        rl_acquire_token();
        grpc::ClientContext c;
        c.set_deadline(std::chrono::system_clock::now() + std::chrono::seconds(5));
        v1beta1::ReportDeviceStatesResponse r;
        auto s = stub->ReportDeviceStates(&c, req, &r);
        if (s.ok()) { ok = true; break; }
        last_code = (int)s.error_code();
        last_msg = s.error_message();
        if (last_code == (int)grpc::StatusCode::UNKNOWN && last_msg.find("too many request") != std::string::npos) {
            std::this_thread::sleep_for(std::chrono::milliseconds(250));
            continue;
        }
        break;
    }
    if (!ok) {
        return -1;
    }
    return 0;
}

int ReportDeviceStates(const char *namespace_, const char *deviceName, const char *state)
{
    if (!namespace_ || !deviceName || !state)
        return -1;
    v1beta1::ReportDeviceStatesRequest req;
    req.set_devicenamespace(namespace_);
    req.set_devicename(deviceName);
    req.set_state(state);
    auto stub = DeviceManagerServiceClient();
    if (!stub) {
        log_error("ReportDeviceStates: DMI client not initialized (no sock)");
        return -1;
    }
    int attempts = 0;
    bool ok = false;
    int last_code = 0;
    std::string last_msg;
    while (attempts < 2 && !ok) {
        attempts++;
        rl_acquire_token();
        grpc::ClientContext c;
        c.set_deadline(std::chrono::system_clock::now() + std::chrono::seconds(5));
        v1beta1::ReportDeviceStatesResponse r;
        auto s = stub->ReportDeviceStates(&c, req, &r);
        if (s.ok()) { ok = true; break; }
        last_code = (int)s.error_code();
        last_msg = s.error_message();
        if (last_code == (int)grpc::StatusCode::UNKNOWN && last_msg.find("too many request") != std::string::npos) {
            std::this_thread::sleep_for(std::chrono::milliseconds(250));
            continue;
        }
        break;
    }

    return 0;
}

static inline long long now_ms() {
    return std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::steady_clock::now().time_since_epoch()).count();
}
static int g_dmi_max_rps = []{
    const char* x = getenv("MAPPER_DMI_MAX_RPS");
    int v = x && *x ? atoi(x) : 5;
    if (v < 1) v = 1;
    if (v > 1000) v = 1000;
    return v;
}();
static std::mutex g_rl_mu;
static int g_rl_tokens = g_dmi_max_rps;
static long long g_rl_window_start_ms = 0;
static void rl_acquire_token() {
    for (;;) {
        const long long t = now_ms();
        std::unique_lock<std::mutex> lk(g_rl_mu);
        if (g_rl_window_start_ms == 0) g_rl_window_start_ms = t;
        if (t - g_rl_window_start_ms >= 1000) {
            g_rl_window_start_ms = t;
            g_rl_tokens = g_dmi_max_rps;
        }
        if (g_rl_tokens > 0) {
            --g_rl_tokens;
            return;
        }
        long long wait_ms = 1000 - (t - g_rl_window_start_ms);
        lk.unlock();
        if (wait_ms < 1) wait_ms = 1;
        std::this_thread::sleep_for(std::chrono::milliseconds((int)wait_ms));
    }
}

static inline const char* infer_type(const std::string& v) {
    if (v == "true" || v == "false" || v == "TRUE" || v == "FALSE") return "bool";
    char *end = nullptr;
    errno = 0;
    (void)strtol(v.c_str(), &end, 10);
    if (end && *end == '\0' && errno == 0) return "int";
    errno = 0;
    end = nullptr;
    (void)strtod(v.c_str(), &end);
    if (end && *end == '\0' && errno == 0) return "float";
    return "string";
}

static UNUSED bool allow_send_twin(const std::string& key, int min_interval_ms) {
    std::lock_guard<std::mutex> lk(g_twin_mu);
    long long t = now_ms();
    for (auto &e : g_twin_rates) {
        if (e.key == key) {
            if (t - e.last_ms < min_interval_ms) return false;
            e.last_ms = t;
            return true;
        }
    }
    for (auto &e : g_twin_rates) {
        if (e.key.empty()) { e.key = key; e.last_ms = t; return true; }
    }
    g_twin_rates[0].key = key; g_twin_rates[0].last_ms = t; return true;
}

static bool value_changed(const std::string& key, const char *val) {
    std::lock_guard<std::mutex> lk(g_twin_val_mu);
    const std::string v = val ? val : "";
    for (auto &e : g_twin_last) {
        if (e.key == key) {
            if (e.val == v) return false;
            e.val = v; return true;
        }
    }
    for (auto &e : g_twin_last) {
        if (e.key.empty()) { e.key = key; e.val = v; return true; }
    }
    g_twin_last[0].key = key; g_twin_last[0].val = v; return true;
}

static TwinBatch* get_batch(const std::string& devkey) {
    for (auto &b : g_batches) if (b.devkey == devkey) return &b;
    for (auto &b : g_batches) if (b.devkey.empty()) { b.devkey = devkey; return &b; }
    return &g_batches[0];
}

static UNUSED int flush_batch_locked(const std::string& ns, const std::string& dev, TwinBatch* b) {
    if (!b || b->kv.empty()) return 0;
    v1beta1::ReportDeviceStatusRequest req;
    req.set_devicename(dev);
    req.set_devicenamespace(ns);
    auto ds = req.mutable_reporteddevice();
    for (auto &it : b->kv) {
        auto twin = ds->add_twins();
        twin->set_propertyname(it.first);
        auto rep = twin->mutable_reported();
        rep->set_value(it.second.val);
        const std::string ty = it.second.typ.empty() ? "string" : it.second.typ;
        (*rep->mutable_metadata())["type"] = ty;
        (*rep->mutable_metadata())["timestamp"] = std::to_string(now_ms());
        auto des = twin->mutable_observeddesired();
        des->set_value(it.second.val);
        (*des->mutable_metadata())["type"] = ty;
        (*des->mutable_metadata())["timestamp"] = std::to_string(now_ms());
    }
    v1beta1::ReportDeviceStatusResponse rsp;
    grpc::ClientContext ctx;
    ctx.set_deadline(std::chrono::system_clock::now() + std::chrono::seconds(5));
    auto stub = DeviceManagerServiceClient();
    if (!stub) return -1;
    auto s = stub->ReportDeviceStatus(&ctx, req, &rsp);
    if (!s.ok()) {
        return -1;
    }
    b->kv.clear();
    b->last_ms = now_ms();
    return 0;
}

static inline void split_devkey(const std::string& devkey, std::string& ns, std::string& dev) {
    auto pos = devkey.find('|');
    if (pos == std::string::npos) { ns = "default"; dev = devkey; return; }
    ns = devkey.substr(0, pos);
    dev = devkey.substr(pos + 1);
}

static void batch_flush_loop() {
    while (!g_batch_stop.load()) {
        std::unique_lock<std::mutex> lk(g_batch_cv_mu);
        g_batch_cv.wait_for(lk, std::chrono::milliseconds(200));
        lk.unlock();
        if (g_disable_twin) {
            std::this_thread::sleep_for(std::chrono::milliseconds(200));
            continue;
        }
        struct Shot { std::string devkey; std::map<std::string, TwinBatchItem> kv; };
        std::vector<Shot> shots;
        const long long tnow = now_ms();
        if (tnow - g_batch_start_ms < g_twin_start_delay_ms) {
            continue;
        }
        {
            std::lock_guard<std::mutex> bl(g_batch_mu);
            for (auto &b : g_batches) {
                if (b.devkey.empty() || b.kv.empty()) continue;
                if (tnow - b.last_ms < g_twin_min_interval_ms) continue;
                shots.push_back(Shot{b.devkey, b.kv});
            }
        }
        for (auto &shot : shots) {
            std::string ns, dev;
            split_devkey(shot.devkey, ns, dev);
            v1beta1::ReportDeviceStatusRequest req;
            req.set_devicenamespace(ns);
            req.set_devicename(dev);
            auto ds = req.mutable_reporteddevice();
            for (auto &kv : shot.kv) {
                auto twin = ds->add_twins();
                twin->set_propertyname(kv.first);
                auto rep = twin->mutable_reported();
                rep->set_value(kv.second.val);
                const char* ty = kv.second.typ.empty() ? infer_type(kv.second.val) : kv.second.typ.c_str();
                (*rep->mutable_metadata())["type"] = ty;
                (*rep->mutable_metadata())["timestamp"] = std::to_string(now_ms());
                auto des = twin->mutable_observeddesired();
                des->set_value(kv.second.val);
                (*des->mutable_metadata())["type"] = ty;
                (*des->mutable_metadata())["timestamp"] = std::to_string(now_ms());
            }
            auto stub = DeviceManagerServiceClient();
            if (!stub) { log_error("ReportDeviceStatus(batch) no client"); continue; }
            int failures = g_twin_failures.load();
            if (failures > 0) {
                int backoff_ms = std::min(2000, 150 << std::min(failures, 5));
                std::this_thread::sleep_for(std::chrono::milliseconds(backoff_ms));
            }
            bool sent = false;
            int last_code = 0;
            std::string last_msg;
            for (int attempt = 0; attempt < 2 && !sent; ++attempt) {
                rl_acquire_token();
                grpc::ClientContext ctx;
                ctx.set_deadline(std::chrono::system_clock::now() + std::chrono::seconds(10));
                v1beta1::ReportDeviceStatusResponse rsp;
                auto s = stub->ReportDeviceStatus(&ctx, req, &rsp);
                if (s.ok()) { sent = true; break; }
                last_code = (int)s.error_code();
                last_msg = s.error_message();
                if (last_code == (int)grpc::StatusCode::UNKNOWN && last_msg.find("too many request") != std::string::npos) {
                    std::this_thread::sleep_for(std::chrono::milliseconds(250));
                    continue;
                }
                break;
            }
            if (sent) {
                 std::lock_guard<std::mutex> bl(g_batch_mu);
                 if (TwinBatch* b = get_batch(shot.devkey)) {
                     for (auto &kv : shot.kv) b->kv.erase(kv.first);
                     b->last_ms = now_ms();
                 }
                 g_twin_failures.store(0);
             } else {
                bool tmr = (last_code == (int)grpc::StatusCode::UNKNOWN && last_msg.find("too many request") != std::string::npos);
                if (!tmr) {
                    int f = std::min(g_twin_failures.load() + 1, 1000);
                    g_twin_failures.store(f);
                    if (f >= g_twin_max_failures) {
                        g_disable_twin = true;
                        log_error("Disable twins after %d consecutive failures to protect edgecore", f);
                    }
                }
                 std::this_thread::sleep_for(std::chrono::milliseconds(300));
             }
            if (g_twin_jitter_ms > 0) {
                std::this_thread::sleep_for(std::chrono::milliseconds(g_twin_jitter_ms));
            }
        }
    }
}

int ReportTwinKV(const char *namespace_, const char *deviceName,
                 const char *propertyName, const char *value, const char *valueType)
{
    if (g_disable_twin) return 0;
    if (!namespace_ || !*namespace_ || !deviceName || !*deviceName || !propertyName || !*propertyName)
        return -1;
    std::string devkey = std::string(namespace_) + "|" + deviceName;
    std::string rk = devkey + "|" + propertyName;
    if (!value_changed(rk, value)) return 0;
    {
        std::lock_guard<std::mutex> lk(g_batch_mu);
        auto *b = get_batch(devkey);
        TwinBatchItem &it = b->kv[propertyName];
        it.val = value ? value : "";
        if (valueType && *valueType) {
            it.typ = valueType;
        } else {
            it.typ = infer_type(it.val);
        }
    }
    g_batch_cv.notify_one();
    return 0;
}



