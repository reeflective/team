// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v3.18.1
// source: client.proto

package proto

import (
	reflect "reflect"
	sync "sync"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// [ Client & User ]
type Client struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ID   uint32 `protobuf:"varint,1,opt,name=ID,proto3" json:"ID,omitempty"`
	Name string `protobuf:"bytes,2,opt,name=Name,proto3" json:"Name,omitempty"`
	User *User  `protobuf:"bytes,3,opt,name=User,proto3" json:"User,omitempty"`
}

func (x *Client) Reset() {
	*x = Client{}
	if protoimpl.UnsafeEnabled {
		mi := &file_client_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Client) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Client) ProtoMessage() {}

func (x *Client) ProtoReflect() protoreflect.Message {
	mi := &file_client_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Client.ProtoReflect.Descriptor instead.
func (*Client) Descriptor() ([]byte, []int) {
	return file_client_proto_rawDescGZIP(), []int{0}
}

func (x *Client) GetID() uint32 {
	if x != nil {
		return x.ID
	}
	return 0
}

func (x *Client) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Client) GetUser() *User {
	if x != nil {
		return x.User
	}
	return nil
}

type Users struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Users []*User `protobuf:"bytes,1,rep,name=Users,proto3" json:"Users,omitempty"`
}

func (x *Users) Reset() {
	*x = Users{}
	if protoimpl.UnsafeEnabled {
		mi := &file_client_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Users) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Users) ProtoMessage() {}

func (x *Users) ProtoReflect() protoreflect.Message {
	mi := &file_client_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Users.ProtoReflect.Descriptor instead.
func (*Users) Descriptor() ([]byte, []int) {
	return file_client_proto_rawDescGZIP(), []int{1}
}

func (x *Users) GetUsers() []*User {
	if x != nil {
		return x.Users
	}
	return nil
}

type User struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name     string `protobuf:"bytes,1,opt,name=Name,proto3" json:"Name,omitempty"`
	Online   bool   `protobuf:"varint,2,opt,name=Online,proto3" json:"Online,omitempty"`
	LastSeen int64  `protobuf:"varint,3,opt,name=LastSeen,proto3" json:"LastSeen,omitempty"`
	Clients  int32  `protobuf:"varint,4,opt,name=Clients,proto3" json:"Clients,omitempty"`
}

func (x *User) Reset() {
	*x = User{}
	if protoimpl.UnsafeEnabled {
		mi := &file_client_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *User) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User) ProtoMessage() {}

func (x *User) ProtoReflect() protoreflect.Message {
	mi := &file_client_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User.ProtoReflect.Descriptor instead.
func (*User) Descriptor() ([]byte, []int) {
	return file_client_proto_rawDescGZIP(), []int{2}
}

func (x *User) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *User) GetOnline() bool {
	if x != nil {
		return x.Online
	}
	return false
}

func (x *User) GetLastSeen() int64 {
	if x != nil {
		return x.LastSeen
	}
	return 0
}

func (x *User) GetClients() int32 {
	if x != nil {
		return x.Clients
	}
	return 0
}

type Version struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Major      int32  `protobuf:"varint,1,opt,name=Major,proto3" json:"Major,omitempty"`
	Minor      int32  `protobuf:"varint,2,opt,name=Minor,proto3" json:"Minor,omitempty"`
	Patch      int32  `protobuf:"varint,3,opt,name=Patch,proto3" json:"Patch,omitempty"`
	Commit     string `protobuf:"bytes,4,opt,name=Commit,proto3" json:"Commit,omitempty"`
	Dirty      bool   `protobuf:"varint,5,opt,name=Dirty,proto3" json:"Dirty,omitempty"`
	CompiledAt int64  `protobuf:"varint,6,opt,name=CompiledAt,proto3" json:"CompiledAt,omitempty"`
	OS         string `protobuf:"bytes,7,opt,name=OS,proto3" json:"OS,omitempty"`
	Arch       string `protobuf:"bytes,8,opt,name=Arch,proto3" json:"Arch,omitempty"`
}

func (x *Version) Reset() {
	*x = Version{}
	if protoimpl.UnsafeEnabled {
		mi := &file_client_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Version) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Version) ProtoMessage() {}

func (x *Version) ProtoReflect() protoreflect.Message {
	mi := &file_client_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Version.ProtoReflect.Descriptor instead.
func (*Version) Descriptor() ([]byte, []int) {
	return file_client_proto_rawDescGZIP(), []int{3}
}

func (x *Version) GetMajor() int32 {
	if x != nil {
		return x.Major
	}
	return 0
}

func (x *Version) GetMinor() int32 {
	if x != nil {
		return x.Minor
	}
	return 0
}

func (x *Version) GetPatch() int32 {
	if x != nil {
		return x.Patch
	}
	return 0
}

func (x *Version) GetCommit() string {
	if x != nil {
		return x.Commit
	}
	return ""
}

func (x *Version) GetDirty() bool {
	if x != nil {
		return x.Dirty
	}
	return false
}

func (x *Version) GetCompiledAt() int64 {
	if x != nil {
		return x.CompiledAt
	}
	return 0
}

func (x *Version) GetOS() string {
	if x != nil {
		return x.OS
	}
	return ""
}

func (x *Version) GetArch() string {
	if x != nil {
		return x.Arch
	}
	return ""
}

// [ Others ]
type Empty struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Empty) Reset() {
	*x = Empty{}
	if protoimpl.UnsafeEnabled {
		mi := &file_client_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_client_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Empty.ProtoReflect.Descriptor instead.
func (*Empty) Descriptor() ([]byte, []int) {
	return file_client_proto_rawDescGZIP(), []int{4}
}

var File_client_proto protoreflect.FileDescriptor

var file_client_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06,
	0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x22, 0x4e, 0x0a, 0x06, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74,
	0x12, 0x0e, 0x0a, 0x02, 0x49, 0x44, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x02, 0x49, 0x44,
	0x12, 0x12, 0x0a, 0x04, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x4e, 0x61, 0x6d, 0x65, 0x12, 0x20, 0x0a, 0x04, 0x55, 0x73, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x2e, 0x55, 0x73, 0x65, 0x72,
	0x52, 0x04, 0x55, 0x73, 0x65, 0x72, 0x22, 0x2b, 0x0a, 0x05, 0x55, 0x73, 0x65, 0x72, 0x73, 0x12,
	0x22, 0x0a, 0x05, 0x55, 0x73, 0x65, 0x72, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0c,
	0x2e, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x52, 0x05, 0x55, 0x73,
	0x65, 0x72, 0x73, 0x22, 0x68, 0x0a, 0x04, 0x55, 0x73, 0x65, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x4e,
	0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x4e, 0x61, 0x6d, 0x65, 0x12,
	0x16, 0x0a, 0x06, 0x4f, 0x6e, 0x6c, 0x69, 0x6e, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x06, 0x4f, 0x6e, 0x6c, 0x69, 0x6e, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x4c, 0x61, 0x73, 0x74, 0x53,
	0x65, 0x65, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x4c, 0x61, 0x73, 0x74, 0x53,
	0x65, 0x65, 0x6e, 0x12, 0x18, 0x0a, 0x07, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x73, 0x22, 0xbd, 0x01,
	0x0a, 0x07, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x14, 0x0a, 0x05, 0x4d, 0x61, 0x6a,
	0x6f, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x4d, 0x61, 0x6a, 0x6f, 0x72, 0x12,
	0x14, 0x0a, 0x05, 0x4d, 0x69, 0x6e, 0x6f, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05,
	0x4d, 0x69, 0x6e, 0x6f, 0x72, 0x12, 0x14, 0x0a, 0x05, 0x50, 0x61, 0x74, 0x63, 0x68, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x50, 0x61, 0x74, 0x63, 0x68, 0x12, 0x16, 0x0a, 0x06, 0x43,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x43, 0x6f, 0x6d,
	0x6d, 0x69, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x44, 0x69, 0x72, 0x74, 0x79, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x05, 0x44, 0x69, 0x72, 0x74, 0x79, 0x12, 0x1e, 0x0a, 0x0a, 0x43, 0x6f, 0x6d,
	0x70, 0x69, 0x6c, 0x65, 0x64, 0x41, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0a, 0x43,
	0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x65, 0x64, 0x41, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x4f, 0x53, 0x18,
	0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x4f, 0x53, 0x12, 0x12, 0x0a, 0x04, 0x41, 0x72, 0x63,
	0x68, 0x18, 0x08, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x41, 0x72, 0x63, 0x68, 0x22, 0x07, 0x0a,
	0x05, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x42, 0x84, 0x01, 0x0a, 0x0a, 0x63, 0x6f, 0x6d, 0x2e, 0x63,
	0x6c, 0x69, 0x65, 0x6e, 0x74, 0x42, 0x0b, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x50, 0x72, 0x6f,
	0x74, 0x6f, 0x50, 0x01, 0x5a, 0x31, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x72, 0x65, 0x65, 0x66, 0x6c, 0x65, 0x63, 0x74, 0x69, 0x76, 0x65, 0x2f, 0x74, 0x65, 0x61,
	0x6d, 0x2f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x2f, 0x67, 0x72, 0x70,
	0x63, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0xa2, 0x02, 0x03, 0x43, 0x58, 0x58, 0xaa, 0x02, 0x06,
	0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0xca, 0x02, 0x06, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0xe2,
	0x02, 0x12, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61,
	0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x06, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_client_proto_rawDescOnce sync.Once
	file_client_proto_rawDescData = file_client_proto_rawDesc
)

func file_client_proto_rawDescGZIP() []byte {
	file_client_proto_rawDescOnce.Do(func() {
		file_client_proto_rawDescData = protoimpl.X.CompressGZIP(file_client_proto_rawDescData)
	})
	return file_client_proto_rawDescData
}

var file_client_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_client_proto_goTypes = []interface{}{
	(*Client)(nil),  // 0: client.Client
	(*Users)(nil),   // 1: client.Users
	(*User)(nil),    // 2: client.User
	(*Version)(nil), // 3: client.Version
	(*Empty)(nil),   // 4: client.Empty
}
var file_client_proto_depIdxs = []int32{
	2, // 0: client.Client.User:type_name -> client.User
	2, // 1: client.Users.Users:type_name -> client.User
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_client_proto_init() }
func file_client_proto_init() {
	if File_client_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_client_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Client); i {
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
		file_client_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Users); i {
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
		file_client_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*User); i {
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
		file_client_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Version); i {
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
		file_client_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Empty); i {
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
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_client_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_client_proto_goTypes,
		DependencyIndexes: file_client_proto_depIdxs,
		MessageInfos:      file_client_proto_msgTypes,
	}.Build()
	File_client_proto = out.File
	file_client_proto_rawDesc = nil
	file_client_proto_goTypes = nil
	file_client_proto_depIdxs = nil
}
