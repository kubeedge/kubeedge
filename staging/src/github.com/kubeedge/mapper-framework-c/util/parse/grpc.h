#ifndef UTIL_PARSE_GRPC_H
#define UTIL_PARSE_GRPC_H

#ifdef __cplusplus
extern "C" {
#endif

#include "dmi/v1beta1/api.pb-c.h"
#include "common/configmaptype.h"
#include "common/datamodel.h"
#include "common/datamethod.h"
#include "common/dataconverter.h"
#include "log/log.h"

// Retrieves the protocol name from a gRPC device object
int get_protocol_name_from_grpc(const V1beta1__Device *device, char **out);

// Builds a ProtocolConfig structure from a gRPC device object
int build_protocol_from_grpc(const V1beta1__Device *device, ProtocolConfig *out);

// Builds an array of Twin structures from a gRPC device object
int build_twins_from_grpc(const V1beta1__Device *device, Twin **out, int *out_count);

// Builds an array of DeviceProperty structures from a gRPC device object
int build_properties_from_grpc(const V1beta1__Device *device, DeviceProperty **out, int *out_count);

// Builds an array of DeviceMethod structures from a gRPC device object
int build_methods_from_grpc(const V1beta1__Device *device, DeviceMethod **out, int *out_count);

// Builds a DeviceModel structure from a gRPC device model object
int get_device_model_from_grpc(const V1beta1__DeviceModel *model, DeviceModel *out);

// Builds a DeviceInstance structure from a gRPC device object
int get_device_from_grpc(const V1beta1__Device *device, const DeviceModel *commonModel, DeviceInstance *out);

// Generates a resource ID from a namespace and name
void get_resource_id(const char *ns, const char *name, char *out, size_t outlen);

#ifdef __cplusplus
}
#endif

#endif // UTIL_PARSE_GRPC_H