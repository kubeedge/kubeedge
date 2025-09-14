#include "stream.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <time.h>
#include <unistd.h>
#include <cjson/cJSON.h>

#ifdef ENABLE_STREAM
// FFmpeg C API（只在启用流处理时包含）
#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libavutil/avutil.h>
#include <libavutil/imgutils.h>
#include <libswscale/swscale.h>
#endif

// 解析流配置（总是可用）
int stream_parse_config(const char *json, StreamConfig *out) {
    if (!json || !out) return -1;
    
    memset(out, 0, sizeof(StreamConfig));
    
    cJSON *root = cJSON_Parse(json);
    if (!root) {
        log_error("Failed to parse stream config JSON");
        return -1;
    }
    
    cJSON *format = cJSON_GetObjectItem(root, "format");
    cJSON *outputDir = cJSON_GetObjectItem(root, "outputDir");
    cJSON *frameCount = cJSON_GetObjectItem(root, "frameCount");
    cJSON *frameInterval = cJSON_GetObjectItem(root, "frameInterval");
    cJSON *videoNum = cJSON_GetObjectItem(root, "videoNum");
    
    out->format = format ? strdup(format->valuestring) : strdup("jpg");
    out->outputDir = outputDir ? strdup(outputDir->valuestring) : strdup("./output");
    out->frameCount = frameCount ? frameCount->valueint : 10;
    out->frameInterval = frameInterval ? frameInterval->valueint : 1000000000; // 1秒
    out->videoNum = videoNum ? videoNum->valueint : 1;
    
    cJSON_Delete(root);
    return 0;
}

// 释放流配置（总是可用）
void stream_free_config(StreamConfig *config) {
    if (!config) return;
    
    free(config->format);
    free(config->outputDir);
    config->format = NULL;
    config->outputDir = NULL;
}

// 生成带时间戳的文件名（总是可用）
char* stream_gen_filename(const char *dir, const char *format) {
    if (!dir || !format) return NULL;
    
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    
    char *filename = malloc(512);
    if (!filename) return NULL;
    
    snprintf(filename, 512, "%s/f%ld%09ld.%s", 
             dir, ts.tv_sec, ts.tv_nsec, format);
    
    return filename;
}

#ifdef ENABLE_STREAM
// ==================== 启用流处理时的实现 ====================

// 保存单个帧为图片
static int save_frame_to_image(AVFrame *frame, int width, int height, 
                              const char *dir, const char *format) {
    char *output_file = stream_gen_filename(dir, format);
    if (!output_file) return -1;
    
    AVFormatContext *output_fmt_ctx = NULL;
    int ret = avformat_alloc_output_context2(&output_fmt_ctx, NULL, NULL, output_file);
    if (ret < 0) {
        log_error("Could not create output context");
        free(output_file);
        return -1;
    }
    
    const AVOutputFormat *ofmt = av_guess_format(NULL, output_file, NULL);
    if (!ofmt) {
        log_error("Could not guess output format");
        avformat_free_context(output_fmt_ctx);
        free(output_file);
        return -1;
    }
    output_fmt_ctx->oformat = ofmt;
    
    // 打开输出文件
    if (!(ofmt->flags & AVFMT_NOFILE)) {
        ret = avio_open(&output_fmt_ctx->pb, output_file, AVIO_FLAG_WRITE);
        if (ret < 0) {
            log_error("Could not open output file '%s'", output_file);
            avformat_free_context(output_fmt_ctx);
            free(output_file);
            return -1;
        }
    }
    
    // 创建视频流
    AVStream *out_stream = avformat_new_stream(output_fmt_ctx, NULL);
    if (!out_stream) {
        log_error("Failed allocating output stream");
        if (!(ofmt->flags & AVFMT_NOFILE)) avio_closep(&output_fmt_ctx->pb);
        avformat_free_context(output_fmt_ctx);
        free(output_file);
        return -1;
    }
    
    // 设置编码器参数
    AVCodecContext *codec_ctx = avcodec_alloc_context3(NULL);
    codec_ctx->codec_id = ofmt->video_codec;
    codec_ctx->codec_type = AVMEDIA_TYPE_VIDEO;
    codec_ctx->pix_fmt = AV_PIX_FMT_YUVJ420P;
    codec_ctx->width = width;
    codec_ctx->height = height;
    codec_ctx->time_base = (AVRational){1, 25};
    
    // 查找编码器
    const AVCodec *codec = avcodec_find_encoder(codec_ctx->codec_id);
    if (!codec) {
        log_error("Codec not found");
        avcodec_free_context(&codec_ctx);
        if (!(ofmt->flags & AVFMT_NOFILE)) avio_closep(&output_fmt_ctx->pb);
        avformat_free_context(output_fmt_ctx);
        free(output_file);
        return -1;
    }
    
    // 打开编码器
    ret = avcodec_open2(codec_ctx, codec, NULL);
    if (ret < 0) {
        log_error("Could not open codec");
        avcodec_free_context(&codec_ctx);
        if (!(ofmt->flags & AVFMT_NOFILE)) avio_closep(&output_fmt_ctx->pb);
        avformat_free_context(output_fmt_ctx);
        free(output_file);
        return -1;
    }
    
    // 复制编码器参数到流
    ret = avcodec_parameters_from_context(out_stream->codecpar, codec_ctx);
    if (ret < 0) {
        log_error("Failed to copy codec parameters");
        avcodec_free_context(&codec_ctx);
        if (!(ofmt->flags & AVFMT_NOFILE)) avio_closep(&output_fmt_ctx->pb);
        avformat_free_context(output_fmt_ctx);
        free(output_file);
        return -1;
    }
    
    // 写文件头
    ret = avformat_write_header(output_fmt_ctx, NULL);
    if (ret < 0) {
        log_error("Error occurred when opening output file");
        avcodec_free_context(&codec_ctx);
        if (!(ofmt->flags & AVFMT_NOFILE)) avio_closep(&output_fmt_ctx->pb);
        avformat_free_context(output_fmt_ctx);
        free(output_file);
        return -1;
    }
    
    // 编码并写入帧
    AVPacket *packet = av_packet_alloc();
    ret = avcodec_send_frame(codec_ctx, frame);
    if (ret < 0) {
        log_error("Error sending frame to encoder");
    } else {
        ret = avcodec_receive_packet(codec_ctx, packet);
        if (ret == 0) {
            packet->stream_index = out_stream->index;
            ret = av_write_frame(output_fmt_ctx, packet);
            if (ret < 0) {
                log_error("Error writing frame");
            }
        }
    }
    
    // 写文件尾
    av_write_trailer(output_fmt_ctx);
    
    // 清理资源
    av_packet_free(&packet);
    avcodec_free_context(&codec_ctx);
    if (!(ofmt->flags & AVFMT_NOFILE)) avio_closep(&output_fmt_ctx->pb);
    avformat_free_context(output_fmt_ctx);
    free(output_file);
    
    return 0;
}

// 保存视频帧为图片
int stream_save_frame(const char *input_url, const char *output_dir, const char *format, 
                     int frame_count, int frame_interval) {
    if (!input_url || !output_dir || !format) return -1;
    
    log_info("Starting frame extraction: %d frames from %s to %s (format: %s, interval: %dns)", 
             frame_count, input_url, output_dir, format, frame_interval);
    
    AVFormatContext *fmt_ctx = NULL;
    AVCodecContext *codec_ctx = NULL;
    AVFrame *frame = NULL;
    AVFrame *frame_rgb = NULL;
    struct SwsContext *sws_ctx = NULL;
    AVPacket *packet = NULL;
    int video_stream_idx = -1;
    int ret = 0;
    uint8_t *buffer = NULL;
    
    // 设置 RTSP 选项
    AVDictionary *opts = NULL;
    av_dict_set(&opts, "rtsp_transport", "tcp", 0);
    av_dict_set(&opts, "max_delay", "5000000", 0);
    av_dict_set(&opts, "stimeout", "10000000", 0); // 10秒超时
    
    // 打开输入流
    ret = avformat_open_input(&fmt_ctx, input_url, NULL, &opts);
    av_dict_free(&opts);
    if (ret < 0) {
        log_error("Unable to open stream %s", input_url);
        return -1;
    }
    
    // 获取流信息
    ret = avformat_find_stream_info(fmt_ctx, NULL);
    if (ret < 0) {
        log_error("Couldn't find stream information");
        goto cleanup;
    }
    
    // 查找视频流
    for (unsigned int i = 0; i < fmt_ctx->nb_streams; i++) {
        if (fmt_ctx->streams[i]->codecpar->codec_type == AVMEDIA_TYPE_VIDEO) {
            video_stream_idx = i;
            break;
        }
    }
    
    if (video_stream_idx == -1) {
        log_error("Couldn't find video stream");
        ret = -1;
        goto cleanup;
    }
    
    // 获取解码器
    const AVCodec *codec = avcodec_find_decoder(fmt_ctx->streams[video_stream_idx]->codecpar->codec_id);
    if (!codec) {
        log_error("Unsupported codec");
        ret = -1;
        goto cleanup;
    }
    
    // 分配解码器上下文
    codec_ctx = avcodec_alloc_context3(codec);
    if (!codec_ctx) {
        log_error("Failed to allocate codec context");
        ret = -1;
        goto cleanup;
    }
    
    // 复制解码器参数
    ret = avcodec_parameters_to_context(codec_ctx, fmt_ctx->streams[video_stream_idx]->codecpar);
    if (ret < 0) {
        log_error("Failed to copy codec parameters");
        goto cleanup;
    }
    
    // 打开解码器
    ret = avcodec_open2(codec_ctx, codec, NULL);
    if (ret < 0) {
        log_error("Could not open codec");
        goto cleanup;
    }
    
    // 分配帧
    frame = av_frame_alloc();
    frame_rgb = av_frame_alloc();
    if (!frame || !frame_rgb) {
        log_error("Could not allocate frames");
        ret = -1;
        goto cleanup;
    }
    
    // 分配图像缓冲区
    int num_bytes = av_image_get_buffer_size(AV_PIX_FMT_YUVJ420P, codec_ctx->width, codec_ctx->height, 1);
    buffer = av_malloc(num_bytes);
    if (!buffer) {
        log_error("Could not allocate image buffer");
        ret = -1;
        goto cleanup;
    }
    
    // 设置帧缓冲区
    av_image_fill_arrays(frame_rgb->data, frame_rgb->linesize, buffer, 
                        AV_PIX_FMT_YUVJ420P, codec_ctx->width, codec_ctx->height, 1);
    
    // 初始化缩放上下文
    sws_ctx = sws_getContext(codec_ctx->width, codec_ctx->height, codec_ctx->pix_fmt,
                            codec_ctx->width, codec_ctx->height, AV_PIX_FMT_YUVJ420P,
                            SWS_BICUBIC, NULL, NULL, NULL);
    if (!sws_ctx) {
        log_error("Could not initialize SWS context");
        ret = -1;
        goto cleanup;
    }
    
    // 分配数据包
    packet = av_packet_alloc();
    if (!packet) {
        log_error("Could not allocate packet");
        ret = -1;
        goto cleanup;
    }
    
    int frame_num = 0;
    int failure_num = 0;
    int failure_count = 5 * frame_count;
    
    // 开始读取和处理帧
    while (frame_num < frame_count && failure_num < failure_count) {
        ret = av_read_frame(fmt_ctx, packet);
        if (ret < 0) {
            log_error("Read frame failed");
            usleep(1000000); // 1秒
            failure_num++;
            continue;
        }
        
        // 检查是否为视频流数据包
        if (packet->stream_index != video_stream_idx) {
            av_packet_unref(packet);
            failure_num++;
            continue;
        }
        
        // 发送数据包到解码器
        ret = avcodec_send_packet(codec_ctx, packet);
        if (ret < 0) {
            log_error("Error while sending packet to decoder");
            av_packet_unref(packet);
            failure_num++;
            continue;
        }
        
        // 接收解码后的帧
        ret = avcodec_receive_frame(codec_ctx, frame);
        if (ret == AVERROR(EAGAIN) || ret == AVERROR_EOF) {
            av_packet_unref(packet);
            failure_num++;
            continue;
        } else if (ret < 0) {
            log_error("Error while receiving frame from decoder");
            av_packet_unref(packet);
            failure_num++;
            continue;
        }
        
        // 转换像素格式
        sws_scale(sws_ctx, (const uint8_t * const*)frame->data, frame->linesize, 
                 0, codec_ctx->height, frame_rgb->data, frame_rgb->linesize);
        
        // 保存帧
        frame_rgb->width = codec_ctx->width;
        frame_rgb->height = codec_ctx->height;
        frame_rgb->format = AV_PIX_FMT_YUVJ420P;
        
        ret = save_frame_to_image(frame_rgb, codec_ctx->width, codec_ctx->height, 
                                 output_dir, format);
        if (ret < 0) {
            log_error("Failed to save frame %d", frame_num + 1);
        } else {
            frame_num++;
            log_info("Saved frame %d/%d", frame_num, frame_count);
        }
        
        av_packet_unref(packet);
        
        // 帧间隔延时
        if (frame_interval > 0) {
            usleep(frame_interval / 1000); // 转换为微秒
        }
    }
    
    if (failure_num >= failure_count) {
        log_error("The number of failed attempts to save frames has reached the upper limit");
        ret = -1;
    } else {
        log_info("Successfully saved %d frames", frame_num);
        ret = 0;
    }
    
cleanup:
    if (buffer) av_free(buffer);
    if (packet) av_packet_free(&packet);
    if (frame) av_frame_free(&frame);
    if (frame_rgb) av_frame_free(&frame_rgb);
    if (sws_ctx) sws_freeContext(sws_ctx);
    if (codec_ctx) avcodec_free_context(&codec_ctx);
    if (fmt_ctx) avformat_close_input(&fmt_ctx);
    
    return ret;
}

// 保存视频片段
int stream_save_video(const char *input_url, const char *output_dir, const char *format, 
                     int frame_count, int video_num) {
    if (!input_url || !output_dir || !format) return -1;
    
    log_info("Starting video segment saving: %d segments from %s to %s (format: %s, frames per segment: %d)", 
             video_num, input_url, output_dir, format, frame_count);
    
    AVFormatContext *input_fmt_ctx = NULL;
    AVFormatContext *output_fmt_ctx = NULL;
    AVPacket *packet = NULL;
    int *stream_mapping = NULL;
    int stream_mapping_size = 0;
    int ret = 0;
    
    // 设置 RTSP 选项
    AVDictionary *opts = NULL;
    av_dict_set(&opts, "rtsp_transport", "tcp", 0);
    av_dict_set(&opts, "max_delay", "5000000", 0);
    av_dict_set(&opts, "stimeout", "10000000", 0); // 10秒超时
    
    // 打开输入流
    ret = avformat_open_input(&input_fmt_ctx, input_url, NULL, &opts);
    av_dict_free(&opts);
    if (ret < 0) {
        log_error("Could not open input stream '%s'", input_url);
        return -1;
    }
    
    // 获取流信息
    ret = avformat_find_stream_info(input_fmt_ctx, NULL);
    if (ret < 0) {
        log_error("Failed to retrieve input stream information");
        goto cleanup;
    }
    
    // 初始化流映射
    stream_mapping_size = input_fmt_ctx->nb_streams;
    stream_mapping = av_calloc(stream_mapping_size, sizeof(*stream_mapping));
    if (!stream_mapping) {
        ret = AVERROR(ENOMEM);
        goto cleanup;
    }
    
    // 分配数据包
    packet = av_packet_alloc();
    if (!packet) {
        ret = AVERROR(ENOMEM);
        goto cleanup;
    }
    
    // 为每个视频片段生成文件
    for (int idx = 0; idx < video_num; idx++) {
        char *output_file = stream_gen_filename(output_dir, format);
        if (!output_file) {
            ret = AVERROR(ENOMEM);
            goto cleanup;
        }
        
        // 分配输出上下文
        ret = avformat_alloc_output_context2(&output_fmt_ctx, NULL, NULL, output_file);
        if (ret < 0) {
            log_error("Could not create output context for segment %d", idx + 1);
            free(output_file);
            goto cleanup;
        }
        
        int stream_index = 0;
        // 为每个输入流创建对应的输出流
        for (unsigned int i = 0; i < input_fmt_ctx->nb_streams; i++) {
            AVStream *in_stream = input_fmt_ctx->streams[i];
            AVCodecParameters *in_codecpar = in_stream->codecpar;
            
            if (in_codecpar->codec_type != AVMEDIA_TYPE_VIDEO &&
                in_codecpar->codec_type != AVMEDIA_TYPE_AUDIO &&
                in_codecpar->codec_type != AVMEDIA_TYPE_SUBTITLE) {
                stream_mapping[i] = -1;
                continue;
            }
            
            stream_mapping[i] = stream_index++;
            
            AVStream *out_stream = avformat_new_stream(output_fmt_ctx, NULL);
            if (!out_stream) {
                log_error("Failed allocating output stream for segment %d", idx + 1);
                ret = AVERROR_UNKNOWN;
                free(output_file);
                goto cleanup;
            }
            
            ret = avcodec_parameters_copy(out_stream->codecpar, in_codecpar);
            if (ret < 0) {
                log_error("Failed to copy codec parameters for segment %d", idx + 1);
                free(output_file);
                goto cleanup;
            }
            out_stream->codecpar->codec_tag = 0;
        }
        
        // 打开输出文件
        if (!(output_fmt_ctx->oformat->flags & AVFMT_NOFILE)) {
            ret = avio_open(&output_fmt_ctx->pb, output_file, AVIO_FLAG_WRITE);
            if (ret < 0) {
                log_error("Could not open output file '%s'", output_file);
                free(output_file);
                goto cleanup;
            }
        }
        
        // 设置分片选项（对于 MP4）
        AVDictionary *output_opts = NULL;
        if (strcmp(format, "mp4") == 0) {
            av_dict_set(&output_opts, "movflags", "frag_keyframe+empty_moov+default_base_moof", 0);
        }
        
        // 写文件头
        ret = avformat_write_header(output_fmt_ctx, &output_opts);
        av_dict_free(&output_opts);
        if (ret < 0) {
            log_error("Error occurred when opening output file for segment %d", idx + 1);
            free(output_file);
            goto cleanup;
        }
        
        // 写入指定数量的帧
        int written_frames = 0;
        while (written_frames < frame_count) {
            ret = av_read_frame(input_fmt_ctx, packet);
            if (ret < 0) {
                log_error("Read frame failed for segment %d", idx + 1);
                break;
            }
            
            int stream_index = packet->stream_index;
            if (stream_index >= stream_mapping_size || stream_mapping[stream_index] < 0) {
                av_packet_unref(packet);
                continue;
            }
            
            packet->stream_index = stream_mapping[stream_index];
            
            AVStream *in_stream = input_fmt_ctx->streams[stream_index];
            AVStream *out_stream = output_fmt_ctx->streams[packet->stream_index];
            
            // 重新计算时间戳
            av_packet_rescale_ts(packet, in_stream->time_base, out_stream->time_base);
            packet->pos = -1;
            
            ret = av_interleaved_write_frame(output_fmt_ctx, packet);
            if (ret < 0) {
                log_error("Error muxing packet for segment %d", idx + 1);
                av_packet_unref(packet);
                continue;
            }
            
            written_frames++;
            av_packet_unref(packet);
        }
        
        // 写文件尾
        av_write_trailer(output_fmt_ctx);
        
        // 关闭输出文件
        if (!(output_fmt_ctx->oformat->flags & AVFMT_NOFILE)) {
            avio_closep(&output_fmt_ctx->pb);
        }
        
        avformat_free_context(output_fmt_ctx);
        output_fmt_ctx = NULL;
        free(output_file);
        
        log_info("Saved video segment %d/%d (%d frames)", idx + 1, video_num, written_frames);
    }
    
    log_info("Successfully saved %d video segments", video_num);
    ret = 0;
    
cleanup:
    if (packet) av_packet_free(&packet);
    if (stream_mapping) av_freep(&stream_mapping);
    if (output_fmt_ctx) {
        if (!(output_fmt_ctx->oformat->flags & AVFMT_NOFILE)) {
            avio_closep(&output_fmt_ctx->pb);
        }
        avformat_free_context(output_fmt_ctx);
    }
    if (input_fmt_ctx) avformat_close_input(&input_fmt_ctx);
    
    return ret;
}

// 流处理 handler（对应 Go 版本的 StreamHandler）
int stream_handler(const Twin *twin, CustomizedClient *client, const VisitorConfig *visitorConfig) {
    if (!twin || !client || !visitorConfig) {
        log_error("Invalid parameters for stream handler");
        return -1;
    }
    
    // 验证 Twin 结构
    if (!twin->propertyName) {
        log_error("Twin propertyName is NULL");
        return -1;
    }
    
    log_info("Processing stream handler for property: %s", twin->propertyName);
    
    // 1. 获取 RTSP URI（从设备获取）
    void *device_data = NULL;
    int ret = GetDeviceData(client, visitorConfig, &device_data);
    if (ret != 0 || !device_data) {
        log_error("Failed to get device data (RTSP URI)");
        return -1;
    }
    
    char *stream_uri = (char*)device_data;
    log_info("Got RTSP URI: %s for property: %s", stream_uri, twin->propertyName);
    
    // 2. 解析流配置（从 visitorConfig 中）
    StreamConfig stream_config;
    if (stream_parse_config(visitorConfig->configData, &stream_config) != 0) {
        log_error("Failed to parse stream config");
        free(device_data);
        return -1;
    }
    
    // 3. 根据 twin 的 propertyName 分发处理（对应 Go 版本的 switch 逻辑）
    ret = -1;
    
    if (strcmp(twin->propertyName, "SaveFrame") == 0) {
        // 保存视频帧（对应 Go 版本的 case "SaveFrame"）
        log_info("Processing SaveFrame for property: %s", twin->propertyName);
        ret = stream_save_frame(stream_uri, stream_config.outputDir, 
                               stream_config.format, stream_config.frameCount, 
                               stream_config.frameInterval);
        
    } else if (strcmp(twin->propertyName, "SaveVideo") == 0) {
        // 保存视频片段（对应 Go 版本的 case "SaveVideo"）
        log_info("Processing SaveVideo for property: %s", twin->propertyName);
        ret = stream_save_video(stream_uri, stream_config.outputDir, 
                               stream_config.format, stream_config.frameCount, 
                               stream_config.videoNum);
        
    } else {
        // 对应 Go 版本的 default case
        log_error("Cannot find the processing method for the corresponding Property %s of the stream data", 
                 twin->propertyName);
        ret = -1;
    }
    
    // 4. 清理资源
    stream_free_config(&stream_config);
    free(device_data);
    
    if (ret == 0) {
        log_info("Successfully processed streaming data by %s", twin->propertyName);
    } else {
        log_error("Failed to process streaming data for %s", twin->propertyName);
    }
    
    return ret;
}

#else
// ==================== 不支持流处理时的实现 ====================

// 不支持流处理时的 handler
int stream_handler_no_support(const Twin *twin, CustomizedClient *client, const VisitorConfig *visitorConfig) {
    (void)twin; (void)client; (void)visitorConfig; // 避免未使用变量警告
    log_error("Need to add the stream flag when compiling if you want to enable stream data processing.");
    return -1;
}

#endif // ENABLE_STREAM