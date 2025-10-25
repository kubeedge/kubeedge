#ifndef STREAM_H
#define STREAM_H

#include <stdint.h>
#include "common/configmaptype.h"
#include "driver/driver.h"

typedef struct
{
    char *format;
    char *outputDir;
    int frameCount;
    int frameInterval;
    int videoNum;
} StreamConfig;

int stream_parse_config(const char *json, StreamConfig *out);
void stream_free_config(StreamConfig *config);
char *stream_gen_filename(const char *dir, const char *format);

#ifdef ENABLE_STREAM

int stream_save_frame(const char *input_url, const char *output_dir, const char *format,
                      int frame_count, int frame_interval);

int stream_save_video(const char *input_url, const char *output_dir, const char *format,
                      int frame_count, int video_num);

int stream_handler(const Twin *twin, CustomizedClient *client, const VisitorConfig *visitorConfig);

static inline int stream_is_supported(void) { return 1; }

#else

static inline int stream_save_frame(const char *input_url, const char *output_dir, const char *format,
                                    int frame_count, int frame_interval)
{
    (void)input_url;
    (void)output_dir;
    (void)format;
    (void)frame_count;
    (void)frame_interval;
    return -1;
}

static inline int stream_save_video(const char *input_url, const char *output_dir, const char *format,
                                    int frame_count, int video_num)
{
    (void)input_url;
    (void)output_dir;
    (void)format;
    (void)frame_count;
    (void)video_num;
    return -1;
}

int stream_handler_no_support(const Twin *twin, CustomizedClient *client, const VisitorConfig *visitorConfig);

static inline int stream_is_supported(void) { return 0; }

#endif

#endif