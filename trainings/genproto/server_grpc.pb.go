// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.12
// source: server.proto

package genproto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// TrainingsClient is the client API for Trainings service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type TrainingsClient interface {
	Init(ctx context.Context, in *InitRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	GetTrainings(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*GetTrainingsResponse, error)
	StartTraining(ctx context.Context, in *StartTrainingRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	NextExercise(ctx context.Context, in *NextExerciseRequest, opts ...grpc.CallOption) (*NextExerciseResponse, error)
	VerifyExercise(ctx context.Context, in *VerifyExerciseRequest, opts ...grpc.CallOption) (Trainings_VerifyExerciseClient, error)
}

type trainingsClient struct {
	cc grpc.ClientConnInterface
}

func NewTrainingsClient(cc grpc.ClientConnInterface) TrainingsClient {
	return &trainingsClient{cc}
}

func (c *trainingsClient) Init(ctx context.Context, in *InitRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/Trainings/Init", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trainingsClient) GetTrainings(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*GetTrainingsResponse, error) {
	out := new(GetTrainingsResponse)
	err := c.cc.Invoke(ctx, "/Trainings/GetTrainings", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trainingsClient) StartTraining(ctx context.Context, in *StartTrainingRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/Trainings/StartTraining", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trainingsClient) NextExercise(ctx context.Context, in *NextExerciseRequest, opts ...grpc.CallOption) (*NextExerciseResponse, error) {
	out := new(NextExerciseResponse)
	err := c.cc.Invoke(ctx, "/Trainings/NextExercise", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trainingsClient) VerifyExercise(ctx context.Context, in *VerifyExerciseRequest, opts ...grpc.CallOption) (Trainings_VerifyExerciseClient, error) {
	stream, err := c.cc.NewStream(ctx, &Trainings_ServiceDesc.Streams[0], "/Trainings/VerifyExercise", opts...)
	if err != nil {
		return nil, err
	}
	x := &trainingsVerifyExerciseClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Trainings_VerifyExerciseClient interface {
	Recv() (*VerifyExerciseResponse, error)
	grpc.ClientStream
}

type trainingsVerifyExerciseClient struct {
	grpc.ClientStream
}

func (x *trainingsVerifyExerciseClient) Recv() (*VerifyExerciseResponse, error) {
	m := new(VerifyExerciseResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// TrainingsServer is the server API for Trainings service.
// All implementations should embed UnimplementedTrainingsServer
// for forward compatibility
type TrainingsServer interface {
	Init(context.Context, *InitRequest) (*emptypb.Empty, error)
	GetTrainings(context.Context, *emptypb.Empty) (*GetTrainingsResponse, error)
	StartTraining(context.Context, *StartTrainingRequest) (*emptypb.Empty, error)
	NextExercise(context.Context, *NextExerciseRequest) (*NextExerciseResponse, error)
	VerifyExercise(*VerifyExerciseRequest, Trainings_VerifyExerciseServer) error
}

// UnimplementedTrainingsServer should be embedded to have forward compatible implementations.
type UnimplementedTrainingsServer struct {
}

func (UnimplementedTrainingsServer) Init(context.Context, *InitRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Init not implemented")
}
func (UnimplementedTrainingsServer) GetTrainings(context.Context, *emptypb.Empty) (*GetTrainingsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTrainings not implemented")
}
func (UnimplementedTrainingsServer) StartTraining(context.Context, *StartTrainingRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartTraining not implemented")
}
func (UnimplementedTrainingsServer) NextExercise(context.Context, *NextExerciseRequest) (*NextExerciseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NextExercise not implemented")
}
func (UnimplementedTrainingsServer) VerifyExercise(*VerifyExerciseRequest, Trainings_VerifyExerciseServer) error {
	return status.Errorf(codes.Unimplemented, "method VerifyExercise not implemented")
}

// UnsafeTrainingsServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to TrainingsServer will
// result in compilation errors.
type UnsafeTrainingsServer interface {
	mustEmbedUnimplementedTrainingsServer()
}

func RegisterTrainingsServer(s grpc.ServiceRegistrar, srv TrainingsServer) {
	s.RegisterService(&Trainings_ServiceDesc, srv)
}

func _Trainings_Init_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrainingsServer).Init(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Trainings/Init",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrainingsServer).Init(ctx, req.(*InitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Trainings_GetTrainings_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrainingsServer).GetTrainings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Trainings/GetTrainings",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrainingsServer).GetTrainings(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Trainings_StartTraining_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartTrainingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrainingsServer).StartTraining(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Trainings/StartTraining",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrainingsServer).StartTraining(ctx, req.(*StartTrainingRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Trainings_NextExercise_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NextExerciseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrainingsServer).NextExercise(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Trainings/NextExercise",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrainingsServer).NextExercise(ctx, req.(*NextExerciseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Trainings_VerifyExercise_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(VerifyExerciseRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(TrainingsServer).VerifyExercise(m, &trainingsVerifyExerciseServer{stream})
}

type Trainings_VerifyExerciseServer interface {
	Send(*VerifyExerciseResponse) error
	grpc.ServerStream
}

type trainingsVerifyExerciseServer struct {
	grpc.ServerStream
}

func (x *trainingsVerifyExerciseServer) Send(m *VerifyExerciseResponse) error {
	return x.ServerStream.SendMsg(m)
}

// Trainings_ServiceDesc is the grpc.ServiceDesc for Trainings service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Trainings_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "Trainings",
	HandlerType: (*TrainingsServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Init",
			Handler:    _Trainings_Init_Handler,
		},
		{
			MethodName: "GetTrainings",
			Handler:    _Trainings_GetTrainings_Handler,
		},
		{
			MethodName: "StartTraining",
			Handler:    _Trainings_StartTraining_Handler,
		},
		{
			MethodName: "NextExercise",
			Handler:    _Trainings_NextExercise_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "VerifyExercise",
			Handler:       _Trainings_VerifyExercise_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "server.proto",
}
