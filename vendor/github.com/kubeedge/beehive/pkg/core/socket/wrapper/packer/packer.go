package packer

import (
	"bytes"
	"encoding/binary"
	"io"

	"k8s.io/klog/v2"
)

const (
	magicSize    = 4
	versionSize  = 2
	reservedSize = 2

	// MessageLenOffest message len offest
	MessageLenOffest = magicSize + versionSize + reservedSize
	// MessageOffset message offset
	MessageOffset = MessageLenOffest + 4
	// HeaderLen header len
	HeaderLen = MessageOffset
)

var (
	headerTags = [HeaderLen]byte{'b', 'e', 'e', 'h', 'v', '1', 'r', 'v', 0, 0, 0, 0}
)

// Packer packer
type Packer struct {
	Magic    [magicSize]byte
	Version  [versionSize]byte
	Reserved [reservedSize]byte
	Length   int32
	Message  []byte
}

// NewPacker new packer
func NewPacker() *Packer {
	return &Packer{
		Magic:    [magicSize]byte{'b', 'e', 'e', 'h'},
		Version:  [versionSize]byte{'v', '1'},
		Reserved: [reservedSize]byte{'r', 'v'},
	}
}

// Validate validate
func (p *Packer) Validate(data []byte) bool {
	if len(data) <= HeaderLen {
		return false
	}
	if !bytes.Equal(data[:magicSize], p.Magic[:magicSize]) {
		return false
	}
	if !bytes.Equal(data[magicSize:magicSize+versionSize], p.Version[:versionSize]) {
		return false
	}
	return true
}

// Write write
func (p *Packer) Write(writer io.Writer) error {
	// fill message len
	headerTags[MessageLenOffest] = byte(uint32(p.Length) >> 24)
	headerTags[MessageLenOffest+1] = byte(uint32(p.Length) >> 16)
	headerTags[MessageLenOffest+2] = byte(uint32(p.Length) >> 8)
	headerTags[MessageLenOffest+3] = byte(uint32(p.Length))
	err := binary.Write(writer, binary.BigEndian, &headerTags)
	if err != nil {
		return err
	}
	err = binary.Write(writer, binary.BigEndian, &p.Message)
	if err != nil {
		return err
	}
	return nil
}

// Read read
func (p *Packer) Read(reader io.Reader) error {
	err := binary.Read(reader, binary.BigEndian, &p.Magic)
	if err != nil {
		return err
	}
	err = binary.Read(reader, binary.BigEndian, &p.Version)
	if err != nil {
		return err
	}
	err = binary.Read(reader, binary.BigEndian, &p.Reserved)
	if err != nil {
		return err
	}
	err = binary.Read(reader, binary.BigEndian, &p.Length)
	if err != nil {
		return err
	}
	err = binary.Read(reader, binary.BigEndian, &p.Message)
	if err != nil {
		return err
	}
	return err
}

// GetMessageLen get message len
func (p *Packer) GetMessageLen(data []byte) int32 {
	length := int32(0)
	if len(data) < MessageOffset {
		return length
	}
	err := binary.Read(bytes.NewReader(data[MessageLenOffest:MessageOffset]), binary.BigEndian, &length)
	if err != nil {
		klog.Errorf("binary Read err %+v", err)
	}
	return length
}
