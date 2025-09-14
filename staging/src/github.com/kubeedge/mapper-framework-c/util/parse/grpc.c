#include "util/parse/grpc.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <cjson/cJSON.h>

// Safely duplicates a string
static char *strdup_safe(const char *s) {
    if (!s) return NULL;
    char *copy = malloc(strlen(s) + 1);
    if (copy) strcpy(copy, s);
    return copy;
}

// Converts a Google Protobuf Any object to a string
static char *any_to_string(const Google__Protobuf__Any *any) {
    if (!any || !any->value.data || any->value.len == 0) return strdup_safe("");
    char *str = malloc(any->value.len + 1);
    memcpy(str, any->value.data, any->value.len);
    str[any->value.len] = '\0';
    return str;
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
            char *valstr = any_to_string(any);
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
                    char *valstr = any_to_string(any);
                    cJSON_AddStringToObject(recvAdapter, entry->key, valstr);
                    free(valstr);
                }
            }
            cJSON_AddItemToObject(visitorConfig, "configData", recvAdapter);
        }
        props[i].visitors = cJSON_PrintUnformatted(visitorConfig);
        cJSON_Delete(visitorConfig);

        props[i].pushMethod = NULL;
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
int get_device_model_from_grpc(const V1beta1__DeviceModel *model, DeviceModel *out) {
    if (!model || !out) return -1;
    out->id = NULL;
    out->name = strdup_safe(model->name);
    out->namespace_ = model->namespace_;
    out->description = NULL;
    if (model->spec && model->spec->n_properties > 0) {
        out->propertiesCount = model->spec->n_properties;
        out->properties = calloc(out->propertiesCount, sizeof(ModelProperty));
        for (int i = 0; i < out->propertiesCount; ++i) {
            V1beta1__ModelProperty *property = model->spec->properties[i];
            out->properties[i].name = strdup_safe(property->name);
            out->properties[i].dataType = strdup_safe(property->type);
            out->properties[i].description = property->description ? strdup_safe(property->description) : NULL;
            out->properties[i].accessMode = property->accessmode ? strdup_safe(property->accessmode) : NULL;
            out->properties[i].minimum = property->minimum ? strdup_safe(property->minimum) : NULL;
            out->properties[i].maximum = property->maximum ? strdup_safe(property->maximum) : NULL;
            out->properties[i].unit = property->unit ? strdup_safe(property->unit) : NULL;
        }
    } else {
        out->properties = NULL;
        out->propertiesCount = 0;
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
    out->namespace_ = device->namespace_;
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
    log_info("final instance data from grpc built");
    return 0;
}

// Generates a resource ID from a namespace and name
void get_resource_id(const char *ns, const char *name, char *out, size_t outlen) {
    snprintf(out, outlen, "%s.%s", ns, name);
}