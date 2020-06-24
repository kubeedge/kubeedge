/*
Copyright 2019 The KubeEdge Authors.

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

package packer

// do packing
func (h *PackageHeader) Pack(buffer *[]byte) {
	*buffer = append(*buffer,
		byte(h.Version>>24),
		byte(h.Version>>16),
		byte(h.Version>>8),
		byte(h.Version),
		byte(h.PackageType),
		byte(h.Flags),
		byte(h.PayloadLen>>24),
		byte(h.PayloadLen>>16),
		byte(h.PayloadLen>>8),
		byte(h.PayloadLen))
}

// do unpacking
func (h *PackageHeader) Unpack(header []byte) {
	h.Version = uint32(header[VersionOffset])<<24 |
		uint32(header[VersionOffset+1])<<16 |
		uint32(header[VersionOffset+2])<<8 |
		uint32(header[VersionOffset+3])
	h.PackageType = PackageType(header[PackageTypeOffset])
	h.Flags = uint8(header[FlagsOffset])
	h.PayloadLen = uint32(header[PayloadLenOffset])<<24 |
		uint32(header[PayloadLenOffset+1])<<16 |
		uint32(header[PayloadLenOffset+2])<<8 |
		uint32(header[PayloadLenOffset+3])
}
