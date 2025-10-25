#include "string_util.h"
#include <stdio.h>
#include <ctype.h>
#include <string.h>
#include <stdlib.h>

void trim_str(char *s) {
    if (!s) return;
    char *p = s;
    while (*p && isspace((unsigned char)*p)) p++;
    if (p != s) memmove(s, p, strlen(p) + 1);
    size_t len = strlen(s);
    while (len > 0 && isspace((unsigned char)s[len - 1])) {
        s[--len] = 0;
    }
}

void sanitize_host(char *s) {
    if (!s) return;
    size_t w = 0;
    for (size_t r = 0; s[r]; ++r) {
        unsigned char c = (unsigned char)s[r];
        if ((c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
            c == '.' || c == '-' || c == '_' || c == ':') {
            s[w++] = (char)c;
        }
    }
    s[w] = 0;
}

void cleanup_escape_prefix(char *s) {
    if (!s) return;
    while (s[0] && (s[0]=='n' || s[0]=='t' || s[0]=='r')) {
        if (s[1]>='0' && s[1]<='9') {
            memmove(s, s+1, strlen(s+1)+1);
            continue;
        }
        if ((s[1]=='n'||s[1]=='t'||s[1]=='r') && (s[2]>='0'&&s[2]<='9')) {
            memmove(s, s+2, strlen(s+2)+1);
            continue;
        }
        break;
    }
}

void normalize_host_port(const char *rawHost, int rawPort,
                         char *outHost, size_t outHostSz, int *outPort) {
    snprintf(outHost, outHostSz, "%s", (rawHost && *rawHost) ? rawHost : "");
    trim_str(outHost);
    sanitize_host(outHost);
    if (outHost[0] == 0) {
        const char *envH = getenv("MAPPER_MODBUS_ADDR");
        if (envH && *envH) {
            snprintf(outHost, outHostSz, "%s", envH);
            trim_str(outHost);
            sanitize_host(outHost);
        }
    }
    if (outHost[0] == 0) {
        snprintf(outHost, outHostSz, "%s", "127.0.0.1");
    }
    int p = rawPort;
    if (p <= 0 || p > 65535) {
        const char *envp = getenv("MAPPER_MODBUS_PORT");
        if (envp && *envp) {
            int ep = atoi(envp);
            if (ep > 0 && ep <= 65535) p = ep;
        }
    }
    if (p <= 0 || p > 65535) p = 1502;
    *outPort = p;
}

void sanitize_id(const char *in, char *out, size_t outsz, const char *fallback)
{
    if (!out || outsz == 0) return;
    if (!in || !*in) {
        snprintf(out, outsz, "%s", fallback);
        return;
    }
    size_t j = 0;
    for (size_t i = 0; in[i] && j + 1 < outsz; ++i) {
        unsigned char c = (unsigned char)in[i];
        if (c >= 'A' && c <= 'Z') c = (unsigned char)(c - 'A' + 'a');
        if ((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
            c == '-' || c == '_' || c == '/')
        {
            out[j++] = (char)c;
        } else {
            out[j++] = '_';
        }
    }
    out[j] = '\0';
    if (j == 0) snprintf(out, outsz, "%s", fallback);
}