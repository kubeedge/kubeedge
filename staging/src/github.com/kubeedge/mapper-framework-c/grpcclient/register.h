#ifndef REGISTER_H
#define REGISTER_H
#include "common/datamodel.h"
#include "common/configmaptype.h"
#ifdef __cplusplus
extern "C" {
#endif

void register_set_dmi_sock(const char *sock_path);
int RegisterMapper(int withData, DeviceInstance **outDeviceList, int *outDeviceCount,
                   DeviceModel **outModelList, int *outModelCount);
int ReportDeviceStatus(const char *namespace_, const char *deviceName, const char *status);
int ReportDeviceStates(const char *namespace_, const char *deviceName, const char *state);
int ReportTwinKV(const char *namespace_, const char *deviceName,
                 const char *propertyName, const char *value, const char *valueType);

#ifdef __cplusplus
}
#endif

#endif // REGISTER_H