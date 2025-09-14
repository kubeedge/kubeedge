#ifndef REGISTER_H
#define REGISTER_H
#include "common/datamodel.h"
#include "common/configmaptype.h"
#ifdef __cplusplus
extern "C" {
#endif

int RegisterMapper(
    int withData,
    DeviceInstance **outDeviceList, int *outDeviceCount,
    DeviceModel **outModelList, int *outModelCount
);

#ifdef __cplusplus
}
#endif

#endif // REGISTER_H