#ifndef COMMON_STRING_UTILS_H
#define COMMON_STRING_UTILS_H

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

void trim_str(char *s);
void sanitize_host(char *s);
void cleanup_escape_prefix(char *s);
void normalize_host_port(const char *rawHost, int rawPort,
                         char *outHost, size_t outHostSz, int *outPort);
void sanitize_id(const char *in, char *out, size_t outsz, const char *fallback);

#ifdef __cplusplus
}
#endif

#endif // COMMON_STRING_UTILS_H