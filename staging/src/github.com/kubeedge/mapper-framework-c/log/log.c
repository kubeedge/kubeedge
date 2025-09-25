#include "log/log.h"
#include <stdarg.h>
#include <stdlib.h>
#include <time.h>
#include <stdio.h>

static LogLevel current_level = LOG_LEVEL_INFO;

void log_init(void) {
}

void log_set_level(LogLevel level) {
    current_level = level;
}

static const char* level_str(LogLevel level) {
    switch (level) {
        case LOG_LEVEL_DEBUG: return "DEBUG";
        case LOG_LEVEL_INFO:  return "INFO";
        case LOG_LEVEL_WARN:  return "WARN";
        case LOG_LEVEL_ERROR: return "ERROR";
        case LOG_LEVEL_FATAL: return "FATAL";
        default: return "UNKNOWN";
    }
}

static void log_write(LogLevel level, const char *fmt, va_list args) {
    if (level < current_level) return;
    time_t now = time(NULL);
    struct tm *t = localtime(&now);
    fprintf(stderr, "[%04d-%02d-%02d %02d:%02d:%02d] %s: ",
        t->tm_year+1900, t->tm_mon+1, t->tm_mday,
        t->tm_hour, t->tm_min, t->tm_sec,
        level_str(level));
    vfprintf(stderr, fmt, args);
    fprintf(stderr, "\n");
    if (level == LOG_LEVEL_FATAL) {
        log_flush();
        exit(EXIT_FAILURE);
    }
}

void log_debug(const char *fmt, ...) {
    va_list args; va_start(args, fmt);
    log_write(LOG_LEVEL_DEBUG, fmt, args);
    va_end(args);
}
void log_info(const char *fmt, ...) {
    va_list args; va_start(args, fmt);
    log_write(LOG_LEVEL_INFO, fmt, args);
    va_end(args);
}
void log_warn(const char *fmt, ...) {
    va_list args; va_start(args, fmt);
    log_write(LOG_LEVEL_WARN, fmt, args);
    va_end(args);
}
void log_error(const char *fmt, ...) {
    va_list args; va_start(args, fmt);
    log_write(LOG_LEVEL_ERROR, fmt, args);
    va_end(args);
}
void log_fatal(const char *fmt, ...) {
    va_list args; va_start(args, fmt);
    log_write(LOG_LEVEL_FATAL, fmt, args);
    va_end(args);
}

void log_flush(void) {
    fflush(stderr);
}