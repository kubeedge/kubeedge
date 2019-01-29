package packet

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"time"

	"github.com/256dpi/mercury"
)

// ErrDetectionOverflow is returned by the Decoder if the next packet couldn't
// be detect from the initial header bytes.
var ErrDetectionOverflow = errors.New("detection overflow")

// ErrReadLimitExceeded can be returned during a Receive if the connection
// exceeded its read limit.
var ErrReadLimitExceeded = errors.New("read limit exceeded")

// An Encoder wraps a Writer and continuously encodes packets.
type Encoder struct {
	writer *mercury.Writer
	buffer bytes.Buffer
}

// NewEncoder creates a new Encoder.
func NewEncoder(writer io.Writer) *Encoder {
	return &Encoder{
		writer: mercury.NewWriter(writer, time.Millisecond),
	}
}

// Write encodes and writes the passed packet to the write buffer.
func (e *Encoder) Write(pkt Generic, async bool) error {
	// reset and eventually grow buffer
	packetLength := pkt.Len()
	e.buffer.Reset()
	e.buffer.Grow(packetLength)
	buf := e.buffer.Bytes()[0:packetLength]

	// encode packet
	_, err := pkt.Encode(buf)
	if err != nil {
		return err
	}

	// write buffer
	if async {
		_, err = e.writer.Write(buf)
	} else {
		_, err = e.writer.WriteAndFlush(buf)
	}
	if err != nil {
		return err
	}

	return nil
}

// Flush flushes the writer buffer.
func (e *Encoder) Flush() error {
	return e.writer.Flush()
}

// A Decoder wraps a Reader and continuously decodes packets.
type Decoder struct {
	Limit int64

	reader *bufio.Reader
	buffer bytes.Buffer
}

// NewDecoder returns a new Decoder.
func NewDecoder(reader io.Reader) *Decoder {
	return &Decoder{
		reader: bufio.NewReader(reader),
	}
}

// Read reads the next packet from the buffered reader.
func (d *Decoder) Read() (Generic, error) {
	// initial detection length
	detectionLength := 2

	for {
		// check length
		if detectionLength > 5 {
			return nil, ErrDetectionOverflow
		}

		// try read detection bytes
		header, err := d.reader.Peek(detectionLength)
		if err == io.EOF && len(header) != 0 {
			// an EOF with some data is unexpected
			return nil, io.ErrUnexpectedEOF
		} else if err != nil {
			return nil, err
		}

		// detect packet
		packetLength, packetType := DetectPacket(header)

		// on zero packet length:
		// increment detection length and try again
		if packetLength <= 0 {
			detectionLength++
			continue
		}

		// check read limit
		if d.Limit > 0 && int64(packetLength) > d.Limit {
			return nil, ErrReadLimitExceeded
		}

		// create packet
		pkt, err := packetType.New()
		if err != nil {
			return nil, err
		}

		// reset and eventually grow buffer
		d.buffer.Reset()
		d.buffer.Grow(packetLength)
		buf := d.buffer.Bytes()[0:packetLength]

		// read whole packet (will not return EOF)
		_, err = io.ReadFull(d.reader, buf)
		if err != nil {
			return nil, err
		}

		// decode buffer
		_, err = pkt.Decode(buf)
		if err != nil {
			return nil, err
		}

		return pkt, nil
	}
}

// A Stream combines an Encoder and Decoder
type Stream struct {
	*Decoder
	*Encoder
}

// NewStream creates a new Stream.
func NewStream(reader io.Reader, writer io.Writer) *Stream {
	return &Stream{
		Decoder: NewDecoder(reader),
		Encoder: NewEncoder(writer),
	}
}
