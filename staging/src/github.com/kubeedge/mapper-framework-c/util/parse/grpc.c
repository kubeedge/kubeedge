#include "util/parse/grpc.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <cjson/cJSON.h>
#include "google/protobuf/wrappers.pb-c.h"
#include "google/protobuf/any.pb-c.h"
#include <ctype.h>

// Safely duplicates a string
static char *strdup_safe(const char *s) {
    if (!s) return NULL;
    char *copy = malloc(strlen(s) + 1);
    if (copy) strcpy(copy, s);
    return copy;
}

static void split_addr_port(const char *addr, char **host_out, int *port_out) {
    if (!addr) return;
    const char *p = strstr(addr, "://");
    const char *s = p ? p + 3 : addr;
    const char *colon = strrchr(s, ':');
    if (colon && colon > s) {
        *host_out = strndup(s, (size_t)(colon - s));
        *port_out = atoi(colon + 1);
    } else {
        *host_out = strdup(s);
        if (port_out && *port_out == 0) {
            *port_out = 80;
        }
    }
}


char *parse_any_to_string(const char *type_url, const ProtobufCBinaryData *value) {
    if (!value || value->len == 0 || !value->data) return NULL;
    if (type_url) {
        if (strstr(type_url, "StringValue")) {
            Google__Protobuf__StringValue *sv = google__protobuf__string_value__unpack(NULL, value->len, value->data);
            if (sv) { char *s = strdup(sv->value ? sv->value : ""); google__protobuf__string_value__free_unpacked(sv, NULL); return s; }
        }
        if (strstr(type_url, "Int32Value")) {
            Google__Protobuf__Int32Value *v = google__protobuf__int32_value__unpack(NULL, value->len, value->data);
            if (v) { char buf[64]; snprintf(buf, sizeof(buf), "%d", v->value); google__protobuf__int32_value__free_unpacked(v, NULL); return strdup(buf); }
        }
        if (strstr(type_url, "Int64Value")) {
            Google__Protobuf__Int64Value *v = google__protobuf__int64_value__unpack(NULL, value->len, value->data);
            if (v) { char buf[64]; snprintf(buf, sizeof(buf), "%lld", (long long)v->value); google__protobuf__int64_value__free_unpacked(v, NULL); return strdup(buf); }
        }
        if (strstr(type_url, "DoubleValue")) {
            Google__Protobuf__DoubleValue *v = google__protobuf__double_value__unpack(NULL, value->len, value->data);
            if (v) { char buf[64]; snprintf(buf, sizeof(buf), "%g", v->value); google__protobuf__double_value__free_unpacked(v, NULL); return strdup(buf); }
        }
        if (strstr(type_url, "BoolValue")) {
            Google__Protobuf__BoolValue *v = google__protobuf__bool_value__unpack(NULL, value->len, value->data);
            if (v) { char *s = strdup(v->value ? "true" : "false"); google__protobuf__bool_value__free_unpacked(v, NULL); return s; }
        }
    }
    char *s = malloc(value->len + 1);
    if (!s) return NULL;
    memcpy(s, value->data, value->len);
    s[value->len] = '\0';
    if (s[0] == '{' || s[0] == '[') {
        cJSON *root = cJSON_Parse(s);
        if (root) {
            cJSON *v = cJSON_GetObjectItem(root, "value");
            if (cJSON_IsString(v) && v->valuestring) { char *res = strdup(v->valuestring); cJSON_Delete(root); free(s); return res; }
            if (cJSON_IsNumber(v)) { char buf[64]; if (v->valuedouble == (double)(long long)v->valuedouble) snprintf(buf, sizeof(buf), "%lld", (long long)v->valuedouble); else snprintf(buf, sizeof(buf), "%g", v->valuedouble); char *res = strdup(buf); cJSON_Delete(root); free(s); return res; }
            cJSON_Delete(root);
        }
    }
    return s;
}

// Retrieves the protocol name from a gRPC device object
int get_protocol_name_from_grpc(const V1beta1__Device *device, char **out) {
    if (!device || !device->spec || !device->spec->protocol || !device->spec->protocol->protocolname) {
        log_error("get_protocol_name_from_grpc: protocol name not found");
        *out = NULL;
        return -1;
    }
    *out = strdup_safe(device->spec->protocol->protocolname);
    return 0;
}

// Builds a ProtocolConfig structure from a gRPC device object
int build_protocol_from_grpc(const V1beta1__Device *device, ProtocolConfig *out) {
    char *protocolName = NULL;
    if (get_protocol_name_from_grpc(device, &protocolName) != 0) return -1;

    cJSON *customizedProtocol = cJSON_CreateObject();
    cJSON_AddStringToObject(customizedProtocol, "protocolName", protocolName);

    if (device->spec->protocol->configdata) {
        cJSON *recvAdapter = cJSON_CreateObject();
        for (size_t i = 0; i < device->spec->protocol->configdata->n_data; ++i) {
            V1beta1__CustomizedValue__DataEntry *entry = device->spec->protocol->configdata->data[i];
            Google__Protobuf__Any *any = entry->value;
            char *valstr = NULL;
            if (any) valstr = parse_any_to_string(any->type_url, &any->value);
             cJSON_AddStringToObject(recvAdapter, entry->key, valstr);
             free(valstr);
        }
        cJSON_AddItemToObject(customizedProtocol, "configData", recvAdapter);
    }
    char *jsonStr = cJSON_PrintUnformatted(customizedProtocol);
    out->protocolName = protocolName;
    out->configData = jsonStr;

    cJSON_Delete(customizedProtocol);
    return 0;
}

// Builds an array of Twin structures from a gRPC device object
int build_twins_from_grpc(const V1beta1__Device *device, Twin **out, int *out_count) {
    if (!device || !device->spec || device->spec->n_properties == 0) {
        *out = NULL;
        *out_count = 0;
        return 0;
    }
    int count = device->spec->n_properties;
    Twin *twins = calloc(count, sizeof(Twin));
    for (int i = 0; i < count; ++i) {
        V1beta1__DeviceProperty *property = device->spec->properties[i];
        twins[i].propertyName = strdup_safe(property->name);
        if (property->desired) {
            twins[i].observedDesired.value = strdup_safe(property->desired->value);
            for (size_t k = 0; k < property->desired->n_metadata; ++k) {
                V1beta1__TwinProperty__MetadataEntry *meta = property->desired->metadata[k];
                if (strcmp(meta->key, "timestamp") == 0)
                    twins[i].observedDesired.metadata.timestamp = strdup_safe(meta->value);
                if (strcmp(meta->key, "type") == 0)
                    twins[i].observedDesired.metadata.type = strdup_safe(meta->value);
            }
        }
    if (property->desired) {
        if (!twins[i].observedDesired.value) {
            if (property->desired->value) {
                twins[i].observedDesired.value = strdup_safe(property->desired->value);
            } 
        }
    }
    }
    *out = twins;
    *out_count = count;
    return 0;
}

// Builds an array of DeviceProperty structures from a gRPC device object
int build_properties_from_grpc(const V1beta1__Device *device, DeviceProperty **out, int *out_count) {
    if (!device || !device->spec || device->spec->n_properties == 0) {
        *out = NULL;
        *out_count = 0;
        return 0;
    }
    char *protocolName = NULL;
    get_protocol_name_from_grpc(device, &protocolName);

    int count = device->spec->n_properties;
    DeviceProperty *props = calloc(count, sizeof(DeviceProperty));
    for (int i = 0; i < count; ++i) {
        V1beta1__DeviceProperty *pptv = device->spec->properties[i];
        props[i].name = strdup_safe(pptv->name);
        props[i].propertyName = strdup_safe(pptv->name);
        props[i].modelName = device->spec->devicemodelreference ? strdup_safe(device->spec->devicemodelreference) : NULL;
        props[i].collectCycle = pptv->collectcycle;
        props[i].reportCycle = pptv->reportcycle;
        props[i].reportToCloud = pptv->reporttocloud;
        props[i].protocol = protocolName ? strdup_safe(protocolName) : NULL;

        cJSON *visitorConfig = cJSON_CreateObject();
        if (pptv->visitors) {
            cJSON_AddStringToObject(visitorConfig, "protocolName", pptv->visitors->protocolname);
            cJSON *recvAdapter = cJSON_CreateObject();
            if (pptv->visitors->configdata) {
                for (size_t j = 0; j < pptv->visitors->configdata->n_data; ++j) {
                    V1beta1__CustomizedValue__DataEntry *entry = pptv->visitors->configdata->data[j];
                    Google__Protobuf__Any *any = entry->value;
                    char *valstr = NULL;
                    if (any) valstr = parse_any_to_string(any->type_url, &any->value);
                     cJSON_AddStringToObject(recvAdapter, entry->key, valstr);
                     free(valstr);
                }
            }
            cJSON_AddItemToObject(visitorConfig, "configData", recvAdapter);
        }
        props[i].visitors = cJSON_PrintUnformatted(visitorConfig);
        cJSON_Delete(visitorConfig);

        if (pptv->pushmethod && pptv->pushmethod->mqtt) {
            char *host = NULL; int port = 1883;
            if (pptv->pushmethod->mqtt->address) split_addr_port(pptv->pushmethod->mqtt->address, &host, &port);
            cJSON *mc = cJSON_CreateObject();
            cJSON_AddStringToObject(mc, "brokerUrl", host ? host : "127.0.0.1");
            cJSON_AddNumberToObject(mc, "port", port);
            if (pptv->pushmethod->mqtt->topic) cJSON_AddStringToObject(mc, "topicPrefix", pptv->pushmethod->mqtt->topic);
            if (pptv->pushmethod->mqtt->qos) cJSON_AddNumberToObject(mc, "qos", (double)pptv->pushmethod->mqtt->qos);
            cJSON_AddNumberToObject(mc, "keepAlive", 60);
            cJSON_AddStringToObject(mc, "clientId", "mapper_c");
            char *mj = cJSON_PrintUnformatted(mc);
            free(host); cJSON_Delete(mc);
            if (!props[i].pushMethod) props[i].pushMethod = calloc(1, sizeof(PushMethodConfig));
            props[i].pushMethod->methodName = strdup_safe("mqtt");
            props[i].pushMethod->methodConfig = mj;
        } else if (pptv->pushmethod && pptv->pushmethod->http) {
            char *host = NULL;
            int port = pptv->pushmethod->http->port ? (int)pptv->pushmethod->http->port : 0;
            if (pptv->pushmethod->http->hostname) split_addr_port(pptv->pushmethod->http->hostname, &host, &port);
             const char *path = pptv->pushmethod->http->requestpath ? pptv->pushmethod->http->requestpath : "/ingest";
             char endpoint[512]; snprintf(endpoint, sizeof(endpoint), "http://%s:%d%s", host ? host : "127.0.0.1", port, path);
             cJSON *hc = cJSON_CreateObject();
             cJSON_AddStringToObject(hc, "endpoint", endpoint);
             cJSON_AddStringToObject(hc, "method", "POST");
             if (pptv->pushmethod->http->timeout) cJSON_AddNumberToObject(hc, "timeout", (double)pptv->pushmethod->http->timeout);
             char *hj = cJSON_PrintUnformatted(hc);
             free(host); cJSON_Delete(hc);
             if (!props[i].pushMethod) props[i].pushMethod = calloc(1, sizeof(PushMethodConfig));
             props[i].pushMethod->methodName = strdup_safe("http");
             props[i].pushMethod->methodConfig = hj;
         } else if (pptv->pushmethod && pptv->pushmethod->otel) {
            cJSON *oc = cJSON_CreateObject();
            if (pptv->pushmethod->otel->endpointurl) cJSON_AddStringToObject(oc, "endpointUrl", pptv->pushmethod->otel->endpointurl);
            char *oj = cJSON_PrintUnformatted(oc);
            cJSON_Delete(oc);
            if (!props[i].pushMethod) props[i].pushMethod = calloc(1, sizeof(PushMethodConfig));
            props[i].pushMethod->methodName = strdup_safe("otel");
            props[i].pushMethod->methodConfig = oj;
        }
        if (pptv->pushmethod && pptv->pushmethod->dbmethod && pptv->pushmethod->dbmethod->mysql && pptv->pushmethod->dbmethod->mysql->mysqlclientconfig) {
            cJSON *mc = cJSON_CreateObject();
            if (pptv->pushmethod->dbmethod->mysql->mysqlclientconfig->addr)     cJSON_AddStringToObject(mc, "addr",     pptv->pushmethod->dbmethod->mysql->mysqlclientconfig->addr);
            if (pptv->pushmethod->dbmethod->mysql->mysqlclientconfig->database) cJSON_AddStringToObject(mc, "database", pptv->pushmethod->dbmethod->mysql->mysqlclientconfig->database);
            if (pptv->pushmethod->dbmethod->mysql->mysqlclientconfig->username) cJSON_AddStringToObject(mc, "userName", pptv->pushmethod->dbmethod->mysql->mysqlclientconfig->username);
            char *mj = cJSON_PrintUnformatted(mc);
            cJSON_Delete(mc);
            if (!props[i].pushMethod) props[i].pushMethod = calloc(1, sizeof(PushMethodConfig));
            if (!props[i].pushMethod->dbMethod) props[i].pushMethod->dbMethod = calloc(1, sizeof(DBMethodConfig));
            if (!props[i].pushMethod->dbMethod->dbConfig) props[i].pushMethod->dbMethod->dbConfig = calloc(1, sizeof(DBConfig));
            props[i].pushMethod->dbMethod->dbMethodName = strdup_safe("mysql");
            props[i].pushMethod->dbMethod->dbConfig->mysqlClientConfig = mj;
        }
        props[i].pProperty = NULL;
    }
    *out = props;
    *out_count = count;
    return 0;
}

// Builds an array of DeviceMethod structures from a gRPC device object
int build_methods_from_grpc(const V1beta1__Device *device, DeviceMethod **out, int *out_count) {
    if (!device || !device->spec || device->spec->n_methods == 0) {
        *out = NULL;
        *out_count = 0;
        return 0;
    }
    int count = device->spec->n_methods;
    DeviceMethod *methods = calloc(count, sizeof(DeviceMethod));
    for (int i = 0; i < count; ++i) {
        V1beta1__DeviceMethod *method = device->spec->methods[i];
        methods[i].name = strdup_safe(method->name);
        methods[i].description = method->description ? strdup_safe(method->description) : NULL;
        methods[i].propertyNamesCount = method->n_propertynames;
        if (method->n_propertynames > 0) {
            methods[i].propertyNames = calloc(method->n_propertynames, sizeof(char*));
            for (size_t j = 0; j < method->n_propertynames; ++j) {
                methods[i].propertyNames[j] = strdup_safe(method->propertynames[j]);
            }
        }
    }
    *out = methods;
    *out_count = count;
    return 0;
}

// Builds a DeviceModel structure from a gRPC device model object
int get_device_model_from_grpc(const V1beta1__DeviceModel *src, DeviceModel *dst) {
    if (!src || !dst) return -1;
    dst->name = src->name ? strdup(src->name) : NULL;
    dst->namespace_ = src->namespace_ ? strdup(src->namespace_) : strdup("default");
    if (dst->namespace_) {
        int ok = 0;
        for (char *p = dst->namespace_; *p; ++p) {
            if (*p >= 32 && *p < 127) { ok = 1; break; }
        }
        if (!ok) {
            free(dst->namespace_);
            dst->namespace_ = strdup("default");
        }
    }
    return 0;
}

// Builds a DeviceInstance structure from a gRPC device object
int get_device_from_grpc(const V1beta1__Device *device, const DeviceModel *commonModel, DeviceInstance *out) {
    if (!device || !out) return -1;
    char *protocolName = NULL;
    get_protocol_name_from_grpc(device, &protocolName);

    out->id = NULL;
    out->name = strdup_safe(device->name);
    out->namespace_ = strdup_safe(device->namespace_);
    if (protocolName) {
        out->protocolName = malloc(strlen(protocolName) + strlen(device->name) + 2);
        sprintf(out->protocolName, "%s-%s", protocolName, device->name);
    } else {
        out->protocolName = NULL;
    }
    build_protocol_from_grpc(device, &out->pProtocol);
    out->model = device->spec->devicemodelreference ? strdup_safe(device->spec->devicemodelreference) : NULL;

    build_twins_from_grpc(device, &out->twins, &out->twinsCount);
    build_properties_from_grpc(device, &out->properties, &out->propertiesCount);
    build_methods_from_grpc(device, &out->methods, &out->methodsCount);

    if (device->status) {
        out->status.reportToCloud = device->status->reporttocloud;
        out->status.reportCycle = device->status->reportcycle;
    }

    if (commonModel) {
        for (int i = 0; i < out->propertiesCount; ++i) {
            for (int j = 0; j < commonModel->propertiesCount; ++j) {
                if (strcmp(out->properties[i].propertyName, commonModel->properties[j].name) == 0) {
                    out->properties[i].pProperty = &commonModel->properties[j];
                    break;
                }
            }
        }
        for (int i = 0; i < out->twinsCount; ++i) {
            for (int j = 0; j < out->propertiesCount; ++j) {
                if (strcmp(out->twins[i].propertyName, out->properties[j].propertyName) == 0) {
                    out->twins[i].property = &out->properties[j];
                    break;
                }
            }
        }
    }
    return 0;
}

// Generates a resource ID from a namespace and name
void get_resource_id(const char *ns, const char *name, char *out, size_t outlen) {
    snprintf(out, outlen, "%s.%s", ns, name);
}