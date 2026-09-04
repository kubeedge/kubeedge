/*
Copyright 2024 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package stream

import (
	"errors"
	"fmt"

	"github.com/sailorvii/goav/avcodec"
	"github.com/sailorvii/goav/avformat"
	"github.com/sailorvii/goav/avutil"
	"k8s.io/klog/v2"
)

// SaveVideo save video.
func SaveVideo(inputFile string, outDir string, format string, frameCount int, videoNum int) error {
	var fragmentedMp4Options int
	//initialize input file with Context
	var inputFmtCtx *avformat.Context

	avformat.AvDictSet(&avformat.Dict, "rtsp_transport", "tcp", 0)
	avformat.AvDictSet(&avformat.Dict, "max_delay", "5000000", 0)

	if avformat.AvformatOpenInput(&inputFmtCtx, inputFile, nil, &avformat.Dict) < 0 {
		return fmt.Errorf("could not open input file '%s", inputFile)
	}
	defer inputFmtCtx.AvformatFreeContext()
	//read stream information

	if inputFmtCtx.AvformatFindStreamInfo(nil) < 0 {
		return errors.New("failed to retrieve input stream information")
	}

	//initialize streamMapping
	streamMappingSize := int(inputFmtCtx.NbStreams())
	streamMapping := make([]int, streamMappingSize)
	var streamIndex int

	validTypeMap := map[avcodec.MediaType]int{
		avformat.AVMEDIA_TYPE_VIDEO:    1,
		avformat.AVMEDIA_TYPE_AUDIO:    1,
		avformat.AVMEDIA_TYPE_SUBTITLE: 1,
	}
	var inCodecParam *avcodec.AvCodecParameters
	defer inCodecParam.AvCodecParametersFree()

	var outputFmtCtx *avformat.Context
	outputFile := GenFileName(outDir, format)
	avformat.AvAllocOutputContext2(&outputFmtCtx, nil, nil, &outputFile)
	if outputFmtCtx == nil {
		return errors.New("Could not create output context")
	}
	defer outputFmtCtx.AvformatFreeContext()

	for index, inStream := range inputFmtCtx.Streams() {
		inCodecParam = inStream.CodecParameters()
		inCodecType := inCodecParam.AvCodecGetType()

		if validTypeMap[inCodecType] == 0 {
			streamMapping[index] = -1
			continue
		}
		streamMapping[index] = streamIndex
		streamIndex++
		outStream := outputFmtCtx.AvformatNewStream(nil)
		if outStream == nil {
			return errors.New("Failed allocating output stream")
		}
		if inCodecParam.AvCodecParametersCopyTo(outStream.CodecParameters()) < 0 {
			return errors.New("Failed to copy codec parameters")
		}
	}

	// initialize opts
	var opts *avutil.Dictionary
	defer opts.AvDictFree()
	if fragmentedMp4Options != 0 {
		opts.AvDictSet("movflags", "frag_keyframe+empty_moov+default_base_moof", 0)
	}
	var packet avcodec.Packet
	defer packet.AvPacketUnref()

	// Capture a set number of video segments
	for idx := 0; idx < videoNum; idx++ {
		outputFile = GenFileName(outDir, format)
		// initialize output file with Context
		outputFmtCtx.AvDumpFormat(0, outputFile, 1)
		if outputFmtCtx.Oformat().GetFlags()&avformat.AVFMT_NOFILE == 0 {
			avIOContext, err := avformat.AvIOOpen(outputFile, avformat.AVIO_FLAG_WRITE)
			if err != nil {
				return fmt.Errorf("could not open output file '%s'", outputFile)
			}
			outputFmtCtx.SetPb(avIOContext)
		}

		if outputFmtCtx.AvformatWriteHeader(&opts) < 0 {
			return errors.New("Error occurred when opening output file")
		}
		// Capture and generate video according to the set number of frames
		for i := 1; i < frameCount; i++ {
			if inputFmtCtx.AvReadFrame(&packet) < 0 {
				return errors.New("read frame failed")
			}
			index := packet.StreamIndex()
			inputStream := inputFmtCtx.Streams()[index]
			if index >= streamMappingSize || streamMapping[index] < 0 {
				continue
			}
			packet.SetStreamIndex(streamMapping[index])
			outputStream := outputFmtCtx.Streams()[index]
			packet.AvPacketRescaleTs(inputStream.TimeBase(), outputStream.TimeBase())
			packet.SetPos(-1)
			if outputFmtCtx.AvInterleavedWriteFrame(&packet) < 0 {
				klog.Error("Error muxing packet")
				continue
			}
		}

		outputFmtCtx.AvWriteTrailer()
		if outputFmtCtx.Oformat().GetFlags()&avformat.AVFMT_NOFILE == 0 {
			if outputFmtCtx.Pb().Close() != nil {
				klog.Error("Error close output context")
				return errors.New("error close output context")
			}
		}
	}
	return nil
}
