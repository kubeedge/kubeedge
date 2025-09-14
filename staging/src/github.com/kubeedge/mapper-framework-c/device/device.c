#include "device.h"
#include "log/log.h"
#include "common/const.h"
#include "data/publish/publisher.h"
#include "data/dbmethod/mysql/recorder.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <pthread.h>
#include <unistd.h>
#include <time.h>
#include <ctype.h>

extern Publisher *g_publisher;


// Function sim_temperature_enabled
static int sim_temperature_enabled(void) {
    const char *v = getenv("MAPPER_SIM_TEMPERATURE");
    return v && (*v == '1' || strcasecmp(v, "true") == 0 || strcasecmp(v, "on") == 0);
}


// Function json_get_str
static int json_get_str(const char *json, const char *key, char *out, size_t outsz) {
    if (!json || !key || !out || outsz == 0) return -1;
    const char *p = strcasestr(json, key);
    if (!p) return -1;
    p = strchr(p, ':');
    if (!p) return -1;
    p++;
    while (*p == ' ' || *p == '\t' || *p == '\r' || *p == '\n') p++;
    int quoted = 0;
    if (*p == '\"') { quoted = 1; p++; }
    size_t i = 0;
    while (*p && i + 1 < outsz) {
        if (quoted) {
            if (*p == '\\' && p[1]) p++;
// Function if
            else if (*p == '\"') break;
            out[i++] = *p++;
        } else {
            if (*p == ',' || *p == '}' || *p == ' ' || *p == '\r' || *p == '\n' || *p == '\t') break;
            out[i++] = *p++;
        }
    }
    out[i] = 0;
    return i > 0 ? 0 : -1;
}


// Function json_get_int
static int json_get_int(const char *json, const char *key, int *out) {
    char buf[32] = {0};
    if (json_get_str(json, key, buf, sizeof(buf)) == 0) { *out = atoi(buf); return 0; }
    return -1;
}


// Function now_iso8601
static void now_iso8601(char ts[32]) {
    time_t t = time(NULL);
    struct tm tm;
    gmtime_r(&t, &tm);
    strftime(ts, 32, "%Y-%m-%dT%H:%M:%SZ", &tm);
}


// Function resolve_offset_by_name
static int resolve_offset_by_name(const char *propName) {
    if (!propName) return -1;
    if (strcmp(propName, "temperature") == 0) return 1;
    if (strcmp(propName, "threshold") == 0) return 2;
    return -1;
}


// Function trim_str
static void trim_str(char *s) {
    if (!s) return;
    char *p = s;
    while (*p && isspace((unsigned char)*p)) p++;
    if (p != s) memmove(s, p, strlen(p) + 1);
    size_t len = strlen(s);
    while (len > 0 && isspace((unsigned char)s[len - 1])) {
        s[--len] = 0;
    }
}


// Function sanitize_host
static void sanitize_host(char *s) {
    if (!s) return;
    size_t w = 0;
    for (size_t r = 0; s[r]; ++r) {
        unsigned char c = (unsigned char)s[r];
        if ((c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
            c == '.' || c == '-' || c == '_' || c == ':') {
            s[w++] = (char)c;
        }
    }
    s[w] = 0;
}


// Function normalize_host_port
static void cleanup_escape_prefix(char *s) {
    if (!s) return;
    // 去掉开头若是 n/t/r 且后面紧跟数字或 1-2 个 n/t/r 再跟数字的情况
    while (s[0] && (s[0]=='n' || s[0]=='t' || s[0]=='r')) {
        if (s[1]>='0' && s[1]<='9') {
            memmove(s, s+1, strlen(s+1)+1);
            continue;
        }
        if ((s[1]=='n'||s[1]=='t'||s[1]=='r') && (s[2]>='0'&&s[2]<='9')) {
            memmove(s, s+2, strlen(s+2)+1);
            continue;
        }
        break;
    }
}
static void normalize_host_port(const char *rawHost, int rawPort,
                                char *outHost, size_t outHostSz, int *outPort) {
    snprintf(outHost, outHostSz, "%s", (rawHost && *rawHost) ? rawHost : "");
    trim_str(outHost);
    sanitize_host(outHost);
    if (outHost[0] == 0) {
        const char *envH = getenv("MAPPER_MODBUS_ADDR");
        if (envH && *envH) {
            snprintf(outHost, outHostSz, "%s", envH);
            trim_str(outHost);
            sanitize_host(outHost);
        }
    }
    if (outHost[0] == 0) {
        snprintf(outHost, outHostSz, "%s", "127.0.0.1");
    }
    int p = rawPort;
    if (p <= 0 || p > 65535) {
        const char *envp = getenv("MAPPER_MODBUS_PORT");
        if (envp && *envp) {
            int ep = atoi(envp);
            if (ep > 0 && ep <= 65535) p = ep;
        }
    }
    if (p <= 0 || p > 65535) p = 1502;
    *outPort = p;
}


// Function simulate_temperature_data
static int simulate_temperature_data(int *current, int *direction) {
    if (!current || !direction) return -1;
    if (*current >= 100) {
        *direction = -1;
    } else if (*current <= 1) {
        *direction = 1;
    }
    *current += *direction;
    return *current;
}
static void *device_data_thread(void *arg) {
    Device *device = (Device*)arg;
    log_info("Device data thread started for device: %s",
             device->instance.name ? device->instance.name : "unknown");

    int simulated_temperature = 1;
    int direction = 1;

    while (device->dataThreadRunning) {
        pthread_mutex_lock(&device->mutex);

        if (strcmp(device->status, DEVICE_STATUS_OK) != 0) {
            pthread_mutex_unlock(&device->mutex);
            usleep(1000000);
            continue;
        }

        for (int i = 0; i < device->instance.twinsCount; i++) {
            Twin *twin = &device->instance.twins[i];
            if (!twin || !twin->propertyName) continue;

            if (strcmp(twin->propertyName, "temperature") == 0 && sim_temperature_enabled()) {
                int simulated_value = simulate_temperature_data(&simulated_temperature, &direction);
                char buf[32];
                snprintf(buf, sizeof(buf), "%d", simulated_value);


                if (!twin->reported.value || strcmp(twin->reported.value, buf) != 0) {
                    free(twin->reported.value);
                    twin->reported.value = strdup(buf);
                    char ts[32];
                    now_iso8601(ts);
                    free(twin->reported.metadata.timestamp);
                    twin->reported.metadata.timestamp = strdup(ts);
                }


                mysql_recorder_record(
                    device->instance.namespace_ ? device->instance.namespace_ : "default",
                    device->instance.name ? device->instance.name : "unknown",
                    twin->propertyName,
                    buf,
                    (long long)time(NULL) * 1000
                );

                continue;
            }


            device_deal_twin(device, twin);
        }

        pthread_mutex_unlock(&device->mutex);
        usleep(5000000);
    }

    log_info("Device data thread stopped for device: %s",
             device->instance.name ? device->instance.name : "unknown");
    return NULL;
}


Device *device_new(const DeviceInstance *instance, const DeviceModel *model) {
    if (!instance || !model) {
        log_error("Invalid parameters for device creation");
        return NULL;
    }

    Device *device = calloc(1, sizeof(Device));
    if (!device) {
        log_error("Failed to allocate memory for device");
        return NULL;
    }


    memset(&device->instance, 0, sizeof(DeviceInstance));


    if (instance->id) device->instance.id = strdup(instance->id);
    if (instance->name) device->instance.name = strdup(instance->name);
    if (instance->namespace_) device->instance.namespace_ = strdup(instance->namespace_);
    if (!device->instance.namespace_ || !*device->instance.namespace_) {
        if (device->instance.namespace_) free(device->instance.namespace_);
        device->instance.namespace_ = strdup("default");
        log_debug("device_new: namespace not provided, default -> 'default' (device=%s)",
                  device->instance.name ? device->instance.name : "(nil)");
    }

    device->instance.namespace_ = strdup("test");
    if (instance->model) device->instance.model = strdup(instance->model);
    if (instance->protocolName) device->instance.protocolName = strdup(instance->protocolName);


    if (instance->pProtocol.protocolName) {
        device->instance.pProtocol.protocolName = strdup(instance->pProtocol.protocolName);
    }
    if (instance->pProtocol.configData) {
        device->instance.pProtocol.configData = strdup(instance->pProtocol.configData);
    }


    if (instance->twins && instance->twinsCount > 0) {
        device->instance.twinsCount = instance->twinsCount;
        device->instance.twins = calloc(instance->twinsCount, sizeof(Twin));

        for (int i = 0; i < instance->twinsCount; i++) {
            Twin *srcTwin = &instance->twins[i];
            Twin *dstTwin = &device->instance.twins[i];

            if (srcTwin->propertyName) {
                dstTwin->propertyName = strdup(srcTwin->propertyName);
            }
            if (srcTwin->observedDesired.value) {
                dstTwin->observedDesired.value = strdup(srcTwin->observedDesired.value);
            }
            if (srcTwin->observedDesired.metadata.timestamp) {
                dstTwin->observedDesired.metadata.timestamp = strdup(srcTwin->observedDesired.metadata.timestamp);
            }
            if (srcTwin->observedDesired.metadata.type) {
                dstTwin->observedDesired.metadata.type = strdup(srcTwin->observedDesired.metadata.type);
            }
            if (srcTwin->reported.value) {
                dstTwin->reported.value = strdup(srcTwin->reported.value);
            }
            if (srcTwin->reported.metadata.timestamp) {
                dstTwin->reported.metadata.timestamp = strdup(srcTwin->reported.metadata.timestamp);
            }
            if (srcTwin->reported.metadata.type) {
                dstTwin->reported.metadata.type = strdup(srcTwin->reported.metadata.type);
            }
            if (srcTwin->property) {
                dstTwin->property = malloc(sizeof(DeviceProperty));
                if (dstTwin->property) {
                    memcpy(dstTwin->property, srcTwin->property, sizeof(DeviceProperty));

                    if (srcTwin->property->name) {
                        dstTwin->property->name = strdup(srcTwin->property->name);
                    }
                }
            }
        }
    }


    if (instance->properties && instance->propertiesCount > 0) {
        device->instance.propertiesCount = instance->propertiesCount;
        device->instance.properties = calloc(instance->propertiesCount, sizeof(DeviceProperty));

        for (int i = 0; i < instance->propertiesCount; i++) {
            DeviceProperty *srcProp = &instance->properties[i];
            DeviceProperty *dstProp = &device->instance.properties[i];


            if (srcProp->name) dstProp->name = strdup(srcProp->name);
            if (srcProp->propertyName) dstProp->propertyName = strdup(srcProp->propertyName);
            if (srcProp->modelName) dstProp->modelName = strdup(srcProp->modelName);
            if (srcProp->protocol) dstProp->protocol = strdup(srcProp->protocol);


            dstProp->collectCycle = srcProp->collectCycle;
            dstProp->reportCycle = srcProp->reportCycle;
            dstProp->reportToCloud = srcProp->reportToCloud;
        }
    }


    if (instance->methods && instance->methodsCount > 0) {
        device->instance.methodsCount = instance->methodsCount;
        device->instance.methods = calloc(instance->methodsCount, sizeof(DeviceMethod));

        for (int i = 0; i < instance->methodsCount; i++) {
            DeviceMethod *srcMethod = &instance->methods[i];
            DeviceMethod *dstMethod = &device->instance.methods[i];

            if (srcMethod->name) dstMethod->name = strdup(srcMethod->name);
            if (srcMethod->description) dstMethod->description = strdup(srcMethod->description);


            if (srcMethod->propertyNames && srcMethod->propertyNamesCount > 0) {
                dstMethod->propertyNamesCount = srcMethod->propertyNamesCount;
                dstMethod->propertyNames = calloc(srcMethod->propertyNamesCount, sizeof(char*));

                for (int j = 0; j < srcMethod->propertyNamesCount; j++) {
                    if (srcMethod->propertyNames[j]) {
                        dstMethod->propertyNames[j] = strdup(srcMethod->propertyNames[j]);
                    }
                }
            }
        }
    }





    memset(&device->model, 0, sizeof(DeviceModel));
    if (model->id) device->model.id = strdup(model->id);
    if (model->name) device->model.name = strdup(model->name);
    if (model->namespace_) device->model.namespace_ = strdup(model->namespace_);
    if (model->description) device->model.description = strdup(model->description);


    if (model->properties && model->propertiesCount > 0) {
        device->model.propertiesCount = model->propertiesCount;
        device->model.properties = calloc(model->propertiesCount, sizeof(ModelProperty));

        for (int i = 0; i < model->propertiesCount; i++) {
            ModelProperty *srcProp = &model->properties[i];
            ModelProperty *dstProp = &device->model.properties[i];

            if (srcProp->name) dstProp->name = strdup(srcProp->name);
            if (srcProp->dataType) dstProp->dataType = strdup(srcProp->dataType);
            if (srcProp->description) dstProp->description = strdup(srcProp->description);
            if (srcProp->accessMode) dstProp->accessMode = strdup(srcProp->accessMode);
            if (srcProp->minimum) dstProp->minimum = strdup(srcProp->minimum);
            if (srcProp->maximum) dstProp->maximum = strdup(srcProp->maximum);
            if (srcProp->unit) dstProp->unit = strdup(srcProp->unit);
        }
    }


    device->status = strdup(DEVICE_STATUS_UNKNOWN);
    device->stopChan = 0;
    device->dataThreadRunning = 0;


    if (pthread_mutex_init(&device->mutex, NULL) != 0) {
        log_error("Failed to initialize device mutex");
        device_free(device);
        return NULL;
    }


    if (device->instance.pProtocol.protocolName) {
        device->client = NewClient(&device->instance.pProtocol);
        if (!device->client) {
            log_error("Failed to create device client");
            device_free(device);
            return NULL;
        }
    }


    if (device->instance.twinsCount == 0 && device->instance.propertiesCount > 0) {
        device->instance.twinsCount = device->instance.propertiesCount;
        device->instance.twins = calloc(device->instance.twinsCount, sizeof(Twin));
        for (int i = 0; i < device->instance.twinsCount; ++i) {
            DeviceProperty *p = &device->instance.properties[i];
            Twin *tw = &device->instance.twins[i];
            tw->propertyName = p->name ? strdup(p->name) : strdup("unknown");
            tw->property = p;
            tw->observedDesired.value = NULL;
            tw->reported.value = NULL;
        }
        log_info("Auto-built %d twins from properties", device->instance.twinsCount);
    }


    if (device->instance.methodsCount == 0 && device->instance.propertiesCount > 0) {
        device->instance.methodsCount = 1;
        device->instance.methods = calloc(1, sizeof(DeviceMethod));
        DeviceMethod *m = &device->instance.methods[0];
        m->name = strdup("SetProperty");
        m->propertyNamesCount = device->instance.propertiesCount;
        m->propertyNames = calloc(m->propertyNamesCount, sizeof(char*));
        for (int i = 0; i < m->propertyNamesCount; ++i) {
            m->propertyNames[i] = device->instance.properties[i].name
                                  ? strdup(device->instance.properties[i].name)
                                  : strdup("unknown");
        }
        log_info("Auto-built default method SetProperty with %d properties", m->propertyNamesCount);
    }

    log_info("Device created successfully: %s", device->instance.name);
    return device;
}


// Destroy device_free
void device_free(Device *device) {
    if (!device) return;


    if (device->dataThreadRunning || device->dataThread) {
        device_stop(device);
    }


    if (device->client) {
        FreeClient(device->client);
    }


    free(device->instance.name);
    free(device->instance.namespace_);
    free(device->instance.model);
    free(device->instance.protocolName);
    free(device->instance.pProtocol.protocolName);
    free(device->instance.pProtocol.configData);


    if (device->instance.twins) {
        for (int i = 0; i < device->instance.twinsCount; i++) {
            Twin *twin = &device->instance.twins[i];
            free(twin->propertyName);
            free(twin->observedDesired.value);
            free(twin->observedDesired.metadata.timestamp);
            free(twin->observedDesired.metadata.type);
            free(twin->reported.value);
            free(twin->reported.metadata.timestamp);
            free(twin->reported.metadata.type);

            if (twin->property) {

                int embedded = 0;
                if (device->instance.properties &&
                    twin->property >= device->instance.properties &&
                    twin->property < device->instance.properties + device->instance.propertiesCount) {
                    embedded = 1;
                }
                if (!embedded) {
                    free(twin->property->name);
                    free(twin->property);
                }

            }
        }
        free(device->instance.twins);
    }


    if (device->instance.properties) {
        for (int i = 0; i < device->instance.propertiesCount; i++) {
            DeviceProperty *prop = &device->instance.properties[i];
            free(prop->name);
            free(prop->propertyName);
            free(prop->modelName);
            free(prop->protocol);
        }
        free(device->instance.properties);
    }


    if (device->instance.methods) {
        for (int i = 0; i < device->instance.methodsCount; i++) {
            DeviceMethod *method = &device->instance.methods[i];
            free(method->name);
            free(method->description);
            if (method->propertyNames) {
                for (int j = 0; j < method->propertyNamesCount; j++) {
                    free(method->propertyNames[j]);
                }
                free(method->propertyNames);
            }
        }
        free(device->instance.methods);
    }


    free(device->model.name);
    free(device->model.namespace_);
    free(device->model.description);
    if (device->model.properties) {
        for (int i = 0; i < device->model.propertiesCount; i++) {
            ModelProperty *prop = &device->model.properties[i];
            free(prop->name);
            free(prop->dataType);
            free(prop->description);
            free(prop->accessMode);
            free(prop->minimum);
            free(prop->maximum);
            free(prop->unit);
        }
        free(device->model.properties);
    }

    free(device->status);
    pthread_mutex_destroy(&device->mutex);
    free(device);
}

// Function device_runtime_rebuild
static void device_runtime_rebuild(Device *device) {
    if (!device) return;
    int rebuild = 0;
    if (device->instance.propertiesCount > 0 &&
        (device->instance.twinsCount == 0 || !device->instance.twins)) {
        rebuild = 1;
    } else {

        for (int i = 0; i < device->instance.twinsCount; ++i) {
            if (device->instance.twins[i].property == NULL) { rebuild = 1; break; }
        }
    }
    if (rebuild) {
        free(device->instance.twins);
        device->instance.twinsCount = device->instance.propertiesCount;
        device->instance.twins = calloc(device->instance.twinsCount, sizeof(Twin));
        for (int i = 0; i < device->instance.twinsCount; ++i) {
            DeviceProperty *p = &device->instance.properties[i];
            Twin *tw = &device->instance.twins[i];
            tw->propertyName = p->name ? strdup(p->name) : strdup("unknown");
            tw->property = p;
        }
        log_warn("Runtime rebuilt %d twins for device %s",
                 device->instance.twinsCount,
                 device->instance.name ? device->instance.name : "(unknown)");
    }
    if (device->instance.methodsCount == 0 && device->instance.propertiesCount > 0) {
        device->instance.methodsCount = 1;
        device->instance.methods = calloc(1, sizeof(DeviceMethod));
        DeviceMethod *m = &device->instance.methods[0];
        m->name = strdup("SetProperty");
        m->propertyNamesCount = device->instance.propertiesCount;
        m->propertyNames = calloc(m->propertyNamesCount, sizeof(char*));
        for (int i = 0; i < m->propertyNamesCount; ++i) {
            m->propertyNames[i] = device->instance.properties[i].name ?
                                  strdup(device->instance.properties[i].name) : strdup("unknown");
        }
        log_warn("Runtime rebuilt default method SetProperty (%d props)", m->propertyNamesCount);
    }
}


// Function device_start
int device_start(Device *device) {
    if (!device) return -1;

    pthread_mutex_lock(&device->mutex);
    device_runtime_rebuild(device);
    log_info("Starting device: %s", device->instance.name);


    if (device->dataThreadRunning) {
        log_warn("Device %s is already running", device->instance.name);
        pthread_mutex_unlock(&device->mutex);
        return 0;
    }


    if (device->client) {
        if (InitDevice(device->client) != 0) {
            log_error("Failed to initialize device client for %s", device->instance.name);
            device_set_status(device, DEVICE_STATUS_OFFLINE);
            pthread_mutex_unlock(&device->mutex);
            return -1;
        }
    }

    device_set_status(device, DEVICE_STATUS_OK);


    device->dataThreadRunning = 1;
    if (pthread_create(&device->dataThread, NULL, device_data_thread, device) != 0) {
        log_error("Failed to create data thread for device %s", device->instance.name);
        device->dataThreadRunning = 0;
        device_set_status(device, DEVICE_STATUS_OFFLINE);
        pthread_mutex_unlock(&device->mutex);
        return -1;
    }



    pthread_mutex_unlock(&device->mutex);

    log_info("Device %s started successfully", device->instance.name);
    return 0;
}


// Function device_stop
int device_stop(Device *device) {
    if (!device) return -1;

    pthread_mutex_lock(&device->mutex);

    log_info("Stopping device: %s", device->instance.name);


    device->stopChan = 1;
    device->dataThreadRunning = 0;


    if (device->client) {
        StopDevice(device->client);
    }


    device_set_status(device, DEVICE_STATUS_OFFLINE);

    pthread_mutex_unlock(&device->mutex);


    if (device->dataThread) {
        for (int i = 0; i < 10; ++i) {

            if (!device->dataThreadRunning) break;
            usleep(50000);
        }
        pthread_cancel(device->dataThread);
        pthread_join(device->dataThread, NULL);
        device->dataThread = 0;
    }

    log_info("Device %s stopped successfully", device->instance.name);
    return 0;
}


// Function device_restart
int device_restart(Device *device) {
    if (!device) return -1;

    log_info("Restarting device: %s", device->instance.name);

    if (device_stop(device) != 0) {
        log_error("Failed to stop device %s during restart", device->instance.name);
        return -1;
    }


    usleep(100000);

    if (device_start(device) != 0) {
        log_error("Failed to start device %s during restart", device->instance.name);
        return -1;
    }

    log_info("Device %s restarted successfully", device->instance.name);
    return 0;
}



// Function device_deal_twin
int device_deal_twin(Device *device, const Twin *twin_in) {
    if (!device || !twin_in) return -1;
    Twin *twin = (Twin*)twin_in;

    if (sim_temperature_enabled() && twin->propertyName &&
        strcmp(twin->propertyName, "temperature") == 0) {
        log_debug("Skip device_deal_twin for simulated temperature");
        return 0;
    }

    const char *prop = twin->propertyName ? twin->propertyName : "(null)";
    const char *desired = twin->observedDesired.value;
    const char *reported = twin->reported.value;

    if (!desired || !*desired) {
        log_debug("Twin %s no desired, skip", prop);
        return 0;
    }
    if (reported && strcmp(reported, desired) == 0) {
        log_debug("Twin %s desired == reported (%s), skip", prop, desired);
        return 0;
    }


    char ip[64] = "127.0.0.1";
    int port = 1502;
    if (device->instance.pProtocol.configData) {
        json_get_str(device->instance.pProtocol.configData, "ip", ip, sizeof(ip));
        json_get_int(device->instance.pProtocol.configData, "port", &port);
        cleanup_escape_prefix(ip);
    }

    int offset = resolve_offset_by_name(twin->propertyName);
    if (offset <= 0) {
        log_warn("Twin %s: cannot resolve offset, skip", prop);
        return 0;
    }

    int value = atoi(desired);


    char hostBuf[128];
    int portFixed;
    normalize_host_port(ip, port, hostBuf, sizeof(hostBuf), &portFixed);

    int timeout_s = 2;
    const char *env_to = getenv("MAPPER_MBPOLL_TIMEOUT_S");
    if (env_to && *env_to) {
        int tv = atoi(env_to);
        if (tv > 0 && tv < 30) timeout_s = tv;
    }

    char cmd[512];
    snprintf(cmd, sizeof(cmd),
             "timeout %d /usr/bin/mbpoll -1 -m tcp %s -p %d -a 1 -t 4 -r %d %d >/dev/null 2>&1",
             timeout_s, hostBuf, portFixed, offset, value);

    log_info("Twin %s: exec cmd=%s", prop, cmd);

    int rcRaw = system(cmd);
    int exit_code = -1;
    if (rcRaw != -1) {
        if (WIFEXITED(rcRaw)) exit_code = WEXITSTATUS(rcRaw);
// Function if
        else if (WIFSIGNALED(rcRaw)) exit_code = 128 + WTERMSIG(rcRaw);
    }
    if (exit_code != 0) {
        log_error("Twin %s: mbpoll write failed (exit=%d raw=%d). cmd=%s",
                  prop, exit_code, rcRaw, cmd);
        return -1;
    }
    log_info("Twin %s: mbpoll write ok host=%s port=%d offset=%d val=%d",
             prop, hostBuf, portFixed, offset, value);


    free(twin->reported.value);
    char ts[32]; now_iso8601(ts);
    twin->reported.value = strdup(desired);
    free(twin->reported.metadata.timestamp);
    twin->reported.metadata.timestamp = strdup(ts);

    log_info("Twin %s write success: offset=%d value=%d (ip=%s:%d); reported updated",
             prop, offset, value, hostBuf, portFixed);

    mysql_recorder_record(
        device->instance.namespace_ ? device->instance.namespace_ : "default",
        device->instance.name ? device->instance.name : "unknown",
        twin->propertyName ? twin->propertyName : "unknown",
        desired,
        (long long)time(NULL) * 1000
    );

    if (g_publisher) {
        DataModel dm = (DataModel){0};
        dm.namespace_   = device->instance.namespace_ ? device->instance.namespace_ : "default";
        dm.deviceName   = device->instance.name ? device->instance.name : "unknown";
        dm.propertyName = twin->propertyName ? twin->propertyName : "unknown";
        dm.type         = "string";
        dm.value        = (char*)desired;
        dm.timeStamp    = (int64_t)time(NULL) * 1000;
        int prc = publisher_publish_data(g_publisher, &dm);
        if (prc != 0) log_warn("Publish failed (write success) for %s", dm.propertyName);
    }
    return 0;
}


DeviceManager *device_manager_new(void) {
    DeviceManager *manager = calloc(1, sizeof(DeviceManager));
    if (!manager) return NULL;

    manager->capacity = 10;
    manager->devices = calloc(manager->capacity, sizeof(Device*));
    if (!manager->devices) {
        free(manager);
        return NULL;
    }

    if (pthread_mutex_init(&manager->managerMutex, NULL) != 0) {
        free(manager->devices);
        free(manager);
        return NULL;
    }

    manager->stopped = 0;
    return manager;
}


// Destroy device_manager_free
void device_manager_free(DeviceManager *manager) {
    if (!manager) return;
    if (!manager->stopped) {
        device_manager_stop_all(manager);
    }
    pthread_mutex_lock(&manager->managerMutex);
    for (int i = 0; i < manager->deviceCount; i++) {
        device_free(manager->devices[i]);
    }
    free(manager->devices);
    pthread_mutex_unlock(&manager->managerMutex);
    pthread_mutex_destroy(&manager->managerMutex);
    free(manager);
}


// Function device_manager_add
int device_manager_add(DeviceManager *manager, Device *device) {
    if (!manager || !device) return -1;

    pthread_mutex_lock(&manager->managerMutex);


    if (manager->deviceCount >= manager->capacity) {
        manager->capacity *= 2;
        Device **newDevices = realloc(manager->devices,
                                     manager->capacity * sizeof(Device*));
        if (!newDevices) {
            pthread_mutex_unlock(&manager->managerMutex);
            return -1;
        }
        manager->devices = newDevices;
    }

    manager->devices[manager->deviceCount++] = device;

    pthread_mutex_unlock(&manager->managerMutex);

    log_info("Device %s added to manager", device->instance.name);
    return 0;
}


// Function device_manager_remove
int device_manager_remove(DeviceManager *manager, const char *deviceId) {
    if (!manager || !deviceId) return -1;

    pthread_mutex_lock(&manager->managerMutex);

    for (int i = 0; i < manager->deviceCount; i++) {
        if (manager->devices[i] && manager->devices[i]->instance.name &&
            strcmp(manager->devices[i]->instance.name, deviceId) == 0) {

            device_free(manager->devices[i]);


            for (int j = i; j < manager->deviceCount - 1; j++) {
                manager->devices[j] = manager->devices[j + 1];
            }
            manager->deviceCount--;

            pthread_mutex_unlock(&manager->managerMutex);
            log_info("Device %s removed from manager", deviceId);
            return 0;
        }
    }

    pthread_mutex_unlock(&manager->managerMutex);
    log_warn("Device %s not found in manager", deviceId);
    return -1;
}


Device *device_manager_get(DeviceManager *manager, const char *deviceId) {
    if (!manager || !deviceId) return NULL;

    pthread_mutex_lock(&manager->managerMutex);

    for (int i = 0; i < manager->deviceCount; i++) {
        if (manager->devices[i] && manager->devices[i]->instance.name &&
            strcmp(manager->devices[i]->instance.name, deviceId) == 0) {
            Device *device = manager->devices[i];
            pthread_mutex_unlock(&manager->managerMutex);
            return device;
        }
    }


    const char *sep = strrchr(deviceId, '.');
    if (!sep) sep = strrchr(deviceId, '/');
    if (sep && *(sep + 1)) {
        const char *shortId = sep + 1;
        for (int i = 0; i < manager->deviceCount; i++) {
            if (manager->devices[i] && manager->devices[i]->instance.name &&
                strcmp(manager->devices[i]->instance.name, shortId) == 0) {
                Device *device = manager->devices[i];
                pthread_mutex_unlock(&manager->managerMutex);
                return device;
            }
        }
    }

    pthread_mutex_unlock(&manager->managerMutex);
    return NULL;
}


// Function device_manager_start_all
int device_manager_start_all(DeviceManager *manager) {
    if (!manager) return -1;

    pthread_mutex_lock(&manager->managerMutex);

    int success = 0;
    for (int i = 0; i < manager->deviceCount; i++) {
        if (device_start(manager->devices[i]) == 0) {
            success++;
        }
    }

    pthread_mutex_unlock(&manager->managerMutex);

    log_info("Started %d/%d devices", success, manager->deviceCount);
    return success == manager->deviceCount ? 0 : -1;
}


// Function device_manager_stop_all
int device_manager_stop_all(DeviceManager *manager) {
    if (!manager) return -1;
    if (manager->stopped) {
        log_debug("device_manager_stop_all: already stopped");
        return 0;
    }
    pthread_mutex_lock(&manager->managerMutex);
    for (int i = 0; i < manager->deviceCount; i++) {
        device_stop(manager->devices[i]);
    }
    pthread_mutex_unlock(&manager->managerMutex);
    manager->stopped = 1;
    log_info("Stopped all devices");
    return 0;
}


// Initialize device_init_from_config
int device_init_from_config(Device *device, const char *configPath) {
    if (!device || !configPath) return -1;


    log_info("Device %s initialized from config: %s", device->instance.name, configPath);
    return 0;
}


// Function device_register_to_edge
int device_register_to_edge(Device *device) {
    if (!device) return -1;


    log_info("Device %s registered to edge core", device->instance.name);
    return 0;
}