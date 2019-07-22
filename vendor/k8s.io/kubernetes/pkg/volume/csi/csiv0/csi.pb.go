/*
Copyright 2018 The Kubernetes Authors.

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

// For backwards compatibility with CSI 0.x we carry a copy of the
// CSI 0.3 client.

package csiv0

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import wrappers "github.com/golang/protobuf/ptypes/wrappers"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type PluginCapability_Service_Type int32

const (
	PluginCapability_Service_UNKNOWN PluginCapability_Service_Type = 0
	// CONTROLLER_SERVICE indicates that the Plugin provides RPCs for
	// the ControllerService. Plugins SHOULD provide this capability.
	// In rare cases certain plugins may wish to omit the
	// ControllerService entirely from their implementation, but such
	// SHOULD NOT be the common case.
	// The presence of this capability determines whether the CO will
	// attempt to invoke the REQUIRED ControllerService RPCs, as well
	// as specific RPCs as indicated by ControllerGetCapabilities.
	PluginCapability_Service_CONTROLLER_SERVICE PluginCapability_Service_Type = 1
	// ACCESSIBILITY_CONSTRAINTS indicates that the volumes for this
	// plugin may not be equally accessible by all nodes in the
	// cluster. The CO MUST use the topology information returned by
	// CreateVolumeRequest along with the topology information
	// returned by NodeGetInfo to ensure that a given volume is
	// accessible from a given node when scheduling workloads.
	PluginCapability_Service_ACCESSIBILITY_CONSTRAINTS PluginCapability_Service_Type = 2
)

var PluginCapability_Service_Type_name = map[int32]string{
	0: "UNKNOWN",
	1: "CONTROLLER_SERVICE",
	2: "ACCESSIBILITY_CONSTRAINTS",
}
var PluginCapability_Service_Type_value = map[string]int32{
	"UNKNOWN":                   0,
	"CONTROLLER_SERVICE":        1,
	"ACCESSIBILITY_CONSTRAINTS": 2,
}

func (x PluginCapability_Service_Type) String() string {
	return proto.EnumName(PluginCapability_Service_Type_name, int32(x))
}
func (PluginCapability_Service_Type) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{4, 0, 0}
}

type VolumeCapability_AccessMode_Mode int32

const (
	VolumeCapability_AccessMode_UNKNOWN VolumeCapability_AccessMode_Mode = 0
	// Can only be published once as read/write on a single node, at
	// any given time.
	VolumeCapability_AccessMode_SINGLE_NODE_WRITER VolumeCapability_AccessMode_Mode = 1
	// Can only be published once as readonly on a single node, at
	// any given time.
	VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY VolumeCapability_AccessMode_Mode = 2
	// Can be published as readonly at multiple nodes simultaneously.
	VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY VolumeCapability_AccessMode_Mode = 3
	// Can be published at multiple nodes simultaneously. Only one of
	// the node can be used as read/write. The rest will be readonly.
	VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER VolumeCapability_AccessMode_Mode = 4
	// Can be published as read/write at multiple nodes
	// simultaneously.
	VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER VolumeCapability_AccessMode_Mode = 5
)

var VolumeCapability_AccessMode_Mode_name = map[int32]string{
	0: "UNKNOWN",
	1: "SINGLE_NODE_WRITER",
	2: "SINGLE_NODE_READER_ONLY",
	3: "MULTI_NODE_READER_ONLY",
	4: "MULTI_NODE_SINGLE_WRITER",
	5: "MULTI_NODE_MULTI_WRITER",
}
var VolumeCapability_AccessMode_Mode_value = map[string]int32{
	"UNKNOWN":                  0,
	"SINGLE_NODE_WRITER":       1,
	"SINGLE_NODE_READER_ONLY":  2,
	"MULTI_NODE_READER_ONLY":   3,
	"MULTI_NODE_SINGLE_WRITER": 4,
	"MULTI_NODE_MULTI_WRITER":  5,
}

func (x VolumeCapability_AccessMode_Mode) String() string {
	return proto.EnumName(VolumeCapability_AccessMode_Mode_name, int32(x))
}
func (VolumeCapability_AccessMode_Mode) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{10, 2, 0}
}

type ControllerServiceCapability_RPC_Type int32

const (
	ControllerServiceCapability_RPC_UNKNOWN                  ControllerServiceCapability_RPC_Type = 0
	ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME     ControllerServiceCapability_RPC_Type = 1
	ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME ControllerServiceCapability_RPC_Type = 2
	ControllerServiceCapability_RPC_LIST_VOLUMES             ControllerServiceCapability_RPC_Type = 3
	ControllerServiceCapability_RPC_GET_CAPACITY             ControllerServiceCapability_RPC_Type = 4
	// Currently the only way to consume a snapshot is to create
	// a volume from it. Therefore plugins supporting
	// CREATE_DELETE_SNAPSHOT MUST support creating volume from
	// snapshot.
	ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT ControllerServiceCapability_RPC_Type = 5
	// LIST_SNAPSHOTS is NOT REQUIRED. For plugins that need to upload
	// a snapshot after it is being cut, LIST_SNAPSHOTS COULD be used
	// with the snapshot_id as the filter to query whether the
	// uploading process is complete or not.
	ControllerServiceCapability_RPC_LIST_SNAPSHOTS ControllerServiceCapability_RPC_Type = 6
)

var ControllerServiceCapability_RPC_Type_name = map[int32]string{
	0: "UNKNOWN",
	1: "CREATE_DELETE_VOLUME",
	2: "PUBLISH_UNPUBLISH_VOLUME",
	3: "LIST_VOLUMES",
	4: "GET_CAPACITY",
	5: "CREATE_DELETE_SNAPSHOT",
	6: "LIST_SNAPSHOTS",
}
var ControllerServiceCapability_RPC_Type_value = map[string]int32{
	"UNKNOWN":                  0,
	"CREATE_DELETE_VOLUME":     1,
	"PUBLISH_UNPUBLISH_VOLUME": 2,
	"LIST_VOLUMES":             3,
	"GET_CAPACITY":             4,
	"CREATE_DELETE_SNAPSHOT":   5,
	"LIST_SNAPSHOTS":           6,
}

func (x ControllerServiceCapability_RPC_Type) String() string {
	return proto.EnumName(ControllerServiceCapability_RPC_Type_name, int32(x))
}
func (ControllerServiceCapability_RPC_Type) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{29, 0, 0}
}

type SnapshotStatus_Type int32

const (
	SnapshotStatus_UNKNOWN SnapshotStatus_Type = 0
	// A snapshot is ready for use.
	SnapshotStatus_READY SnapshotStatus_Type = 1
	// A snapshot is cut and is now being uploaded.
	// Some cloud providers and storage systems uploads the snapshot
	// to the cloud after the snapshot is cut. During this phase,
	// `thaw` can be done so the application can be running again if
	// `freeze` was done before taking the snapshot.
	SnapshotStatus_UPLOADING SnapshotStatus_Type = 2
	// An error occurred during the snapshot uploading process.
	// This error status is specific for uploading because
	// `CreateSnaphot` is a blocking call before the snapshot is
	// cut and therefore it SHOULD NOT come back with an error
	// status when an error occurs. Instead a gRPC error code SHALL
	// be returned by `CreateSnapshot` when an error occurs before
	// a snapshot is cut.
	SnapshotStatus_ERROR_UPLOADING SnapshotStatus_Type = 3
)

var SnapshotStatus_Type_name = map[int32]string{
	0: "UNKNOWN",
	1: "READY",
	2: "UPLOADING",
	3: "ERROR_UPLOADING",
}
var SnapshotStatus_Type_value = map[string]int32{
	"UNKNOWN":         0,
	"READY":           1,
	"UPLOADING":       2,
	"ERROR_UPLOADING": 3,
}

func (x SnapshotStatus_Type) String() string {
	return proto.EnumName(SnapshotStatus_Type_name, int32(x))
}
func (SnapshotStatus_Type) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{33, 0}
}

type NodeServiceCapability_RPC_Type int32

const (
	NodeServiceCapability_RPC_UNKNOWN              NodeServiceCapability_RPC_Type = 0
	NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME NodeServiceCapability_RPC_Type = 1
)

var NodeServiceCapability_RPC_Type_name = map[int32]string{
	0: "UNKNOWN",
	1: "STAGE_UNSTAGE_VOLUME",
}
var NodeServiceCapability_RPC_Type_value = map[string]int32{
	"UNKNOWN":              0,
	"STAGE_UNSTAGE_VOLUME": 1,
}

func (x NodeServiceCapability_RPC_Type) String() string {
	return proto.EnumName(NodeServiceCapability_RPC_Type_name, int32(x))
}
func (NodeServiceCapability_RPC_Type) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{50, 0, 0}
}

type GetPluginInfoRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetPluginInfoRequest) Reset()         { *m = GetPluginInfoRequest{} }
func (m *GetPluginInfoRequest) String() string { return proto.CompactTextString(m) }
func (*GetPluginInfoRequest) ProtoMessage()    {}
func (*GetPluginInfoRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{0}
}
func (m *GetPluginInfoRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetPluginInfoRequest.Unmarshal(m, b)
}
func (m *GetPluginInfoRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetPluginInfoRequest.Marshal(b, m, deterministic)
}
func (dst *GetPluginInfoRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetPluginInfoRequest.Merge(dst, src)
}
func (m *GetPluginInfoRequest) XXX_Size() int {
	return xxx_messageInfo_GetPluginInfoRequest.Size(m)
}
func (m *GetPluginInfoRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetPluginInfoRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetPluginInfoRequest proto.InternalMessageInfo

type GetPluginInfoResponse struct {
	// The name MUST follow reverse domain name notation format
	// (https://en.wikipedia.org/wiki/Reverse_domain_name_notation).
	// It SHOULD include the plugin's host company name and the plugin
	// name, to minimize the possibility of collisions. It MUST be 63
	// characters or less, beginning and ending with an alphanumeric
	// character ([a-z0-9A-Z]) with dashes (-), underscores (_),
	// dots (.), and alphanumerics between. This field is REQUIRED.
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// This field is REQUIRED. Value of this field is opaque to the CO.
	VendorVersion string `protobuf:"bytes,2,opt,name=vendor_version,json=vendorVersion" json:"vendor_version,omitempty"`
	// This field is OPTIONAL. Values are opaque to the CO.
	Manifest             map[string]string `protobuf:"bytes,3,rep,name=manifest" json:"manifest,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *GetPluginInfoResponse) Reset()         { *m = GetPluginInfoResponse{} }
func (m *GetPluginInfoResponse) String() string { return proto.CompactTextString(m) }
func (*GetPluginInfoResponse) ProtoMessage()    {}
func (*GetPluginInfoResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{1}
}
func (m *GetPluginInfoResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetPluginInfoResponse.Unmarshal(m, b)
}
func (m *GetPluginInfoResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetPluginInfoResponse.Marshal(b, m, deterministic)
}
func (dst *GetPluginInfoResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetPluginInfoResponse.Merge(dst, src)
}
func (m *GetPluginInfoResponse) XXX_Size() int {
	return xxx_messageInfo_GetPluginInfoResponse.Size(m)
}
func (m *GetPluginInfoResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GetPluginInfoResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GetPluginInfoResponse proto.InternalMessageInfo

func (m *GetPluginInfoResponse) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *GetPluginInfoResponse) GetVendorVersion() string {
	if m != nil {
		return m.VendorVersion
	}
	return ""
}

func (m *GetPluginInfoResponse) GetManifest() map[string]string {
	if m != nil {
		return m.Manifest
	}
	return nil
}

type GetPluginCapabilitiesRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetPluginCapabilitiesRequest) Reset()         { *m = GetPluginCapabilitiesRequest{} }
func (m *GetPluginCapabilitiesRequest) String() string { return proto.CompactTextString(m) }
func (*GetPluginCapabilitiesRequest) ProtoMessage()    {}
func (*GetPluginCapabilitiesRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{2}
}
func (m *GetPluginCapabilitiesRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetPluginCapabilitiesRequest.Unmarshal(m, b)
}
func (m *GetPluginCapabilitiesRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetPluginCapabilitiesRequest.Marshal(b, m, deterministic)
}
func (dst *GetPluginCapabilitiesRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetPluginCapabilitiesRequest.Merge(dst, src)
}
func (m *GetPluginCapabilitiesRequest) XXX_Size() int {
	return xxx_messageInfo_GetPluginCapabilitiesRequest.Size(m)
}
func (m *GetPluginCapabilitiesRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetPluginCapabilitiesRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetPluginCapabilitiesRequest proto.InternalMessageInfo

type GetPluginCapabilitiesResponse struct {
	// All the capabilities that the controller service supports. This
	// field is OPTIONAL.
	Capabilities         []*PluginCapability `protobuf:"bytes,2,rep,name=capabilities" json:"capabilities,omitempty"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *GetPluginCapabilitiesResponse) Reset()         { *m = GetPluginCapabilitiesResponse{} }
func (m *GetPluginCapabilitiesResponse) String() string { return proto.CompactTextString(m) }
func (*GetPluginCapabilitiesResponse) ProtoMessage()    {}
func (*GetPluginCapabilitiesResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{3}
}
func (m *GetPluginCapabilitiesResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetPluginCapabilitiesResponse.Unmarshal(m, b)
}
func (m *GetPluginCapabilitiesResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetPluginCapabilitiesResponse.Marshal(b, m, deterministic)
}
func (dst *GetPluginCapabilitiesResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetPluginCapabilitiesResponse.Merge(dst, src)
}
func (m *GetPluginCapabilitiesResponse) XXX_Size() int {
	return xxx_messageInfo_GetPluginCapabilitiesResponse.Size(m)
}
func (m *GetPluginCapabilitiesResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GetPluginCapabilitiesResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GetPluginCapabilitiesResponse proto.InternalMessageInfo

func (m *GetPluginCapabilitiesResponse) GetCapabilities() []*PluginCapability {
	if m != nil {
		return m.Capabilities
	}
	return nil
}

// Specifies a capability of the plugin.
type PluginCapability struct {
	// Types that are valid to be assigned to Type:
	//	*PluginCapability_Service_
	Type                 isPluginCapability_Type `protobuf_oneof:"type"`
	XXX_NoUnkeyedLiteral struct{}                `json:"-"`
	XXX_unrecognized     []byte                  `json:"-"`
	XXX_sizecache        int32                   `json:"-"`
}

func (m *PluginCapability) Reset()         { *m = PluginCapability{} }
func (m *PluginCapability) String() string { return proto.CompactTextString(m) }
func (*PluginCapability) ProtoMessage()    {}
func (*PluginCapability) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{4}
}
func (m *PluginCapability) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PluginCapability.Unmarshal(m, b)
}
func (m *PluginCapability) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PluginCapability.Marshal(b, m, deterministic)
}
func (dst *PluginCapability) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PluginCapability.Merge(dst, src)
}
func (m *PluginCapability) XXX_Size() int {
	return xxx_messageInfo_PluginCapability.Size(m)
}
func (m *PluginCapability) XXX_DiscardUnknown() {
	xxx_messageInfo_PluginCapability.DiscardUnknown(m)
}

var xxx_messageInfo_PluginCapability proto.InternalMessageInfo

type isPluginCapability_Type interface {
	isPluginCapability_Type()
}

type PluginCapability_Service_ struct {
	Service *PluginCapability_Service `protobuf:"bytes,1,opt,name=service,oneof"`
}

func (*PluginCapability_Service_) isPluginCapability_Type() {}

func (m *PluginCapability) GetType() isPluginCapability_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (m *PluginCapability) GetService() *PluginCapability_Service {
	if x, ok := m.GetType().(*PluginCapability_Service_); ok {
		return x.Service
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*PluginCapability) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _PluginCapability_OneofMarshaler, _PluginCapability_OneofUnmarshaler, _PluginCapability_OneofSizer, []interface{}{
		(*PluginCapability_Service_)(nil),
	}
}

func _PluginCapability_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*PluginCapability)
	// type
	switch x := m.Type.(type) {
	case *PluginCapability_Service_:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Service); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("PluginCapability.Type has unexpected type %T", x)
	}
	return nil
}

func _PluginCapability_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*PluginCapability)
	switch tag {
	case 1: // type.service
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(PluginCapability_Service)
		err := b.DecodeMessage(msg)
		m.Type = &PluginCapability_Service_{msg}
		return true, err
	default:
		return false, nil
	}
}

func _PluginCapability_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*PluginCapability)
	// type
	switch x := m.Type.(type) {
	case *PluginCapability_Service_:
		s := proto.Size(x.Service)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type PluginCapability_Service struct {
	Type                 PluginCapability_Service_Type `protobuf:"varint,1,opt,name=type,enum=csi.v0.PluginCapability_Service_Type" json:"type,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                      `json:"-"`
	XXX_unrecognized     []byte                        `json:"-"`
	XXX_sizecache        int32                         `json:"-"`
}

func (m *PluginCapability_Service) Reset()         { *m = PluginCapability_Service{} }
func (m *PluginCapability_Service) String() string { return proto.CompactTextString(m) }
func (*PluginCapability_Service) ProtoMessage()    {}
func (*PluginCapability_Service) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{4, 0}
}
func (m *PluginCapability_Service) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PluginCapability_Service.Unmarshal(m, b)
}
func (m *PluginCapability_Service) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PluginCapability_Service.Marshal(b, m, deterministic)
}
func (dst *PluginCapability_Service) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PluginCapability_Service.Merge(dst, src)
}
func (m *PluginCapability_Service) XXX_Size() int {
	return xxx_messageInfo_PluginCapability_Service.Size(m)
}
func (m *PluginCapability_Service) XXX_DiscardUnknown() {
	xxx_messageInfo_PluginCapability_Service.DiscardUnknown(m)
}

var xxx_messageInfo_PluginCapability_Service proto.InternalMessageInfo

func (m *PluginCapability_Service) GetType() PluginCapability_Service_Type {
	if m != nil {
		return m.Type
	}
	return PluginCapability_Service_UNKNOWN
}

type ProbeRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ProbeRequest) Reset()         { *m = ProbeRequest{} }
func (m *ProbeRequest) String() string { return proto.CompactTextString(m) }
func (*ProbeRequest) ProtoMessage()    {}
func (*ProbeRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{5}
}
func (m *ProbeRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ProbeRequest.Unmarshal(m, b)
}
func (m *ProbeRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ProbeRequest.Marshal(b, m, deterministic)
}
func (dst *ProbeRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ProbeRequest.Merge(dst, src)
}
func (m *ProbeRequest) XXX_Size() int {
	return xxx_messageInfo_ProbeRequest.Size(m)
}
func (m *ProbeRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ProbeRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ProbeRequest proto.InternalMessageInfo

type ProbeResponse struct {
	// Readiness allows a plugin to report its initialization status back
	// to the CO. Initialization for some plugins MAY be time consuming
	// and it is important for a CO to distinguish between the following
	// cases:
	//
	// 1) The plugin is in an unhealthy state and MAY need restarting. In
	//    this case a gRPC error code SHALL be returned.
	// 2) The plugin is still initializing, but is otherwise perfectly
	//    healthy. In this case a successful response SHALL be returned
	//    with a readiness value of `false`. Calls to the plugin's
	//    Controller and/or Node services MAY fail due to an incomplete
	//    initialization state.
	// 3) The plugin has finished initializing and is ready to service
	//    calls to its Controller and/or Node services. A successful
	//    response is returned with a readiness value of `true`.
	//
	// This field is OPTIONAL. If not present, the caller SHALL assume
	// that the plugin is in a ready state and is accepting calls to its
	// Controller and/or Node services (according to the plugin's reported
	// capabilities).
	Ready                *wrappers.BoolValue `protobuf:"bytes,1,opt,name=ready" json:"ready,omitempty"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *ProbeResponse) Reset()         { *m = ProbeResponse{} }
func (m *ProbeResponse) String() string { return proto.CompactTextString(m) }
func (*ProbeResponse) ProtoMessage()    {}
func (*ProbeResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{6}
}
func (m *ProbeResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ProbeResponse.Unmarshal(m, b)
}
func (m *ProbeResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ProbeResponse.Marshal(b, m, deterministic)
}
func (dst *ProbeResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ProbeResponse.Merge(dst, src)
}
func (m *ProbeResponse) XXX_Size() int {
	return xxx_messageInfo_ProbeResponse.Size(m)
}
func (m *ProbeResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ProbeResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ProbeResponse proto.InternalMessageInfo

func (m *ProbeResponse) GetReady() *wrappers.BoolValue {
	if m != nil {
		return m.Ready
	}
	return nil
}

type CreateVolumeRequest struct {
	// The suggested name for the storage space. This field is REQUIRED.
	// It serves two purposes:
	// 1) Idempotency - This name is generated by the CO to achieve
	//    idempotency. If `CreateVolume` fails, the volume may or may not
	//    be provisioned. In this case, the CO may call `CreateVolume`
	//    again, with the same name, to ensure the volume exists. The
	//    Plugin should ensure that multiple `CreateVolume` calls for the
	//    same name do not result in more than one piece of storage
	//    provisioned corresponding to that name. If a Plugin is unable to
	//    enforce idempotency, the CO's error recovery logic could result
	//    in multiple (unused) volumes being provisioned.
	// 2) Suggested name - Some storage systems allow callers to specify
	//    an identifier by which to refer to the newly provisioned
	//    storage. If a storage system supports this, it can optionally
	//    use this name as the identifier for the new volume.
	Name          string         `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	CapacityRange *CapacityRange `protobuf:"bytes,2,opt,name=capacity_range,json=capacityRange" json:"capacity_range,omitempty"`
	// The capabilities that the provisioned volume MUST have: the Plugin
	// MUST provision a volume that could satisfy ALL of the
	// capabilities specified in this list. The Plugin MUST assume that
	// the CO MAY use the  provisioned volume later with ANY of the
	// capabilities specified in this list. This also enables the CO to do
	// early validation: if ANY of the specified volume capabilities are
	// not supported by the Plugin, the call SHALL fail. This field is
	// REQUIRED.
	VolumeCapabilities []*VolumeCapability `protobuf:"bytes,3,rep,name=volume_capabilities,json=volumeCapabilities" json:"volume_capabilities,omitempty"`
	// Plugin specific parameters passed in as opaque key-value pairs.
	// This field is OPTIONAL. The Plugin is responsible for parsing and
	// validating these parameters. COs will treat these as opaque.
	Parameters map[string]string `protobuf:"bytes,4,rep,name=parameters" json:"parameters,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// Secrets required by plugin to complete volume creation request.
	// This field is OPTIONAL. Refer to the `Secrets Requirements`
	// section on how to use this field.
	ControllerCreateSecrets map[string]string `protobuf:"bytes,5,rep,name=controller_create_secrets,json=controllerCreateSecrets" json:"controller_create_secrets,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// If specified, the new volume will be pre-populated with data from
	// this source. This field is OPTIONAL.
	VolumeContentSource *VolumeContentSource `protobuf:"bytes,6,opt,name=volume_content_source,json=volumeContentSource" json:"volume_content_source,omitempty"`
	// Specifies where (regions, zones, racks, etc.) the provisioned
	// volume MUST be accessible from.
	// An SP SHALL advertise the requirements for topological
	// accessibility information in documentation. COs SHALL only specify
	// topological accessibility information supported by the SP.
	// This field is OPTIONAL.
	// This field SHALL NOT be specified unless the SP has the
	// ACCESSIBILITY_CONSTRAINTS plugin capability.
	// If this field is not specified and the SP has the
	// ACCESSIBILITY_CONSTRAINTS plugin capability, the SP MAY choose
	// where the provisioned volume is accessible from.
	AccessibilityRequirements *TopologyRequirement `protobuf:"bytes,7,opt,name=accessibility_requirements,json=accessibilityRequirements" json:"accessibility_requirements,omitempty"`
	XXX_NoUnkeyedLiteral      struct{}             `json:"-"`
	XXX_unrecognized          []byte               `json:"-"`
	XXX_sizecache             int32                `json:"-"`
}

func (m *CreateVolumeRequest) Reset()         { *m = CreateVolumeRequest{} }
func (m *CreateVolumeRequest) String() string { return proto.CompactTextString(m) }
func (*CreateVolumeRequest) ProtoMessage()    {}
func (*CreateVolumeRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{7}
}
func (m *CreateVolumeRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CreateVolumeRequest.Unmarshal(m, b)
}
func (m *CreateVolumeRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CreateVolumeRequest.Marshal(b, m, deterministic)
}
func (dst *CreateVolumeRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CreateVolumeRequest.Merge(dst, src)
}
func (m *CreateVolumeRequest) XXX_Size() int {
	return xxx_messageInfo_CreateVolumeRequest.Size(m)
}
func (m *CreateVolumeRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_CreateVolumeRequest.DiscardUnknown(m)
}

var xxx_messageInfo_CreateVolumeRequest proto.InternalMessageInfo

func (m *CreateVolumeRequest) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *CreateVolumeRequest) GetCapacityRange() *CapacityRange {
	if m != nil {
		return m.CapacityRange
	}
	return nil
}

func (m *CreateVolumeRequest) GetVolumeCapabilities() []*VolumeCapability {
	if m != nil {
		return m.VolumeCapabilities
	}
	return nil
}

func (m *CreateVolumeRequest) GetParameters() map[string]string {
	if m != nil {
		return m.Parameters
	}
	return nil
}

func (m *CreateVolumeRequest) GetControllerCreateSecrets() map[string]string {
	if m != nil {
		return m.ControllerCreateSecrets
	}
	return nil
}

func (m *CreateVolumeRequest) GetVolumeContentSource() *VolumeContentSource {
	if m != nil {
		return m.VolumeContentSource
	}
	return nil
}

func (m *CreateVolumeRequest) GetAccessibilityRequirements() *TopologyRequirement {
	if m != nil {
		return m.AccessibilityRequirements
	}
	return nil
}

// Specifies what source the volume will be created from. One of the
// type fields MUST be specified.
type VolumeContentSource struct {
	// Types that are valid to be assigned to Type:
	//	*VolumeContentSource_Snapshot
	Type                 isVolumeContentSource_Type `protobuf_oneof:"type"`
	XXX_NoUnkeyedLiteral struct{}                   `json:"-"`
	XXX_unrecognized     []byte                     `json:"-"`
	XXX_sizecache        int32                      `json:"-"`
}

func (m *VolumeContentSource) Reset()         { *m = VolumeContentSource{} }
func (m *VolumeContentSource) String() string { return proto.CompactTextString(m) }
func (*VolumeContentSource) ProtoMessage()    {}
func (*VolumeContentSource) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{8}
}
func (m *VolumeContentSource) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_VolumeContentSource.Unmarshal(m, b)
}
func (m *VolumeContentSource) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_VolumeContentSource.Marshal(b, m, deterministic)
}
func (dst *VolumeContentSource) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VolumeContentSource.Merge(dst, src)
}
func (m *VolumeContentSource) XXX_Size() int {
	return xxx_messageInfo_VolumeContentSource.Size(m)
}
func (m *VolumeContentSource) XXX_DiscardUnknown() {
	xxx_messageInfo_VolumeContentSource.DiscardUnknown(m)
}

var xxx_messageInfo_VolumeContentSource proto.InternalMessageInfo

type isVolumeContentSource_Type interface {
	isVolumeContentSource_Type()
}

type VolumeContentSource_Snapshot struct {
	Snapshot *VolumeContentSource_SnapshotSource `protobuf:"bytes,1,opt,name=snapshot,oneof"`
}

func (*VolumeContentSource_Snapshot) isVolumeContentSource_Type() {}

func (m *VolumeContentSource) GetType() isVolumeContentSource_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (m *VolumeContentSource) GetSnapshot() *VolumeContentSource_SnapshotSource {
	if x, ok := m.GetType().(*VolumeContentSource_Snapshot); ok {
		return x.Snapshot
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*VolumeContentSource) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _VolumeContentSource_OneofMarshaler, _VolumeContentSource_OneofUnmarshaler, _VolumeContentSource_OneofSizer, []interface{}{
		(*VolumeContentSource_Snapshot)(nil),
	}
}

func _VolumeContentSource_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*VolumeContentSource)
	// type
	switch x := m.Type.(type) {
	case *VolumeContentSource_Snapshot:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Snapshot); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("VolumeContentSource.Type has unexpected type %T", x)
	}
	return nil
}

func _VolumeContentSource_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*VolumeContentSource)
	switch tag {
	case 1: // type.snapshot
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(VolumeContentSource_SnapshotSource)
		err := b.DecodeMessage(msg)
		m.Type = &VolumeContentSource_Snapshot{msg}
		return true, err
	default:
		return false, nil
	}
}

func _VolumeContentSource_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*VolumeContentSource)
	// type
	switch x := m.Type.(type) {
	case *VolumeContentSource_Snapshot:
		s := proto.Size(x.Snapshot)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type VolumeContentSource_SnapshotSource struct {
	// Contains identity information for the existing source snapshot.
	// This field is REQUIRED. Plugin is REQUIRED to support creating
	// volume from snapshot if it supports the capability
	// CREATE_DELETE_SNAPSHOT.
	Id                   string   `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *VolumeContentSource_SnapshotSource) Reset()         { *m = VolumeContentSource_SnapshotSource{} }
func (m *VolumeContentSource_SnapshotSource) String() string { return proto.CompactTextString(m) }
func (*VolumeContentSource_SnapshotSource) ProtoMessage()    {}
func (*VolumeContentSource_SnapshotSource) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{8, 0}
}
func (m *VolumeContentSource_SnapshotSource) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_VolumeContentSource_SnapshotSource.Unmarshal(m, b)
}
func (m *VolumeContentSource_SnapshotSource) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_VolumeContentSource_SnapshotSource.Marshal(b, m, deterministic)
}
func (dst *VolumeContentSource_SnapshotSource) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VolumeContentSource_SnapshotSource.Merge(dst, src)
}
func (m *VolumeContentSource_SnapshotSource) XXX_Size() int {
	return xxx_messageInfo_VolumeContentSource_SnapshotSource.Size(m)
}
func (m *VolumeContentSource_SnapshotSource) XXX_DiscardUnknown() {
	xxx_messageInfo_VolumeContentSource_SnapshotSource.DiscardUnknown(m)
}

var xxx_messageInfo_VolumeContentSource_SnapshotSource proto.InternalMessageInfo

func (m *VolumeContentSource_SnapshotSource) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

type CreateVolumeResponse struct {
	// Contains all attributes of the newly created volume that are
	// relevant to the CO along with information required by the Plugin
	// to uniquely identify the volume. This field is REQUIRED.
	Volume               *Volume  `protobuf:"bytes,1,opt,name=volume" json:"volume,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CreateVolumeResponse) Reset()         { *m = CreateVolumeResponse{} }
func (m *CreateVolumeResponse) String() string { return proto.CompactTextString(m) }
func (*CreateVolumeResponse) ProtoMessage()    {}
func (*CreateVolumeResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{9}
}
func (m *CreateVolumeResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CreateVolumeResponse.Unmarshal(m, b)
}
func (m *CreateVolumeResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CreateVolumeResponse.Marshal(b, m, deterministic)
}
func (dst *CreateVolumeResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CreateVolumeResponse.Merge(dst, src)
}
func (m *CreateVolumeResponse) XXX_Size() int {
	return xxx_messageInfo_CreateVolumeResponse.Size(m)
}
func (m *CreateVolumeResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_CreateVolumeResponse.DiscardUnknown(m)
}

var xxx_messageInfo_CreateVolumeResponse proto.InternalMessageInfo

func (m *CreateVolumeResponse) GetVolume() *Volume {
	if m != nil {
		return m.Volume
	}
	return nil
}

// Specify a capability of a volume.
type VolumeCapability struct {
	// Specifies what API the volume will be accessed using. One of the
	// following fields MUST be specified.
	//
	// Types that are valid to be assigned to AccessType:
	//	*VolumeCapability_Block
	//	*VolumeCapability_Mount
	AccessType isVolumeCapability_AccessType `protobuf_oneof:"access_type"`
	// This is a REQUIRED field.
	AccessMode           *VolumeCapability_AccessMode `protobuf:"bytes,3,opt,name=access_mode,json=accessMode" json:"access_mode,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                     `json:"-"`
	XXX_unrecognized     []byte                       `json:"-"`
	XXX_sizecache        int32                        `json:"-"`
}

func (m *VolumeCapability) Reset()         { *m = VolumeCapability{} }
func (m *VolumeCapability) String() string { return proto.CompactTextString(m) }
func (*VolumeCapability) ProtoMessage()    {}
func (*VolumeCapability) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{10}
}
func (m *VolumeCapability) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_VolumeCapability.Unmarshal(m, b)
}
func (m *VolumeCapability) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_VolumeCapability.Marshal(b, m, deterministic)
}
func (dst *VolumeCapability) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VolumeCapability.Merge(dst, src)
}
func (m *VolumeCapability) XXX_Size() int {
	return xxx_messageInfo_VolumeCapability.Size(m)
}
func (m *VolumeCapability) XXX_DiscardUnknown() {
	xxx_messageInfo_VolumeCapability.DiscardUnknown(m)
}

var xxx_messageInfo_VolumeCapability proto.InternalMessageInfo

type isVolumeCapability_AccessType interface {
	isVolumeCapability_AccessType()
}

type VolumeCapability_Block struct {
	Block *VolumeCapability_BlockVolume `protobuf:"bytes,1,opt,name=block,oneof"`
}
type VolumeCapability_Mount struct {
	Mount *VolumeCapability_MountVolume `protobuf:"bytes,2,opt,name=mount,oneof"`
}

func (*VolumeCapability_Block) isVolumeCapability_AccessType() {}
func (*VolumeCapability_Mount) isVolumeCapability_AccessType() {}

func (m *VolumeCapability) GetAccessType() isVolumeCapability_AccessType {
	if m != nil {
		return m.AccessType
	}
	return nil
}

func (m *VolumeCapability) GetBlock() *VolumeCapability_BlockVolume {
	if x, ok := m.GetAccessType().(*VolumeCapability_Block); ok {
		return x.Block
	}
	return nil
}

func (m *VolumeCapability) GetMount() *VolumeCapability_MountVolume {
	if x, ok := m.GetAccessType().(*VolumeCapability_Mount); ok {
		return x.Mount
	}
	return nil
}

func (m *VolumeCapability) GetAccessMode() *VolumeCapability_AccessMode {
	if m != nil {
		return m.AccessMode
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*VolumeCapability) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _VolumeCapability_OneofMarshaler, _VolumeCapability_OneofUnmarshaler, _VolumeCapability_OneofSizer, []interface{}{
		(*VolumeCapability_Block)(nil),
		(*VolumeCapability_Mount)(nil),
	}
}

func _VolumeCapability_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*VolumeCapability)
	// access_type
	switch x := m.AccessType.(type) {
	case *VolumeCapability_Block:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Block); err != nil {
			return err
		}
	case *VolumeCapability_Mount:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Mount); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("VolumeCapability.AccessType has unexpected type %T", x)
	}
	return nil
}

func _VolumeCapability_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*VolumeCapability)
	switch tag {
	case 1: // access_type.block
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(VolumeCapability_BlockVolume)
		err := b.DecodeMessage(msg)
		m.AccessType = &VolumeCapability_Block{msg}
		return true, err
	case 2: // access_type.mount
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(VolumeCapability_MountVolume)
		err := b.DecodeMessage(msg)
		m.AccessType = &VolumeCapability_Mount{msg}
		return true, err
	default:
		return false, nil
	}
}

func _VolumeCapability_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*VolumeCapability)
	// access_type
	switch x := m.AccessType.(type) {
	case *VolumeCapability_Block:
		s := proto.Size(x.Block)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *VolumeCapability_Mount:
		s := proto.Size(x.Mount)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

// Indicate that the volume will be accessed via the block device API.
type VolumeCapability_BlockVolume struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *VolumeCapability_BlockVolume) Reset()         { *m = VolumeCapability_BlockVolume{} }
func (m *VolumeCapability_BlockVolume) String() string { return proto.CompactTextString(m) }
func (*VolumeCapability_BlockVolume) ProtoMessage()    {}
func (*VolumeCapability_BlockVolume) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{10, 0}
}
func (m *VolumeCapability_BlockVolume) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_VolumeCapability_BlockVolume.Unmarshal(m, b)
}
func (m *VolumeCapability_BlockVolume) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_VolumeCapability_BlockVolume.Marshal(b, m, deterministic)
}
func (dst *VolumeCapability_BlockVolume) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VolumeCapability_BlockVolume.Merge(dst, src)
}
func (m *VolumeCapability_BlockVolume) XXX_Size() int {
	return xxx_messageInfo_VolumeCapability_BlockVolume.Size(m)
}
func (m *VolumeCapability_BlockVolume) XXX_DiscardUnknown() {
	xxx_messageInfo_VolumeCapability_BlockVolume.DiscardUnknown(m)
}

var xxx_messageInfo_VolumeCapability_BlockVolume proto.InternalMessageInfo

// Indicate that the volume will be accessed via the filesystem API.
type VolumeCapability_MountVolume struct {
	// The filesystem type. This field is OPTIONAL.
	// An empty string is equal to an unspecified field value.
	FsType string `protobuf:"bytes,1,opt,name=fs_type,json=fsType" json:"fs_type,omitempty"`
	// The mount options that can be used for the volume. This field is
	// OPTIONAL. `mount_flags` MAY contain sensitive information.
	// Therefore, the CO and the Plugin MUST NOT leak this information
	// to untrusted entities. The total size of this repeated field
	// SHALL NOT exceed 4 KiB.
	MountFlags           []string `protobuf:"bytes,2,rep,name=mount_flags,json=mountFlags" json:"mount_flags,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *VolumeCapability_MountVolume) Reset()         { *m = VolumeCapability_MountVolume{} }
func (m *VolumeCapability_MountVolume) String() string { return proto.CompactTextString(m) }
func (*VolumeCapability_MountVolume) ProtoMessage()    {}
func (*VolumeCapability_MountVolume) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{10, 1}
}
func (m *VolumeCapability_MountVolume) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_VolumeCapability_MountVolume.Unmarshal(m, b)
}
func (m *VolumeCapability_MountVolume) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_VolumeCapability_MountVolume.Marshal(b, m, deterministic)
}
func (dst *VolumeCapability_MountVolume) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VolumeCapability_MountVolume.Merge(dst, src)
}
func (m *VolumeCapability_MountVolume) XXX_Size() int {
	return xxx_messageInfo_VolumeCapability_MountVolume.Size(m)
}
func (m *VolumeCapability_MountVolume) XXX_DiscardUnknown() {
	xxx_messageInfo_VolumeCapability_MountVolume.DiscardUnknown(m)
}

var xxx_messageInfo_VolumeCapability_MountVolume proto.InternalMessageInfo

func (m *VolumeCapability_MountVolume) GetFsType() string {
	if m != nil {
		return m.FsType
	}
	return ""
}

func (m *VolumeCapability_MountVolume) GetMountFlags() []string {
	if m != nil {
		return m.MountFlags
	}
	return nil
}

// Specify how a volume can be accessed.
type VolumeCapability_AccessMode struct {
	// This field is REQUIRED.
	Mode                 VolumeCapability_AccessMode_Mode `protobuf:"varint,1,opt,name=mode,enum=csi.v0.VolumeCapability_AccessMode_Mode" json:"mode,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                         `json:"-"`
	XXX_unrecognized     []byte                           `json:"-"`
	XXX_sizecache        int32                            `json:"-"`
}

func (m *VolumeCapability_AccessMode) Reset()         { *m = VolumeCapability_AccessMode{} }
func (m *VolumeCapability_AccessMode) String() string { return proto.CompactTextString(m) }
func (*VolumeCapability_AccessMode) ProtoMessage()    {}
func (*VolumeCapability_AccessMode) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{10, 2}
}
func (m *VolumeCapability_AccessMode) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_VolumeCapability_AccessMode.Unmarshal(m, b)
}
func (m *VolumeCapability_AccessMode) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_VolumeCapability_AccessMode.Marshal(b, m, deterministic)
}
func (dst *VolumeCapability_AccessMode) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VolumeCapability_AccessMode.Merge(dst, src)
}
func (m *VolumeCapability_AccessMode) XXX_Size() int {
	return xxx_messageInfo_VolumeCapability_AccessMode.Size(m)
}
func (m *VolumeCapability_AccessMode) XXX_DiscardUnknown() {
	xxx_messageInfo_VolumeCapability_AccessMode.DiscardUnknown(m)
}

var xxx_messageInfo_VolumeCapability_AccessMode proto.InternalMessageInfo

func (m *VolumeCapability_AccessMode) GetMode() VolumeCapability_AccessMode_Mode {
	if m != nil {
		return m.Mode
	}
	return VolumeCapability_AccessMode_UNKNOWN
}

// The capacity of the storage space in bytes. To specify an exact size,
// `required_bytes` and `limit_bytes` SHALL be set to the same value. At
// least one of the these fields MUST be specified.
type CapacityRange struct {
	// Volume MUST be at least this big. This field is OPTIONAL.
	// A value of 0 is equal to an unspecified field value.
	// The value of this field MUST NOT be negative.
	RequiredBytes int64 `protobuf:"varint,1,opt,name=required_bytes,json=requiredBytes" json:"required_bytes,omitempty"`
	// Volume MUST not be bigger than this. This field is OPTIONAL.
	// A value of 0 is equal to an unspecified field value.
	// The value of this field MUST NOT be negative.
	LimitBytes           int64    `protobuf:"varint,2,opt,name=limit_bytes,json=limitBytes" json:"limit_bytes,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CapacityRange) Reset()         { *m = CapacityRange{} }
func (m *CapacityRange) String() string { return proto.CompactTextString(m) }
func (*CapacityRange) ProtoMessage()    {}
func (*CapacityRange) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{11}
}
func (m *CapacityRange) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CapacityRange.Unmarshal(m, b)
}
func (m *CapacityRange) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CapacityRange.Marshal(b, m, deterministic)
}
func (dst *CapacityRange) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CapacityRange.Merge(dst, src)
}
func (m *CapacityRange) XXX_Size() int {
	return xxx_messageInfo_CapacityRange.Size(m)
}
func (m *CapacityRange) XXX_DiscardUnknown() {
	xxx_messageInfo_CapacityRange.DiscardUnknown(m)
}

var xxx_messageInfo_CapacityRange proto.InternalMessageInfo

func (m *CapacityRange) GetRequiredBytes() int64 {
	if m != nil {
		return m.RequiredBytes
	}
	return 0
}

func (m *CapacityRange) GetLimitBytes() int64 {
	if m != nil {
		return m.LimitBytes
	}
	return 0
}

// The information about a provisioned volume.
type Volume struct {
	// The capacity of the volume in bytes. This field is OPTIONAL. If not
	// set (value of 0), it indicates that the capacity of the volume is
	// unknown (e.g., NFS share).
	// The value of this field MUST NOT be negative.
	CapacityBytes int64 `protobuf:"varint,1,opt,name=capacity_bytes,json=capacityBytes" json:"capacity_bytes,omitempty"`
	// Contains identity information for the created volume. This field is
	// REQUIRED. The identity information will be used by the CO in
	// subsequent calls to refer to the provisioned volume.
	Id string `protobuf:"bytes,2,opt,name=id" json:"id,omitempty"`
	// Attributes reflect static properties of a volume and MUST be passed
	// to volume validation and publishing calls.
	// Attributes SHALL be opaque to a CO. Attributes SHALL NOT be mutable
	// and SHALL be safe for the CO to cache. Attributes SHOULD NOT
	// contain sensitive information. Attributes MAY NOT uniquely identify
	// a volume. A volume uniquely identified by `id` SHALL always report
	// the same attributes. This field is OPTIONAL and when present MUST
	// be passed to volume validation and publishing calls.
	Attributes map[string]string `protobuf:"bytes,3,rep,name=attributes" json:"attributes,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// If specified, indicates that the volume is not empty and is
	// pre-populated with data from the specified source.
	// This field is OPTIONAL.
	ContentSource *VolumeContentSource `protobuf:"bytes,4,opt,name=content_source,json=contentSource" json:"content_source,omitempty"`
	// Specifies where (regions, zones, racks, etc.) the provisioned
	// volume is accessible from.
	// A plugin that returns this field MUST also set the
	// ACCESSIBILITY_CONSTRAINTS plugin capability.
	// An SP MAY specify multiple topologies to indicate the volume is
	// accessible from multiple locations.
	// COs MAY use this information along with the topology information
	// returned by NodeGetInfo to ensure that a given volume is accessible
	// from a given node when scheduling workloads.
	// This field is OPTIONAL. If it is not specified, the CO MAY assume
	// the volume is equally accessible from all nodes in the cluster and
	// may schedule workloads referencing the volume on any available
	// node.
	//
	// Example 1:
	//   accessible_topology = {"region": "R1", "zone": "Z2"}
	// Indicates a volume accessible only from the "region" "R1" and the
	// "zone" "Z2".
	//
	// Example 2:
	//   accessible_topology =
	//     {"region": "R1", "zone": "Z2"},
	//     {"region": "R1", "zone": "Z3"}
	// Indicates a volume accessible from both "zone" "Z2" and "zone" "Z3"
	// in the "region" "R1".
	AccessibleTopology   []*Topology `protobuf:"bytes,5,rep,name=accessible_topology,json=accessibleTopology" json:"accessible_topology,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *Volume) Reset()         { *m = Volume{} }
func (m *Volume) String() string { return proto.CompactTextString(m) }
func (*Volume) ProtoMessage()    {}
func (*Volume) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{12}
}
func (m *Volume) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Volume.Unmarshal(m, b)
}
func (m *Volume) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Volume.Marshal(b, m, deterministic)
}
func (dst *Volume) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Volume.Merge(dst, src)
}
func (m *Volume) XXX_Size() int {
	return xxx_messageInfo_Volume.Size(m)
}
func (m *Volume) XXX_DiscardUnknown() {
	xxx_messageInfo_Volume.DiscardUnknown(m)
}

var xxx_messageInfo_Volume proto.InternalMessageInfo

func (m *Volume) GetCapacityBytes() int64 {
	if m != nil {
		return m.CapacityBytes
	}
	return 0
}

func (m *Volume) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Volume) GetAttributes() map[string]string {
	if m != nil {
		return m.Attributes
	}
	return nil
}

func (m *Volume) GetContentSource() *VolumeContentSource {
	if m != nil {
		return m.ContentSource
	}
	return nil
}

func (m *Volume) GetAccessibleTopology() []*Topology {
	if m != nil {
		return m.AccessibleTopology
	}
	return nil
}

type TopologyRequirement struct {
	// Specifies the list of topologies the provisioned volume MUST be
	// accessible from.
	// This field is OPTIONAL. If TopologyRequirement is specified either
	// requisite or preferred or both MUST be specified.
	//
	// If requisite is specified, the provisioned volume MUST be
	// accessible from at least one of the requisite topologies.
	//
	// Given
	//   x = number of topologies provisioned volume is accessible from
	//   n = number of requisite topologies
	// The CO MUST ensure n >= 1. The SP MUST ensure x >= 1
	// If x==n, than the SP MUST make the provisioned volume available to
	// all topologies from the list of requisite topologies. If it is
	// unable to do so, the SP MUST fail the CreateVolume call.
	// For example, if a volume should be accessible from a single zone,
	// and requisite =
	//   {"region": "R1", "zone": "Z2"}
	// then the provisioned volume MUST be accessible from the "region"
	// "R1" and the "zone" "Z2".
	// Similarly, if a volume should be accessible from two zones, and
	// requisite =
	//   {"region": "R1", "zone": "Z2"},
	//   {"region": "R1", "zone": "Z3"}
	// then the provisioned volume MUST be accessible from the "region"
	// "R1" and both "zone" "Z2" and "zone" "Z3".
	//
	// If x<n, than the SP SHALL choose x unique topologies from the list
	// of requisite topologies. If it is unable to do so, the SP MUST fail
	// the CreateVolume call.
	// For example, if a volume should be accessible from a single zone,
	// and requisite =
	//   {"region": "R1", "zone": "Z2"},
	//   {"region": "R1", "zone": "Z3"}
	// then the SP may choose to make the provisioned volume available in
	// either the "zone" "Z2" or the "zone" "Z3" in the "region" "R1".
	// Similarly, if a volume should be accessible from two zones, and
	// requisite =
	//   {"region": "R1", "zone": "Z2"},
	//   {"region": "R1", "zone": "Z3"},
	//   {"region": "R1", "zone": "Z4"}
	// then the provisioned volume MUST be accessible from any combination
	// of two unique topologies: e.g. "R1/Z2" and "R1/Z3", or "R1/Z2" and
	//  "R1/Z4", or "R1/Z3" and "R1/Z4".
	//
	// If x>n, than the SP MUST make the provisioned volume available from
	// all topologies from the list of requisite topologies and MAY choose
	// the remaining x-n unique topologies from the list of all possible
	// topologies. If it is unable to do so, the SP MUST fail the
	// CreateVolume call.
	// For example, if a volume should be accessible from two zones, and
	// requisite =
	//   {"region": "R1", "zone": "Z2"}
	// then the provisioned volume MUST be accessible from the "region"
	// "R1" and the "zone" "Z2" and the SP may select the second zone
	// independently, e.g. "R1/Z4".
	Requisite []*Topology `protobuf:"bytes,1,rep,name=requisite" json:"requisite,omitempty"`
	// Specifies the list of topologies the CO would prefer the volume to
	// be provisioned in.
	//
	// This field is OPTIONAL. If TopologyRequirement is specified either
	// requisite or preferred or both MUST be specified.
	//
	// An SP MUST attempt to make the provisioned volume available using
	// the preferred topologies in order from first to last.
	//
	// If requisite is specified, all topologies in preferred list MUST
	// also be present in the list of requisite topologies.
	//
	// If the SP is unable to to make the provisioned volume available
	// from any of the preferred topologies, the SP MAY choose a topology
	// from the list of requisite topologies.
	// If the list of requisite topologies is not specified, then the SP
	// MAY choose from the list of all possible topologies.
	// If the list of requisite topologies is specified and the SP is
	// unable to to make the provisioned volume available from any of the
	// requisite topologies it MUST fail the CreateVolume call.
	//
	// Example 1:
	// Given a volume should be accessible from a single zone, and
	// requisite =
	//   {"region": "R1", "zone": "Z2"},
	//   {"region": "R1", "zone": "Z3"}
	// preferred =
	//   {"region": "R1", "zone": "Z3"}
	// then the the SP SHOULD first attempt to make the provisioned volume
	// available from "zone" "Z3" in the "region" "R1" and fall back to
	// "zone" "Z2" in the "region" "R1" if that is not possible.
	//
	// Example 2:
	// Given a volume should be accessible from a single zone, and
	// requisite =
	//   {"region": "R1", "zone": "Z2"},
	//   {"region": "R1", "zone": "Z3"},
	//   {"region": "R1", "zone": "Z4"},
	//   {"region": "R1", "zone": "Z5"}
	// preferred =
	//   {"region": "R1", "zone": "Z4"},
	//   {"region": "R1", "zone": "Z2"}
	// then the the SP SHOULD first attempt to make the provisioned volume
	// accessible from "zone" "Z4" in the "region" "R1" and fall back to
	// "zone" "Z2" in the "region" "R1" if that is not possible. If that
	// is not possible, the SP may choose between either the "zone"
	// "Z3" or "Z5" in the "region" "R1".
	//
	// Example 3:
	// Given a volume should be accessible from TWO zones (because an
	// opaque parameter in CreateVolumeRequest, for example, specifies
	// the volume is accessible from two zones, aka synchronously
	// replicated), and
	// requisite =
	//   {"region": "R1", "zone": "Z2"},
	//   {"region": "R1", "zone": "Z3"},
	//   {"region": "R1", "zone": "Z4"},
	//   {"region": "R1", "zone": "Z5"}
	// preferred =
	//   {"region": "R1", "zone": "Z5"},
	//   {"region": "R1", "zone": "Z3"}
	// then the the SP SHOULD first attempt to make the provisioned volume
	// accessible from the combination of the two "zones" "Z5" and "Z3" in
	// the "region" "R1". If that's not possible, it should fall back to
	// a combination of "Z5" and other possibilities from the list of
	// requisite. If that's not possible, it should fall back  to a
	// combination of "Z3" and other possibilities from the list of
	// requisite. If that's not possible, it should fall back  to a
	// combination of other possibilities from the list of requisite.
	Preferred            []*Topology `protobuf:"bytes,2,rep,name=preferred" json:"preferred,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *TopologyRequirement) Reset()         { *m = TopologyRequirement{} }
func (m *TopologyRequirement) String() string { return proto.CompactTextString(m) }
func (*TopologyRequirement) ProtoMessage()    {}
func (*TopologyRequirement) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{13}
}
func (m *TopologyRequirement) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TopologyRequirement.Unmarshal(m, b)
}
func (m *TopologyRequirement) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TopologyRequirement.Marshal(b, m, deterministic)
}
func (dst *TopologyRequirement) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TopologyRequirement.Merge(dst, src)
}
func (m *TopologyRequirement) XXX_Size() int {
	return xxx_messageInfo_TopologyRequirement.Size(m)
}
func (m *TopologyRequirement) XXX_DiscardUnknown() {
	xxx_messageInfo_TopologyRequirement.DiscardUnknown(m)
}

var xxx_messageInfo_TopologyRequirement proto.InternalMessageInfo

func (m *TopologyRequirement) GetRequisite() []*Topology {
	if m != nil {
		return m.Requisite
	}
	return nil
}

func (m *TopologyRequirement) GetPreferred() []*Topology {
	if m != nil {
		return m.Preferred
	}
	return nil
}

// Topology is a map of topological domains to topological segments.
// A topological domain is a sub-division of a cluster, like "region",
// "zone", "rack", etc.
// A topological segment is a specific instance of a topological domain,
// like "zone3", "rack3", etc.
// For example {"com.company/zone": "Z1", "com.company/rack": "R3"}
// Valid keys have two segments: an optional prefix and name, separated
// by a slash (/), for example: "com.company.example/zone".
// The key name segment is required. The prefix is optional.
// Both the key name and the prefix MUST each be 63 characters or less,
// begin and end with an alphanumeric character ([a-z0-9A-Z]) and
// contain only dashes (-), underscores (_), dots (.), or alphanumerics
// in between, for example "zone".
// The key prefix MUST follow reverse domain name notation format
// (https://en.wikipedia.org/wiki/Reverse_domain_name_notation).
// The key prefix SHOULD include the plugin's host company name and/or
// the plugin name, to minimize the possibility of collisions with keys
// from other plugins.
// If a key prefix is specified, it MUST be identical across all
// topology keys returned by the SP (across all RPCs).
// Keys MUST be case-insensitive. Meaning the keys "Zone" and "zone"
// MUST not both exist.
// Each value (topological segment) MUST contain 1 or more strings.
// Each string MUST be 63 characters or less and begin and end with an
// alphanumeric character with '-', '_', '.', or alphanumerics in
// between.
type Topology struct {
	Segments             map[string]string `protobuf:"bytes,1,rep,name=segments" json:"segments,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *Topology) Reset()         { *m = Topology{} }
func (m *Topology) String() string { return proto.CompactTextString(m) }
func (*Topology) ProtoMessage()    {}
func (*Topology) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{14}
}
func (m *Topology) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Topology.Unmarshal(m, b)
}
func (m *Topology) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Topology.Marshal(b, m, deterministic)
}
func (dst *Topology) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Topology.Merge(dst, src)
}
func (m *Topology) XXX_Size() int {
	return xxx_messageInfo_Topology.Size(m)
}
func (m *Topology) XXX_DiscardUnknown() {
	xxx_messageInfo_Topology.DiscardUnknown(m)
}

var xxx_messageInfo_Topology proto.InternalMessageInfo

func (m *Topology) GetSegments() map[string]string {
	if m != nil {
		return m.Segments
	}
	return nil
}

type DeleteVolumeRequest struct {
	// The ID of the volume to be deprovisioned.
	// This field is REQUIRED.
	VolumeId string `protobuf:"bytes,1,opt,name=volume_id,json=volumeId" json:"volume_id,omitempty"`
	// Secrets required by plugin to complete volume deletion request.
	// This field is OPTIONAL. Refer to the `Secrets Requirements`
	// section on how to use this field.
	ControllerDeleteSecrets map[string]string `protobuf:"bytes,2,rep,name=controller_delete_secrets,json=controllerDeleteSecrets" json:"controller_delete_secrets,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	XXX_NoUnkeyedLiteral    struct{}          `json:"-"`
	XXX_unrecognized        []byte            `json:"-"`
	XXX_sizecache           int32             `json:"-"`
}

func (m *DeleteVolumeRequest) Reset()         { *m = DeleteVolumeRequest{} }
func (m *DeleteVolumeRequest) String() string { return proto.CompactTextString(m) }
func (*DeleteVolumeRequest) ProtoMessage()    {}
func (*DeleteVolumeRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{15}
}
func (m *DeleteVolumeRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DeleteVolumeRequest.Unmarshal(m, b)
}
func (m *DeleteVolumeRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DeleteVolumeRequest.Marshal(b, m, deterministic)
}
func (dst *DeleteVolumeRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeleteVolumeRequest.Merge(dst, src)
}
func (m *DeleteVolumeRequest) XXX_Size() int {
	return xxx_messageInfo_DeleteVolumeRequest.Size(m)
}
func (m *DeleteVolumeRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DeleteVolumeRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DeleteVolumeRequest proto.InternalMessageInfo

func (m *DeleteVolumeRequest) GetVolumeId() string {
	if m != nil {
		return m.VolumeId
	}
	return ""
}

func (m *DeleteVolumeRequest) GetControllerDeleteSecrets() map[string]string {
	if m != nil {
		return m.ControllerDeleteSecrets
	}
	return nil
}

type DeleteVolumeResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DeleteVolumeResponse) Reset()         { *m = DeleteVolumeResponse{} }
func (m *DeleteVolumeResponse) String() string { return proto.CompactTextString(m) }
func (*DeleteVolumeResponse) ProtoMessage()    {}
func (*DeleteVolumeResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{16}
}
func (m *DeleteVolumeResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DeleteVolumeResponse.Unmarshal(m, b)
}
func (m *DeleteVolumeResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DeleteVolumeResponse.Marshal(b, m, deterministic)
}
func (dst *DeleteVolumeResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeleteVolumeResponse.Merge(dst, src)
}
func (m *DeleteVolumeResponse) XXX_Size() int {
	return xxx_messageInfo_DeleteVolumeResponse.Size(m)
}
func (m *DeleteVolumeResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_DeleteVolumeResponse.DiscardUnknown(m)
}

var xxx_messageInfo_DeleteVolumeResponse proto.InternalMessageInfo

type ControllerPublishVolumeRequest struct {
	// The ID of the volume to be used on a node.
	// This field is REQUIRED.
	VolumeId string `protobuf:"bytes,1,opt,name=volume_id,json=volumeId" json:"volume_id,omitempty"`
	// The ID of the node. This field is REQUIRED. The CO SHALL set this
	// field to match the node ID returned by `NodeGetInfo`.
	NodeId string `protobuf:"bytes,2,opt,name=node_id,json=nodeId" json:"node_id,omitempty"`
	// The capability of the volume the CO expects the volume to have.
	// This is a REQUIRED field.
	VolumeCapability *VolumeCapability `protobuf:"bytes,3,opt,name=volume_capability,json=volumeCapability" json:"volume_capability,omitempty"`
	// Whether to publish the volume in readonly mode. This field is
	// REQUIRED.
	Readonly bool `protobuf:"varint,4,opt,name=readonly" json:"readonly,omitempty"`
	// Secrets required by plugin to complete controller publish volume
	// request. This field is OPTIONAL. Refer to the
	// `Secrets Requirements` section on how to use this field.
	ControllerPublishSecrets map[string]string `protobuf:"bytes,5,rep,name=controller_publish_secrets,json=controllerPublishSecrets" json:"controller_publish_secrets,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// Attributes of the volume to be used on a node. This field is
	// OPTIONAL and MUST match the attributes of the Volume identified
	// by `volume_id`.
	VolumeAttributes     map[string]string `protobuf:"bytes,6,rep,name=volume_attributes,json=volumeAttributes" json:"volume_attributes,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *ControllerPublishVolumeRequest) Reset()         { *m = ControllerPublishVolumeRequest{} }
func (m *ControllerPublishVolumeRequest) String() string { return proto.CompactTextString(m) }
func (*ControllerPublishVolumeRequest) ProtoMessage()    {}
func (*ControllerPublishVolumeRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{17}
}
func (m *ControllerPublishVolumeRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ControllerPublishVolumeRequest.Unmarshal(m, b)
}
func (m *ControllerPublishVolumeRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ControllerPublishVolumeRequest.Marshal(b, m, deterministic)
}
func (dst *ControllerPublishVolumeRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ControllerPublishVolumeRequest.Merge(dst, src)
}
func (m *ControllerPublishVolumeRequest) XXX_Size() int {
	return xxx_messageInfo_ControllerPublishVolumeRequest.Size(m)
}
func (m *ControllerPublishVolumeRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ControllerPublishVolumeRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ControllerPublishVolumeRequest proto.InternalMessageInfo

func (m *ControllerPublishVolumeRequest) GetVolumeId() string {
	if m != nil {
		return m.VolumeId
	}
	return ""
}

func (m *ControllerPublishVolumeRequest) GetNodeId() string {
	if m != nil {
		return m.NodeId
	}
	return ""
}

func (m *ControllerPublishVolumeRequest) GetVolumeCapability() *VolumeCapability {
	if m != nil {
		return m.VolumeCapability
	}
	return nil
}

func (m *ControllerPublishVolumeRequest) GetReadonly() bool {
	if m != nil {
		return m.Readonly
	}
	return false
}

func (m *ControllerPublishVolumeRequest) GetControllerPublishSecrets() map[string]string {
	if m != nil {
		return m.ControllerPublishSecrets
	}
	return nil
}

func (m *ControllerPublishVolumeRequest) GetVolumeAttributes() map[string]string {
	if m != nil {
		return m.VolumeAttributes
	}
	return nil
}

type ControllerPublishVolumeResponse struct {
	// The SP specific information that will be passed to the Plugin in
	// the subsequent `NodeStageVolume` or `NodePublishVolume` calls
	// for the given volume.
	// This information is opaque to the CO. This field is OPTIONAL.
	PublishInfo          map[string]string `protobuf:"bytes,1,rep,name=publish_info,json=publishInfo" json:"publish_info,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *ControllerPublishVolumeResponse) Reset()         { *m = ControllerPublishVolumeResponse{} }
func (m *ControllerPublishVolumeResponse) String() string { return proto.CompactTextString(m) }
func (*ControllerPublishVolumeResponse) ProtoMessage()    {}
func (*ControllerPublishVolumeResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{18}
}
func (m *ControllerPublishVolumeResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ControllerPublishVolumeResponse.Unmarshal(m, b)
}
func (m *ControllerPublishVolumeResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ControllerPublishVolumeResponse.Marshal(b, m, deterministic)
}
func (dst *ControllerPublishVolumeResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ControllerPublishVolumeResponse.Merge(dst, src)
}
func (m *ControllerPublishVolumeResponse) XXX_Size() int {
	return xxx_messageInfo_ControllerPublishVolumeResponse.Size(m)
}
func (m *ControllerPublishVolumeResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ControllerPublishVolumeResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ControllerPublishVolumeResponse proto.InternalMessageInfo

func (m *ControllerPublishVolumeResponse) GetPublishInfo() map[string]string {
	if m != nil {
		return m.PublishInfo
	}
	return nil
}

type ControllerUnpublishVolumeRequest struct {
	// The ID of the volume. This field is REQUIRED.
	VolumeId string `protobuf:"bytes,1,opt,name=volume_id,json=volumeId" json:"volume_id,omitempty"`
	// The ID of the node. This field is OPTIONAL. The CO SHOULD set this
	// field to match the node ID returned by `NodeGetInfo` or leave it
	// unset. If the value is set, the SP MUST unpublish the volume from
	// the specified node. If the value is unset, the SP MUST unpublish
	// the volume from all nodes it is published to.
	NodeId string `protobuf:"bytes,2,opt,name=node_id,json=nodeId" json:"node_id,omitempty"`
	// Secrets required by plugin to complete controller unpublish volume
	// request. This SHOULD be the same secrets passed to the
	// ControllerPublishVolume call for the specified volume.
	// This field is OPTIONAL. Refer to the `Secrets Requirements`
	// section on how to use this field.
	ControllerUnpublishSecrets map[string]string `protobuf:"bytes,3,rep,name=controller_unpublish_secrets,json=controllerUnpublishSecrets" json:"controller_unpublish_secrets,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	XXX_NoUnkeyedLiteral       struct{}          `json:"-"`
	XXX_unrecognized           []byte            `json:"-"`
	XXX_sizecache              int32             `json:"-"`
}

func (m *ControllerUnpublishVolumeRequest) Reset()         { *m = ControllerUnpublishVolumeRequest{} }
func (m *ControllerUnpublishVolumeRequest) String() string { return proto.CompactTextString(m) }
func (*ControllerUnpublishVolumeRequest) ProtoMessage()    {}
func (*ControllerUnpublishVolumeRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{19}
}
func (m *ControllerUnpublishVolumeRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ControllerUnpublishVolumeRequest.Unmarshal(m, b)
}
func (m *ControllerUnpublishVolumeRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ControllerUnpublishVolumeRequest.Marshal(b, m, deterministic)
}
func (dst *ControllerUnpublishVolumeRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ControllerUnpublishVolumeRequest.Merge(dst, src)
}
func (m *ControllerUnpublishVolumeRequest) XXX_Size() int {
	return xxx_messageInfo_ControllerUnpublishVolumeRequest.Size(m)
}
func (m *ControllerUnpublishVolumeRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ControllerUnpublishVolumeRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ControllerUnpublishVolumeRequest proto.InternalMessageInfo

func (m *ControllerUnpublishVolumeRequest) GetVolumeId() string {
	if m != nil {
		return m.VolumeId
	}
	return ""
}

func (m *ControllerUnpublishVolumeRequest) GetNodeId() string {
	if m != nil {
		return m.NodeId
	}
	return ""
}

func (m *ControllerUnpublishVolumeRequest) GetControllerUnpublishSecrets() map[string]string {
	if m != nil {
		return m.ControllerUnpublishSecrets
	}
	return nil
}

type ControllerUnpublishVolumeResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ControllerUnpublishVolumeResponse) Reset()         { *m = ControllerUnpublishVolumeResponse{} }
func (m *ControllerUnpublishVolumeResponse) String() string { return proto.CompactTextString(m) }
func (*ControllerUnpublishVolumeResponse) ProtoMessage()    {}
func (*ControllerUnpublishVolumeResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{20}
}
func (m *ControllerUnpublishVolumeResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ControllerUnpublishVolumeResponse.Unmarshal(m, b)
}
func (m *ControllerUnpublishVolumeResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ControllerUnpublishVolumeResponse.Marshal(b, m, deterministic)
}
func (dst *ControllerUnpublishVolumeResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ControllerUnpublishVolumeResponse.Merge(dst, src)
}
func (m *ControllerUnpublishVolumeResponse) XXX_Size() int {
	return xxx_messageInfo_ControllerUnpublishVolumeResponse.Size(m)
}
func (m *ControllerUnpublishVolumeResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ControllerUnpublishVolumeResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ControllerUnpublishVolumeResponse proto.InternalMessageInfo

type ValidateVolumeCapabilitiesRequest struct {
	// The ID of the volume to check. This field is REQUIRED.
	VolumeId string `protobuf:"bytes,1,opt,name=volume_id,json=volumeId" json:"volume_id,omitempty"`
	// The capabilities that the CO wants to check for the volume. This
	// call SHALL return "supported" only if all the volume capabilities
	// specified below are supported. This field is REQUIRED.
	VolumeCapabilities []*VolumeCapability `protobuf:"bytes,2,rep,name=volume_capabilities,json=volumeCapabilities" json:"volume_capabilities,omitempty"`
	// Attributes of the volume to check. This field is OPTIONAL and MUST
	// match the attributes of the Volume identified by `volume_id`.
	VolumeAttributes map[string]string `protobuf:"bytes,3,rep,name=volume_attributes,json=volumeAttributes" json:"volume_attributes,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// Specifies where (regions, zones, racks, etc.) the caller believes
	// the volume is accessible from.
	// A caller MAY specify multiple topologies to indicate they believe
	// the volume to be accessible from multiple locations.
	// This field is OPTIONAL. This field SHALL NOT be set unless the
	// plugin advertises the ACCESSIBILITY_CONSTRAINTS capability.
	AccessibleTopology   []*Topology `protobuf:"bytes,4,rep,name=accessible_topology,json=accessibleTopology" json:"accessible_topology,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *ValidateVolumeCapabilitiesRequest) Reset()         { *m = ValidateVolumeCapabilitiesRequest{} }
func (m *ValidateVolumeCapabilitiesRequest) String() string { return proto.CompactTextString(m) }
func (*ValidateVolumeCapabilitiesRequest) ProtoMessage()    {}
func (*ValidateVolumeCapabilitiesRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{21}
}
func (m *ValidateVolumeCapabilitiesRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ValidateVolumeCapabilitiesRequest.Unmarshal(m, b)
}
func (m *ValidateVolumeCapabilitiesRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ValidateVolumeCapabilitiesRequest.Marshal(b, m, deterministic)
}
func (dst *ValidateVolumeCapabilitiesRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ValidateVolumeCapabilitiesRequest.Merge(dst, src)
}
func (m *ValidateVolumeCapabilitiesRequest) XXX_Size() int {
	return xxx_messageInfo_ValidateVolumeCapabilitiesRequest.Size(m)
}
func (m *ValidateVolumeCapabilitiesRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ValidateVolumeCapabilitiesRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ValidateVolumeCapabilitiesRequest proto.InternalMessageInfo

func (m *ValidateVolumeCapabilitiesRequest) GetVolumeId() string {
	if m != nil {
		return m.VolumeId
	}
	return ""
}

func (m *ValidateVolumeCapabilitiesRequest) GetVolumeCapabilities() []*VolumeCapability {
	if m != nil {
		return m.VolumeCapabilities
	}
	return nil
}

func (m *ValidateVolumeCapabilitiesRequest) GetVolumeAttributes() map[string]string {
	if m != nil {
		return m.VolumeAttributes
	}
	return nil
}

func (m *ValidateVolumeCapabilitiesRequest) GetAccessibleTopology() []*Topology {
	if m != nil {
		return m.AccessibleTopology
	}
	return nil
}

type ValidateVolumeCapabilitiesResponse struct {
	// True if the Plugin supports the specified capabilities for the
	// given volume. This field is REQUIRED.
	Supported bool `protobuf:"varint,1,opt,name=supported" json:"supported,omitempty"`
	// Message to the CO if `supported` above is false. This field is
	// OPTIONAL.
	// An empty string is equal to an unspecified field value.
	Message              string   `protobuf:"bytes,2,opt,name=message" json:"message,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ValidateVolumeCapabilitiesResponse) Reset()         { *m = ValidateVolumeCapabilitiesResponse{} }
func (m *ValidateVolumeCapabilitiesResponse) String() string { return proto.CompactTextString(m) }
func (*ValidateVolumeCapabilitiesResponse) ProtoMessage()    {}
func (*ValidateVolumeCapabilitiesResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{22}
}
func (m *ValidateVolumeCapabilitiesResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ValidateVolumeCapabilitiesResponse.Unmarshal(m, b)
}
func (m *ValidateVolumeCapabilitiesResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ValidateVolumeCapabilitiesResponse.Marshal(b, m, deterministic)
}
func (dst *ValidateVolumeCapabilitiesResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ValidateVolumeCapabilitiesResponse.Merge(dst, src)
}
func (m *ValidateVolumeCapabilitiesResponse) XXX_Size() int {
	return xxx_messageInfo_ValidateVolumeCapabilitiesResponse.Size(m)
}
func (m *ValidateVolumeCapabilitiesResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ValidateVolumeCapabilitiesResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ValidateVolumeCapabilitiesResponse proto.InternalMessageInfo

func (m *ValidateVolumeCapabilitiesResponse) GetSupported() bool {
	if m != nil {
		return m.Supported
	}
	return false
}

func (m *ValidateVolumeCapabilitiesResponse) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

type ListVolumesRequest struct {
	// If specified (non-zero value), the Plugin MUST NOT return more
	// entries than this number in the response. If the actual number of
	// entries is more than this number, the Plugin MUST set `next_token`
	// in the response which can be used to get the next page of entries
	// in the subsequent `ListVolumes` call. This field is OPTIONAL. If
	// not specified (zero value), it means there is no restriction on the
	// number of entries that can be returned.
	// The value of this field MUST NOT be negative.
	MaxEntries int32 `protobuf:"varint,1,opt,name=max_entries,json=maxEntries" json:"max_entries,omitempty"`
	// A token to specify where to start paginating. Set this field to
	// `next_token` returned by a previous `ListVolumes` call to get the
	// next page of entries. This field is OPTIONAL.
	// An empty string is equal to an unspecified field value.
	StartingToken        string   `protobuf:"bytes,2,opt,name=starting_token,json=startingToken" json:"starting_token,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListVolumesRequest) Reset()         { *m = ListVolumesRequest{} }
func (m *ListVolumesRequest) String() string { return proto.CompactTextString(m) }
func (*ListVolumesRequest) ProtoMessage()    {}
func (*ListVolumesRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{23}
}
func (m *ListVolumesRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListVolumesRequest.Unmarshal(m, b)
}
func (m *ListVolumesRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListVolumesRequest.Marshal(b, m, deterministic)
}
func (dst *ListVolumesRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListVolumesRequest.Merge(dst, src)
}
func (m *ListVolumesRequest) XXX_Size() int {
	return xxx_messageInfo_ListVolumesRequest.Size(m)
}
func (m *ListVolumesRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListVolumesRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListVolumesRequest proto.InternalMessageInfo

func (m *ListVolumesRequest) GetMaxEntries() int32 {
	if m != nil {
		return m.MaxEntries
	}
	return 0
}

func (m *ListVolumesRequest) GetStartingToken() string {
	if m != nil {
		return m.StartingToken
	}
	return ""
}

type ListVolumesResponse struct {
	Entries []*ListVolumesResponse_Entry `protobuf:"bytes,1,rep,name=entries" json:"entries,omitempty"`
	// This token allows you to get the next page of entries for
	// `ListVolumes` request. If the number of entries is larger than
	// `max_entries`, use the `next_token` as a value for the
	// `starting_token` field in the next `ListVolumes` request. This
	// field is OPTIONAL.
	// An empty string is equal to an unspecified field value.
	NextToken            string   `protobuf:"bytes,2,opt,name=next_token,json=nextToken" json:"next_token,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListVolumesResponse) Reset()         { *m = ListVolumesResponse{} }
func (m *ListVolumesResponse) String() string { return proto.CompactTextString(m) }
func (*ListVolumesResponse) ProtoMessage()    {}
func (*ListVolumesResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{24}
}
func (m *ListVolumesResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListVolumesResponse.Unmarshal(m, b)
}
func (m *ListVolumesResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListVolumesResponse.Marshal(b, m, deterministic)
}
func (dst *ListVolumesResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListVolumesResponse.Merge(dst, src)
}
func (m *ListVolumesResponse) XXX_Size() int {
	return xxx_messageInfo_ListVolumesResponse.Size(m)
}
func (m *ListVolumesResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ListVolumesResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ListVolumesResponse proto.InternalMessageInfo

func (m *ListVolumesResponse) GetEntries() []*ListVolumesResponse_Entry {
	if m != nil {
		return m.Entries
	}
	return nil
}

func (m *ListVolumesResponse) GetNextToken() string {
	if m != nil {
		return m.NextToken
	}
	return ""
}

type ListVolumesResponse_Entry struct {
	Volume               *Volume  `protobuf:"bytes,1,opt,name=volume" json:"volume,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListVolumesResponse_Entry) Reset()         { *m = ListVolumesResponse_Entry{} }
func (m *ListVolumesResponse_Entry) String() string { return proto.CompactTextString(m) }
func (*ListVolumesResponse_Entry) ProtoMessage()    {}
func (*ListVolumesResponse_Entry) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{24, 0}
}
func (m *ListVolumesResponse_Entry) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListVolumesResponse_Entry.Unmarshal(m, b)
}
func (m *ListVolumesResponse_Entry) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListVolumesResponse_Entry.Marshal(b, m, deterministic)
}
func (dst *ListVolumesResponse_Entry) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListVolumesResponse_Entry.Merge(dst, src)
}
func (m *ListVolumesResponse_Entry) XXX_Size() int {
	return xxx_messageInfo_ListVolumesResponse_Entry.Size(m)
}
func (m *ListVolumesResponse_Entry) XXX_DiscardUnknown() {
	xxx_messageInfo_ListVolumesResponse_Entry.DiscardUnknown(m)
}

var xxx_messageInfo_ListVolumesResponse_Entry proto.InternalMessageInfo

func (m *ListVolumesResponse_Entry) GetVolume() *Volume {
	if m != nil {
		return m.Volume
	}
	return nil
}

type GetCapacityRequest struct {
	// If specified, the Plugin SHALL report the capacity of the storage
	// that can be used to provision volumes that satisfy ALL of the
	// specified `volume_capabilities`. These are the same
	// `volume_capabilities` the CO will use in `CreateVolumeRequest`.
	// This field is OPTIONAL.
	VolumeCapabilities []*VolumeCapability `protobuf:"bytes,1,rep,name=volume_capabilities,json=volumeCapabilities" json:"volume_capabilities,omitempty"`
	// If specified, the Plugin SHALL report the capacity of the storage
	// that can be used to provision volumes with the given Plugin
	// specific `parameters`. These are the same `parameters` the CO will
	// use in `CreateVolumeRequest`. This field is OPTIONAL.
	Parameters map[string]string `protobuf:"bytes,2,rep,name=parameters" json:"parameters,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// If specified, the Plugin SHALL report the capacity of the storage
	// that can be used to provision volumes that in the specified
	// `accessible_topology`. This is the same as the
	// `accessible_topology` the CO returns in a `CreateVolumeResponse`.
	// This field is OPTIONAL. This field SHALL NOT be set unless the
	// plugin advertises the ACCESSIBILITY_CONSTRAINTS capability.
	AccessibleTopology   *Topology `protobuf:"bytes,3,opt,name=accessible_topology,json=accessibleTopology" json:"accessible_topology,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *GetCapacityRequest) Reset()         { *m = GetCapacityRequest{} }
func (m *GetCapacityRequest) String() string { return proto.CompactTextString(m) }
func (*GetCapacityRequest) ProtoMessage()    {}
func (*GetCapacityRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{25}
}
func (m *GetCapacityRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetCapacityRequest.Unmarshal(m, b)
}
func (m *GetCapacityRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetCapacityRequest.Marshal(b, m, deterministic)
}
func (dst *GetCapacityRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetCapacityRequest.Merge(dst, src)
}
func (m *GetCapacityRequest) XXX_Size() int {
	return xxx_messageInfo_GetCapacityRequest.Size(m)
}
func (m *GetCapacityRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetCapacityRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetCapacityRequest proto.InternalMessageInfo

func (m *GetCapacityRequest) GetVolumeCapabilities() []*VolumeCapability {
	if m != nil {
		return m.VolumeCapabilities
	}
	return nil
}

func (m *GetCapacityRequest) GetParameters() map[string]string {
	if m != nil {
		return m.Parameters
	}
	return nil
}

func (m *GetCapacityRequest) GetAccessibleTopology() *Topology {
	if m != nil {
		return m.AccessibleTopology
	}
	return nil
}

type GetCapacityResponse struct {
	// The available capacity, in bytes, of the storage that can be used
	// to provision volumes. If `volume_capabilities` or `parameters` is
	// specified in the request, the Plugin SHALL take those into
	// consideration when calculating the available capacity of the
	// storage. This field is REQUIRED.
	// The value of this field MUST NOT be negative.
	AvailableCapacity    int64    `protobuf:"varint,1,opt,name=available_capacity,json=availableCapacity" json:"available_capacity,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetCapacityResponse) Reset()         { *m = GetCapacityResponse{} }
func (m *GetCapacityResponse) String() string { return proto.CompactTextString(m) }
func (*GetCapacityResponse) ProtoMessage()    {}
func (*GetCapacityResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{26}
}
func (m *GetCapacityResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetCapacityResponse.Unmarshal(m, b)
}
func (m *GetCapacityResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetCapacityResponse.Marshal(b, m, deterministic)
}
func (dst *GetCapacityResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetCapacityResponse.Merge(dst, src)
}
func (m *GetCapacityResponse) XXX_Size() int {
	return xxx_messageInfo_GetCapacityResponse.Size(m)
}
func (m *GetCapacityResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GetCapacityResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GetCapacityResponse proto.InternalMessageInfo

func (m *GetCapacityResponse) GetAvailableCapacity() int64 {
	if m != nil {
		return m.AvailableCapacity
	}
	return 0
}

type ControllerGetCapabilitiesRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ControllerGetCapabilitiesRequest) Reset()         { *m = ControllerGetCapabilitiesRequest{} }
func (m *ControllerGetCapabilitiesRequest) String() string { return proto.CompactTextString(m) }
func (*ControllerGetCapabilitiesRequest) ProtoMessage()    {}
func (*ControllerGetCapabilitiesRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{27}
}
func (m *ControllerGetCapabilitiesRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ControllerGetCapabilitiesRequest.Unmarshal(m, b)
}
func (m *ControllerGetCapabilitiesRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ControllerGetCapabilitiesRequest.Marshal(b, m, deterministic)
}
func (dst *ControllerGetCapabilitiesRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ControllerGetCapabilitiesRequest.Merge(dst, src)
}
func (m *ControllerGetCapabilitiesRequest) XXX_Size() int {
	return xxx_messageInfo_ControllerGetCapabilitiesRequest.Size(m)
}
func (m *ControllerGetCapabilitiesRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ControllerGetCapabilitiesRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ControllerGetCapabilitiesRequest proto.InternalMessageInfo

type ControllerGetCapabilitiesResponse struct {
	// All the capabilities that the controller service supports. This
	// field is OPTIONAL.
	Capabilities         []*ControllerServiceCapability `protobuf:"bytes,2,rep,name=capabilities" json:"capabilities,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                       `json:"-"`
	XXX_unrecognized     []byte                         `json:"-"`
	XXX_sizecache        int32                          `json:"-"`
}

func (m *ControllerGetCapabilitiesResponse) Reset()         { *m = ControllerGetCapabilitiesResponse{} }
func (m *ControllerGetCapabilitiesResponse) String() string { return proto.CompactTextString(m) }
func (*ControllerGetCapabilitiesResponse) ProtoMessage()    {}
func (*ControllerGetCapabilitiesResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{28}
}
func (m *ControllerGetCapabilitiesResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ControllerGetCapabilitiesResponse.Unmarshal(m, b)
}
func (m *ControllerGetCapabilitiesResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ControllerGetCapabilitiesResponse.Marshal(b, m, deterministic)
}
func (dst *ControllerGetCapabilitiesResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ControllerGetCapabilitiesResponse.Merge(dst, src)
}
func (m *ControllerGetCapabilitiesResponse) XXX_Size() int {
	return xxx_messageInfo_ControllerGetCapabilitiesResponse.Size(m)
}
func (m *ControllerGetCapabilitiesResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ControllerGetCapabilitiesResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ControllerGetCapabilitiesResponse proto.InternalMessageInfo

func (m *ControllerGetCapabilitiesResponse) GetCapabilities() []*ControllerServiceCapability {
	if m != nil {
		return m.Capabilities
	}
	return nil
}

// Specifies a capability of the controller service.
type ControllerServiceCapability struct {
	// Types that are valid to be assigned to Type:
	//	*ControllerServiceCapability_Rpc
	Type                 isControllerServiceCapability_Type `protobuf_oneof:"type"`
	XXX_NoUnkeyedLiteral struct{}                           `json:"-"`
	XXX_unrecognized     []byte                             `json:"-"`
	XXX_sizecache        int32                              `json:"-"`
}

func (m *ControllerServiceCapability) Reset()         { *m = ControllerServiceCapability{} }
func (m *ControllerServiceCapability) String() string { return proto.CompactTextString(m) }
func (*ControllerServiceCapability) ProtoMessage()    {}
func (*ControllerServiceCapability) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{29}
}
func (m *ControllerServiceCapability) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ControllerServiceCapability.Unmarshal(m, b)
}
func (m *ControllerServiceCapability) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ControllerServiceCapability.Marshal(b, m, deterministic)
}
func (dst *ControllerServiceCapability) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ControllerServiceCapability.Merge(dst, src)
}
func (m *ControllerServiceCapability) XXX_Size() int {
	return xxx_messageInfo_ControllerServiceCapability.Size(m)
}
func (m *ControllerServiceCapability) XXX_DiscardUnknown() {
	xxx_messageInfo_ControllerServiceCapability.DiscardUnknown(m)
}

var xxx_messageInfo_ControllerServiceCapability proto.InternalMessageInfo

type isControllerServiceCapability_Type interface {
	isControllerServiceCapability_Type()
}

type ControllerServiceCapability_Rpc struct {
	Rpc *ControllerServiceCapability_RPC `protobuf:"bytes,1,opt,name=rpc,oneof"`
}

func (*ControllerServiceCapability_Rpc) isControllerServiceCapability_Type() {}

func (m *ControllerServiceCapability) GetType() isControllerServiceCapability_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (m *ControllerServiceCapability) GetRpc() *ControllerServiceCapability_RPC {
	if x, ok := m.GetType().(*ControllerServiceCapability_Rpc); ok {
		return x.Rpc
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*ControllerServiceCapability) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _ControllerServiceCapability_OneofMarshaler, _ControllerServiceCapability_OneofUnmarshaler, _ControllerServiceCapability_OneofSizer, []interface{}{
		(*ControllerServiceCapability_Rpc)(nil),
	}
}

func _ControllerServiceCapability_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*ControllerServiceCapability)
	// type
	switch x := m.Type.(type) {
	case *ControllerServiceCapability_Rpc:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Rpc); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("ControllerServiceCapability.Type has unexpected type %T", x)
	}
	return nil
}

func _ControllerServiceCapability_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*ControllerServiceCapability)
	switch tag {
	case 1: // type.rpc
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(ControllerServiceCapability_RPC)
		err := b.DecodeMessage(msg)
		m.Type = &ControllerServiceCapability_Rpc{msg}
		return true, err
	default:
		return false, nil
	}
}

func _ControllerServiceCapability_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*ControllerServiceCapability)
	// type
	switch x := m.Type.(type) {
	case *ControllerServiceCapability_Rpc:
		s := proto.Size(x.Rpc)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type ControllerServiceCapability_RPC struct {
	Type                 ControllerServiceCapability_RPC_Type `protobuf:"varint,1,opt,name=type,enum=csi.v0.ControllerServiceCapability_RPC_Type" json:"type,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                             `json:"-"`
	XXX_unrecognized     []byte                               `json:"-"`
	XXX_sizecache        int32                                `json:"-"`
}

func (m *ControllerServiceCapability_RPC) Reset()         { *m = ControllerServiceCapability_RPC{} }
func (m *ControllerServiceCapability_RPC) String() string { return proto.CompactTextString(m) }
func (*ControllerServiceCapability_RPC) ProtoMessage()    {}
func (*ControllerServiceCapability_RPC) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{29, 0}
}
func (m *ControllerServiceCapability_RPC) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ControllerServiceCapability_RPC.Unmarshal(m, b)
}
func (m *ControllerServiceCapability_RPC) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ControllerServiceCapability_RPC.Marshal(b, m, deterministic)
}
func (dst *ControllerServiceCapability_RPC) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ControllerServiceCapability_RPC.Merge(dst, src)
}
func (m *ControllerServiceCapability_RPC) XXX_Size() int {
	return xxx_messageInfo_ControllerServiceCapability_RPC.Size(m)
}
func (m *ControllerServiceCapability_RPC) XXX_DiscardUnknown() {
	xxx_messageInfo_ControllerServiceCapability_RPC.DiscardUnknown(m)
}

var xxx_messageInfo_ControllerServiceCapability_RPC proto.InternalMessageInfo

func (m *ControllerServiceCapability_RPC) GetType() ControllerServiceCapability_RPC_Type {
	if m != nil {
		return m.Type
	}
	return ControllerServiceCapability_RPC_UNKNOWN
}

type CreateSnapshotRequest struct {
	// The ID of the source volume to be snapshotted.
	// This field is REQUIRED.
	SourceVolumeId string `protobuf:"bytes,1,opt,name=source_volume_id,json=sourceVolumeId" json:"source_volume_id,omitempty"`
	// The suggested name for the snapshot. This field is REQUIRED for
	// idempotency.
	Name string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	// Secrets required by plugin to complete snapshot creation request.
	// This field is OPTIONAL. Refer to the `Secrets Requirements`
	// section on how to use this field.
	CreateSnapshotSecrets map[string]string `protobuf:"bytes,3,rep,name=create_snapshot_secrets,json=createSnapshotSecrets" json:"create_snapshot_secrets,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// Plugin specific parameters passed in as opaque key-value pairs.
	// This field is OPTIONAL. The Plugin is responsible for parsing and
	// validating these parameters. COs will treat these as opaque.
	// Use cases for opaque parameters:
	// - Specify a policy to automatically clean up the snapshot.
	// - Specify an expiration date for the snapshot.
	// - Specify whether the snapshot is readonly or read/write.
	// - Specify if the snapshot should be replicated to some place.
	// - Specify primary or secondary for replication systems that
	//   support snapshotting only on primary.
	Parameters           map[string]string `protobuf:"bytes,4,rep,name=parameters" json:"parameters,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *CreateSnapshotRequest) Reset()         { *m = CreateSnapshotRequest{} }
func (m *CreateSnapshotRequest) String() string { return proto.CompactTextString(m) }
func (*CreateSnapshotRequest) ProtoMessage()    {}
func (*CreateSnapshotRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{30}
}
func (m *CreateSnapshotRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CreateSnapshotRequest.Unmarshal(m, b)
}
func (m *CreateSnapshotRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CreateSnapshotRequest.Marshal(b, m, deterministic)
}
func (dst *CreateSnapshotRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CreateSnapshotRequest.Merge(dst, src)
}
func (m *CreateSnapshotRequest) XXX_Size() int {
	return xxx_messageInfo_CreateSnapshotRequest.Size(m)
}
func (m *CreateSnapshotRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_CreateSnapshotRequest.DiscardUnknown(m)
}

var xxx_messageInfo_CreateSnapshotRequest proto.InternalMessageInfo

func (m *CreateSnapshotRequest) GetSourceVolumeId() string {
	if m != nil {
		return m.SourceVolumeId
	}
	return ""
}

func (m *CreateSnapshotRequest) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *CreateSnapshotRequest) GetCreateSnapshotSecrets() map[string]string {
	if m != nil {
		return m.CreateSnapshotSecrets
	}
	return nil
}

func (m *CreateSnapshotRequest) GetParameters() map[string]string {
	if m != nil {
		return m.Parameters
	}
	return nil
}

type CreateSnapshotResponse struct {
	// Contains all attributes of the newly created snapshot that are
	// relevant to the CO along with information required by the Plugin
	// to uniquely identify the snapshot. This field is REQUIRED.
	Snapshot             *Snapshot `protobuf:"bytes,1,opt,name=snapshot" json:"snapshot,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *CreateSnapshotResponse) Reset()         { *m = CreateSnapshotResponse{} }
func (m *CreateSnapshotResponse) String() string { return proto.CompactTextString(m) }
func (*CreateSnapshotResponse) ProtoMessage()    {}
func (*CreateSnapshotResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{31}
}
func (m *CreateSnapshotResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CreateSnapshotResponse.Unmarshal(m, b)
}
func (m *CreateSnapshotResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CreateSnapshotResponse.Marshal(b, m, deterministic)
}
func (dst *CreateSnapshotResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CreateSnapshotResponse.Merge(dst, src)
}
func (m *CreateSnapshotResponse) XXX_Size() int {
	return xxx_messageInfo_CreateSnapshotResponse.Size(m)
}
func (m *CreateSnapshotResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_CreateSnapshotResponse.DiscardUnknown(m)
}

var xxx_messageInfo_CreateSnapshotResponse proto.InternalMessageInfo

func (m *CreateSnapshotResponse) GetSnapshot() *Snapshot {
	if m != nil {
		return m.Snapshot
	}
	return nil
}

// The information about a provisioned snapshot.
type Snapshot struct {
	// This is the complete size of the snapshot in bytes. The purpose of
	// this field is to give CO guidance on how much space is needed to
	// create a volume from this snapshot. The size of the volume MUST NOT
	// be less than the size of the source snapshot. This field is
	// OPTIONAL. If this field is not set, it indicates that this size is
	// unknown. The value of this field MUST NOT be negative and a size of
	// zero means it is unspecified.
	SizeBytes int64 `protobuf:"varint,1,opt,name=size_bytes,json=sizeBytes" json:"size_bytes,omitempty"`
	// Uniquely identifies a snapshot and is generated by the plugin. It
	// will not change over time. This field is REQUIRED. The identity
	// information will be used by the CO in subsequent calls to refer to
	// the provisioned snapshot.
	Id string `protobuf:"bytes,2,opt,name=id" json:"id,omitempty"`
	// Identity information for the source volume. Note that creating a
	// snapshot from a snapshot is not supported here so the source has to
	// be a volume. This field is REQUIRED.
	SourceVolumeId string `protobuf:"bytes,3,opt,name=source_volume_id,json=sourceVolumeId" json:"source_volume_id,omitempty"`
	// Timestamp when the point-in-time snapshot is taken on the storage
	// system. The format of this field should be a Unix nanoseconds time
	// encoded as an int64. On Unix, the command `date +%s%N` returns the
	// current time in nanoseconds since 1970-01-01 00:00:00 UTC. This
	// field is REQUIRED.
	CreatedAt int64 `protobuf:"varint,4,opt,name=created_at,json=createdAt" json:"created_at,omitempty"`
	// The status of a snapshot.
	Status               *SnapshotStatus `protobuf:"bytes,5,opt,name=status" json:"status,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *Snapshot) Reset()         { *m = Snapshot{} }
func (m *Snapshot) String() string { return proto.CompactTextString(m) }
func (*Snapshot) ProtoMessage()    {}
func (*Snapshot) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{32}
}
func (m *Snapshot) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Snapshot.Unmarshal(m, b)
}
func (m *Snapshot) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Snapshot.Marshal(b, m, deterministic)
}
func (dst *Snapshot) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Snapshot.Merge(dst, src)
}
func (m *Snapshot) XXX_Size() int {
	return xxx_messageInfo_Snapshot.Size(m)
}
func (m *Snapshot) XXX_DiscardUnknown() {
	xxx_messageInfo_Snapshot.DiscardUnknown(m)
}

var xxx_messageInfo_Snapshot proto.InternalMessageInfo

func (m *Snapshot) GetSizeBytes() int64 {
	if m != nil {
		return m.SizeBytes
	}
	return 0
}

func (m *Snapshot) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Snapshot) GetSourceVolumeId() string {
	if m != nil {
		return m.SourceVolumeId
	}
	return ""
}

func (m *Snapshot) GetCreatedAt() int64 {
	if m != nil {
		return m.CreatedAt
	}
	return 0
}

func (m *Snapshot) GetStatus() *SnapshotStatus {
	if m != nil {
		return m.Status
	}
	return nil
}

// The status of a snapshot.
type SnapshotStatus struct {
	// This field is REQUIRED.
	Type SnapshotStatus_Type `protobuf:"varint,1,opt,name=type,enum=csi.v0.SnapshotStatus_Type" json:"type,omitempty"`
	// Additional information to describe why a snapshot ended up in the
	// `ERROR_UPLOADING` status. This field is OPTIONAL.
	Details              string   `protobuf:"bytes,2,opt,name=details" json:"details,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SnapshotStatus) Reset()         { *m = SnapshotStatus{} }
func (m *SnapshotStatus) String() string { return proto.CompactTextString(m) }
func (*SnapshotStatus) ProtoMessage()    {}
func (*SnapshotStatus) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{33}
}
func (m *SnapshotStatus) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SnapshotStatus.Unmarshal(m, b)
}
func (m *SnapshotStatus) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SnapshotStatus.Marshal(b, m, deterministic)
}
func (dst *SnapshotStatus) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SnapshotStatus.Merge(dst, src)
}
func (m *SnapshotStatus) XXX_Size() int {
	return xxx_messageInfo_SnapshotStatus.Size(m)
}
func (m *SnapshotStatus) XXX_DiscardUnknown() {
	xxx_messageInfo_SnapshotStatus.DiscardUnknown(m)
}

var xxx_messageInfo_SnapshotStatus proto.InternalMessageInfo

func (m *SnapshotStatus) GetType() SnapshotStatus_Type {
	if m != nil {
		return m.Type
	}
	return SnapshotStatus_UNKNOWN
}

func (m *SnapshotStatus) GetDetails() string {
	if m != nil {
		return m.Details
	}
	return ""
}

type DeleteSnapshotRequest struct {
	// The ID of the snapshot to be deleted.
	// This field is REQUIRED.
	SnapshotId string `protobuf:"bytes,1,opt,name=snapshot_id,json=snapshotId" json:"snapshot_id,omitempty"`
	// Secrets required by plugin to complete snapshot deletion request.
	// This field is OPTIONAL. Refer to the `Secrets Requirements`
	// section on how to use this field.
	DeleteSnapshotSecrets map[string]string `protobuf:"bytes,2,rep,name=delete_snapshot_secrets,json=deleteSnapshotSecrets" json:"delete_snapshot_secrets,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	XXX_NoUnkeyedLiteral  struct{}          `json:"-"`
	XXX_unrecognized      []byte            `json:"-"`
	XXX_sizecache         int32             `json:"-"`
}

func (m *DeleteSnapshotRequest) Reset()         { *m = DeleteSnapshotRequest{} }
func (m *DeleteSnapshotRequest) String() string { return proto.CompactTextString(m) }
func (*DeleteSnapshotRequest) ProtoMessage()    {}
func (*DeleteSnapshotRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{34}
}
func (m *DeleteSnapshotRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DeleteSnapshotRequest.Unmarshal(m, b)
}
func (m *DeleteSnapshotRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DeleteSnapshotRequest.Marshal(b, m, deterministic)
}
func (dst *DeleteSnapshotRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeleteSnapshotRequest.Merge(dst, src)
}
func (m *DeleteSnapshotRequest) XXX_Size() int {
	return xxx_messageInfo_DeleteSnapshotRequest.Size(m)
}
func (m *DeleteSnapshotRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DeleteSnapshotRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DeleteSnapshotRequest proto.InternalMessageInfo

func (m *DeleteSnapshotRequest) GetSnapshotId() string {
	if m != nil {
		return m.SnapshotId
	}
	return ""
}

func (m *DeleteSnapshotRequest) GetDeleteSnapshotSecrets() map[string]string {
	if m != nil {
		return m.DeleteSnapshotSecrets
	}
	return nil
}

type DeleteSnapshotResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DeleteSnapshotResponse) Reset()         { *m = DeleteSnapshotResponse{} }
func (m *DeleteSnapshotResponse) String() string { return proto.CompactTextString(m) }
func (*DeleteSnapshotResponse) ProtoMessage()    {}
func (*DeleteSnapshotResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{35}
}
func (m *DeleteSnapshotResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DeleteSnapshotResponse.Unmarshal(m, b)
}
func (m *DeleteSnapshotResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DeleteSnapshotResponse.Marshal(b, m, deterministic)
}
func (dst *DeleteSnapshotResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeleteSnapshotResponse.Merge(dst, src)
}
func (m *DeleteSnapshotResponse) XXX_Size() int {
	return xxx_messageInfo_DeleteSnapshotResponse.Size(m)
}
func (m *DeleteSnapshotResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_DeleteSnapshotResponse.DiscardUnknown(m)
}

var xxx_messageInfo_DeleteSnapshotResponse proto.InternalMessageInfo

// List all snapshots on the storage system regardless of how they were
// created.
type ListSnapshotsRequest struct {
	// If specified (non-zero value), the Plugin MUST NOT return more
	// entries than this number in the response. If the actual number of
	// entries is more than this number, the Plugin MUST set `next_token`
	// in the response which can be used to get the next page of entries
	// in the subsequent `ListSnapshots` call. This field is OPTIONAL. If
	// not specified (zero value), it means there is no restriction on the
	// number of entries that can be returned.
	// The value of this field MUST NOT be negative.
	MaxEntries int32 `protobuf:"varint,1,opt,name=max_entries,json=maxEntries" json:"max_entries,omitempty"`
	// A token to specify where to start paginating. Set this field to
	// `next_token` returned by a previous `ListSnapshots` call to get the
	// next page of entries. This field is OPTIONAL.
	// An empty string is equal to an unspecified field value.
	StartingToken string `protobuf:"bytes,2,opt,name=starting_token,json=startingToken" json:"starting_token,omitempty"`
	// Identity information for the source volume. This field is OPTIONAL.
	// It can be used to list snapshots by volume.
	SourceVolumeId string `protobuf:"bytes,3,opt,name=source_volume_id,json=sourceVolumeId" json:"source_volume_id,omitempty"`
	// Identity information for a specific snapshot. This field is
	// OPTIONAL. It can be used to list only a specific snapshot.
	// ListSnapshots will return with current snapshot information
	// and will not block if the snapshot is being uploaded.
	SnapshotId           string   `protobuf:"bytes,4,opt,name=snapshot_id,json=snapshotId" json:"snapshot_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListSnapshotsRequest) Reset()         { *m = ListSnapshotsRequest{} }
func (m *ListSnapshotsRequest) String() string { return proto.CompactTextString(m) }
func (*ListSnapshotsRequest) ProtoMessage()    {}
func (*ListSnapshotsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{36}
}
func (m *ListSnapshotsRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListSnapshotsRequest.Unmarshal(m, b)
}
func (m *ListSnapshotsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListSnapshotsRequest.Marshal(b, m, deterministic)
}
func (dst *ListSnapshotsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListSnapshotsRequest.Merge(dst, src)
}
func (m *ListSnapshotsRequest) XXX_Size() int {
	return xxx_messageInfo_ListSnapshotsRequest.Size(m)
}
func (m *ListSnapshotsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListSnapshotsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListSnapshotsRequest proto.InternalMessageInfo

func (m *ListSnapshotsRequest) GetMaxEntries() int32 {
	if m != nil {
		return m.MaxEntries
	}
	return 0
}

func (m *ListSnapshotsRequest) GetStartingToken() string {
	if m != nil {
		return m.StartingToken
	}
	return ""
}

func (m *ListSnapshotsRequest) GetSourceVolumeId() string {
	if m != nil {
		return m.SourceVolumeId
	}
	return ""
}

func (m *ListSnapshotsRequest) GetSnapshotId() string {
	if m != nil {
		return m.SnapshotId
	}
	return ""
}

type ListSnapshotsResponse struct {
	Entries []*ListSnapshotsResponse_Entry `protobuf:"bytes,1,rep,name=entries" json:"entries,omitempty"`
	// This token allows you to get the next page of entries for
	// `ListSnapshots` request. If the number of entries is larger than
	// `max_entries`, use the `next_token` as a value for the
	// `starting_token` field in the next `ListSnapshots` request. This
	// field is OPTIONAL.
	// An empty string is equal to an unspecified field value.
	NextToken            string   `protobuf:"bytes,2,opt,name=next_token,json=nextToken" json:"next_token,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListSnapshotsResponse) Reset()         { *m = ListSnapshotsResponse{} }
func (m *ListSnapshotsResponse) String() string { return proto.CompactTextString(m) }
func (*ListSnapshotsResponse) ProtoMessage()    {}
func (*ListSnapshotsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{37}
}
func (m *ListSnapshotsResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListSnapshotsResponse.Unmarshal(m, b)
}
func (m *ListSnapshotsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListSnapshotsResponse.Marshal(b, m, deterministic)
}
func (dst *ListSnapshotsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListSnapshotsResponse.Merge(dst, src)
}
func (m *ListSnapshotsResponse) XXX_Size() int {
	return xxx_messageInfo_ListSnapshotsResponse.Size(m)
}
func (m *ListSnapshotsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ListSnapshotsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ListSnapshotsResponse proto.InternalMessageInfo

func (m *ListSnapshotsResponse) GetEntries() []*ListSnapshotsResponse_Entry {
	if m != nil {
		return m.Entries
	}
	return nil
}

func (m *ListSnapshotsResponse) GetNextToken() string {
	if m != nil {
		return m.NextToken
	}
	return ""
}

type ListSnapshotsResponse_Entry struct {
	Snapshot             *Snapshot `protobuf:"bytes,1,opt,name=snapshot" json:"snapshot,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *ListSnapshotsResponse_Entry) Reset()         { *m = ListSnapshotsResponse_Entry{} }
func (m *ListSnapshotsResponse_Entry) String() string { return proto.CompactTextString(m) }
func (*ListSnapshotsResponse_Entry) ProtoMessage()    {}
func (*ListSnapshotsResponse_Entry) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{37, 0}
}
func (m *ListSnapshotsResponse_Entry) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListSnapshotsResponse_Entry.Unmarshal(m, b)
}
func (m *ListSnapshotsResponse_Entry) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListSnapshotsResponse_Entry.Marshal(b, m, deterministic)
}
func (dst *ListSnapshotsResponse_Entry) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListSnapshotsResponse_Entry.Merge(dst, src)
}
func (m *ListSnapshotsResponse_Entry) XXX_Size() int {
	return xxx_messageInfo_ListSnapshotsResponse_Entry.Size(m)
}
func (m *ListSnapshotsResponse_Entry) XXX_DiscardUnknown() {
	xxx_messageInfo_ListSnapshotsResponse_Entry.DiscardUnknown(m)
}

var xxx_messageInfo_ListSnapshotsResponse_Entry proto.InternalMessageInfo

func (m *ListSnapshotsResponse_Entry) GetSnapshot() *Snapshot {
	if m != nil {
		return m.Snapshot
	}
	return nil
}

type NodeStageVolumeRequest struct {
	// The ID of the volume to publish. This field is REQUIRED.
	VolumeId string `protobuf:"bytes,1,opt,name=volume_id,json=volumeId" json:"volume_id,omitempty"`
	// The CO SHALL set this field to the value returned by
	// `ControllerPublishVolume` if the corresponding Controller Plugin
	// has `PUBLISH_UNPUBLISH_VOLUME` controller capability, and SHALL be
	// left unset if the corresponding Controller Plugin does not have
	// this capability. This is an OPTIONAL field.
	PublishInfo map[string]string `protobuf:"bytes,2,rep,name=publish_info,json=publishInfo" json:"publish_info,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// The path to which the volume will be published. It MUST be an
	// absolute path in the root filesystem of the process serving this
	// request. The CO SHALL ensure that there is only one
	// staging_target_path per volume.
	// This is a REQUIRED field.
	StagingTargetPath string `protobuf:"bytes,3,opt,name=staging_target_path,json=stagingTargetPath" json:"staging_target_path,omitempty"`
	// The capability of the volume the CO expects the volume to have.
	// This is a REQUIRED field.
	VolumeCapability *VolumeCapability `protobuf:"bytes,4,opt,name=volume_capability,json=volumeCapability" json:"volume_capability,omitempty"`
	// Secrets required by plugin to complete node stage volume request.
	// This field is OPTIONAL. Refer to the `Secrets Requirements`
	// section on how to use this field.
	NodeStageSecrets map[string]string `protobuf:"bytes,5,rep,name=node_stage_secrets,json=nodeStageSecrets" json:"node_stage_secrets,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// Attributes of the volume to publish. This field is OPTIONAL and
	// MUST match the attributes of the `Volume` identified by
	// `volume_id`.
	VolumeAttributes     map[string]string `protobuf:"bytes,6,rep,name=volume_attributes,json=volumeAttributes" json:"volume_attributes,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *NodeStageVolumeRequest) Reset()         { *m = NodeStageVolumeRequest{} }
func (m *NodeStageVolumeRequest) String() string { return proto.CompactTextString(m) }
func (*NodeStageVolumeRequest) ProtoMessage()    {}
func (*NodeStageVolumeRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{38}
}
func (m *NodeStageVolumeRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodeStageVolumeRequest.Unmarshal(m, b)
}
func (m *NodeStageVolumeRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodeStageVolumeRequest.Marshal(b, m, deterministic)
}
func (dst *NodeStageVolumeRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeStageVolumeRequest.Merge(dst, src)
}
func (m *NodeStageVolumeRequest) XXX_Size() int {
	return xxx_messageInfo_NodeStageVolumeRequest.Size(m)
}
func (m *NodeStageVolumeRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeStageVolumeRequest.DiscardUnknown(m)
}

var xxx_messageInfo_NodeStageVolumeRequest proto.InternalMessageInfo

func (m *NodeStageVolumeRequest) GetVolumeId() string {
	if m != nil {
		return m.VolumeId
	}
	return ""
}

func (m *NodeStageVolumeRequest) GetPublishInfo() map[string]string {
	if m != nil {
		return m.PublishInfo
	}
	return nil
}

func (m *NodeStageVolumeRequest) GetStagingTargetPath() string {
	if m != nil {
		return m.StagingTargetPath
	}
	return ""
}

func (m *NodeStageVolumeRequest) GetVolumeCapability() *VolumeCapability {
	if m != nil {
		return m.VolumeCapability
	}
	return nil
}

func (m *NodeStageVolumeRequest) GetNodeStageSecrets() map[string]string {
	if m != nil {
		return m.NodeStageSecrets
	}
	return nil
}

func (m *NodeStageVolumeRequest) GetVolumeAttributes() map[string]string {
	if m != nil {
		return m.VolumeAttributes
	}
	return nil
}

type NodeStageVolumeResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NodeStageVolumeResponse) Reset()         { *m = NodeStageVolumeResponse{} }
func (m *NodeStageVolumeResponse) String() string { return proto.CompactTextString(m) }
func (*NodeStageVolumeResponse) ProtoMessage()    {}
func (*NodeStageVolumeResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{39}
}
func (m *NodeStageVolumeResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodeStageVolumeResponse.Unmarshal(m, b)
}
func (m *NodeStageVolumeResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodeStageVolumeResponse.Marshal(b, m, deterministic)
}
func (dst *NodeStageVolumeResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeStageVolumeResponse.Merge(dst, src)
}
func (m *NodeStageVolumeResponse) XXX_Size() int {
	return xxx_messageInfo_NodeStageVolumeResponse.Size(m)
}
func (m *NodeStageVolumeResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeStageVolumeResponse.DiscardUnknown(m)
}

var xxx_messageInfo_NodeStageVolumeResponse proto.InternalMessageInfo

type NodeUnstageVolumeRequest struct {
	// The ID of the volume. This field is REQUIRED.
	VolumeId string `protobuf:"bytes,1,opt,name=volume_id,json=volumeId" json:"volume_id,omitempty"`
	// The path at which the volume was published. It MUST be an absolute
	// path in the root filesystem of the process serving this request.
	// This is a REQUIRED field.
	StagingTargetPath    string   `protobuf:"bytes,2,opt,name=staging_target_path,json=stagingTargetPath" json:"staging_target_path,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NodeUnstageVolumeRequest) Reset()         { *m = NodeUnstageVolumeRequest{} }
func (m *NodeUnstageVolumeRequest) String() string { return proto.CompactTextString(m) }
func (*NodeUnstageVolumeRequest) ProtoMessage()    {}
func (*NodeUnstageVolumeRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{40}
}
func (m *NodeUnstageVolumeRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodeUnstageVolumeRequest.Unmarshal(m, b)
}
func (m *NodeUnstageVolumeRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodeUnstageVolumeRequest.Marshal(b, m, deterministic)
}
func (dst *NodeUnstageVolumeRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeUnstageVolumeRequest.Merge(dst, src)
}
func (m *NodeUnstageVolumeRequest) XXX_Size() int {
	return xxx_messageInfo_NodeUnstageVolumeRequest.Size(m)
}
func (m *NodeUnstageVolumeRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeUnstageVolumeRequest.DiscardUnknown(m)
}

var xxx_messageInfo_NodeUnstageVolumeRequest proto.InternalMessageInfo

func (m *NodeUnstageVolumeRequest) GetVolumeId() string {
	if m != nil {
		return m.VolumeId
	}
	return ""
}

func (m *NodeUnstageVolumeRequest) GetStagingTargetPath() string {
	if m != nil {
		return m.StagingTargetPath
	}
	return ""
}

type NodeUnstageVolumeResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NodeUnstageVolumeResponse) Reset()         { *m = NodeUnstageVolumeResponse{} }
func (m *NodeUnstageVolumeResponse) String() string { return proto.CompactTextString(m) }
func (*NodeUnstageVolumeResponse) ProtoMessage()    {}
func (*NodeUnstageVolumeResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{41}
}
func (m *NodeUnstageVolumeResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodeUnstageVolumeResponse.Unmarshal(m, b)
}
func (m *NodeUnstageVolumeResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodeUnstageVolumeResponse.Marshal(b, m, deterministic)
}
func (dst *NodeUnstageVolumeResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeUnstageVolumeResponse.Merge(dst, src)
}
func (m *NodeUnstageVolumeResponse) XXX_Size() int {
	return xxx_messageInfo_NodeUnstageVolumeResponse.Size(m)
}
func (m *NodeUnstageVolumeResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeUnstageVolumeResponse.DiscardUnknown(m)
}

var xxx_messageInfo_NodeUnstageVolumeResponse proto.InternalMessageInfo

type NodePublishVolumeRequest struct {
	// The ID of the volume to publish. This field is REQUIRED.
	VolumeId string `protobuf:"bytes,1,opt,name=volume_id,json=volumeId" json:"volume_id,omitempty"`
	// The CO SHALL set this field to the value returned by
	// `ControllerPublishVolume` if the corresponding Controller Plugin
	// has `PUBLISH_UNPUBLISH_VOLUME` controller capability, and SHALL be
	// left unset if the corresponding Controller Plugin does not have
	// this capability. This is an OPTIONAL field.
	PublishInfo map[string]string `protobuf:"bytes,2,rep,name=publish_info,json=publishInfo" json:"publish_info,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// The path to which the device was mounted by `NodeStageVolume`.
	// It MUST be an absolute path in the root filesystem of the process
	// serving this request.
	// It MUST be set if the Node Plugin implements the
	// `STAGE_UNSTAGE_VOLUME` node capability.
	// This is an OPTIONAL field.
	StagingTargetPath string `protobuf:"bytes,3,opt,name=staging_target_path,json=stagingTargetPath" json:"staging_target_path,omitempty"`
	// The path to which the volume will be published. It MUST be an
	// absolute path in the root filesystem of the process serving this
	// request. The CO SHALL ensure uniqueness of target_path per volume.
	// The CO SHALL ensure that the path exists, and that the process
	// serving the request has `read` and `write` permissions to the path.
	// This is a REQUIRED field.
	TargetPath string `protobuf:"bytes,4,opt,name=target_path,json=targetPath" json:"target_path,omitempty"`
	// The capability of the volume the CO expects the volume to have.
	// This is a REQUIRED field.
	VolumeCapability *VolumeCapability `protobuf:"bytes,5,opt,name=volume_capability,json=volumeCapability" json:"volume_capability,omitempty"`
	// Whether to publish the volume in readonly mode. This field is
	// REQUIRED.
	Readonly bool `protobuf:"varint,6,opt,name=readonly" json:"readonly,omitempty"`
	// Secrets required by plugin to complete node publish volume request.
	// This field is OPTIONAL. Refer to the `Secrets Requirements`
	// section on how to use this field.
	NodePublishSecrets map[string]string `protobuf:"bytes,7,rep,name=node_publish_secrets,json=nodePublishSecrets" json:"node_publish_secrets,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// Attributes of the volume to publish. This field is OPTIONAL and
	// MUST match the attributes of the Volume identified by
	// `volume_id`.
	VolumeAttributes     map[string]string `protobuf:"bytes,8,rep,name=volume_attributes,json=volumeAttributes" json:"volume_attributes,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *NodePublishVolumeRequest) Reset()         { *m = NodePublishVolumeRequest{} }
func (m *NodePublishVolumeRequest) String() string { return proto.CompactTextString(m) }
func (*NodePublishVolumeRequest) ProtoMessage()    {}
func (*NodePublishVolumeRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{42}
}
func (m *NodePublishVolumeRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodePublishVolumeRequest.Unmarshal(m, b)
}
func (m *NodePublishVolumeRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodePublishVolumeRequest.Marshal(b, m, deterministic)
}
func (dst *NodePublishVolumeRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodePublishVolumeRequest.Merge(dst, src)
}
func (m *NodePublishVolumeRequest) XXX_Size() int {
	return xxx_messageInfo_NodePublishVolumeRequest.Size(m)
}
func (m *NodePublishVolumeRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_NodePublishVolumeRequest.DiscardUnknown(m)
}

var xxx_messageInfo_NodePublishVolumeRequest proto.InternalMessageInfo

func (m *NodePublishVolumeRequest) GetVolumeId() string {
	if m != nil {
		return m.VolumeId
	}
	return ""
}

func (m *NodePublishVolumeRequest) GetPublishInfo() map[string]string {
	if m != nil {
		return m.PublishInfo
	}
	return nil
}

func (m *NodePublishVolumeRequest) GetStagingTargetPath() string {
	if m != nil {
		return m.StagingTargetPath
	}
	return ""
}

func (m *NodePublishVolumeRequest) GetTargetPath() string {
	if m != nil {
		return m.TargetPath
	}
	return ""
}

func (m *NodePublishVolumeRequest) GetVolumeCapability() *VolumeCapability {
	if m != nil {
		return m.VolumeCapability
	}
	return nil
}

func (m *NodePublishVolumeRequest) GetReadonly() bool {
	if m != nil {
		return m.Readonly
	}
	return false
}

func (m *NodePublishVolumeRequest) GetNodePublishSecrets() map[string]string {
	if m != nil {
		return m.NodePublishSecrets
	}
	return nil
}

func (m *NodePublishVolumeRequest) GetVolumeAttributes() map[string]string {
	if m != nil {
		return m.VolumeAttributes
	}
	return nil
}

type NodePublishVolumeResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NodePublishVolumeResponse) Reset()         { *m = NodePublishVolumeResponse{} }
func (m *NodePublishVolumeResponse) String() string { return proto.CompactTextString(m) }
func (*NodePublishVolumeResponse) ProtoMessage()    {}
func (*NodePublishVolumeResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{43}
}
func (m *NodePublishVolumeResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodePublishVolumeResponse.Unmarshal(m, b)
}
func (m *NodePublishVolumeResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodePublishVolumeResponse.Marshal(b, m, deterministic)
}
func (dst *NodePublishVolumeResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodePublishVolumeResponse.Merge(dst, src)
}
func (m *NodePublishVolumeResponse) XXX_Size() int {
	return xxx_messageInfo_NodePublishVolumeResponse.Size(m)
}
func (m *NodePublishVolumeResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_NodePublishVolumeResponse.DiscardUnknown(m)
}

var xxx_messageInfo_NodePublishVolumeResponse proto.InternalMessageInfo

type NodeUnpublishVolumeRequest struct {
	// The ID of the volume. This field is REQUIRED.
	VolumeId string `protobuf:"bytes,1,opt,name=volume_id,json=volumeId" json:"volume_id,omitempty"`
	// The path at which the volume was published. It MUST be an absolute
	// path in the root filesystem of the process serving this request.
	// This is a REQUIRED field.
	TargetPath           string   `protobuf:"bytes,2,opt,name=target_path,json=targetPath" json:"target_path,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NodeUnpublishVolumeRequest) Reset()         { *m = NodeUnpublishVolumeRequest{} }
func (m *NodeUnpublishVolumeRequest) String() string { return proto.CompactTextString(m) }
func (*NodeUnpublishVolumeRequest) ProtoMessage()    {}
func (*NodeUnpublishVolumeRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{44}
}
func (m *NodeUnpublishVolumeRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodeUnpublishVolumeRequest.Unmarshal(m, b)
}
func (m *NodeUnpublishVolumeRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodeUnpublishVolumeRequest.Marshal(b, m, deterministic)
}
func (dst *NodeUnpublishVolumeRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeUnpublishVolumeRequest.Merge(dst, src)
}
func (m *NodeUnpublishVolumeRequest) XXX_Size() int {
	return xxx_messageInfo_NodeUnpublishVolumeRequest.Size(m)
}
func (m *NodeUnpublishVolumeRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeUnpublishVolumeRequest.DiscardUnknown(m)
}

var xxx_messageInfo_NodeUnpublishVolumeRequest proto.InternalMessageInfo

func (m *NodeUnpublishVolumeRequest) GetVolumeId() string {
	if m != nil {
		return m.VolumeId
	}
	return ""
}

func (m *NodeUnpublishVolumeRequest) GetTargetPath() string {
	if m != nil {
		return m.TargetPath
	}
	return ""
}

type NodeUnpublishVolumeResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NodeUnpublishVolumeResponse) Reset()         { *m = NodeUnpublishVolumeResponse{} }
func (m *NodeUnpublishVolumeResponse) String() string { return proto.CompactTextString(m) }
func (*NodeUnpublishVolumeResponse) ProtoMessage()    {}
func (*NodeUnpublishVolumeResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{45}
}
func (m *NodeUnpublishVolumeResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodeUnpublishVolumeResponse.Unmarshal(m, b)
}
func (m *NodeUnpublishVolumeResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodeUnpublishVolumeResponse.Marshal(b, m, deterministic)
}
func (dst *NodeUnpublishVolumeResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeUnpublishVolumeResponse.Merge(dst, src)
}
func (m *NodeUnpublishVolumeResponse) XXX_Size() int {
	return xxx_messageInfo_NodeUnpublishVolumeResponse.Size(m)
}
func (m *NodeUnpublishVolumeResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeUnpublishVolumeResponse.DiscardUnknown(m)
}

var xxx_messageInfo_NodeUnpublishVolumeResponse proto.InternalMessageInfo

type NodeGetIdRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NodeGetIdRequest) Reset()         { *m = NodeGetIdRequest{} }
func (m *NodeGetIdRequest) String() string { return proto.CompactTextString(m) }
func (*NodeGetIdRequest) ProtoMessage()    {}
func (*NodeGetIdRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{46}
}
func (m *NodeGetIdRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodeGetIdRequest.Unmarshal(m, b)
}
func (m *NodeGetIdRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodeGetIdRequest.Marshal(b, m, deterministic)
}
func (dst *NodeGetIdRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeGetIdRequest.Merge(dst, src)
}
func (m *NodeGetIdRequest) XXX_Size() int {
	return xxx_messageInfo_NodeGetIdRequest.Size(m)
}
func (m *NodeGetIdRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeGetIdRequest.DiscardUnknown(m)
}

var xxx_messageInfo_NodeGetIdRequest proto.InternalMessageInfo

type NodeGetIdResponse struct {
	// The ID of the node as understood by the SP which SHALL be used by
	// CO in subsequent `ControllerPublishVolume`.
	// This is a REQUIRED field.
	NodeId               string   `protobuf:"bytes,1,opt,name=node_id,json=nodeId" json:"node_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NodeGetIdResponse) Reset()         { *m = NodeGetIdResponse{} }
func (m *NodeGetIdResponse) String() string { return proto.CompactTextString(m) }
func (*NodeGetIdResponse) ProtoMessage()    {}
func (*NodeGetIdResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{47}
}
func (m *NodeGetIdResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodeGetIdResponse.Unmarshal(m, b)
}
func (m *NodeGetIdResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodeGetIdResponse.Marshal(b, m, deterministic)
}
func (dst *NodeGetIdResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeGetIdResponse.Merge(dst, src)
}
func (m *NodeGetIdResponse) XXX_Size() int {
	return xxx_messageInfo_NodeGetIdResponse.Size(m)
}
func (m *NodeGetIdResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeGetIdResponse.DiscardUnknown(m)
}

var xxx_messageInfo_NodeGetIdResponse proto.InternalMessageInfo

func (m *NodeGetIdResponse) GetNodeId() string {
	if m != nil {
		return m.NodeId
	}
	return ""
}

type NodeGetCapabilitiesRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NodeGetCapabilitiesRequest) Reset()         { *m = NodeGetCapabilitiesRequest{} }
func (m *NodeGetCapabilitiesRequest) String() string { return proto.CompactTextString(m) }
func (*NodeGetCapabilitiesRequest) ProtoMessage()    {}
func (*NodeGetCapabilitiesRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{48}
}
func (m *NodeGetCapabilitiesRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodeGetCapabilitiesRequest.Unmarshal(m, b)
}
func (m *NodeGetCapabilitiesRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodeGetCapabilitiesRequest.Marshal(b, m, deterministic)
}
func (dst *NodeGetCapabilitiesRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeGetCapabilitiesRequest.Merge(dst, src)
}
func (m *NodeGetCapabilitiesRequest) XXX_Size() int {
	return xxx_messageInfo_NodeGetCapabilitiesRequest.Size(m)
}
func (m *NodeGetCapabilitiesRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeGetCapabilitiesRequest.DiscardUnknown(m)
}

var xxx_messageInfo_NodeGetCapabilitiesRequest proto.InternalMessageInfo

type NodeGetCapabilitiesResponse struct {
	// All the capabilities that the node service supports. This field
	// is OPTIONAL.
	Capabilities         []*NodeServiceCapability `protobuf:"bytes,1,rep,name=capabilities" json:"capabilities,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                 `json:"-"`
	XXX_unrecognized     []byte                   `json:"-"`
	XXX_sizecache        int32                    `json:"-"`
}

func (m *NodeGetCapabilitiesResponse) Reset()         { *m = NodeGetCapabilitiesResponse{} }
func (m *NodeGetCapabilitiesResponse) String() string { return proto.CompactTextString(m) }
func (*NodeGetCapabilitiesResponse) ProtoMessage()    {}
func (*NodeGetCapabilitiesResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{49}
}
func (m *NodeGetCapabilitiesResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodeGetCapabilitiesResponse.Unmarshal(m, b)
}
func (m *NodeGetCapabilitiesResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodeGetCapabilitiesResponse.Marshal(b, m, deterministic)
}
func (dst *NodeGetCapabilitiesResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeGetCapabilitiesResponse.Merge(dst, src)
}
func (m *NodeGetCapabilitiesResponse) XXX_Size() int {
	return xxx_messageInfo_NodeGetCapabilitiesResponse.Size(m)
}
func (m *NodeGetCapabilitiesResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeGetCapabilitiesResponse.DiscardUnknown(m)
}

var xxx_messageInfo_NodeGetCapabilitiesResponse proto.InternalMessageInfo

func (m *NodeGetCapabilitiesResponse) GetCapabilities() []*NodeServiceCapability {
	if m != nil {
		return m.Capabilities
	}
	return nil
}

// Specifies a capability of the node service.
type NodeServiceCapability struct {
	// Types that are valid to be assigned to Type:
	//	*NodeServiceCapability_Rpc
	Type                 isNodeServiceCapability_Type `protobuf_oneof:"type"`
	XXX_NoUnkeyedLiteral struct{}                     `json:"-"`
	XXX_unrecognized     []byte                       `json:"-"`
	XXX_sizecache        int32                        `json:"-"`
}

func (m *NodeServiceCapability) Reset()         { *m = NodeServiceCapability{} }
func (m *NodeServiceCapability) String() string { return proto.CompactTextString(m) }
func (*NodeServiceCapability) ProtoMessage()    {}
func (*NodeServiceCapability) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{50}
}
func (m *NodeServiceCapability) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodeServiceCapability.Unmarshal(m, b)
}
func (m *NodeServiceCapability) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodeServiceCapability.Marshal(b, m, deterministic)
}
func (dst *NodeServiceCapability) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeServiceCapability.Merge(dst, src)
}
func (m *NodeServiceCapability) XXX_Size() int {
	return xxx_messageInfo_NodeServiceCapability.Size(m)
}
func (m *NodeServiceCapability) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeServiceCapability.DiscardUnknown(m)
}

var xxx_messageInfo_NodeServiceCapability proto.InternalMessageInfo

type isNodeServiceCapability_Type interface {
	isNodeServiceCapability_Type()
}

type NodeServiceCapability_Rpc struct {
	Rpc *NodeServiceCapability_RPC `protobuf:"bytes,1,opt,name=rpc,oneof"`
}

func (*NodeServiceCapability_Rpc) isNodeServiceCapability_Type() {}

func (m *NodeServiceCapability) GetType() isNodeServiceCapability_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (m *NodeServiceCapability) GetRpc() *NodeServiceCapability_RPC {
	if x, ok := m.GetType().(*NodeServiceCapability_Rpc); ok {
		return x.Rpc
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*NodeServiceCapability) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _NodeServiceCapability_OneofMarshaler, _NodeServiceCapability_OneofUnmarshaler, _NodeServiceCapability_OneofSizer, []interface{}{
		(*NodeServiceCapability_Rpc)(nil),
	}
}

func _NodeServiceCapability_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*NodeServiceCapability)
	// type
	switch x := m.Type.(type) {
	case *NodeServiceCapability_Rpc:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Rpc); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("NodeServiceCapability.Type has unexpected type %T", x)
	}
	return nil
}

func _NodeServiceCapability_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*NodeServiceCapability)
	switch tag {
	case 1: // type.rpc
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(NodeServiceCapability_RPC)
		err := b.DecodeMessage(msg)
		m.Type = &NodeServiceCapability_Rpc{msg}
		return true, err
	default:
		return false, nil
	}
}

func _NodeServiceCapability_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*NodeServiceCapability)
	// type
	switch x := m.Type.(type) {
	case *NodeServiceCapability_Rpc:
		s := proto.Size(x.Rpc)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type NodeServiceCapability_RPC struct {
	Type                 NodeServiceCapability_RPC_Type `protobuf:"varint,1,opt,name=type,enum=csi.v0.NodeServiceCapability_RPC_Type" json:"type,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                       `json:"-"`
	XXX_unrecognized     []byte                         `json:"-"`
	XXX_sizecache        int32                          `json:"-"`
}

func (m *NodeServiceCapability_RPC) Reset()         { *m = NodeServiceCapability_RPC{} }
func (m *NodeServiceCapability_RPC) String() string { return proto.CompactTextString(m) }
func (*NodeServiceCapability_RPC) ProtoMessage()    {}
func (*NodeServiceCapability_RPC) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{50, 0}
}
func (m *NodeServiceCapability_RPC) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodeServiceCapability_RPC.Unmarshal(m, b)
}
func (m *NodeServiceCapability_RPC) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodeServiceCapability_RPC.Marshal(b, m, deterministic)
}
func (dst *NodeServiceCapability_RPC) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeServiceCapability_RPC.Merge(dst, src)
}
func (m *NodeServiceCapability_RPC) XXX_Size() int {
	return xxx_messageInfo_NodeServiceCapability_RPC.Size(m)
}
func (m *NodeServiceCapability_RPC) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeServiceCapability_RPC.DiscardUnknown(m)
}

var xxx_messageInfo_NodeServiceCapability_RPC proto.InternalMessageInfo

func (m *NodeServiceCapability_RPC) GetType() NodeServiceCapability_RPC_Type {
	if m != nil {
		return m.Type
	}
	return NodeServiceCapability_RPC_UNKNOWN
}

type NodeGetInfoRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NodeGetInfoRequest) Reset()         { *m = NodeGetInfoRequest{} }
func (m *NodeGetInfoRequest) String() string { return proto.CompactTextString(m) }
func (*NodeGetInfoRequest) ProtoMessage()    {}
func (*NodeGetInfoRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{51}
}
func (m *NodeGetInfoRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodeGetInfoRequest.Unmarshal(m, b)
}
func (m *NodeGetInfoRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodeGetInfoRequest.Marshal(b, m, deterministic)
}
func (dst *NodeGetInfoRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeGetInfoRequest.Merge(dst, src)
}
func (m *NodeGetInfoRequest) XXX_Size() int {
	return xxx_messageInfo_NodeGetInfoRequest.Size(m)
}
func (m *NodeGetInfoRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeGetInfoRequest.DiscardUnknown(m)
}

var xxx_messageInfo_NodeGetInfoRequest proto.InternalMessageInfo

type NodeGetInfoResponse struct {
	// The ID of the node as understood by the SP which SHALL be used by
	// CO in subsequent calls to `ControllerPublishVolume`.
	// This is a REQUIRED field.
	NodeId string `protobuf:"bytes,1,opt,name=node_id,json=nodeId" json:"node_id,omitempty"`
	// Maximum number of volumes that controller can publish to the node.
	// If value is not set or zero CO SHALL decide how many volumes of
	// this type can be published by the controller to the node. The
	// plugin MUST NOT set negative values here.
	// This field is OPTIONAL.
	MaxVolumesPerNode int64 `protobuf:"varint,2,opt,name=max_volumes_per_node,json=maxVolumesPerNode" json:"max_volumes_per_node,omitempty"`
	// Specifies where (regions, zones, racks, etc.) the node is
	// accessible from.
	// A plugin that returns this field MUST also set the
	// ACCESSIBILITY_CONSTRAINTS plugin capability.
	// COs MAY use this information along with the topology information
	// returned in CreateVolumeResponse to ensure that a given volume is
	// accessible from a given node when scheduling workloads.
	// This field is OPTIONAL. If it is not specified, the CO MAY assume
	// the node is not subject to any topological constraint, and MAY
	// schedule workloads that reference any volume V, such that there are
	// no topological constraints declared for V.
	//
	// Example 1:
	//   accessible_topology =
	//     {"region": "R1", "zone": "R2"}
	// Indicates the node exists within the "region" "R1" and the "zone"
	// "Z2".
	AccessibleTopology   *Topology `protobuf:"bytes,3,opt,name=accessible_topology,json=accessibleTopology" json:"accessible_topology,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *NodeGetInfoResponse) Reset()         { *m = NodeGetInfoResponse{} }
func (m *NodeGetInfoResponse) String() string { return proto.CompactTextString(m) }
func (*NodeGetInfoResponse) ProtoMessage()    {}
func (*NodeGetInfoResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_csi_31237507707d37ec, []int{52}
}
func (m *NodeGetInfoResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NodeGetInfoResponse.Unmarshal(m, b)
}
func (m *NodeGetInfoResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NodeGetInfoResponse.Marshal(b, m, deterministic)
}
func (dst *NodeGetInfoResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeGetInfoResponse.Merge(dst, src)
}
func (m *NodeGetInfoResponse) XXX_Size() int {
	return xxx_messageInfo_NodeGetInfoResponse.Size(m)
}
func (m *NodeGetInfoResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeGetInfoResponse.DiscardUnknown(m)
}

var xxx_messageInfo_NodeGetInfoResponse proto.InternalMessageInfo

func (m *NodeGetInfoResponse) GetNodeId() string {
	if m != nil {
		return m.NodeId
	}
	return ""
}

func (m *NodeGetInfoResponse) GetMaxVolumesPerNode() int64 {
	if m != nil {
		return m.MaxVolumesPerNode
	}
	return 0
}

func (m *NodeGetInfoResponse) GetAccessibleTopology() *Topology {
	if m != nil {
		return m.AccessibleTopology
	}
	return nil
}

func init() {
	proto.RegisterType((*GetPluginInfoRequest)(nil), "csi.v0.GetPluginInfoRequest")
	proto.RegisterType((*GetPluginInfoResponse)(nil), "csi.v0.GetPluginInfoResponse")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.GetPluginInfoResponse.ManifestEntry")
	proto.RegisterType((*GetPluginCapabilitiesRequest)(nil), "csi.v0.GetPluginCapabilitiesRequest")
	proto.RegisterType((*GetPluginCapabilitiesResponse)(nil), "csi.v0.GetPluginCapabilitiesResponse")
	proto.RegisterType((*PluginCapability)(nil), "csi.v0.PluginCapability")
	proto.RegisterType((*PluginCapability_Service)(nil), "csi.v0.PluginCapability.Service")
	proto.RegisterType((*ProbeRequest)(nil), "csi.v0.ProbeRequest")
	proto.RegisterType((*ProbeResponse)(nil), "csi.v0.ProbeResponse")
	proto.RegisterType((*CreateVolumeRequest)(nil), "csi.v0.CreateVolumeRequest")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.CreateVolumeRequest.ControllerCreateSecretsEntry")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.CreateVolumeRequest.ParametersEntry")
	proto.RegisterType((*VolumeContentSource)(nil), "csi.v0.VolumeContentSource")
	proto.RegisterType((*VolumeContentSource_SnapshotSource)(nil), "csi.v0.VolumeContentSource.SnapshotSource")
	proto.RegisterType((*CreateVolumeResponse)(nil), "csi.v0.CreateVolumeResponse")
	proto.RegisterType((*VolumeCapability)(nil), "csi.v0.VolumeCapability")
	proto.RegisterType((*VolumeCapability_BlockVolume)(nil), "csi.v0.VolumeCapability.BlockVolume")
	proto.RegisterType((*VolumeCapability_MountVolume)(nil), "csi.v0.VolumeCapability.MountVolume")
	proto.RegisterType((*VolumeCapability_AccessMode)(nil), "csi.v0.VolumeCapability.AccessMode")
	proto.RegisterType((*CapacityRange)(nil), "csi.v0.CapacityRange")
	proto.RegisterType((*Volume)(nil), "csi.v0.Volume")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.Volume.AttributesEntry")
	proto.RegisterType((*TopologyRequirement)(nil), "csi.v0.TopologyRequirement")
	proto.RegisterType((*Topology)(nil), "csi.v0.Topology")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.Topology.SegmentsEntry")
	proto.RegisterType((*DeleteVolumeRequest)(nil), "csi.v0.DeleteVolumeRequest")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.DeleteVolumeRequest.ControllerDeleteSecretsEntry")
	proto.RegisterType((*DeleteVolumeResponse)(nil), "csi.v0.DeleteVolumeResponse")
	proto.RegisterType((*ControllerPublishVolumeRequest)(nil), "csi.v0.ControllerPublishVolumeRequest")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.ControllerPublishVolumeRequest.ControllerPublishSecretsEntry")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.ControllerPublishVolumeRequest.VolumeAttributesEntry")
	proto.RegisterType((*ControllerPublishVolumeResponse)(nil), "csi.v0.ControllerPublishVolumeResponse")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.ControllerPublishVolumeResponse.PublishInfoEntry")
	proto.RegisterType((*ControllerUnpublishVolumeRequest)(nil), "csi.v0.ControllerUnpublishVolumeRequest")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.ControllerUnpublishVolumeRequest.ControllerUnpublishSecretsEntry")
	proto.RegisterType((*ControllerUnpublishVolumeResponse)(nil), "csi.v0.ControllerUnpublishVolumeResponse")
	proto.RegisterType((*ValidateVolumeCapabilitiesRequest)(nil), "csi.v0.ValidateVolumeCapabilitiesRequest")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.ValidateVolumeCapabilitiesRequest.VolumeAttributesEntry")
	proto.RegisterType((*ValidateVolumeCapabilitiesResponse)(nil), "csi.v0.ValidateVolumeCapabilitiesResponse")
	proto.RegisterType((*ListVolumesRequest)(nil), "csi.v0.ListVolumesRequest")
	proto.RegisterType((*ListVolumesResponse)(nil), "csi.v0.ListVolumesResponse")
	proto.RegisterType((*ListVolumesResponse_Entry)(nil), "csi.v0.ListVolumesResponse.Entry")
	proto.RegisterType((*GetCapacityRequest)(nil), "csi.v0.GetCapacityRequest")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.GetCapacityRequest.ParametersEntry")
	proto.RegisterType((*GetCapacityResponse)(nil), "csi.v0.GetCapacityResponse")
	proto.RegisterType((*ControllerGetCapabilitiesRequest)(nil), "csi.v0.ControllerGetCapabilitiesRequest")
	proto.RegisterType((*ControllerGetCapabilitiesResponse)(nil), "csi.v0.ControllerGetCapabilitiesResponse")
	proto.RegisterType((*ControllerServiceCapability)(nil), "csi.v0.ControllerServiceCapability")
	proto.RegisterType((*ControllerServiceCapability_RPC)(nil), "csi.v0.ControllerServiceCapability.RPC")
	proto.RegisterType((*CreateSnapshotRequest)(nil), "csi.v0.CreateSnapshotRequest")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.CreateSnapshotRequest.CreateSnapshotSecretsEntry")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.CreateSnapshotRequest.ParametersEntry")
	proto.RegisterType((*CreateSnapshotResponse)(nil), "csi.v0.CreateSnapshotResponse")
	proto.RegisterType((*Snapshot)(nil), "csi.v0.Snapshot")
	proto.RegisterType((*SnapshotStatus)(nil), "csi.v0.SnapshotStatus")
	proto.RegisterType((*DeleteSnapshotRequest)(nil), "csi.v0.DeleteSnapshotRequest")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.DeleteSnapshotRequest.DeleteSnapshotSecretsEntry")
	proto.RegisterType((*DeleteSnapshotResponse)(nil), "csi.v0.DeleteSnapshotResponse")
	proto.RegisterType((*ListSnapshotsRequest)(nil), "csi.v0.ListSnapshotsRequest")
	proto.RegisterType((*ListSnapshotsResponse)(nil), "csi.v0.ListSnapshotsResponse")
	proto.RegisterType((*ListSnapshotsResponse_Entry)(nil), "csi.v0.ListSnapshotsResponse.Entry")
	proto.RegisterType((*NodeStageVolumeRequest)(nil), "csi.v0.NodeStageVolumeRequest")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.NodeStageVolumeRequest.NodeStageSecretsEntry")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.NodeStageVolumeRequest.PublishInfoEntry")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.NodeStageVolumeRequest.VolumeAttributesEntry")
	proto.RegisterType((*NodeStageVolumeResponse)(nil), "csi.v0.NodeStageVolumeResponse")
	proto.RegisterType((*NodeUnstageVolumeRequest)(nil), "csi.v0.NodeUnstageVolumeRequest")
	proto.RegisterType((*NodeUnstageVolumeResponse)(nil), "csi.v0.NodeUnstageVolumeResponse")
	proto.RegisterType((*NodePublishVolumeRequest)(nil), "csi.v0.NodePublishVolumeRequest")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.NodePublishVolumeRequest.NodePublishSecretsEntry")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.NodePublishVolumeRequest.PublishInfoEntry")
	proto.RegisterMapType((map[string]string)(nil), "csi.v0.NodePublishVolumeRequest.VolumeAttributesEntry")
	proto.RegisterType((*NodePublishVolumeResponse)(nil), "csi.v0.NodePublishVolumeResponse")
	proto.RegisterType((*NodeUnpublishVolumeRequest)(nil), "csi.v0.NodeUnpublishVolumeRequest")
	proto.RegisterType((*NodeUnpublishVolumeResponse)(nil), "csi.v0.NodeUnpublishVolumeResponse")
	proto.RegisterType((*NodeGetIdRequest)(nil), "csi.v0.NodeGetIdRequest")
	proto.RegisterType((*NodeGetIdResponse)(nil), "csi.v0.NodeGetIdResponse")
	proto.RegisterType((*NodeGetCapabilitiesRequest)(nil), "csi.v0.NodeGetCapabilitiesRequest")
	proto.RegisterType((*NodeGetCapabilitiesResponse)(nil), "csi.v0.NodeGetCapabilitiesResponse")
	proto.RegisterType((*NodeServiceCapability)(nil), "csi.v0.NodeServiceCapability")
	proto.RegisterType((*NodeServiceCapability_RPC)(nil), "csi.v0.NodeServiceCapability.RPC")
	proto.RegisterType((*NodeGetInfoRequest)(nil), "csi.v0.NodeGetInfoRequest")
	proto.RegisterType((*NodeGetInfoResponse)(nil), "csi.v0.NodeGetInfoResponse")
	proto.RegisterEnum("csi.v0.PluginCapability_Service_Type", PluginCapability_Service_Type_name, PluginCapability_Service_Type_value)
	proto.RegisterEnum("csi.v0.VolumeCapability_AccessMode_Mode", VolumeCapability_AccessMode_Mode_name, VolumeCapability_AccessMode_Mode_value)
	proto.RegisterEnum("csi.v0.ControllerServiceCapability_RPC_Type", ControllerServiceCapability_RPC_Type_name, ControllerServiceCapability_RPC_Type_value)
	proto.RegisterEnum("csi.v0.SnapshotStatus_Type", SnapshotStatus_Type_name, SnapshotStatus_Type_value)
	proto.RegisterEnum("csi.v0.NodeServiceCapability_RPC_Type", NodeServiceCapability_RPC_Type_name, NodeServiceCapability_RPC_Type_value)
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Identity service

type IdentityClient interface {
	GetPluginInfo(ctx context.Context, in *GetPluginInfoRequest, opts ...grpc.CallOption) (*GetPluginInfoResponse, error)
	GetPluginCapabilities(ctx context.Context, in *GetPluginCapabilitiesRequest, opts ...grpc.CallOption) (*GetPluginCapabilitiesResponse, error)
	Probe(ctx context.Context, in *ProbeRequest, opts ...grpc.CallOption) (*ProbeResponse, error)
}

type identityClient struct {
	cc *grpc.ClientConn
}

func NewIdentityClient(cc *grpc.ClientConn) IdentityClient {
	return &identityClient{cc}
}

func (c *identityClient) GetPluginInfo(ctx context.Context, in *GetPluginInfoRequest, opts ...grpc.CallOption) (*GetPluginInfoResponse, error) {
	out := new(GetPluginInfoResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Identity/GetPluginInfo", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *identityClient) GetPluginCapabilities(ctx context.Context, in *GetPluginCapabilitiesRequest, opts ...grpc.CallOption) (*GetPluginCapabilitiesResponse, error) {
	out := new(GetPluginCapabilitiesResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Identity/GetPluginCapabilities", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *identityClient) Probe(ctx context.Context, in *ProbeRequest, opts ...grpc.CallOption) (*ProbeResponse, error) {
	out := new(ProbeResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Identity/Probe", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Identity service

type IdentityServer interface {
	GetPluginInfo(context.Context, *GetPluginInfoRequest) (*GetPluginInfoResponse, error)
	GetPluginCapabilities(context.Context, *GetPluginCapabilitiesRequest) (*GetPluginCapabilitiesResponse, error)
	Probe(context.Context, *ProbeRequest) (*ProbeResponse, error)
}

func RegisterIdentityServer(s *grpc.Server, srv IdentityServer) {
	s.RegisterService(&_Identity_serviceDesc, srv)
}

func _Identity_GetPluginInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetPluginInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IdentityServer).GetPluginInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Identity/GetPluginInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IdentityServer).GetPluginInfo(ctx, req.(*GetPluginInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Identity_GetPluginCapabilities_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetPluginCapabilitiesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IdentityServer).GetPluginCapabilities(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Identity/GetPluginCapabilities",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IdentityServer).GetPluginCapabilities(ctx, req.(*GetPluginCapabilitiesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Identity_Probe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProbeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IdentityServer).Probe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Identity/Probe",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IdentityServer).Probe(ctx, req.(*ProbeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Identity_serviceDesc = grpc.ServiceDesc{
	ServiceName: "csi.v0.Identity",
	HandlerType: (*IdentityServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetPluginInfo",
			Handler:    _Identity_GetPluginInfo_Handler,
		},
		{
			MethodName: "GetPluginCapabilities",
			Handler:    _Identity_GetPluginCapabilities_Handler,
		},
		{
			MethodName: "Probe",
			Handler:    _Identity_Probe_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "csi.proto",
}

// Client API for Controller service

type ControllerClient interface {
	CreateVolume(ctx context.Context, in *CreateVolumeRequest, opts ...grpc.CallOption) (*CreateVolumeResponse, error)
	DeleteVolume(ctx context.Context, in *DeleteVolumeRequest, opts ...grpc.CallOption) (*DeleteVolumeResponse, error)
	ControllerPublishVolume(ctx context.Context, in *ControllerPublishVolumeRequest, opts ...grpc.CallOption) (*ControllerPublishVolumeResponse, error)
	ControllerUnpublishVolume(ctx context.Context, in *ControllerUnpublishVolumeRequest, opts ...grpc.CallOption) (*ControllerUnpublishVolumeResponse, error)
	ValidateVolumeCapabilities(ctx context.Context, in *ValidateVolumeCapabilitiesRequest, opts ...grpc.CallOption) (*ValidateVolumeCapabilitiesResponse, error)
	ListVolumes(ctx context.Context, in *ListVolumesRequest, opts ...grpc.CallOption) (*ListVolumesResponse, error)
	GetCapacity(ctx context.Context, in *GetCapacityRequest, opts ...grpc.CallOption) (*GetCapacityResponse, error)
	ControllerGetCapabilities(ctx context.Context, in *ControllerGetCapabilitiesRequest, opts ...grpc.CallOption) (*ControllerGetCapabilitiesResponse, error)
	CreateSnapshot(ctx context.Context, in *CreateSnapshotRequest, opts ...grpc.CallOption) (*CreateSnapshotResponse, error)
	DeleteSnapshot(ctx context.Context, in *DeleteSnapshotRequest, opts ...grpc.CallOption) (*DeleteSnapshotResponse, error)
	ListSnapshots(ctx context.Context, in *ListSnapshotsRequest, opts ...grpc.CallOption) (*ListSnapshotsResponse, error)
}

type controllerClient struct {
	cc *grpc.ClientConn
}

func NewControllerClient(cc *grpc.ClientConn) ControllerClient {
	return &controllerClient{cc}
}

func (c *controllerClient) CreateVolume(ctx context.Context, in *CreateVolumeRequest, opts ...grpc.CallOption) (*CreateVolumeResponse, error) {
	out := new(CreateVolumeResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Controller/CreateVolume", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) DeleteVolume(ctx context.Context, in *DeleteVolumeRequest, opts ...grpc.CallOption) (*DeleteVolumeResponse, error) {
	out := new(DeleteVolumeResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Controller/DeleteVolume", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) ControllerPublishVolume(ctx context.Context, in *ControllerPublishVolumeRequest, opts ...grpc.CallOption) (*ControllerPublishVolumeResponse, error) {
	out := new(ControllerPublishVolumeResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Controller/ControllerPublishVolume", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) ControllerUnpublishVolume(ctx context.Context, in *ControllerUnpublishVolumeRequest, opts ...grpc.CallOption) (*ControllerUnpublishVolumeResponse, error) {
	out := new(ControllerUnpublishVolumeResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Controller/ControllerUnpublishVolume", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) ValidateVolumeCapabilities(ctx context.Context, in *ValidateVolumeCapabilitiesRequest, opts ...grpc.CallOption) (*ValidateVolumeCapabilitiesResponse, error) {
	out := new(ValidateVolumeCapabilitiesResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Controller/ValidateVolumeCapabilities", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) ListVolumes(ctx context.Context, in *ListVolumesRequest, opts ...grpc.CallOption) (*ListVolumesResponse, error) {
	out := new(ListVolumesResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Controller/ListVolumes", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) GetCapacity(ctx context.Context, in *GetCapacityRequest, opts ...grpc.CallOption) (*GetCapacityResponse, error) {
	out := new(GetCapacityResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Controller/GetCapacity", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) ControllerGetCapabilities(ctx context.Context, in *ControllerGetCapabilitiesRequest, opts ...grpc.CallOption) (*ControllerGetCapabilitiesResponse, error) {
	out := new(ControllerGetCapabilitiesResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Controller/ControllerGetCapabilities", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) CreateSnapshot(ctx context.Context, in *CreateSnapshotRequest, opts ...grpc.CallOption) (*CreateSnapshotResponse, error) {
	out := new(CreateSnapshotResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Controller/CreateSnapshot", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) DeleteSnapshot(ctx context.Context, in *DeleteSnapshotRequest, opts ...grpc.CallOption) (*DeleteSnapshotResponse, error) {
	out := new(DeleteSnapshotResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Controller/DeleteSnapshot", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) ListSnapshots(ctx context.Context, in *ListSnapshotsRequest, opts ...grpc.CallOption) (*ListSnapshotsResponse, error) {
	out := new(ListSnapshotsResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Controller/ListSnapshots", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Controller service

type ControllerServer interface {
	CreateVolume(context.Context, *CreateVolumeRequest) (*CreateVolumeResponse, error)
	DeleteVolume(context.Context, *DeleteVolumeRequest) (*DeleteVolumeResponse, error)
	ControllerPublishVolume(context.Context, *ControllerPublishVolumeRequest) (*ControllerPublishVolumeResponse, error)
	ControllerUnpublishVolume(context.Context, *ControllerUnpublishVolumeRequest) (*ControllerUnpublishVolumeResponse, error)
	ValidateVolumeCapabilities(context.Context, *ValidateVolumeCapabilitiesRequest) (*ValidateVolumeCapabilitiesResponse, error)
	ListVolumes(context.Context, *ListVolumesRequest) (*ListVolumesResponse, error)
	GetCapacity(context.Context, *GetCapacityRequest) (*GetCapacityResponse, error)
	ControllerGetCapabilities(context.Context, *ControllerGetCapabilitiesRequest) (*ControllerGetCapabilitiesResponse, error)
	CreateSnapshot(context.Context, *CreateSnapshotRequest) (*CreateSnapshotResponse, error)
	DeleteSnapshot(context.Context, *DeleteSnapshotRequest) (*DeleteSnapshotResponse, error)
	ListSnapshots(context.Context, *ListSnapshotsRequest) (*ListSnapshotsResponse, error)
}

func RegisterControllerServer(s *grpc.Server, srv ControllerServer) {
	s.RegisterService(&_Controller_serviceDesc, srv)
}

func _Controller_CreateVolume_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateVolumeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).CreateVolume(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Controller/CreateVolume",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).CreateVolume(ctx, req.(*CreateVolumeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_DeleteVolume_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteVolumeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).DeleteVolume(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Controller/DeleteVolume",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).DeleteVolume(ctx, req.(*DeleteVolumeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_ControllerPublishVolume_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ControllerPublishVolumeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).ControllerPublishVolume(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Controller/ControllerPublishVolume",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).ControllerPublishVolume(ctx, req.(*ControllerPublishVolumeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_ControllerUnpublishVolume_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ControllerUnpublishVolumeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).ControllerUnpublishVolume(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Controller/ControllerUnpublishVolume",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).ControllerUnpublishVolume(ctx, req.(*ControllerUnpublishVolumeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_ValidateVolumeCapabilities_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ValidateVolumeCapabilitiesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).ValidateVolumeCapabilities(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Controller/ValidateVolumeCapabilities",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).ValidateVolumeCapabilities(ctx, req.(*ValidateVolumeCapabilitiesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_ListVolumes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListVolumesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).ListVolumes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Controller/ListVolumes",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).ListVolumes(ctx, req.(*ListVolumesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_GetCapacity_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetCapacityRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).GetCapacity(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Controller/GetCapacity",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).GetCapacity(ctx, req.(*GetCapacityRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_ControllerGetCapabilities_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ControllerGetCapabilitiesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).ControllerGetCapabilities(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Controller/ControllerGetCapabilities",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).ControllerGetCapabilities(ctx, req.(*ControllerGetCapabilitiesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_CreateSnapshot_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateSnapshotRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).CreateSnapshot(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Controller/CreateSnapshot",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).CreateSnapshot(ctx, req.(*CreateSnapshotRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_DeleteSnapshot_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteSnapshotRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).DeleteSnapshot(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Controller/DeleteSnapshot",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).DeleteSnapshot(ctx, req.(*DeleteSnapshotRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_ListSnapshots_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListSnapshotsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).ListSnapshots(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Controller/ListSnapshots",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).ListSnapshots(ctx, req.(*ListSnapshotsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Controller_serviceDesc = grpc.ServiceDesc{
	ServiceName: "csi.v0.Controller",
	HandlerType: (*ControllerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateVolume",
			Handler:    _Controller_CreateVolume_Handler,
		},
		{
			MethodName: "DeleteVolume",
			Handler:    _Controller_DeleteVolume_Handler,
		},
		{
			MethodName: "ControllerPublishVolume",
			Handler:    _Controller_ControllerPublishVolume_Handler,
		},
		{
			MethodName: "ControllerUnpublishVolume",
			Handler:    _Controller_ControllerUnpublishVolume_Handler,
		},
		{
			MethodName: "ValidateVolumeCapabilities",
			Handler:    _Controller_ValidateVolumeCapabilities_Handler,
		},
		{
			MethodName: "ListVolumes",
			Handler:    _Controller_ListVolumes_Handler,
		},
		{
			MethodName: "GetCapacity",
			Handler:    _Controller_GetCapacity_Handler,
		},
		{
			MethodName: "ControllerGetCapabilities",
			Handler:    _Controller_ControllerGetCapabilities_Handler,
		},
		{
			MethodName: "CreateSnapshot",
			Handler:    _Controller_CreateSnapshot_Handler,
		},
		{
			MethodName: "DeleteSnapshot",
			Handler:    _Controller_DeleteSnapshot_Handler,
		},
		{
			MethodName: "ListSnapshots",
			Handler:    _Controller_ListSnapshots_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "csi.proto",
}

// Client API for Node service

type NodeClient interface {
	NodeStageVolume(ctx context.Context, in *NodeStageVolumeRequest, opts ...grpc.CallOption) (*NodeStageVolumeResponse, error)
	NodeUnstageVolume(ctx context.Context, in *NodeUnstageVolumeRequest, opts ...grpc.CallOption) (*NodeUnstageVolumeResponse, error)
	NodePublishVolume(ctx context.Context, in *NodePublishVolumeRequest, opts ...grpc.CallOption) (*NodePublishVolumeResponse, error)
	NodeUnpublishVolume(ctx context.Context, in *NodeUnpublishVolumeRequest, opts ...grpc.CallOption) (*NodeUnpublishVolumeResponse, error)
	// NodeGetId is being deprecated in favor of NodeGetInfo and will be
	// removed in CSI 1.0. Existing drivers, however, may depend on this
	// RPC call and hence this RPC call MUST be implemented by the CSI
	// plugin prior to v1.0.
	NodeGetId(ctx context.Context, in *NodeGetIdRequest, opts ...grpc.CallOption) (*NodeGetIdResponse, error)
	NodeGetCapabilities(ctx context.Context, in *NodeGetCapabilitiesRequest, opts ...grpc.CallOption) (*NodeGetCapabilitiesResponse, error)
	// Prior to CSI 1.0 - CSI plugins MUST implement both NodeGetId and
	// NodeGetInfo RPC calls.
	NodeGetInfo(ctx context.Context, in *NodeGetInfoRequest, opts ...grpc.CallOption) (*NodeGetInfoResponse, error)
}

type nodeClient struct {
	cc *grpc.ClientConn
}

func NewNodeClient(cc *grpc.ClientConn) NodeClient {
	return &nodeClient{cc}
}

func (c *nodeClient) NodeStageVolume(ctx context.Context, in *NodeStageVolumeRequest, opts ...grpc.CallOption) (*NodeStageVolumeResponse, error) {
	out := new(NodeStageVolumeResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Node/NodeStageVolume", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeClient) NodeUnstageVolume(ctx context.Context, in *NodeUnstageVolumeRequest, opts ...grpc.CallOption) (*NodeUnstageVolumeResponse, error) {
	out := new(NodeUnstageVolumeResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Node/NodeUnstageVolume", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeClient) NodePublishVolume(ctx context.Context, in *NodePublishVolumeRequest, opts ...grpc.CallOption) (*NodePublishVolumeResponse, error) {
	out := new(NodePublishVolumeResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Node/NodePublishVolume", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeClient) NodeUnpublishVolume(ctx context.Context, in *NodeUnpublishVolumeRequest, opts ...grpc.CallOption) (*NodeUnpublishVolumeResponse, error) {
	out := new(NodeUnpublishVolumeResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Node/NodeUnpublishVolume", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Deprecated: Do not use.
func (c *nodeClient) NodeGetId(ctx context.Context, in *NodeGetIdRequest, opts ...grpc.CallOption) (*NodeGetIdResponse, error) {
	out := new(NodeGetIdResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Node/NodeGetId", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeClient) NodeGetCapabilities(ctx context.Context, in *NodeGetCapabilitiesRequest, opts ...grpc.CallOption) (*NodeGetCapabilitiesResponse, error) {
	out := new(NodeGetCapabilitiesResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Node/NodeGetCapabilities", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeClient) NodeGetInfo(ctx context.Context, in *NodeGetInfoRequest, opts ...grpc.CallOption) (*NodeGetInfoResponse, error) {
	out := new(NodeGetInfoResponse)
	err := grpc.Invoke(ctx, "/csi.v0.Node/NodeGetInfo", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Node service

type NodeServer interface {
	NodeStageVolume(context.Context, *NodeStageVolumeRequest) (*NodeStageVolumeResponse, error)
	NodeUnstageVolume(context.Context, *NodeUnstageVolumeRequest) (*NodeUnstageVolumeResponse, error)
	NodePublishVolume(context.Context, *NodePublishVolumeRequest) (*NodePublishVolumeResponse, error)
	NodeUnpublishVolume(context.Context, *NodeUnpublishVolumeRequest) (*NodeUnpublishVolumeResponse, error)
	// NodeGetId is being deprecated in favor of NodeGetInfo and will be
	// removed in CSI 1.0. Existing drivers, however, may depend on this
	// RPC call and hence this RPC call MUST be implemented by the CSI
	// plugin prior to v1.0.
	NodeGetId(context.Context, *NodeGetIdRequest) (*NodeGetIdResponse, error)
	NodeGetCapabilities(context.Context, *NodeGetCapabilitiesRequest) (*NodeGetCapabilitiesResponse, error)
	// Prior to CSI 1.0 - CSI plugins MUST implement both NodeGetId and
	// NodeGetInfo RPC calls.
	NodeGetInfo(context.Context, *NodeGetInfoRequest) (*NodeGetInfoResponse, error)
}

func RegisterNodeServer(s *grpc.Server, srv NodeServer) {
	s.RegisterService(&_Node_serviceDesc, srv)
}

func _Node_NodeStageVolume_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeStageVolumeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServer).NodeStageVolume(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Node/NodeStageVolume",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServer).NodeStageVolume(ctx, req.(*NodeStageVolumeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Node_NodeUnstageVolume_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeUnstageVolumeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServer).NodeUnstageVolume(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Node/NodeUnstageVolume",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServer).NodeUnstageVolume(ctx, req.(*NodeUnstageVolumeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Node_NodePublishVolume_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodePublishVolumeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServer).NodePublishVolume(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Node/NodePublishVolume",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServer).NodePublishVolume(ctx, req.(*NodePublishVolumeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Node_NodeUnpublishVolume_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeUnpublishVolumeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServer).NodeUnpublishVolume(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Node/NodeUnpublishVolume",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServer).NodeUnpublishVolume(ctx, req.(*NodeUnpublishVolumeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Node_NodeGetId_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeGetIdRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServer).NodeGetId(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Node/NodeGetId",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServer).NodeGetId(ctx, req.(*NodeGetIdRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Node_NodeGetCapabilities_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeGetCapabilitiesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServer).NodeGetCapabilities(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Node/NodeGetCapabilities",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServer).NodeGetCapabilities(ctx, req.(*NodeGetCapabilitiesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Node_NodeGetInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeGetInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServer).NodeGetInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/csi.v0.Node/NodeGetInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServer).NodeGetInfo(ctx, req.(*NodeGetInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Node_serviceDesc = grpc.ServiceDesc{
	ServiceName: "csi.v0.Node",
	HandlerType: (*NodeServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "NodeStageVolume",
			Handler:    _Node_NodeStageVolume_Handler,
		},
		{
			MethodName: "NodeUnstageVolume",
			Handler:    _Node_NodeUnstageVolume_Handler,
		},
		{
			MethodName: "NodePublishVolume",
			Handler:    _Node_NodePublishVolume_Handler,
		},
		{
			MethodName: "NodeUnpublishVolume",
			Handler:    _Node_NodeUnpublishVolume_Handler,
		},
		{
			MethodName: "NodeGetId",
			Handler:    _Node_NodeGetId_Handler,
		},
		{
			MethodName: "NodeGetCapabilities",
			Handler:    _Node_NodeGetCapabilities_Handler,
		},
		{
			MethodName: "NodeGetInfo",
			Handler:    _Node_NodeGetInfo_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "csi.proto",
}

func init() { proto.RegisterFile("csi.proto", fileDescriptor_csi_31237507707d37ec) }

var fileDescriptor_csi_31237507707d37ec = []byte{
	// 2932 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xcc, 0x1a, 0x4d, 0x73, 0x23, 0x47,
	0xd5, 0xa3, 0x0f, 0xdb, 0x7a, 0x5e, 0x3b, 0xda, 0xf6, 0x97, 0x3c, 0xb6, 0x77, 0xbd, 0xb3, 0xd9,
	0x64, 0x13, 0x12, 0x6d, 0x30, 0x24, 0x15, 0x92, 0x4d, 0x40, 0x96, 0x15, 0x5b, 0x59, 0x5b, 0x36,
	0x23, 0xd9, 0xa9, 0x5d, 0x42, 0x4d, 0xc6, 0x52, 0x5b, 0x3b, 0xac, 0x3c, 0xa3, 0xcc, 0x8c, 0xcc,
	0x9a, 0x1b, 0x70, 0x01, 0x4e, 0xf0, 0x0b, 0x52, 0x95, 0x1b, 0x14, 0xb9, 0x50, 0xdc, 0xa8, 0xe2,
	0x46, 0x15, 0x27, 0xce, 0x9c, 0xb8, 0xa7, 0xe0, 0xc8, 0x89, 0x2a, 0xaa, 0xa8, 0x9e, 0xee, 0x19,
	0x4d, 0xb7, 0x7a, 0xf4, 0x91, 0xdd, 0x4a, 0x71, 0x92, 0xe6, 0x7d, 0xf5, 0xeb, 0xd7, 0xef, 0xbd,
	0x7e, 0xef, 0xcd, 0x40, 0xae, 0xe9, 0x59, 0xc5, 0xae, 0xeb, 0xf8, 0x0e, 0x9a, 0x26, 0x7f, 0x2f,
	0xdf, 0x50, 0x6f, 0xb4, 0x1d, 0xa7, 0xdd, 0xc1, 0xf7, 0x02, 0xe8, 0x59, 0xef, 0xfc, 0xde, 0x8f,
	0x5d, 0xb3, 0xdb, 0xc5, 0xae, 0x47, 0xe9, 0xb4, 0x15, 0x58, 0xda, 0xc3, 0xfe, 0x71, 0xa7, 0xd7,
	0xb6, 0xec, 0xaa, 0x7d, 0xee, 0xe8, 0xf8, 0xd3, 0x1e, 0xf6, 0x7c, 0xed, 0xef, 0x0a, 0x2c, 0x0b,
	0x08, 0xaf, 0xeb, 0xd8, 0x1e, 0x46, 0x08, 0x32, 0xb6, 0x79, 0x81, 0x0b, 0xca, 0x96, 0x72, 0x37,
	0xa7, 0x07, 0xff, 0xd1, 0x1d, 0x58, 0xb8, 0xc4, 0x76, 0xcb, 0x71, 0x8d, 0x4b, 0xec, 0x7a, 0x96,
	0x63, 0x17, 0x52, 0x01, 0x76, 0x9e, 0x42, 0x4f, 0x29, 0x10, 0xed, 0xc1, 0xec, 0x85, 0x69, 0x5b,
	0xe7, 0xd8, 0xf3, 0x0b, 0xe9, 0xad, 0xf4, 0xdd, 0xb9, 0xed, 0x6f, 0x14, 0xa9, 0x9e, 0x45, 0xe9,
	0x5a, 0xc5, 0x43, 0x46, 0x5d, 0xb1, 0x7d, 0xf7, 0x4a, 0x8f, 0x98, 0xd5, 0x77, 0x61, 0x9e, 0x43,
	0xa1, 0x3c, 0xa4, 0x9f, 0xe0, 0x2b, 0xa6, 0x13, 0xf9, 0x8b, 0x96, 0x20, 0x7b, 0x69, 0x76, 0x7a,
	0x98, 0x69, 0x42, 0x1f, 0xde, 0x49, 0xbd, 0xad, 0x68, 0x37, 0x60, 0x23, 0x5a, 0xad, 0x6c, 0x76,
	0xcd, 0x33, 0xab, 0x63, 0xf9, 0x16, 0xf6, 0xc2, 0xad, 0xff, 0x10, 0x36, 0x13, 0xf0, 0xcc, 0x02,
	0xf7, 0xe1, 0x5a, 0x33, 0x06, 0x2f, 0xa4, 0x82, 0xad, 0x14, 0xc2, 0xad, 0x08, 0x9c, 0x57, 0x3a,
	0x47, 0xad, 0xfd, 0x53, 0x81, 0xbc, 0x48, 0x82, 0xee, 0xc3, 0x8c, 0x87, 0xdd, 0x4b, 0xab, 0x49,
	0xed, 0x3a, 0xb7, 0xbd, 0x95, 0x24, 0xad, 0x58, 0xa7, 0x74, 0xfb, 0x53, 0x7a, 0xc8, 0xa2, 0xfe,
	0x5a, 0x81, 0x19, 0x06, 0x46, 0xdf, 0x81, 0x8c, 0x7f, 0xd5, 0xa5, 0x62, 0x16, 0xb6, 0xef, 0x8c,
	0x12, 0x53, 0x6c, 0x5c, 0x75, 0xb1, 0x1e, 0xb0, 0x68, 0x1f, 0x42, 0x86, 0x3c, 0xa1, 0x39, 0x98,
	0x39, 0xa9, 0x3d, 0xa8, 0x1d, 0x7d, 0x54, 0xcb, 0x4f, 0xa1, 0x15, 0x40, 0xe5, 0xa3, 0x5a, 0x43,
	0x3f, 0x3a, 0x38, 0xa8, 0xe8, 0x46, 0xbd, 0xa2, 0x9f, 0x56, 0xcb, 0x95, 0xbc, 0x82, 0x36, 0x61,
	0xad, 0x54, 0x2e, 0x57, 0xea, 0xf5, 0xea, 0x4e, 0xf5, 0xa0, 0xda, 0x78, 0x68, 0x94, 0x8f, 0x6a,
	0xf5, 0x86, 0x5e, 0xaa, 0xd6, 0x1a, 0xf5, 0x7c, 0x6a, 0x67, 0x9a, 0xaa, 0xa1, 0x2d, 0xc0, 0xb5,
	0x63, 0xd7, 0x39, 0xc3, 0xa1, 0x71, 0x4b, 0x30, 0xcf, 0x9e, 0x99, 0x31, 0xdf, 0x80, 0xac, 0x8b,
	0xcd, 0xd6, 0x15, 0xdb, 0xb7, 0x5a, 0xa4, 0x0e, 0x5b, 0x0c, 0x1d, 0xb6, 0xb8, 0xe3, 0x38, 0x9d,
	0x53, 0x72, 0x78, 0x3a, 0x25, 0xd4, 0xbe, 0xc8, 0xc2, 0x62, 0xd9, 0xc5, 0xa6, 0x8f, 0x4f, 0x9d,
	0x4e, 0xef, 0x22, 0x14, 0x2d, 0x75, 0xcc, 0xfb, 0xb0, 0x40, 0x8c, 0xdf, 0xb4, 0xfc, 0x2b, 0xc3,
	0x35, 0xed, 0x36, 0x75, 0x87, 0xb9, 0xed, 0xe5, 0xd0, 0x2e, 0x65, 0x86, 0xd5, 0x09, 0x52, 0x9f,
	0x6f, 0xc6, 0x1f, 0x51, 0x15, 0x16, 0x2f, 0x83, 0x25, 0x0c, 0xee, 0xbc, 0xd3, 0xfc, 0x79, 0x53,
	0x2d, 0x62, 0xe7, 0x8d, 0x2e, 0x79, 0x88, 0x85, 0x3d, 0xf4, 0x00, 0xa0, 0x6b, 0xba, 0xe6, 0x05,
	0xf6, 0xb1, 0xeb, 0x15, 0x32, 0xbc, 0xf3, 0x4b, 0x76, 0x53, 0x3c, 0x8e, 0xa8, 0xa9, 0xf3, 0xc7,
	0xd8, 0x91, 0x0f, 0x6b, 0x4d, 0xc7, 0xf6, 0x5d, 0xa7, 0xd3, 0xc1, 0xae, 0xd1, 0x0c, 0xb8, 0x0d,
	0x0f, 0x37, 0x5d, 0xec, 0x7b, 0x85, 0x6c, 0x20, 0xfb, 0xed, 0x61, 0xb2, 0xcb, 0x11, 0x33, 0xc5,
	0xd6, 0x29, 0x2b, 0x5d, 0x68, 0xb5, 0x29, 0xc7, 0xa2, 0x23, 0x58, 0x0e, 0xad, 0xe1, 0xd8, 0x3e,
	0xb6, 0x7d, 0xc3, 0x73, 0x7a, 0x6e, 0x13, 0x17, 0xa6, 0x03, 0x93, 0xae, 0x0b, 0xf6, 0xa0, 0x34,
	0xf5, 0x80, 0x44, 0x67, 0x76, 0xe4, 0x80, 0xe8, 0x11, 0xa8, 0x66, 0xb3, 0x89, 0x3d, 0xcf, 0xa2,
	0x86, 0x33, 0x5c, 0xfc, 0x69, 0xcf, 0x72, 0xf1, 0x05, 0xb6, 0x7d, 0xaf, 0x30, 0xc3, 0x4b, 0x6d,
	0x38, 0x5d, 0xa7, 0xe3, 0xb4, 0xaf, 0xf4, 0x3e, 0x8d, 0xbe, 0xc6, 0xb1, 0xc7, 0x30, 0x9e, 0xfa,
	0x1e, 0xbc, 0x20, 0x58, 0x70, 0x92, 0x1c, 0xa1, 0x7e, 0x08, 0x1b, 0xc3, 0x8c, 0x34, 0x51, 0xbe,
	0xf9, 0xa5, 0x02, 0x8b, 0x12, 0x9b, 0xa0, 0x7d, 0x98, 0xf5, 0x6c, 0xb3, 0xeb, 0x3d, 0x76, 0x7c,
	0xe6, 0xfc, 0xaf, 0x0e, 0x31, 0x61, 0xb1, 0xce, 0x68, 0xe9, 0xe3, 0xfe, 0x94, 0x1e, 0x71, 0xab,
	0x5b, 0xb0, 0xc0, 0x63, 0xd1, 0x02, 0xa4, 0xac, 0x16, 0x53, 0x2f, 0x65, 0xb5, 0xa2, 0x70, 0x7c,
	0x1f, 0x96, 0x78, 0x87, 0x60, 0x51, 0xf8, 0x12, 0x4c, 0xd3, 0x13, 0x62, 0x9a, 0x2c, 0xf0, 0x9a,
	0xe8, 0x0c, 0xab, 0xfd, 0x2e, 0x03, 0x79, 0xd1, 0xdf, 0xd1, 0x7d, 0xc8, 0x9e, 0x75, 0x9c, 0xe6,
	0x13, 0xc6, 0xfb, 0x62, 0x52, 0x60, 0x14, 0x77, 0x08, 0x15, 0x85, 0xee, 0x4f, 0xe9, 0x94, 0x89,
	0x70, 0x5f, 0x38, 0x3d, 0xdb, 0x67, 0x91, 0x99, 0xcc, 0x7d, 0x48, 0xa8, 0xfa, 0xdc, 0x01, 0x13,
	0xda, 0x85, 0x39, 0xea, 0x04, 0xc6, 0x85, 0xd3, 0xc2, 0x85, 0x74, 0x20, 0xe3, 0x76, 0xa2, 0x8c,
	0x52, 0x40, 0x7b, 0xe8, 0xb4, 0xb0, 0x0e, 0x66, 0xf4, 0x5f, 0x9d, 0x87, 0xb9, 0x98, 0x6e, 0xea,
	0x1e, 0xcc, 0xc5, 0x16, 0x43, 0xab, 0x30, 0x73, 0xee, 0x19, 0x51, 0x56, 0xcd, 0xe9, 0xd3, 0xe7,
	0x5e, 0x90, 0x28, 0x6f, 0xc2, 0x5c, 0xa0, 0x85, 0x71, 0xde, 0x31, 0xdb, 0xf4, 0x1e, 0xc8, 0xe9,
	0x10, 0x80, 0x3e, 0x20, 0x10, 0xf5, 0x5f, 0x0a, 0x40, 0x7f, 0x49, 0x74, 0x1f, 0x32, 0x81, 0x96,
	0x34, 0x37, 0xdf, 0x1d, 0x43, 0xcb, 0x62, 0xa0, 0x6a, 0xc0, 0xa5, 0x7d, 0xa6, 0x40, 0x26, 0x10,
	0x23, 0xe6, 0xe7, 0x7a, 0xb5, 0xb6, 0x77, 0x50, 0x31, 0x6a, 0x47, 0xbb, 0x15, 0xe3, 0x23, 0xbd,
	0xda, 0xa8, 0xe8, 0x79, 0x05, 0xad, 0xc3, 0x6a, 0x1c, 0xae, 0x57, 0x4a, 0xbb, 0x15, 0xdd, 0x38,
	0xaa, 0x1d, 0x3c, 0xcc, 0xa7, 0x90, 0x0a, 0x2b, 0x87, 0x27, 0x07, 0x8d, 0xea, 0x20, 0x2e, 0x8d,
	0x36, 0xa0, 0x10, 0xc3, 0x31, 0x19, 0x4c, 0x6c, 0x86, 0x88, 0x8d, 0x61, 0xe9, 0x5f, 0x86, 0xcc,
	0xee, 0xcc, 0x47, 0x87, 0x11, 0x38, 0xdb, 0x47, 0x30, 0xcf, 0xa5, 0x57, 0x52, 0x26, 0xb0, 0x10,
	0x6f, 0x19, 0x67, 0x57, 0x3e, 0xf6, 0x02, 0x4b, 0xa4, 0xf5, 0xf9, 0x10, 0xba, 0x43, 0x80, 0xc4,
	0xac, 0x1d, 0xeb, 0xc2, 0xf2, 0x19, 0x4d, 0x2a, 0xa0, 0x81, 0x00, 0x14, 0x10, 0x68, 0x7f, 0x49,
	0xc1, 0x34, 0x3b, 0x9b, 0x3b, 0xb1, 0x04, 0xcf, 0x89, 0x0c, 0xa1, 0x54, 0x24, 0x8d, 0x87, 0x54,
	0x18, 0x0f, 0xe8, 0x7d, 0x00, 0xd3, 0xf7, 0x5d, 0xeb, 0xac, 0xe7, 0x47, 0x09, 0xfd, 0x06, 0x7f,
	0x1e, 0xc5, 0x52, 0x44, 0xc0, 0x32, 0x70, 0x9f, 0x03, 0xed, 0xc0, 0x82, 0x90, 0x04, 0x33, 0xa3,
	0x93, 0xe0, 0x7c, 0x93, 0x8b, 0xff, 0x12, 0x2c, 0x86, 0xf9, 0xab, 0x83, 0x0d, 0x9f, 0xe5, 0x37,
	0x96, 0xbf, 0xf3, 0x03, 0x79, 0x0f, 0xf5, 0x89, 0x43, 0x18, 0xc9, 0x72, 0x82, 0x96, 0x13, 0x65,
	0xa6, 0x1e, 0x2c, 0x4a, 0xd2, 0x2a, 0x2a, 0x42, 0x2e, 0x38, 0x10, 0xcf, 0xf2, 0x89, 0xaf, 0xca,
	0xd5, 0xe9, 0x93, 0x10, 0xfa, 0xae, 0x8b, 0xcf, 0xb1, 0xeb, 0xe2, 0x16, 0x2b, 0x86, 0x24, 0xf4,
	0x11, 0x89, 0xf6, 0x73, 0x05, 0x66, 0x43, 0x38, 0x7a, 0x07, 0x66, 0x3d, 0xdc, 0xa6, 0x29, 0x5f,
	0xe1, 0xcf, 0x21, 0xa4, 0x29, 0xd6, 0x19, 0x01, 0x2b, 0x03, 0x43, 0x7a, 0x52, 0x06, 0x72, 0xa8,
	0x89, 0x36, 0xff, 0x6f, 0x05, 0x16, 0x77, 0x71, 0x07, 0x8b, 0x65, 0xc4, 0x3a, 0xe4, 0xd8, 0x35,
	0x17, 0x65, 0xd0, 0x59, 0x0a, 0xa8, 0xb6, 0x84, 0x9b, 0xb7, 0x15, 0xb0, 0x47, 0x37, 0x6f, 0x8a,
	0xbf, 0x79, 0x25, 0xc2, 0x63, 0x37, 0x2f, 0xc5, 0x26, 0xdd, 0xbc, 0x1c, 0x96, 0xbf, 0x8d, 0x06,
	0x19, 0x27, 0xda, 0xf6, 0x0a, 0x2c, 0xf1, 0x8a, 0xd1, 0x1b, 0x40, 0xfb, 0x53, 0x06, 0x6e, 0xf4,
	0x17, 0x39, 0xee, 0x9d, 0x75, 0x2c, 0xef, 0xf1, 0x04, 0x96, 0x59, 0x85, 0x19, 0xdb, 0x69, 0x05,
	0x28, 0xba, 0xe6, 0x34, 0x79, 0xac, 0xb6, 0x50, 0x05, 0xae, 0x8b, 0x45, 0xd4, 0x15, 0xcb, 0xd3,
	0xc9, 0x25, 0x54, 0xfe, 0x52, 0xbc, 0x64, 0x54, 0x98, 0x25, 0xe5, 0x9f, 0x63, 0x77, 0xae, 0x82,
	0x58, 0x9b, 0xd5, 0xa3, 0x67, 0xf4, 0x33, 0x05, 0xd4, 0xd8, 0xb1, 0x74, 0xa9, 0xf2, 0x42, 0x45,
	0xb4, 0x1b, 0x55, 0x44, 0x43, 0x77, 0x39, 0x88, 0xe6, 0xce, 0xa8, 0xd0, 0x4c, 0x40, 0x23, 0x2b,
	0xda, 0x67, 0x2c, 0xb3, 0x4c, 0x07, 0x4b, 0xdf, 0x1f, 0x73, 0x69, 0xfa, 0x24, 0xe6, 0x1d, 0x66,
	0x8b, 0x3e, 0x58, 0x7d, 0x00, 0x9b, 0x43, 0xb5, 0x9c, 0xa8, 0xd4, 0x29, 0xc3, 0xb2, 0x74, 0xdd,
	0x89, 0xbc, 0xea, 0xcf, 0x0a, 0xdc, 0x4c, 0xdc, 0x1c, 0xab, 0x31, 0x7e, 0x00, 0xd7, 0xc2, 0x93,
	0xb1, 0xec, 0x73, 0x87, 0x45, 0xfb, 0xdb, 0x23, 0x6d, 0xc3, 0x7a, 0x41, 0x06, 0x25, 0xfd, 0x21,
	0xb5, 0xcb, 0x5c, 0xb7, 0x0f, 0x51, 0xdf, 0x87, 0xbc, 0x48, 0x30, 0xd1, 0x06, 0xfe, 0x98, 0x82,
	0xad, 0xbe, 0x06, 0x27, 0x76, 0xf7, 0xf9, 0x05, 0xc0, 0xaf, 0x14, 0xd8, 0x88, 0x79, 0x67, 0xcf,
	0x16, 0xfd, 0x93, 0x5e, 0x3f, 0xfb, 0x83, 0x86, 0x90, 0xab, 0x21, 0x23, 0xe0, 0x7c, 0x34, 0x16,
	0x0b, 0x22, 0x81, 0x7a, 0x18, 0x3f, 0x27, 0x29, 0xfb, 0x44, 0x66, 0xbb, 0x0d, 0xb7, 0x86, 0xa8,
	0xcb, 0x52, 0xcb, 0x4f, 0xd3, 0x70, 0xeb, 0xd4, 0xec, 0x58, 0xad, 0xa8, 0xee, 0x94, 0xb4, 0xdd,
	0xc3, 0x8d, 0x9b, 0xd0, 0x89, 0xa5, 0xbe, 0x42, 0x27, 0xd6, 0x91, 0xc5, 0x29, 0x3d, 0x82, 0xef,
	0x46, 0x82, 0x46, 0x69, 0x3b, 0x6e, 0xa8, 0x26, 0x5d, 0xf2, 0x99, 0x09, 0x2e, 0xf9, 0xe7, 0x12,
	0xa0, 0x1f, 0x83, 0x36, 0x6c, 0x53, 0x2c, 0x44, 0x37, 0x20, 0xe7, 0xf5, 0xba, 0x5d, 0xc7, 0xf5,
	0x31, 0x3d, 0x83, 0x59, 0xbd, 0x0f, 0x40, 0x05, 0x98, 0xb9, 0xc0, 0x9e, 0x67, 0xb6, 0x43, 0xf9,
	0xe1, 0xa3, 0xf6, 0x31, 0xa0, 0x03, 0xcb, 0x63, 0xf5, 0x72, 0x74, 0xa2, 0xa4, 0x3c, 0x36, 0x9f,
	0x1a, 0xd8, 0xf6, 0x5d, 0x8b, 0x15, 0x66, 0x59, 0x1d, 0x2e, 0xcc, 0xa7, 0x15, 0x0a, 0x21, 0xc5,
	0x9b, 0xe7, 0x9b, 0xae, 0x6f, 0xd9, 0x6d, 0xc3, 0x77, 0x9e, 0xe0, 0x68, 0x6c, 0x14, 0x42, 0x1b,
	0x04, 0xa8, 0x7d, 0xae, 0xc0, 0x22, 0x27, 0x9e, 0x69, 0xfb, 0x2e, 0xcc, 0xf4, 0x65, 0x13, 0x7b,
	0xde, 0x0a, 0xed, 0x29, 0xa1, 0x2e, 0xd2, 0x13, 0x0a, 0x39, 0xd0, 0x26, 0x80, 0x8d, 0x9f, 0xfa,
	0xdc, 0xba, 0x39, 0x02, 0x09, 0xd6, 0x54, 0xef, 0x41, 0x96, 0x1a, 0x79, 0xdc, 0xce, 0xe8, 0x8b,
	0x14, 0xa0, 0x3d, 0xec, 0x47, 0x05, 0x2f, 0xb3, 0x41, 0x82, 0xe3, 0x2a, 0x5f, 0xc1, 0x71, 0x3f,
	0xe4, 0x46, 0x08, 0xd4, 0xf5, 0x5f, 0x8d, 0xcd, 0xcf, 0x84, 0xa5, 0x87, 0x4e, 0x10, 0x12, 0xdc,
	0x92, 0x5e, 0xcb, 0x63, 0xd7, 0x9e, 0xcf, 0xd0, 0x61, 0x6b, 0xbb, 0xb0, 0xc8, 0xe9, 0xcc, 0xce,
	0xf4, 0x75, 0x40, 0xe6, 0xa5, 0x69, 0x75, 0x4c, 0xa2, 0x57, 0x58, 0xc3, 0xb3, 0x9a, 0xfe, 0x7a,
	0x84, 0x09, 0xd9, 0x34, 0x2d, 0x9e, 0xb5, 0x99, 0x3c, 0x71, 0x9e, 0xd7, 0x89, 0xe7, 0xa8, 0x01,
	0x1a, 0xb6, 0xee, 0x9e, 0x74, 0xa6, 0x77, 0x7b, 0x30, 0x27, 0xb3, 0xb9, 0x59, 0xe2, 0x78, 0xef,
	0x6f, 0x29, 0x58, 0x1f, 0x42, 0x8d, 0xde, 0x85, 0xb4, 0xdb, 0x6d, 0x32, 0x67, 0x7a, 0x79, 0x0c,
	0xf9, 0x45, 0xfd, 0xb8, 0xbc, 0x3f, 0xa5, 0x13, 0x2e, 0xf5, 0x4b, 0x05, 0xd2, 0xfa, 0x71, 0x19,
	0x7d, 0x8f, 0x1b, 0xf2, 0xbd, 0x36, 0xa6, 0x94, 0xf8, 0xac, 0x8f, 0x34, 0x93, 0x83, 0xc3, 0xbe,
	0x02, 0x2c, 0x95, 0xf5, 0x4a, 0xa9, 0x51, 0x31, 0x76, 0x2b, 0x07, 0x95, 0x46, 0xc5, 0x38, 0x3d,
	0x3a, 0x38, 0x39, 0xac, 0xe4, 0x15, 0xd2, 0x15, 0x1e, 0x9f, 0xec, 0x1c, 0x54, 0xeb, 0xfb, 0xc6,
	0x49, 0x2d, 0xfc, 0xc7, 0xb0, 0x29, 0x94, 0x87, 0x6b, 0x07, 0xd5, 0x7a, 0x83, 0x01, 0xea, 0xf9,
	0x34, 0x81, 0xec, 0x55, 0x1a, 0x46, 0xb9, 0x74, 0x5c, 0x2a, 0x57, 0x1b, 0x0f, 0xf3, 0x19, 0xd2,
	0x73, 0xf2, 0xb2, 0xeb, 0xb5, 0xd2, 0x71, 0x7d, 0xff, 0xa8, 0x91, 0xcf, 0x22, 0x04, 0x0b, 0x01,
	0x7f, 0x08, 0xaa, 0xe7, 0xa7, 0xa3, 0x91, 0xc5, 0x67, 0x69, 0x58, 0x66, 0x13, 0x18, 0x36, 0xe3,
	0x08, 0x63, 0xeb, 0x2e, 0xe4, 0x69, 0xf3, 0x65, 0x88, 0x17, 0xc7, 0x02, 0x85, 0x9f, 0x86, 0xd7,
	0x47, 0x38, 0x1a, 0x4c, 0xc5, 0x46, 0x83, 0x5d, 0x58, 0x0d, 0x27, 0x67, 0x4c, 0xae, 0x70, 0x21,
	0x0b, 0x23, 0x34, 0x61, 0x75, 0x01, 0xca, 0x5d, 0xc0, 0xcb, 0x4d, 0x19, 0x0e, 0x1d, 0x4a, 0x66,
	0x80, 0xaf, 0x0f, 0x5f, 0x64, 0x48, 0x0c, 0xab, 0xfb, 0xa0, 0x26, 0xeb, 0x30, 0x51, 0x09, 0xf8,
	0x8c, 0xa1, 0xfc, 0x01, 0xac, 0x88, 0xda, 0xb3, 0xa8, 0x7a, 0x6d, 0x60, 0xc4, 0x15, 0xe5, 0x96,
	0x88, 0x36, 0xa2, 0xd0, 0xfe, 0xa0, 0xc0, 0x6c, 0x08, 0x26, 0xf9, 0xd9, 0xb3, 0x7e, 0x82, 0xb9,
	0xa6, 0x3e, 0x47, 0x20, 0xf2, 0x86, 0x5e, 0xe6, 0x0b, 0x69, 0xa9, 0x2f, 0x6c, 0x02, 0xd0, 0xe3,
	0x69, 0x19, 0xa6, 0x1f, 0xb4, 0x12, 0x69, 0x3d, 0xc7, 0x20, 0x25, 0xd2, 0xfc, 0x4e, 0x7b, 0xbe,
	0xe9, 0xf7, 0x48, 0xdb, 0x40, 0x14, 0x5e, 0x11, 0x15, 0xae, 0x07, 0x58, 0x9d, 0x51, 0x91, 0x40,
	0x5a, 0xe0, 0x51, 0xe8, 0x1e, 0x17, 0x9d, 0xeb, 0x72, 0x01, 0xb1, 0x60, 0x24, 0x17, 0x6b, 0x0b,
	0xfb, 0xa6, 0xd5, 0xf1, 0xc2, 0x8b, 0x95, 0x3d, 0x6a, 0x3b, 0xb2, 0x28, 0xcd, 0x41, 0x56, 0xaf,
	0x94, 0x76, 0x1f, 0xe6, 0x15, 0x34, 0x0f, 0xb9, 0x93, 0xe3, 0x83, 0xa3, 0xd2, 0x6e, 0xb5, 0xb6,
	0x97, 0x4f, 0xa1, 0x45, 0x78, 0xa1, 0xa2, 0xeb, 0x47, 0xba, 0xd1, 0x07, 0xa6, 0x49, 0xa3, 0xbb,
	0xcc, 0x9a, 0x46, 0x21, 0x80, 0x6e, 0xc2, 0x5c, 0xe4, 0xfb, 0x51, 0xec, 0x40, 0x08, 0xaa, 0xb6,
	0x48, 0x8c, 0x84, 0x3d, 0xae, 0x18, 0x23, 0xd2, 0x66, 0x57, 0x74, 0x5f, 0x1e, 0xca, 0xc7, 0x48,
	0x4b, 0x86, 0x23, 0x4e, 0x9d, 0xcc, 0x34, 0x91, 0x57, 0x16, 0x60, 0x45, 0x54, 0x8a, 0xd5, 0xa3,
	0xbf, 0x55, 0x60, 0x89, 0x54, 0x08, 0x21, 0xe2, 0x79, 0x17, 0x2c, 0x13, 0x38, 0xa3, 0x70, 0x02,
	0x19, 0xf1, 0x04, 0xb4, 0xdf, 0x2b, 0xb0, 0x2c, 0xe8, 0xca, 0x62, 0xeb, 0x3d, 0xb1, 0xfa, 0xb9,
	0x1d, 0xaf, 0x7e, 0x06, 0xe8, 0x27, 0xac, 0x7f, 0xde, 0x0c, 0xeb, 0x9f, 0xc9, 0x42, 0xf8, 0x37,
	0x59, 0x58, 0xa9, 0x39, 0x2d, 0x5c, 0xf7, 0xcd, 0xf6, 0x24, 0x73, 0x15, 0x5d, 0xe8, 0x0d, 0xa9,
	0x77, 0xdd, 0x0b, 0x57, 0x92, 0x8b, 0x1c, 0xde, 0x12, 0xa2, 0x22, 0x2c, 0x7a, 0xbe, 0xd9, 0x0e,
	0xce, 0xca, 0x74, 0xdb, 0xd8, 0x37, 0xba, 0xa6, 0xff, 0x98, 0x1d, 0xc4, 0x75, 0x86, 0x6a, 0x04,
	0x98, 0x63, 0xd3, 0x7f, 0x2c, 0x1f, 0x54, 0x64, 0x26, 0x1e, 0x54, 0x9c, 0x01, 0x0a, 0xfa, 0x40,
	0xb2, 0x80, 0xf8, 0x56, 0xe6, 0xdb, 0x23, 0x36, 0x14, 0x81, 0xb9, 0x50, 0xc9, 0xdb, 0x02, 0x18,
	0x99, 0xc9, 0xb3, 0x86, 0x51, 0x4b, 0x8c, 0x3b, 0x63, 0x78, 0xc6, 0x86, 0x9a, 0x74, 0x2d, 0xd2,
	0xdd, 0x7c, 0xfd, 0xb3, 0x89, 0x35, 0x58, 0x1d, 0xb0, 0x05, 0xcb, 0x04, 0x6d, 0x28, 0x10, 0xd4,
	0x89, 0xed, 0x4d, 0xe8, 0xaf, 0x09, 0xbe, 0x95, 0x4a, 0xf0, 0x2d, 0x6d, 0x1d, 0xd6, 0x24, 0x0b,
	0x31, 0x2d, 0xfe, 0x91, 0xa5, 0x6a, 0x4c, 0x3e, 0x74, 0x6b, 0x48, 0xc3, 0xe6, 0x9b, 0x71, 0x17,
	0x90, 0x0e, 0x9a, 0x9e, 0x6f, 0xe0, 0xdc, 0x84, 0xb9, 0x38, 0x1d, 0x4b, 0x62, 0xfe, 0x88, 0xc8,
	0xca, 0x3e, 0xd3, 0x08, 0x70, 0x5a, 0x18, 0x01, 0xfe, 0x08, 0x96, 0x82, 0xa8, 0x13, 0x67, 0x2b,
	0x33, 0xfc, 0x35, 0x95, 0x68, 0x91, 0x18, 0x82, 0x8b, 0xbd, 0x20, 0x96, 0x85, 0x49, 0x5f, 0x53,
	0x16, 0x7d, 0xb3, 0xc1, 0x42, 0x6f, 0x8d, 0x5c, 0xe8, 0xeb, 0x8a, 0xbf, 0x0a, 0xf5, 0xfa, 0xff,
	0x8b, 0xe9, 0x20, 0xf3, 0x7e, 0xe9, 0x5c, 0x4f, 0x7b, 0x04, 0x2a, 0x0d, 0x8d, 0xc9, 0x47, 0x6e,
	0x82, 0xe3, 0xa5, 0x44, 0xc7, 0xd3, 0x36, 0x61, 0x5d, 0x2a, 0x9b, 0x2d, 0x8d, 0x20, 0x4f, 0xd0,
	0x7b, 0xd8, 0xaf, 0xb6, 0xc2, 0x6e, 0xf1, 0x35, 0xb8, 0x1e, 0x83, 0xb1, 0xbb, 0x36, 0x36, 0xdb,
	0x53, 0xe2, 0xb3, 0x3d, 0x6d, 0x83, 0x2a, 0x9f, 0xd0, 0x79, 0x7e, 0x42, 0x97, 0x4f, 0xea, 0x39,
	0x4b, 0x42, 0xcf, 0x49, 0xaf, 0xf1, 0x4d, 0x2e, 0x81, 0x8f, 0xe8, 0x36, 0xff, 0xaa, 0xb0, 0x34,
	0x3b, 0xd0, 0x67, 0xbe, 0x19, 0xef, 0x33, 0x6f, 0x0d, 0x95, 0x19, 0xef, 0x30, 0xbb, 0xb4, 0xc1,
	0x7c, 0x87, 0x2b, 0x61, 0x5f, 0x1a, 0xc9, 0x1e, 0x6f, 0x2d, 0x5f, 0x4f, 0xe8, 0x2c, 0xeb, 0x8d,
	0xd2, 0x5e, 0xc5, 0x38, 0xa9, 0xd1, 0xdf, 0xb0, 0xb3, 0x8c, 0xfa, 0xbc, 0x25, 0x40, 0xa1, 0xe1,
	0x63, 0xdf, 0x21, 0x7d, 0xae, 0xc0, 0x22, 0x07, 0x1e, 0x71, 0x22, 0xe8, 0x1e, 0x2c, 0x91, 0x1a,
	0x8e, 0xfa, 0x88, 0x67, 0x74, 0xb1, 0x6b, 0x10, 0x0c, 0x7b, 0x8b, 0x78, 0xfd, 0xc2, 0x7c, 0xca,
	0x06, 0x43, 0xc7, 0xd8, 0x25, 0x82, 0x9f, 0xc3, 0x28, 0x64, 0xfb, 0x3f, 0x0a, 0xcc, 0x56, 0x5b,
	0xd8, 0xf6, 0x89, 0xe1, 0x6b, 0x30, 0xcf, 0x7d, 0xcc, 0x84, 0x36, 0x12, 0xbe, 0x71, 0x0a, 0x36,
	0xa8, 0x6e, 0x0e, 0xfd, 0x02, 0x4a, 0x9b, 0x42, 0xe7, 0xb1, 0x0f, 0xb1, 0xb8, 0x79, 0xd0, 0x8b,
	0x03, 0x9c, 0x12, 0x1f, 0x54, 0xef, 0x8c, 0xa0, 0x8a, 0xd6, 0x79, 0x0b, 0xb2, 0xc1, 0x97, 0x39,
	0x68, 0x29, 0xfa, 0x66, 0x28, 0xf6, 0xe1, 0x8e, 0xba, 0x2c, 0x40, 0x43, 0xbe, 0xed, 0xff, 0xce,
	0x00, 0xf4, 0x07, 0x0f, 0xe8, 0x01, 0x5c, 0x8b, 0x7f, 0x61, 0x80, 0xd6, 0x87, 0x7c, 0x88, 0xa2,
	0x6e, 0xc8, 0x91, 0x91, 0x4e, 0x0f, 0xe0, 0x5a, 0xfc, 0x65, 0x55, 0x5f, 0x98, 0xe4, 0xdd, 0x5a,
	0x5f, 0x98, 0xf4, 0xfd, 0xd6, 0x14, 0xea, 0xc0, 0x6a, 0xc2, 0x3b, 0x06, 0xf4, 0xd2, 0x78, 0x2f,
	0x68, 0xd4, 0x97, 0xc7, 0x7c, 0x59, 0xa1, 0x4d, 0x21, 0x17, 0xd6, 0x12, 0x27, 0xe3, 0xe8, 0xee,
	0xb8, 0xb3, 0x7e, 0xf5, 0x95, 0x31, 0x28, 0xa3, 0x35, 0x7b, 0xa0, 0x26, 0x0f, 0x79, 0xd1, 0x2b,
	0x63, 0x4f, 0xb7, 0xd5, 0x57, 0xc7, 0x21, 0x8d, 0x96, 0xdd, 0x87, 0xb9, 0xd8, 0xc0, 0x15, 0xa9,
	0xd2, 0x29, 0x2c, 0x15, 0xbc, 0x3e, 0x64, 0x42, 0x4b, 0x25, 0xc5, 0x86, 0x82, 0x7d, 0x49, 0x83,
	0xd3, 0xcd, 0xbe, 0x24, 0xc9, 0x14, 0x51, 0x34, 0xbf, 0x90, 0x80, 0x65, 0xe6, 0x97, 0x67, 0x70,
	0x99, 0xf9, 0x13, 0xb2, 0xb9, 0x36, 0x85, 0xbe, 0x0f, 0x0b, 0xfc, 0x1c, 0x04, 0x6d, 0x0e, 0x9d,
	0xee, 0xa8, 0x37, 0x92, 0xd0, 0x71, 0x91, 0x7c, 0x13, 0xdb, 0x17, 0x29, 0xed, 0xb8, 0xfb, 0x22,
	0x13, 0x7a, 0xdf, 0x29, 0x92, 0x9f, 0xb8, 0x06, 0xb1, 0x9f, 0x9f, 0x64, 0x3d, 0x71, 0x3f, 0x3f,
	0x49, 0xbb, 0x4a, 0x6d, 0x6a, 0xfb, 0xcb, 0x0c, 0x64, 0x82, 0x44, 0xda, 0x80, 0x17, 0x84, 0x3a,
	0x1b, 0xdd, 0x18, 0xde, 0x8c, 0xa8, 0x37, 0x13, 0xf1, 0x91, 0xba, 0x8f, 0xe8, 0x7d, 0xcc, 0x55,
	0xce, 0x68, 0x2b, 0xce, 0x27, 0xab, 0xde, 0xd5, 0x5b, 0x43, 0x28, 0x44, 0xd9, 0x7c, 0x2e, 0xd8,
	0x1a, 0x55, 0xc2, 0xf1, 0xb2, 0x93, 0xe2, 0xff, 0x13, 0x7a, 0x6f, 0x89, 0x91, 0xaf, 0xf1, 0x7a,
	0x49, 0x63, 0xfe, 0xf6, 0x50, 0x9a, 0x68, 0x85, 0x0a, 0xe4, 0xa2, 0x4a, 0x05, 0x15, 0xe2, 0x3c,
	0xf1, 0x82, 0x46, 0x5d, 0x93, 0x60, 0x98, 0x8c, 0xf4, 0x2f, 0x52, 0x4a, 0xa8, 0xa8, 0x18, 0x23,
	0x9a, 0xc0, 0x26, 0x8b, 0x8e, 0xdb, 0x43, 0x69, 0xe2, 0x51, 0x1d, 0xbb, 0xc2, 0xfb, 0x51, 0x3d,
	0x78, 0xdd, 0xf7, 0xa3, 0x5a, 0x72, 0xe7, 0x6b, 0x53, 0x3b, 0xd9, 0x47, 0xe9, 0xa6, 0x67, 0x9d,
	0x4d, 0x07, 0x1f, 0x87, 0x7e, 0xeb, 0x7f, 0x01, 0x00, 0x00, 0xff, 0xff, 0x9c, 0x3b, 0x5a, 0x51,
	0xf0, 0x2c, 0x00, 0x00,
}
