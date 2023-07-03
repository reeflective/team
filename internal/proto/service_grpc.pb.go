// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.18.1
// source: service.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Team_GetVersion_FullMethodName = "/client.Team/GetVersion"
	Team_ClientLog_FullMethodName  = "/client.Team/ClientLog"
	Team_GetUsers_FullMethodName   = "/client.Team/GetUsers"
)

// TeamClient is the client API for Team service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type TeamClient interface {
	GetVersion(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Version, error)
	ClientLog(ctx context.Context, opts ...grpc.CallOption) (Team_ClientLogClient, error)
	GetUsers(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Users, error)
}

type teamClient struct {
	cc grpc.ClientConnInterface
}

func NewTeamClient(cc grpc.ClientConnInterface) TeamClient {
	return &teamClient{cc}
}

func (c *teamClient) GetVersion(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Version, error) {
	out := new(Version)
	err := c.cc.Invoke(ctx, Team_GetVersion_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *teamClient) ClientLog(ctx context.Context, opts ...grpc.CallOption) (Team_ClientLogClient, error) {
	stream, err := c.cc.NewStream(ctx, &Team_ServiceDesc.Streams[0], Team_ClientLog_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &teamClientLogClient{stream}
	return x, nil
}

type Team_ClientLogClient interface {
	Send(*LogData) error
	CloseAndRecv() (*Empty, error)
	grpc.ClientStream
}

type teamClientLogClient struct {
	grpc.ClientStream
}

func (x *teamClientLogClient) Send(m *LogData) error {
	return x.ClientStream.SendMsg(m)
}

func (x *teamClientLogClient) CloseAndRecv() (*Empty, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(Empty)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *teamClient) GetUsers(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Users, error) {
	out := new(Users)
	err := c.cc.Invoke(ctx, Team_GetUsers_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// TeamServer is the server API for Team service.
// All implementations must embed UnimplementedTeamServer
// for forward compatibility
type TeamServer interface {
	GetVersion(context.Context, *Empty) (*Version, error)
	ClientLog(Team_ClientLogServer) error
	GetUsers(context.Context, *Empty) (*Users, error)
	mustEmbedUnimplementedTeamServer()
}

// UnimplementedTeamServer must be embedded to have forward compatible implementations.
type UnimplementedTeamServer struct {
}

func (UnimplementedTeamServer) GetVersion(context.Context, *Empty) (*Version, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVersion not implemented")
}
func (UnimplementedTeamServer) ClientLog(Team_ClientLogServer) error {
	return status.Errorf(codes.Unimplemented, "method ClientLog not implemented")
}
func (UnimplementedTeamServer) GetUsers(context.Context, *Empty) (*Users, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUsers not implemented")
}
func (UnimplementedTeamServer) mustEmbedUnimplementedTeamServer() {}

// UnsafeTeamServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to TeamServer will
// result in compilation errors.
type UnsafeTeamServer interface {
	mustEmbedUnimplementedTeamServer()
}

func RegisterTeamServer(s grpc.ServiceRegistrar, srv TeamServer) {
	s.RegisterService(&Team_ServiceDesc, srv)
}

func _Team_GetVersion_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TeamServer).GetVersion(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Team_GetVersion_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TeamServer).GetVersion(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Team_ClientLog_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(TeamServer).ClientLog(&teamClientLogServer{stream})
}

type Team_ClientLogServer interface {
	SendAndClose(*Empty) error
	Recv() (*LogData, error)
	grpc.ServerStream
}

type teamClientLogServer struct {
	grpc.ServerStream
}

func (x *teamClientLogServer) SendAndClose(m *Empty) error {
	return x.ServerStream.SendMsg(m)
}

func (x *teamClientLogServer) Recv() (*LogData, error) {
	m := new(LogData)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _Team_GetUsers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TeamServer).GetUsers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Team_GetUsers_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TeamServer).GetUsers(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// Team_ServiceDesc is the grpc.ServiceDesc for Team service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Team_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "client.Team",
	HandlerType: (*TeamServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetVersion",
			Handler:    _Team_GetVersion_Handler,
		},
		{
			MethodName: "GetUsers",
			Handler:    _Team_GetUsers_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "ClientLog",
			Handler:       _Team_ClientLog_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "service.proto",
}
