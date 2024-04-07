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
	"time"
	"unsafe"

	"github.com/sailorvii/goav/avcodec"
	"github.com/sailorvii/goav/avformat"
	"github.com/sailorvii/goav/avutil"
	"github.com/sailorvii/goav/swscale"
	"k8s.io/klog/v2"
)

// GenFileName generate file name with current time. Formate f<year><month><day><hour><minute><second><millisecond>.<format>
func GenFileName(dir string, format string) string {
	return fmt.Sprintf("%s/f%s.%s", dir, time.Now().Format(time.RFC3339Nano), format)
}

func save(frame *avutil.Frame, width int, height int, dir string, format string) error {
	// Save video frames to picture file
	outputFile := GenFileName(dir, format)
	var outputFmtCtx *avformat.Context
	avformat.AvAllocOutputContext2(&outputFmtCtx, nil, nil, &outputFile)
	if outputFmtCtx == nil {
		return errors.New("Could not create output context")
	}
	defer outputFmtCtx.AvformatFreeContext()

	ofmt := avformat.AvGuessFormat("", outputFile, "")
	outputFmtCtx.SetOformat(ofmt)

	avIOContext, err := avformat.AvIOOpen(outputFile, avformat.AVIO_FLAG_WRITE)
	if err != nil {
		return fmt.Errorf("Could not open output file '%s'", outputFile)
	}
	outputFmtCtx.SetPb(avIOContext)

	outStream := outputFmtCtx.AvformatNewStream(nil)
	if outStream == nil {
		return errors.New("Failed allocating output stream")
	}

	// Set the frame format
	pCodecCtx := outStream.Codec()
	pCodecCtx.SetCodecId(ofmt.GetVideoCodec())
	pCodecCtx.SetCodecType(avformat.AVMEDIA_TYPE_VIDEO)
	pCodecCtx.SetPixelFormat(avcodec.AV_PIX_FMT_YUVJ420P)
	pCodecCtx.SetWidth(width)
	pCodecCtx.SetHeight(height)
	pCodecCtx.SetTimeBase(1, 25)
	outputFmtCtx.AvDumpFormat(0, outputFile, 1)

	// Get video codec
	pCodec := avcodec.AvcodecFindEncoder(pCodecCtx.CodecId())
	if pCodec == nil {
		return errors.New("Codec not found.")
	}
	defer pCodecCtx.AvcodecClose()

	// open video codec
	cctx := avcodec.Context(*pCodecCtx)
	defer cctx.AvcodecClose()
	if cctx.AvcodecOpen2(pCodec, nil) < 0 {
		return errors.New("Could not open codec.")
	}

	outputFmtCtx.AvformatWriteHeader(nil)
	ySize := width * height

	// Write media data to media files
	var packet avcodec.Packet
	packet.AvNewPacket(ySize * 3)
	defer packet.AvPacketUnref()
	var gotPicture int
	if cctx.AvcodecEncodeVideo2(&packet, frame, &gotPicture) < 0 {
		return errors.New("Encode Error")
	}
	if gotPicture == 1 {
		packet.SetStreamIndex(outStream.Index())
		outputFmtCtx.AvWriteFrame(&packet)
	}

	outputFmtCtx.AvWriteTrailer()
	if outputFmtCtx.Oformat().GetFlags()&avformat.AVFMT_NOFILE == 0 {
		if err = outputFmtCtx.Pb().Close(); err != nil {
			return fmt.Errorf("close output fmt context failed: %v", err)
		}
	}
	return nil
}

// SaveFrame save frame.
func SaveFrame(input string, outDir string, format string, frameCount int, frameInterval int) error {
	// Open video file
	avformat.AvDictSet(&avformat.Dict, "rtsp_transport", "tcp", 0)
	avformat.AvDictSet(&avformat.Dict, "max_delay", "5000000", 0)

	pFormatContext := avformat.AvformatAllocContext()
	if avformat.AvformatOpenInput(&pFormatContext, input, nil, &avformat.Dict) != 0 {
		return fmt.Errorf("Unable to open file %s", input)
	}
	// Retrieve stream information
	if pFormatContext.AvformatFindStreamInfo(nil) < 0 {
		return errors.New("Couldn't find stream information")
	}
	// Dump information about file onto standard error
	pFormatContext.AvDumpFormat(0, input, 0)
	// Find the first video stream
	streamIndex := -1
	for i := 0; i < int(pFormatContext.NbStreams()); i++ {
		if pFormatContext.Streams()[i].CodecParameters().AvCodecGetType() == avformat.AVMEDIA_TYPE_VIDEO {
			streamIndex = i
			break
		}
	}
	if streamIndex == -1 {
		return errors.New("couldn't find video stream")
	}
	// Get a pointer to the codec context for the video stream
	pCodecCtxOrig := pFormatContext.Streams()[streamIndex].Codec()
	// Find the decoder for the video stream
	pCodec := avcodec.AvcodecFindDecoder(pCodecCtxOrig.CodecId())
	if pCodec == nil {
		return errors.New("unsupported codec")
	}
	// Copy context
	pCodecCtx := pCodec.AvcodecAllocContext3()
	if pCodecCtx.AvcodecCopyContext((*avcodec.Context)(unsafe.Pointer(pCodecCtxOrig))) != 0 {
		return errors.New("couldn't copy codec context")
	}

	// Open codec
	if pCodecCtx.AvcodecOpen2(pCodec, nil) < 0 {
		return errors.New("could not open codec")
	}

	// Allocate video frame
	pFrame := avutil.AvFrameAlloc()

	// Allocate an AVFrame structure
	pFrameRGB := avutil.AvFrameAlloc()
	if pFrameRGB == nil {
		return errors.New("unable to allocate RGB Frame")
	}
	// Determine required buffer size and allocate buffer
	numBytes := uintptr(avcodec.AvpictureGetSize(avcodec.AV_PIX_FMT_YUVJ420P, pCodecCtx.Width(),
		pCodecCtx.Height()))
	buffer := avutil.AvMalloc(numBytes)

	// Assign appropriate parts of buffer to image planes in pFrameRGB
	// Note that pFrameRGB is an AVFrame, but AVFrame is a superset
	// of AVPicture
	avp := (*avcodec.Picture)(unsafe.Pointer(pFrameRGB))
	avp.AvpictureFill((*uint8)(buffer), avcodec.AV_PIX_FMT_YUVJ420P, pCodecCtx.Width(), pCodecCtx.Height())

	// initialize SWS context for software scaling
	swsCtx := swscale.SwsGetcontext(
		pCodecCtx.Width(),
		pCodecCtx.Height(),
		(swscale.PixelFormat)(pCodecCtx.PixFmt()),
		pCodecCtx.Width(),
		pCodecCtx.Height(),
		avcodec.AV_PIX_FMT_YUVJ420P,
		avcodec.SWS_BICUBIC,
		nil,
		nil,
		nil,
	)
	frameNum := 0
	failureNum := 0
	failureCount := 5 * frameCount
	packet := avcodec.AvPacketAlloc()
	// Start capturing and saving video frames
	for {
		if failureNum >= failureCount {
			klog.Error("the number of failed attempts to save frames has reached the upper limit")
			return errors.New("the number of failed attempts to save frames has reached the upper limit")
		}

		if pFormatContext.AvReadFrame(packet) < 0 {
			klog.Error("Read frame failed")
			time.Sleep(time.Second)
			continue
		}

		// Is this a packet from the video stream?
		if packet.StreamIndex() != streamIndex {
			failureNum++
			continue
		}

		// Decode video frame
		response := pCodecCtx.AvcodecSendPacket(packet)
		if response < 0 {
			klog.Errorf("Error while sending a packet to the decoder: %s", avutil.ErrorFromCode(response))
			failureNum++
			continue
		}
		response = pCodecCtx.AvcodecReceiveFrame((*avutil.Frame)(unsafe.Pointer(pFrame)))
		if response == avutil.AvErrorEAGAIN || response == avutil.AvErrorEOF {
			failureNum++
			continue
		} else if response < 0 {
			klog.Errorf("Error while receiving a frame from the decoder: %s", avutil.ErrorFromCode(response))
			failureNum++
			continue
		}
		// Convert the image from its native format to RGB
		swscale.SwsScale2(swsCtx, avutil.Data(pFrame),
			avutil.Linesize(pFrame), 0, pCodecCtx.Height(),
			avutil.Data(pFrameRGB), avutil.Linesize(pFrameRGB))

		// Save the frame to disk
		err := save(pFrameRGB, pCodecCtx.Width(), pCodecCtx.Height(), outDir, format)
		if err != nil {
			klog.Error(err)
			continue
		}
		frameNum++
		if frameNum >= frameCount {
			return nil
		}
		time.Sleep(time.Nanosecond * time.Duration(frameInterval))
	}
}
