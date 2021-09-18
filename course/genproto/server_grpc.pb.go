// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package genproto

import (
	context "context"
	empty "github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ServerClient is the client API for Server service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ServerClient interface {
	GetCourses(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*GetCoursesResponse, error)
	StartCourse(ctx context.Context, in *StartCourseRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	NextExercise(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*NextExerciseResponse, error)
	VerifyExercise(ctx context.Context, in *VerifyExerciseRequest, opts ...grpc.CallOption) (Server_VerifyExerciseClient, error)
}

type serverClient struct {
	cc grpc.ClientConnInterface
}

func NewServerClient(cc grpc.ClientConnInterface) ServerClient {
	return &serverClient{cc}
}

func (c *serverClient) GetCourses(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*GetCoursesResponse, error) {
	out := new(GetCoursesResponse)
	err := c.cc.Invoke(ctx, "/Server/GetCourses", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serverClient) StartCourse(ctx context.Context, in *StartCourseRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/Server/StartCourse", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serverClient) NextExercise(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*NextExerciseResponse, error) {
	out := new(NextExerciseResponse)
	err := c.cc.Invoke(ctx, "/Server/NextExercise", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serverClient) VerifyExercise(ctx context.Context, in *VerifyExerciseRequest, opts ...grpc.CallOption) (Server_VerifyExerciseClient, error) {
	stream, err := c.cc.NewStream(ctx, &Server_ServiceDesc.Streams[0], "/Server/VerifyExercise", opts...)
	if err != nil {
		return nil, err
	}
	x := &serverVerifyExerciseClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Server_VerifyExerciseClient interface {
	Recv() (*VerifyExerciseResponse, error)
	grpc.ClientStream
}

type serverVerifyExerciseClient struct {
	grpc.ClientStream
}

func (x *serverVerifyExerciseClient) Recv() (*VerifyExerciseResponse, error) {
	m := new(VerifyExerciseResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ServerServer is the server API for Server service.
// All implementations should embed UnimplementedServerServer
// for forward compatibility
type ServerServer interface {
	GetCourses(context.Context, *empty.Empty) (*GetCoursesResponse, error)
	StartCourse(context.Context, *StartCourseRequest) (*empty.Empty, error)
	NextExercise(context.Context, *empty.Empty) (*NextExerciseResponse, error)
	VerifyExercise(*VerifyExerciseRequest, Server_VerifyExerciseServer) error
}

// UnimplementedServerServer should be embedded to have forward compatible implementations.
type UnimplementedServerServer struct {
}

func (UnimplementedServerServer) GetCourses(context.Context, *empty.Empty) (*GetCoursesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCourses not implemented")
}
func (UnimplementedServerServer) StartCourse(context.Context, *StartCourseRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartCourse not implemented")
}
func (UnimplementedServerServer) NextExercise(context.Context, *empty.Empty) (*NextExerciseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NextExercise not implemented")
}
func (UnimplementedServerServer) VerifyExercise(*VerifyExerciseRequest, Server_VerifyExerciseServer) error {
	return status.Errorf(codes.Unimplemented, "method VerifyExercise not implemented")
}

// UnsafeServerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ServerServer will
// result in compilation errors.
type UnsafeServerServer interface {
	mustEmbedUnimplementedServerServer()
}

func RegisterServerServer(s grpc.ServiceRegistrar, srv ServerServer) {
	s.RegisterService(&Server_ServiceDesc, srv)
}

func _Server_GetCourses_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(empty.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServerServer).GetCourses(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Server/GetCourses",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServerServer).GetCourses(ctx, req.(*empty.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Server_StartCourse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartCourseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServerServer).StartCourse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Server/StartCourse",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServerServer).StartCourse(ctx, req.(*StartCourseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Server_NextExercise_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(empty.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServerServer).NextExercise(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Server/NextExercise",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServerServer).NextExercise(ctx, req.(*empty.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Server_VerifyExercise_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(VerifyExerciseRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ServerServer).VerifyExercise(m, &serverVerifyExerciseServer{stream})
}

type Server_VerifyExerciseServer interface {
	Send(*VerifyExerciseResponse) error
	grpc.ServerStream
}

type serverVerifyExerciseServer struct {
	grpc.ServerStream
}

func (x *serverVerifyExerciseServer) Send(m *VerifyExerciseResponse) error {
	return x.ServerStream.SendMsg(m)
}

// Server_ServiceDesc is the grpc.ServiceDesc for Server service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Server_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "Server",
	HandlerType: (*ServerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetCourses",
			Handler:    _Server_GetCourses_Handler,
		},
		{
			MethodName: "StartCourse",
			Handler:    _Server_StartCourse_Handler,
		},
		{
			MethodName: "NextExercise",
			Handler:    _Server_NextExercise_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "VerifyExercise",
			Handler:       _Server_VerifyExercise_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "server.proto",
}
