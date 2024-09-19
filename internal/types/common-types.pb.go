// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.12
// source: common-types.proto

package types

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type DeviceType int32

const (
	DeviceType_TELTONIKA DeviceType = 0
	DeviceType_WANWAY    DeviceType = 1
	DeviceType_CONCOX    DeviceType = 2
)

// Enum value maps for DeviceType.
var (
	DeviceType_name = map[int32]string{
		0: "TELTONIKA",
		1: "WANWAY",
		2: "CONCOX",
	}
	DeviceType_value = map[string]int32{
		"TELTONIKA": 0,
		"WANWAY":    1,
		"CONCOX":    2,
	}
)

func (x DeviceType) Enum() *DeviceType {
	p := new(DeviceType)
	*p = x
	return p
}

func (x DeviceType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (DeviceType) Descriptor() protoreflect.EnumDescriptor {
	return file_common_types_proto_enumTypes[0].Descriptor()
}

func (DeviceType) Type() protoreflect.EnumType {
	return &file_common_types_proto_enumTypes[0]
}

func (x DeviceType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use DeviceType.Descriptor instead.
func (DeviceType) EnumDescriptor() ([]byte, []int) {
	return file_common_types_proto_rawDescGZIP(), []int{0}
}

type DeviceProtocolType int32

const (
	DeviceProtocolType_FM1200 DeviceProtocolType = 0
	DeviceProtocolType_GT06   DeviceProtocolType = 1
	DeviceProtocolType_TR06   DeviceProtocolType = 2
)

// Enum value maps for DeviceProtocolType.
var (
	DeviceProtocolType_name = map[int32]string{
		0: "FM1200",
		1: "GT06",
		2: "TR06",
	}
	DeviceProtocolType_value = map[string]int32{
		"FM1200": 0,
		"GT06":   1,
		"TR06":   2,
	}
)

func (x DeviceProtocolType) Enum() *DeviceProtocolType {
	p := new(DeviceProtocolType)
	*p = x
	return p
}

func (x DeviceProtocolType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (DeviceProtocolType) Descriptor() protoreflect.EnumDescriptor {
	return file_common_types_proto_enumTypes[1].Descriptor()
}

func (DeviceProtocolType) Type() protoreflect.EnumType {
	return &file_common_types_proto_enumTypes[1]
}

func (x DeviceProtocolType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use DeviceProtocolType.Descriptor instead.
func (DeviceProtocolType) EnumDescriptor() ([]byte, []int) {
	return file_common_types_proto_rawDescGZIP(), []int{1}
}

type DeviceStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Imei             string                 `protobuf:"bytes,1,opt,name=imei,proto3" json:"imei,omitempty"`
	DeviceType       DeviceType             `protobuf:"varint,2,opt,name=device_type,json=deviceType,proto3,enum=types.DeviceType" json:"device_type,omitempty"`
	Timestamp        *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	MessageType      string                 `protobuf:"bytes,4,opt,name=message_type,json=messageType,proto3" json:"message_type,omitempty"`
	Position         *GPSPosition           `protobuf:"bytes,5,opt,name=position,proto3" json:"position,omitempty"`
	VehicleStatus    *VehicleStatus         `protobuf:"bytes,6,opt,name=vehicle_status,json=vehicleStatus,proto3" json:"vehicle_status,omitempty"`
	BatteryLevel     int32                  `protobuf:"varint,7,opt,name=battery_level,json=batteryLevel,proto3" json:"battery_level,omitempty"`
	Temperature      float32                `protobuf:"fixed32,8,opt,name=temperature,proto3" json:"temperature,omitempty"`
	Odometer         int32                  `protobuf:"varint,9,opt,name=odometer,proto3" json:"odometer,omitempty"`
	FuelLevel        int32                  `protobuf:"varint,10,opt,name=fuel_level,json=fuelLevel,proto3" json:"fuel_level,omitempty"`
	IdentificationId string                 `protobuf:"bytes,11,opt,name=identification_id,json=identificationId,proto3" json:"identification_id,omitempty"`
	// Types that are assignable to RawData:
	//
	//	*DeviceStatus_WanwayPacket
	//	*DeviceStatus_TeltonikaPacket
	//	*DeviceStatus_ConcoxPacket
	RawData isDeviceStatus_RawData `protobuf_oneof:"raw_data"`
}

func (x *DeviceStatus) Reset() {
	*x = DeviceStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_types_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeviceStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeviceStatus) ProtoMessage() {}

func (x *DeviceStatus) ProtoReflect() protoreflect.Message {
	mi := &file_common_types_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeviceStatus.ProtoReflect.Descriptor instead.
func (*DeviceStatus) Descriptor() ([]byte, []int) {
	return file_common_types_proto_rawDescGZIP(), []int{0}
}

func (x *DeviceStatus) GetImei() string {
	if x != nil {
		return x.Imei
	}
	return ""
}

func (x *DeviceStatus) GetDeviceType() DeviceType {
	if x != nil {
		return x.DeviceType
	}
	return DeviceType_TELTONIKA
}

func (x *DeviceStatus) GetTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *DeviceStatus) GetMessageType() string {
	if x != nil {
		return x.MessageType
	}
	return ""
}

func (x *DeviceStatus) GetPosition() *GPSPosition {
	if x != nil {
		return x.Position
	}
	return nil
}

func (x *DeviceStatus) GetVehicleStatus() *VehicleStatus {
	if x != nil {
		return x.VehicleStatus
	}
	return nil
}

func (x *DeviceStatus) GetBatteryLevel() int32 {
	if x != nil {
		return x.BatteryLevel
	}
	return 0
}

func (x *DeviceStatus) GetTemperature() float32 {
	if x != nil {
		return x.Temperature
	}
	return 0
}

func (x *DeviceStatus) GetOdometer() int32 {
	if x != nil {
		return x.Odometer
	}
	return 0
}

func (x *DeviceStatus) GetFuelLevel() int32 {
	if x != nil {
		return x.FuelLevel
	}
	return 0
}

func (x *DeviceStatus) GetIdentificationId() string {
	if x != nil {
		return x.IdentificationId
	}
	return ""
}

func (m *DeviceStatus) GetRawData() isDeviceStatus_RawData {
	if m != nil {
		return m.RawData
	}
	return nil
}

func (x *DeviceStatus) GetWanwayPacket() *WanwayPacket {
	if x, ok := x.GetRawData().(*DeviceStatus_WanwayPacket); ok {
		return x.WanwayPacket
	}
	return nil
}

func (x *DeviceStatus) GetTeltonikaPacket() *TeltonikaPacket {
	if x, ok := x.GetRawData().(*DeviceStatus_TeltonikaPacket); ok {
		return x.TeltonikaPacket
	}
	return nil
}

func (x *DeviceStatus) GetConcoxPacket() *ConcoxPacket {
	if x, ok := x.GetRawData().(*DeviceStatus_ConcoxPacket); ok {
		return x.ConcoxPacket
	}
	return nil
}

type isDeviceStatus_RawData interface {
	isDeviceStatus_RawData()
}

type DeviceStatus_WanwayPacket struct {
	WanwayPacket *WanwayPacket `protobuf:"bytes,12,opt,name=wanway_packet,json=wanwayPacket,proto3,oneof"`
}

type DeviceStatus_TeltonikaPacket struct {
	TeltonikaPacket *TeltonikaPacket `protobuf:"bytes,13,opt,name=teltonika_packet,json=teltonikaPacket,proto3,oneof"`
}

type DeviceStatus_ConcoxPacket struct {
	ConcoxPacket *ConcoxPacket `protobuf:"bytes,14,opt,name=concox_packet,json=concoxPacket,proto3,oneof"`
}

func (*DeviceStatus_WanwayPacket) isDeviceStatus_RawData() {}

func (*DeviceStatus_TeltonikaPacket) isDeviceStatus_RawData() {}

func (*DeviceStatus_ConcoxPacket) isDeviceStatus_RawData() {}

type GPSPosition struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Latitude   float32  `protobuf:"fixed32,1,opt,name=latitude,proto3" json:"latitude,omitempty"`
	Longitude  float32  `protobuf:"fixed32,2,opt,name=longitude,proto3" json:"longitude,omitempty"`
	Altitude   float32  `protobuf:"fixed32,3,opt,name=altitude,proto3" json:"altitude,omitempty"`
	Speed      *float32 `protobuf:"fixed32,4,opt,name=speed,proto3,oneof" json:"speed,omitempty"`
	Course     float32  `protobuf:"fixed32,5,opt,name=course,proto3" json:"course,omitempty"`
	Satellites int32    `protobuf:"varint,6,opt,name=satellites,proto3" json:"satellites,omitempty"`
}

func (x *GPSPosition) Reset() {
	*x = GPSPosition{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_types_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GPSPosition) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GPSPosition) ProtoMessage() {}

func (x *GPSPosition) ProtoReflect() protoreflect.Message {
	mi := &file_common_types_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GPSPosition.ProtoReflect.Descriptor instead.
func (*GPSPosition) Descriptor() ([]byte, []int) {
	return file_common_types_proto_rawDescGZIP(), []int{1}
}

func (x *GPSPosition) GetLatitude() float32 {
	if x != nil {
		return x.Latitude
	}
	return 0
}

func (x *GPSPosition) GetLongitude() float32 {
	if x != nil {
		return x.Longitude
	}
	return 0
}

func (x *GPSPosition) GetAltitude() float32 {
	if x != nil {
		return x.Altitude
	}
	return 0
}

func (x *GPSPosition) GetSpeed() float32 {
	if x != nil && x.Speed != nil {
		return *x.Speed
	}
	return 0
}

func (x *GPSPosition) GetCourse() float32 {
	if x != nil {
		return x.Course
	}
	return 0
}

func (x *GPSPosition) GetSatellites() int32 {
	if x != nil {
		return x.Satellites
	}
	return 0
}

type VehicleStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ignition     *bool `protobuf:"varint,1,opt,name=ignition,proto3,oneof" json:"ignition,omitempty"`
	Overspeeding bool  `protobuf:"varint,2,opt,name=overspeeding,proto3" json:"overspeeding,omitempty"`
	RashDriving  bool  `protobuf:"varint,3,opt,name=rash_driving,json=rashDriving,proto3" json:"rash_driving,omitempty"`
}

func (x *VehicleStatus) Reset() {
	*x = VehicleStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_types_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VehicleStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VehicleStatus) ProtoMessage() {}

func (x *VehicleStatus) ProtoReflect() protoreflect.Message {
	mi := &file_common_types_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VehicleStatus.ProtoReflect.Descriptor instead.
func (*VehicleStatus) Descriptor() ([]byte, []int) {
	return file_common_types_proto_rawDescGZIP(), []int{2}
}

func (x *VehicleStatus) GetIgnition() bool {
	if x != nil && x.Ignition != nil {
		return *x.Ignition
	}
	return false
}

func (x *VehicleStatus) GetOverspeeding() bool {
	if x != nil {
		return x.Overspeeding
	}
	return false
}

func (x *VehicleStatus) GetRashDriving() bool {
	if x != nil {
		return x.RashDriving
	}
	return false
}

type WanwayPacket struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RawData []byte `protobuf:"bytes,1,opt,name=raw_data,json=rawData,proto3" json:"raw_data,omitempty"`
}

func (x *WanwayPacket) Reset() {
	*x = WanwayPacket{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_types_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WanwayPacket) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WanwayPacket) ProtoMessage() {}

func (x *WanwayPacket) ProtoReflect() protoreflect.Message {
	mi := &file_common_types_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WanwayPacket.ProtoReflect.Descriptor instead.
func (*WanwayPacket) Descriptor() ([]byte, []int) {
	return file_common_types_proto_rawDescGZIP(), []int{3}
}

func (x *WanwayPacket) GetRawData() []byte {
	if x != nil {
		return x.RawData
	}
	return nil
}

type TeltonikaPacket struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RawData []byte `protobuf:"bytes,1,opt,name=raw_data,json=rawData,proto3" json:"raw_data,omitempty"`
}

func (x *TeltonikaPacket) Reset() {
	*x = TeltonikaPacket{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_types_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TeltonikaPacket) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TeltonikaPacket) ProtoMessage() {}

func (x *TeltonikaPacket) ProtoReflect() protoreflect.Message {
	mi := &file_common_types_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TeltonikaPacket.ProtoReflect.Descriptor instead.
func (*TeltonikaPacket) Descriptor() ([]byte, []int) {
	return file_common_types_proto_rawDescGZIP(), []int{4}
}

func (x *TeltonikaPacket) GetRawData() []byte {
	if x != nil {
		return x.RawData
	}
	return nil
}

type ConcoxPacket struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RawData []byte `protobuf:"bytes,1,opt,name=raw_data,json=rawData,proto3" json:"raw_data,omitempty"`
}

func (x *ConcoxPacket) Reset() {
	*x = ConcoxPacket{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_types_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConcoxPacket) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConcoxPacket) ProtoMessage() {}

func (x *ConcoxPacket) ProtoReflect() protoreflect.Message {
	mi := &file_common_types_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConcoxPacket.ProtoReflect.Descriptor instead.
func (*ConcoxPacket) Descriptor() ([]byte, []int) {
	return file_common_types_proto_rawDescGZIP(), []int{5}
}

func (x *ConcoxPacket) GetRawData() []byte {
	if x != nil {
		return x.RawData
	}
	return nil
}

type DeviceResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Imei     string `protobuf:"bytes,1,opt,name=imei,proto3" json:"imei,omitempty"`
	Response string `protobuf:"bytes,2,opt,name=response,proto3" json:"response,omitempty"`
}

func (x *DeviceResponse) Reset() {
	*x = DeviceResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_types_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeviceResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeviceResponse) ProtoMessage() {}

func (x *DeviceResponse) ProtoReflect() protoreflect.Message {
	mi := &file_common_types_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeviceResponse.ProtoReflect.Descriptor instead.
func (*DeviceResponse) Descriptor() ([]byte, []int) {
	return file_common_types_proto_rawDescGZIP(), []int{6}
}

func (x *DeviceResponse) GetImei() string {
	if x != nil {
		return x.Imei
	}
	return ""
}

func (x *DeviceResponse) GetResponse() string {
	if x != nil {
		return x.Response
	}
	return ""
}

var File_common_types_proto protoreflect.FileDescriptor

var file_common_types_proto_rawDesc = []byte{
	0x0a, 0x12, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2d, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x74, 0x79, 0x70, 0x65, 0x73, 0x1a, 0x1f, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x98, 0x05, 0x0a,
	0x0c, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x12, 0x0a,
	0x04, 0x69, 0x6d, 0x65, 0x69, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x69, 0x6d, 0x65,
	0x69, 0x12, 0x32, 0x0a, 0x0b, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x74, 0x79, 0x70, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x11, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x44,
	0x65, 0x76, 0x69, 0x63, 0x65, 0x54, 0x79, 0x70, 0x65, 0x52, 0x0a, 0x64, 0x65, 0x76, 0x69, 0x63,
	0x65, 0x54, 0x79, 0x70, 0x65, 0x12, 0x38, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61,
	0x6d, 0x70, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12,
	0x21, 0x0a, 0x0c, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x2e, 0x0a, 0x08, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x47, 0x50, 0x53,
	0x50, 0x6f, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x08, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x69,
	0x6f, 0x6e, 0x12, 0x3b, 0x0a, 0x0e, 0x76, 0x65, 0x68, 0x69, 0x63, 0x6c, 0x65, 0x5f, 0x73, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x74, 0x79, 0x70,
	0x65, 0x73, 0x2e, 0x56, 0x65, 0x68, 0x69, 0x63, 0x6c, 0x65, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x52, 0x0d, 0x76, 0x65, 0x68, 0x69, 0x63, 0x6c, 0x65, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12,
	0x23, 0x0a, 0x0d, 0x62, 0x61, 0x74, 0x74, 0x65, 0x72, 0x79, 0x5f, 0x6c, 0x65, 0x76, 0x65, 0x6c,
	0x18, 0x07, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0c, 0x62, 0x61, 0x74, 0x74, 0x65, 0x72, 0x79, 0x4c,
	0x65, 0x76, 0x65, 0x6c, 0x12, 0x20, 0x0a, 0x0b, 0x74, 0x65, 0x6d, 0x70, 0x65, 0x72, 0x61, 0x74,
	0x75, 0x72, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x02, 0x52, 0x0b, 0x74, 0x65, 0x6d, 0x70, 0x65,
	0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x6f, 0x64, 0x6f, 0x6d, 0x65, 0x74,
	0x65, 0x72, 0x18, 0x09, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x6f, 0x64, 0x6f, 0x6d, 0x65, 0x74,
	0x65, 0x72, 0x12, 0x1d, 0x0a, 0x0a, 0x66, 0x75, 0x65, 0x6c, 0x5f, 0x6c, 0x65, 0x76, 0x65, 0x6c,
	0x18, 0x0a, 0x20, 0x01, 0x28, 0x05, 0x52, 0x09, 0x66, 0x75, 0x65, 0x6c, 0x4c, 0x65, 0x76, 0x65,
	0x6c, 0x12, 0x2b, 0x0a, 0x11, 0x69, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x09, 0x52, 0x10, 0x69, 0x64,
	0x65, 0x6e, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x3a,
	0x0a, 0x0d, 0x77, 0x61, 0x6e, 0x77, 0x61, 0x79, 0x5f, 0x70, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x18,
	0x0c, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x57, 0x61,
	0x6e, 0x77, 0x61, 0x79, 0x50, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x48, 0x00, 0x52, 0x0c, 0x77, 0x61,
	0x6e, 0x77, 0x61, 0x79, 0x50, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x12, 0x43, 0x0a, 0x10, 0x74, 0x65,
	0x6c, 0x74, 0x6f, 0x6e, 0x69, 0x6b, 0x61, 0x5f, 0x70, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x18, 0x0d,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x54, 0x65, 0x6c,
	0x74, 0x6f, 0x6e, 0x69, 0x6b, 0x61, 0x50, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x48, 0x00, 0x52, 0x0f,
	0x74, 0x65, 0x6c, 0x74, 0x6f, 0x6e, 0x69, 0x6b, 0x61, 0x50, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x12,
	0x3a, 0x0a, 0x0d, 0x63, 0x6f, 0x6e, 0x63, 0x6f, 0x78, 0x5f, 0x70, 0x61, 0x63, 0x6b, 0x65, 0x74,
	0x18, 0x0e, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x43,
	0x6f, 0x6e, 0x63, 0x6f, 0x78, 0x50, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x48, 0x00, 0x52, 0x0c, 0x63,
	0x6f, 0x6e, 0x63, 0x6f, 0x78, 0x50, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x42, 0x0a, 0x0a, 0x08, 0x72,
	0x61, 0x77, 0x5f, 0x64, 0x61, 0x74, 0x61, 0x22, 0xc0, 0x01, 0x0a, 0x0b, 0x47, 0x50, 0x53, 0x50,
	0x6f, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x6c, 0x61, 0x74, 0x69, 0x74,
	0x75, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x02, 0x52, 0x08, 0x6c, 0x61, 0x74, 0x69, 0x74,
	0x75, 0x64, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x6c, 0x6f, 0x6e, 0x67, 0x69, 0x74, 0x75, 0x64, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x02, 0x52, 0x09, 0x6c, 0x6f, 0x6e, 0x67, 0x69, 0x74, 0x75, 0x64,
	0x65, 0x12, 0x1a, 0x0a, 0x08, 0x61, 0x6c, 0x74, 0x69, 0x74, 0x75, 0x64, 0x65, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x02, 0x52, 0x08, 0x61, 0x6c, 0x74, 0x69, 0x74, 0x75, 0x64, 0x65, 0x12, 0x19, 0x0a,
	0x05, 0x73, 0x70, 0x65, 0x65, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x02, 0x48, 0x00, 0x52, 0x05,
	0x73, 0x70, 0x65, 0x65, 0x64, 0x88, 0x01, 0x01, 0x12, 0x16, 0x0a, 0x06, 0x63, 0x6f, 0x75, 0x72,
	0x73, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x02, 0x52, 0x06, 0x63, 0x6f, 0x75, 0x72, 0x73, 0x65,
	0x12, 0x1e, 0x0a, 0x0a, 0x73, 0x61, 0x74, 0x65, 0x6c, 0x6c, 0x69, 0x74, 0x65, 0x73, 0x18, 0x06,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x0a, 0x73, 0x61, 0x74, 0x65, 0x6c, 0x6c, 0x69, 0x74, 0x65, 0x73,
	0x42, 0x08, 0x0a, 0x06, 0x5f, 0x73, 0x70, 0x65, 0x65, 0x64, 0x22, 0x84, 0x01, 0x0a, 0x0d, 0x56,
	0x65, 0x68, 0x69, 0x63, 0x6c, 0x65, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x1f, 0x0a, 0x08,
	0x69, 0x67, 0x6e, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x48, 0x00,
	0x52, 0x08, 0x69, 0x67, 0x6e, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x88, 0x01, 0x01, 0x12, 0x22, 0x0a,
	0x0c, 0x6f, 0x76, 0x65, 0x72, 0x73, 0x70, 0x65, 0x65, 0x64, 0x69, 0x6e, 0x67, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x0c, 0x6f, 0x76, 0x65, 0x72, 0x73, 0x70, 0x65, 0x65, 0x64, 0x69, 0x6e,
	0x67, 0x12, 0x21, 0x0a, 0x0c, 0x72, 0x61, 0x73, 0x68, 0x5f, 0x64, 0x72, 0x69, 0x76, 0x69, 0x6e,
	0x67, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0b, 0x72, 0x61, 0x73, 0x68, 0x44, 0x72, 0x69,
	0x76, 0x69, 0x6e, 0x67, 0x42, 0x0b, 0x0a, 0x09, 0x5f, 0x69, 0x67, 0x6e, 0x69, 0x74, 0x69, 0x6f,
	0x6e, 0x22, 0x29, 0x0a, 0x0c, 0x57, 0x61, 0x6e, 0x77, 0x61, 0x79, 0x50, 0x61, 0x63, 0x6b, 0x65,
	0x74, 0x12, 0x19, 0x0a, 0x08, 0x72, 0x61, 0x77, 0x5f, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x07, 0x72, 0x61, 0x77, 0x44, 0x61, 0x74, 0x61, 0x22, 0x2c, 0x0a, 0x0f,
	0x54, 0x65, 0x6c, 0x74, 0x6f, 0x6e, 0x69, 0x6b, 0x61, 0x50, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x12,
	0x19, 0x0a, 0x08, 0x72, 0x61, 0x77, 0x5f, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x07, 0x72, 0x61, 0x77, 0x44, 0x61, 0x74, 0x61, 0x22, 0x29, 0x0a, 0x0c, 0x43, 0x6f,
	0x6e, 0x63, 0x6f, 0x78, 0x50, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x12, 0x19, 0x0a, 0x08, 0x72, 0x61,
	0x77, 0x5f, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x72, 0x61,
	0x77, 0x44, 0x61, 0x74, 0x61, 0x22, 0x40, 0x0a, 0x0e, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x69, 0x6d, 0x65, 0x69, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x69, 0x6d, 0x65, 0x69, 0x12, 0x1a, 0x0a, 0x08, 0x72,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x72,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2a, 0x33, 0x0a, 0x0a, 0x44, 0x65, 0x76, 0x69, 0x63,
	0x65, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0d, 0x0a, 0x09, 0x54, 0x45, 0x4c, 0x54, 0x4f, 0x4e, 0x49,
	0x4b, 0x41, 0x10, 0x00, 0x12, 0x0a, 0x0a, 0x06, 0x57, 0x41, 0x4e, 0x57, 0x41, 0x59, 0x10, 0x01,
	0x12, 0x0a, 0x0a, 0x06, 0x43, 0x4f, 0x4e, 0x43, 0x4f, 0x58, 0x10, 0x02, 0x2a, 0x34, 0x0a, 0x12,
	0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x0a, 0x0a, 0x06, 0x46, 0x4d, 0x31, 0x32, 0x30, 0x30, 0x10, 0x00, 0x12, 0x08,
	0x0a, 0x04, 0x47, 0x54, 0x30, 0x36, 0x10, 0x01, 0x12, 0x08, 0x0a, 0x04, 0x54, 0x52, 0x30, 0x36,
	0x10, 0x02, 0x42, 0x37, 0x5a, 0x35, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x34, 0x30, 0x34, 0x6d, 0x69, 0x6e, 0x64, 0x73, 0x2f, 0x61, 0x76, 0x6c, 0x2d, 0x72, 0x65,
	0x63, 0x65, 0x69, 0x76, 0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f,
	0x74, 0x79, 0x70, 0x65, 0x73, 0x3b, 0x74, 0x79, 0x70, 0x65, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_common_types_proto_rawDescOnce sync.Once
	file_common_types_proto_rawDescData = file_common_types_proto_rawDesc
)

func file_common_types_proto_rawDescGZIP() []byte {
	file_common_types_proto_rawDescOnce.Do(func() {
		file_common_types_proto_rawDescData = protoimpl.X.CompressGZIP(file_common_types_proto_rawDescData)
	})
	return file_common_types_proto_rawDescData
}

var file_common_types_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_common_types_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_common_types_proto_goTypes = []interface{}{
	(DeviceType)(0),               // 0: types.DeviceType
	(DeviceProtocolType)(0),       // 1: types.DeviceProtocolType
	(*DeviceStatus)(nil),          // 2: types.DeviceStatus
	(*GPSPosition)(nil),           // 3: types.GPSPosition
	(*VehicleStatus)(nil),         // 4: types.VehicleStatus
	(*WanwayPacket)(nil),          // 5: types.WanwayPacket
	(*TeltonikaPacket)(nil),       // 6: types.TeltonikaPacket
	(*ConcoxPacket)(nil),          // 7: types.ConcoxPacket
	(*DeviceResponse)(nil),        // 8: types.DeviceResponse
	(*timestamppb.Timestamp)(nil), // 9: google.protobuf.Timestamp
}
var file_common_types_proto_depIdxs = []int32{
	0, // 0: types.DeviceStatus.device_type:type_name -> types.DeviceType
	9, // 1: types.DeviceStatus.timestamp:type_name -> google.protobuf.Timestamp
	3, // 2: types.DeviceStatus.position:type_name -> types.GPSPosition
	4, // 3: types.DeviceStatus.vehicle_status:type_name -> types.VehicleStatus
	5, // 4: types.DeviceStatus.wanway_packet:type_name -> types.WanwayPacket
	6, // 5: types.DeviceStatus.teltonika_packet:type_name -> types.TeltonikaPacket
	7, // 6: types.DeviceStatus.concox_packet:type_name -> types.ConcoxPacket
	7, // [7:7] is the sub-list for method output_type
	7, // [7:7] is the sub-list for method input_type
	7, // [7:7] is the sub-list for extension type_name
	7, // [7:7] is the sub-list for extension extendee
	0, // [0:7] is the sub-list for field type_name
}

func init() { file_common_types_proto_init() }
func file_common_types_proto_init() {
	if File_common_types_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_common_types_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeviceStatus); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_types_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GPSPosition); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_types_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VehicleStatus); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_types_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WanwayPacket); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_types_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TeltonikaPacket); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_types_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ConcoxPacket); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_types_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeviceResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_common_types_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*DeviceStatus_WanwayPacket)(nil),
		(*DeviceStatus_TeltonikaPacket)(nil),
		(*DeviceStatus_ConcoxPacket)(nil),
	}
	file_common_types_proto_msgTypes[1].OneofWrappers = []interface{}{}
	file_common_types_proto_msgTypes[2].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_common_types_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_common_types_proto_goTypes,
		DependencyIndexes: file_common_types_proto_depIdxs,
		EnumInfos:         file_common_types_proto_enumTypes,
		MessageInfos:      file_common_types_proto_msgTypes,
	}.Build()
	File_common_types_proto = out.File
	file_common_types_proto_rawDesc = nil
	file_common_types_proto_goTypes = nil
	file_common_types_proto_depIdxs = nil
}
