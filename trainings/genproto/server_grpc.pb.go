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

// TrainingsServerClient is the client API for TrainingsServer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type TrainingsServerClient interface {
	Init(ctx context.Context, in *InitRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	GetTrainings(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*GetTrainingsResponse, error)
	StartTraining(ctx context.Context, in *StartTrainingRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	NextExercise(ctx context.Context, in *NextExerciseRequest, opts ...grpc.CallOption) (*NextExerciseResponse, error)
	VerifyExercise(ctx context.Context, in *VerifyExerciseRequest, opts ...grpc.CallOption) (TrainingsServer_VerifyExerciseClient, error)
}

type trainingsServerClient struct {
	cc grpc.ClientConnInterface
}

func NewTrainingsServerClient(cc grpc.ClientConnInterface) TrainingsServerClient {
	return &trainingsServerClient{cc}
}

func (c *trainingsServerClient) Init(ctx context.Context, in *InitRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/TrainingsServer/Init", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trainingsServerClient) GetTrainings(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*GetTrainingsResponse, error) {
	out := new(GetTrainingsResponse)
	err := c.cc.Invoke(ctx, "/TrainingsServer/GetTrainings", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trainingsServerClient) StartTraining(ctx context.Context, in *StartTrainingRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/TrainingsServer/StartTraining", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trainingsServerClient) NextExercise(ctx context.Context, in *NextExerciseRequest, opts ...grpc.CallOption) (*NextExerciseResponse, error) {
	out := new(NextExerciseResponse)
	err := c.cc.Invoke(ctx, "/TrainingsServer/NextExercise", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trainingsServerClient) VerifyExercise(ctx context.Context, in *VerifyExerciseRequest, opts ...grpc.CallOption) (TrainingsServer_VerifyExerciseClient, error) {
	stream, err := c.cc.NewStream(ctx, &TrainingsServer_ServiceDesc.Streams[0], "/TrainingsServer/VerifyExercise", opts...)
	if err != nil {
		return nil, err
	}
	x := &trainingsServerVerifyExerciseClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type TrainingsServer_VerifyExerciseClient interface {
	Recv() (*VerifyExerciseResponse, error)
	grpc.ClientStream
}

type trainingsServerVerifyExerciseClient struct {
	grpc.ClientStream
}

func (x *trainingsServerVerifyExerciseClient) Recv() (*VerifyExerciseResponse, error) {
	m := new(VerifyExerciseResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// TrainingsServerServer is the server API for TrainingsServer service.
// All implementations should embed UnimplementedTrainingsServerServer
// for forward compatibility
type TrainingsServerServer interface {
	Init(context.Context, *InitRequest) (*empty.Empty, error)
	GetTrainings(context.Context, *empty.Empty) (*GetTrainingsResponse, error)
	StartTraining(context.Context, *StartTrainingRequest) (*empty.Empty, error)
	NextExercise(context.Context, *NextExerciseRequest) (*NextExerciseResponse, error)
	VerifyExercise(*VerifyExerciseRequest, TrainingsServer_VerifyExerciseServer) error
}

// UnimplementedTrainingsServerServer should be embedded to have forward compatible implementations.
type UnimplementedTrainingsServerServer struct {
}

func (UnimplementedTrainingsServerServer) Init(context.Context, *InitRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Init not implemented")
}
func (UnimplementedTrainingsServerServer) GetTrainings(context.Context, *empty.Empty) (*GetTrainingsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTrainings not implemented")
}
func (UnimplementedTrainingsServerServer) StartTraining(context.Context, *StartTrainingRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartTraining not implemented")
}
func (UnimplementedTrainingsServerServer) NextExercise(context.Context, *NextExerciseRequest) (*NextExerciseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NextExercise not implemented")
}
func (UnimplementedTrainingsServerServer) VerifyExercise(*VerifyExerciseRequest, TrainingsServer_VerifyExerciseServer) error {
	return status.Errorf(codes.Unimplemented, "method VerifyExercise not implemented")
}

// UnsafeTrainingsServerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to TrainingsServerServer will
// result in compilation errors.
type UnsafeTrainingsServerServer interface {
	mustEmbedUnimplementedTrainingsServerServer()
}

func RegisterTrainingsServerServer(s grpc.ServiceRegistrar, srv TrainingsServerServer) {
	s.RegisterService(&TrainingsServer_ServiceDesc, srv)
}

func _TrainingsServer_Init_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrainingsServerServer).Init(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/TrainingsServer/Init",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrainingsServerServer).Init(ctx, req.(*InitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _TrainingsServer_GetTrainings_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(empty.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrainingsServerServer).GetTrainings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/TrainingsServer/GetTrainings",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrainingsServerServer).GetTrainings(ctx, req.(*empty.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _TrainingsServer_StartTraining_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartTrainingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrainingsServerServer).StartTraining(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/TrainingsServer/StartTraining",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrainingsServerServer).StartTraining(ctx, req.(*StartTrainingRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _TrainingsServer_NextExercise_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NextExerciseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrainingsServerServer).NextExercise(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/TrainingsServer/NextExercise",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrainingsServerServer).NextExercise(ctx, req.(*NextExerciseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _TrainingsServer_VerifyExercise_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(VerifyExerciseRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(TrainingsServerServer).VerifyExercise(m, &trainingsServerVerifyExerciseServer{stream})
}

type TrainingsServer_VerifyExerciseServer interface {
	Send(*VerifyExerciseResponse) error
	grpc.ServerStream
}

type trainingsServerVerifyExerciseServer struct {
	grpc.ServerStream
}

func (x *trainingsServerVerifyExerciseServer) Send(m *VerifyExerciseResponse) error {
	return x.ServerStream.SendMsg(m)
}

// TrainingsServer_ServiceDesc is the grpc.ServiceDesc for TrainingsServer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var TrainingsServer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "TrainingsServer",
	HandlerType: (*TrainingsServerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Init",
			Handler:    _TrainingsServer_Init_Handler,
		},
		{
			MethodName: "GetTrainings",
			Handler:    _TrainingsServer_GetTrainings_Handler,
		},
		{
			MethodName: "StartTraining",
			Handler:    _TrainingsServer_StartTraining_Handler,
		},
		{
			MethodName: "NextExercise",
			Handler:    _TrainingsServer_NextExercise_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "VerifyExercise",
			Handler:       _TrainingsServer_VerifyExercise_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "server.proto",
}
