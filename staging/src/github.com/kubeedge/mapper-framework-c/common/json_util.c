#include "json_util.h"
#include <string.h>
#include <stdlib.h>
#include <ctype.h>


static const char *find_key_case_insensitive(const char *json, const char *key) {
    if (!json || !key) return NULL;
    return strcasestr(json, key);
}

int json_get_str(const char *json, const char *key, char *out, size_t outsz) {
    if (!json || !key || !out || outsz == 0) return -1;
    const char *p = find_key_case_insensitive(json, key);
    if (!p) return -1;
    p = strchr(p, ':');
    if (!p) return -1;
    p++;
    while (*p && (*p == ' ' || *p == '\t' || *p == '\r' || *p == '\n')) p++;
    int quoted = 0;
    if (*p == '\"') { quoted = 1; p++; }
    size_t i = 0;
    while (*p && i + 1 < outsz) {
        if (quoted) {
            if (*p == '\\' && p[1]) p++;
            else if (*p == '\"') break;
            out[i++] = *p++;
        } else {
            if (*p == ',' || *p == '}' || *p == ' ' || *p == '\r' || *p == '\n' || *p == '\t') break;
            out[i++] = *p++;
        }
    }
    out[i] = '\0';
    return i > 0 ? 0 : -1;
}

int json_get_int(const char *json, const char *key, int *out) {
    char buf[32] = {0};
    if (json_get_str(json, key, buf, sizeof(buf)) == 0) {
        *out = atoi(buf);
        return 0;
    }
    return -1;
}

const char *json_get_raw_object(const char *json, const char *objectKey) {
    if (!json || !objectKey) return NULL;
    const char *cfg = find_key_case_insensitive(json, objectKey);
    if (!cfg) return NULL;
    const char *p = strchr(cfg, ':');
    if (!p) return NULL;
    p++;
    while (*p && *p != '{') p++;
    if (*p == '{') return p;
    return NULL;
}

int json_get_int_in_object(const char *json, const char *objectKey, const char *key, int *out) {
    if (!json || !key) return -1;
    if (json_get_int(json, key, out) == 0) return 0;
    if (!objectKey) return -1;
    const char *obj = json_get_raw_object(json, objectKey);
    if (!obj) return -1;
    return json_get_int(obj, key, out);
}