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

const (
	// pakcage version
	MajorVersion = 1
	MinorVersion = 1
	FixVersion   = 1
)

// make up version
func makeUpVersion(major, minor, fix uint8) uint32 {
	return uint32(major)<<24 | uint32(minor)<<16 | uint32(fix)<<8
}

// break down version
func breadDownVersion(version uint32) (uint8, uint8, uint8) {
	return uint8(version >> 24), uint8(version >> 16), uint8(version >> 8)
}
