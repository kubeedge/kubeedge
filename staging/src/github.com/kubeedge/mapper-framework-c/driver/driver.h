#ifndef DRIVER_DRIVER_H
#define DRIVER_DRIVER_H

#ifdef __cplusplus
extern "C"
{
#endif

#include "devicetype.h"

    CustomizedClient *NewClient(const ProtocolConfig *protocol);
    void FreeClient(CustomizedClient *client);

    int InitDevice(CustomizedClient *client);
    int GetDeviceData(CustomizedClient *client, const VisitorConfig *visitor, void **out_data);
    int DeviceDataWrite(CustomizedClient *client, const VisitorConfig *visitor, const char *deviceMethodName, const char *propertyName, const void *data);
    int SetDeviceData(CustomizedClient *client, const void *data, const VisitorConfig *visitor);
    int StopDevice(CustomizedClient *client);
    const char *GetDeviceStates(CustomizedClient *client);

#ifdef __cplusplus
}
#endif

#endif // DRIVER_DRIVER_H