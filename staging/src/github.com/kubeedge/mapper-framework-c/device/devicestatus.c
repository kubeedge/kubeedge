#include "device/devicestatus.h"
#include "device/device.h"
#include "common/const.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <time.h>

static long long now_ms(void) {
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    return (long long)ts.tv_sec * 1000LL + ts.tv_nsec / 1000000LL;
}

int device_status_update(Device *device, const char *newStatus) {
    if (!device) return -1;
    if (!newStatus || !*newStatus) newStatus = DEVICE_STATUS_UNKNOWN;
    if (!device->status) {
        device->status = strdup(newStatus);
        log_info("Device %s status init -> %s",
                 device->instance.name ? device->instance.name : "(null)", newStatus);
        return 0;
    }
    if (strcmp(device->status, newStatus) == 0) return 0;
    char *old = device->status;
    device->status = strdup(newStatus);
    log_info("Device %s status %s -> %s",
             device->instance.name ? device->instance.name : "(null)",
             old, newStatus);
    free(old);
    return 0;
}

const char *device_status_get_current(Device *device) {
    if (!device || !device->status) return DEVICE_STATUS_UNKNOWN;
    return device->status;
}

int device_status_check_change(Device *device, const char *currentStatus) {
    if (!device || !currentStatus) return -1;
    if (!device->status) return 1;
    return strcmp(device->status, currentStatus) != 0;
}

int device_status_handle_offline(Device *device) {
    return device_status_update(device, DEVICE_STATUS_OFFLINE);
}

int device_status_handle_online(Device *device) {
    return device_status_update(device, DEVICE_STATUS_OK);
}

/* 占位实现 */
long long device_status_get_last_update_time(Device *device) {
    (void)device;
    return now_ms();
}

int device_status_start_health_monitor(Device *device) {
    (void)device;
    return 0;
}

int device_status_stop_health_monitor(Device *device) {
    (void)device;
    return 0;
}

int device_status_health_check(Device *device) {
    (void)device;
    return 0;
}

int device_status_send_event(Device *device, const char *eventType, const char *message) {
    log_info("Device %s event %s: %s",
             device && device->instance.name ? device->instance.name : "(null)",
             eventType ? eventType : "(nil)",
             message ? message : "");
    return 0;
}
int device_set_status(Device *device, const char *newStatus) {
    return device_status_update(device, newStatus);
}

const char *device_get_status(Device *device) {
    return device_status_get_current(device);
}