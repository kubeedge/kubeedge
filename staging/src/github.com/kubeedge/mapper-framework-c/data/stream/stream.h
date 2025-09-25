#ifndef STREAM_H
#define STREAM_H

#include <stdint.h>
#include "common/configmaptype.h"  // 包含 Twin 定义
#include "driver/driver.h"         // 包含 CustomizedClient 和 VisitorConfig 定义

// 流处理配置结构体
typedef struct {
    char *format;        // 输出格式（jpg, mp4等）
    char *outputDir;     // 输出目录
    int frameCount;      // 帧数量
    int frameInterval;   // 帧间隔（纳秒）
    int videoNum;        // 视频片段数量
} StreamConfig;

// 基础流处理函数（总是可用）
int stream_parse_config(const char *json, StreamConfig *out);
void stream_free_config(StreamConfig *config);
char* stream_gen_filename(const char *dir, const char *format);

// 条件编译的流处理函数
#ifdef ENABLE_STREAM
// 图片帧处理
int stream_save_frame(const char *input_url, const char *output_dir, const char *format, 
                     int frame_count, int frame_interval);

// 视频处理
int stream_save_video(const char *input_url, const char *output_dir, const char *format, 
                     int frame_count, int video_num);

// 流处理 handler（对应 Go 版本的 StreamHandler）
int stream_handler(const Twin *twin, CustomizedClient *client, const VisitorConfig *visitorConfig);

// 检查是否支持流处理
static inline int stream_is_supported(void) { return 1; }

#else
// 不支持流处理时的占位函数
static inline int stream_save_frame(const char *input_url, const char *output_dir, const char *format, 
                                   int frame_count, int frame_interval) {
    (void)input_url; (void)output_dir; (void)format; (void)frame_count; (void)frame_interval;
    return -1; // 不支持
}

static inline int stream_save_video(const char *input_url, const char *output_dir, const char *format, 
                                   int frame_count, int video_num) {
    (void)input_url; (void)output_dir; (void)format; (void)frame_count; (void)video_num;
    return -1; // 不支持
}

// 不支持流处理时的 handler
int stream_handler_no_support(const Twin *twin, CustomizedClient *client, const VisitorConfig *visitorConfig);

// 检查是否支持流处理
static inline int stream_is_supported(void) { return 0; }

#endif // ENABLE_STREAM

#endif // STREAM_H