#ifndef _COMMON_JSON_UTILS_H_
#define _COMMON_JSON_UTILS_H_

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif


int json_get_str(const char *json, const char *key, char *out, size_t outsz);

int json_get_int(const char *json, const char *key, int *out);

const char *json_get_raw_object(const char *json, const char *objectKey);

int json_get_int_in_object(const char *json, const char *objectKey, const char *key, int *out);

#ifdef __cplusplus
}
#endif

#endif // _COMMON_JSON_UTILS_H_