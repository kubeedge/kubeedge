#ifndef UTIL_PARSE_TYPE_H
#define UTIL_PARSE_TYPE_H

#include "dmi/v1beta1/api.pb-c.h"
#include "common/configmaptype.h"
#include "common/eventtype.h"

// Twin <-> gRPC Twin
V1beta1__Twin **ConvTwinsToGrpc(const Twin *twins, int twin_count, int *out_count);
Twin *ConvGrpcToTwins(V1beta1__Twin **twins, int twin_count, const Twin *src_twins, int src_count, int *out_count);

// MsgTwin map <-> gRPC Twin
V1beta1__Twin **ConvMsgTwinToGrpc(const char **names, MsgTwin **msgTwins, int count, int *out_count);

#endif // PARSE_TYPE_H