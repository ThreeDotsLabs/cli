// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.19.1
// source: server.proto

package genproto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type InitRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Token string `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"`
}

func (x *InitRequest) Reset() {
	*x = InitRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_server_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InitRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InitRequest) ProtoMessage() {}

func (x *InitRequest) ProtoReflect() protoreflect.Message {
	mi := &file_server_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InitRequest.ProtoReflect.Descriptor instead.
func (*InitRequest) Descriptor() ([]byte, []int) {
	return file_server_proto_rawDescGZIP(), []int{0}
}

func (x *InitRequest) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

type Training struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *Training) Reset() {
	*x = Training{}
	if protoimpl.UnsafeEnabled {
		mi := &file_server_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Training) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Training) ProtoMessage() {}

func (x *Training) ProtoReflect() protoreflect.Message {
	mi := &file_server_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Training.ProtoReflect.Descriptor instead.
func (*Training) Descriptor() ([]byte, []int) {
	return file_server_proto_rawDescGZIP(), []int{1}
}

func (x *Training) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type GetTrainingsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Trainings []*Training `protobuf:"bytes,1,rep,name=trainings,proto3" json:"trainings,omitempty"`
}

func (x *GetTrainingsResponse) Reset() {
	*x = GetTrainingsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_server_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetTrainingsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetTrainingsResponse) ProtoMessage() {}

func (x *GetTrainingsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_server_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetTrainingsResponse.ProtoReflect.Descriptor instead.
func (*GetTrainingsResponse) Descriptor() ([]byte, []int) {
	return file_server_proto_rawDescGZIP(), []int{2}
}

func (x *GetTrainingsResponse) GetTrainings() []*Training {
	if x != nil {
		return x.Trainings
	}
	return nil
}

type StartTrainingRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TrainingName string `protobuf:"bytes,1,opt,name=training_name,json=trainingName,proto3" json:"training_name,omitempty"`
	Token        string `protobuf:"bytes,2,opt,name=token,proto3" json:"token,omitempty"`
}

func (x *StartTrainingRequest) Reset() {
	*x = StartTrainingRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_server_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StartTrainingRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StartTrainingRequest) ProtoMessage() {}

func (x *StartTrainingRequest) ProtoReflect() protoreflect.Message {
	mi := &file_server_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StartTrainingRequest.ProtoReflect.Descriptor instead.
func (*StartTrainingRequest) Descriptor() ([]byte, []int) {
	return file_server_proto_rawDescGZIP(), []int{3}
}

func (x *StartTrainingRequest) GetTrainingName() string {
	if x != nil {
		return x.TrainingName
	}
	return ""
}

func (x *StartTrainingRequest) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

type StartTrainingResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *StartTrainingResponse) Reset() {
	*x = StartTrainingResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_server_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StartTrainingResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StartTrainingResponse) ProtoMessage() {}

func (x *StartTrainingResponse) ProtoReflect() protoreflect.Message {
	mi := &file_server_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StartTrainingResponse.ProtoReflect.Descriptor instead.
func (*StartTrainingResponse) Descriptor() ([]byte, []int) {
	return file_server_proto_rawDescGZIP(), []int{4}
}

type NextExerciseRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TrainingName      string `protobuf:"bytes,1,opt,name=training_name,json=trainingName,proto3" json:"training_name,omitempty"`
	CurrentExerciseId string `protobuf:"bytes,2,opt,name=current_exercise_id,json=currentExerciseId,proto3" json:"current_exercise_id,omitempty"`
	Token             string `protobuf:"bytes,3,opt,name=token,proto3" json:"token,omitempty"`
}

func (x *NextExerciseRequest) Reset() {
	*x = NextExerciseRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_server_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NextExerciseRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NextExerciseRequest) ProtoMessage() {}

func (x *NextExerciseRequest) ProtoReflect() protoreflect.Message {
	mi := &file_server_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NextExerciseRequest.ProtoReflect.Descriptor instead.
func (*NextExerciseRequest) Descriptor() ([]byte, []int) {
	return file_server_proto_rawDescGZIP(), []int{5}
}

func (x *NextExerciseRequest) GetTrainingName() string {
	if x != nil {
		return x.TrainingName
	}
	return ""
}

func (x *NextExerciseRequest) GetCurrentExerciseId() string {
	if x != nil {
		return x.CurrentExerciseId
	}
	return ""
}

func (x *NextExerciseRequest) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

type NextExerciseResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Dir           string  `protobuf:"bytes,1,opt,name=dir,proto3" json:"dir,omitempty"`
	ExerciseId    string  `protobuf:"bytes,2,opt,name=exercise_id,json=exerciseId,proto3" json:"exercise_id,omitempty"`
	FilesToCreate []*File `protobuf:"bytes,3,rep,name=files_to_create,json=filesToCreate,proto3" json:"files_to_create,omitempty"`
}

func (x *NextExerciseResponse) Reset() {
	*x = NextExerciseResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_server_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NextExerciseResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NextExerciseResponse) ProtoMessage() {}

func (x *NextExerciseResponse) ProtoReflect() protoreflect.Message {
	mi := &file_server_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NextExerciseResponse.ProtoReflect.Descriptor instead.
func (*NextExerciseResponse) Descriptor() ([]byte, []int) {
	return file_server_proto_rawDescGZIP(), []int{6}
}

func (x *NextExerciseResponse) GetDir() string {
	if x != nil {
		return x.Dir
	}
	return ""
}

func (x *NextExerciseResponse) GetExerciseId() string {
	if x != nil {
		return x.ExerciseId
	}
	return ""
}

func (x *NextExerciseResponse) GetFilesToCreate() []*File {
	if x != nil {
		return x.FilesToCreate
	}
	return nil
}

type NextExercise struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Dir           string  `protobuf:"bytes,1,opt,name=dir,proto3" json:"dir,omitempty"`
	FilesToCreate []*File `protobuf:"bytes,2,rep,name=files_to_create,json=filesToCreate,proto3" json:"files_to_create,omitempty"`
}

func (x *NextExercise) Reset() {
	*x = NextExercise{}
	if protoimpl.UnsafeEnabled {
		mi := &file_server_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NextExercise) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NextExercise) ProtoMessage() {}

func (x *NextExercise) ProtoReflect() protoreflect.Message {
	mi := &file_server_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NextExercise.ProtoReflect.Descriptor instead.
func (*NextExercise) Descriptor() ([]byte, []int) {
	return file_server_proto_rawDescGZIP(), []int{7}
}

func (x *NextExercise) GetDir() string {
	if x != nil {
		return x.Dir
	}
	return ""
}

func (x *NextExercise) GetFilesToCreate() []*File {
	if x != nil {
		return x.FilesToCreate
	}
	return nil
}

type VerifyExerciseRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ExerciseId string  `protobuf:"bytes,2,opt,name=exercise_id,json=exerciseId,proto3" json:"exercise_id,omitempty"`
	Files      []*File `protobuf:"bytes,3,rep,name=files,proto3" json:"files,omitempty"`
	Token      string  `protobuf:"bytes,4,opt,name=token,proto3" json:"token,omitempty"`
}

func (x *VerifyExerciseRequest) Reset() {
	*x = VerifyExerciseRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_server_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VerifyExerciseRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VerifyExerciseRequest) ProtoMessage() {}

func (x *VerifyExerciseRequest) ProtoReflect() protoreflect.Message {
	mi := &file_server_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VerifyExerciseRequest.ProtoReflect.Descriptor instead.
func (*VerifyExerciseRequest) Descriptor() ([]byte, []int) {
	return file_server_proto_rawDescGZIP(), []int{8}
}

func (x *VerifyExerciseRequest) GetExerciseId() string {
	if x != nil {
		return x.ExerciseId
	}
	return ""
}

func (x *VerifyExerciseRequest) GetFiles() []*File {
	if x != nil {
		return x.Files
	}
	return nil
}

func (x *VerifyExerciseRequest) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

type File struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Path    string `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	Content string `protobuf:"bytes,2,opt,name=content,proto3" json:"content,omitempty"`
}

func (x *File) Reset() {
	*x = File{}
	if protoimpl.UnsafeEnabled {
		mi := &file_server_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *File) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*File) ProtoMessage() {}

func (x *File) ProtoReflect() protoreflect.Message {
	mi := &file_server_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use File.ProtoReflect.Descriptor instead.
func (*File) Descriptor() ([]byte, []int) {
	return file_server_proto_rawDescGZIP(), []int{9}
}

func (x *File) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *File) GetContent() string {
	if x != nil {
		return x.Content
	}
	return ""
}

type VerifyExerciseResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Finished       bool   `protobuf:"varint,1,opt,name=finished,proto3" json:"finished,omitempty"`
	Successful     bool   `protobuf:"varint,2,opt,name=successful,proto3" json:"successful,omitempty"`
	Stdout         string `protobuf:"bytes,3,opt,name=stdout,proto3" json:"stdout,omitempty"`
	Stderr         string `protobuf:"bytes,4,opt,name=stderr,proto3" json:"stderr,omitempty"`
	LastExercise   bool   `protobuf:"varint,5,opt,name=last_exercise,json=lastExercise,proto3" json:"last_exercise,omitempty"`
	VerificationId string `protobuf:"bytes,6,opt,name=verification_id,json=verificationId,proto3" json:"verification_id,omitempty"`
}

func (x *VerifyExerciseResponse) Reset() {
	*x = VerifyExerciseResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_server_proto_msgTypes[10]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VerifyExerciseResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VerifyExerciseResponse) ProtoMessage() {}

func (x *VerifyExerciseResponse) ProtoReflect() protoreflect.Message {
	mi := &file_server_proto_msgTypes[10]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VerifyExerciseResponse.ProtoReflect.Descriptor instead.
func (*VerifyExerciseResponse) Descriptor() ([]byte, []int) {
	return file_server_proto_rawDescGZIP(), []int{10}
}

func (x *VerifyExerciseResponse) GetFinished() bool {
	if x != nil {
		return x.Finished
	}
	return false
}

func (x *VerifyExerciseResponse) GetSuccessful() bool {
	if x != nil {
		return x.Successful
	}
	return false
}

func (x *VerifyExerciseResponse) GetStdout() string {
	if x != nil {
		return x.Stdout
	}
	return ""
}

func (x *VerifyExerciseResponse) GetStderr() string {
	if x != nil {
		return x.Stderr
	}
	return ""
}

func (x *VerifyExerciseResponse) GetLastExercise() bool {
	if x != nil {
		return x.LastExercise
	}
	return false
}

func (x *VerifyExerciseResponse) GetVerificationId() string {
	if x != nil {
		return x.VerificationId
	}
	return ""
}

var File_server_proto protoreflect.FileDescriptor

var file_server_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f,
	0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x23, 0x0a, 0x0b, 0x49,
	0x6e, 0x69, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f,
	0x6b, 0x65, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
	0x22, 0x1a, 0x0a, 0x08, 0x54, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x12, 0x0e, 0x0a, 0x02,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x22, 0x3f, 0x0a, 0x14,
	0x47, 0x65, 0x74, 0x54, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x73, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x27, 0x0a, 0x09, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67,
	0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x09, 0x2e, 0x54, 0x72, 0x61, 0x69, 0x6e, 0x69,
	0x6e, 0x67, 0x52, 0x09, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x73, 0x22, 0x51, 0x0a,
	0x14, 0x53, 0x74, 0x61, 0x72, 0x74, 0x54, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x23, 0x0a, 0x0d, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e,
	0x67, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x74, 0x72,
	0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f,
	0x6b, 0x65, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
	0x22, 0x17, 0x0a, 0x15, 0x53, 0x74, 0x61, 0x72, 0x74, 0x54, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e,
	0x67, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x80, 0x01, 0x0a, 0x13, 0x4e, 0x65,
	0x78, 0x74, 0x45, 0x78, 0x65, 0x72, 0x63, 0x69, 0x73, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x23, 0x0a, 0x0d, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x5f, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x69,
	0x6e, 0x67, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x2e, 0x0a, 0x13, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e,
	0x74, 0x5f, 0x65, 0x78, 0x65, 0x72, 0x63, 0x69, 0x73, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x11, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x45, 0x78, 0x65, 0x72,
	0x63, 0x69, 0x73, 0x65, 0x49, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x22, 0x78, 0x0a, 0x14,
	0x4e, 0x65, 0x78, 0x74, 0x45, 0x78, 0x65, 0x72, 0x63, 0x69, 0x73, 0x65, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x64, 0x69, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x03, 0x64, 0x69, 0x72, 0x12, 0x1f, 0x0a, 0x0b, 0x65, 0x78, 0x65, 0x72, 0x63, 0x69,
	0x73, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x65, 0x78, 0x65,
	0x72, 0x63, 0x69, 0x73, 0x65, 0x49, 0x64, 0x12, 0x2d, 0x0a, 0x0f, 0x66, 0x69, 0x6c, 0x65, 0x73,
	0x5f, 0x74, 0x6f, 0x5f, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x05, 0x2e, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x0d, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x54, 0x6f,
	0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x22, 0x4f, 0x0a, 0x0c, 0x4e, 0x65, 0x78, 0x74, 0x45, 0x78,
	0x65, 0x72, 0x63, 0x69, 0x73, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x64, 0x69, 0x72, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x03, 0x64, 0x69, 0x72, 0x12, 0x2d, 0x0a, 0x0f, 0x66, 0x69, 0x6c, 0x65,
	0x73, 0x5f, 0x74, 0x6f, 0x5f, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x18, 0x02, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x05, 0x2e, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x0d, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x54,
	0x6f, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x22, 0x6b, 0x0a, 0x15, 0x56, 0x65, 0x72, 0x69, 0x66,
	0x79, 0x45, 0x78, 0x65, 0x72, 0x63, 0x69, 0x73, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x1f, 0x0a, 0x0b, 0x65, 0x78, 0x65, 0x72, 0x63, 0x69, 0x73, 0x65, 0x5f, 0x69, 0x64, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x65, 0x78, 0x65, 0x72, 0x63, 0x69, 0x73, 0x65, 0x49,
	0x64, 0x12, 0x1b, 0x0a, 0x05, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x05, 0x2e, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x05, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x12, 0x14,
	0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74,
	0x6f, 0x6b, 0x65, 0x6e, 0x22, 0x34, 0x0a, 0x04, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x12, 0x0a, 0x04,
	0x70, 0x61, 0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x61, 0x74, 0x68,
	0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x22, 0xd2, 0x01, 0x0a, 0x16, 0x56,
	0x65, 0x72, 0x69, 0x66, 0x79, 0x45, 0x78, 0x65, 0x72, 0x63, 0x69, 0x73, 0x65, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x66, 0x69, 0x6e, 0x69, 0x73, 0x68, 0x65,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x66, 0x69, 0x6e, 0x69, 0x73, 0x68, 0x65,
	0x64, 0x12, 0x1e, 0x0a, 0x0a, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x66, 0x75, 0x6c, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0a, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x66, 0x75,
	0x6c, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x74, 0x64, 0x6f, 0x75, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x06, 0x73, 0x74, 0x64, 0x6f, 0x75, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x74, 0x64,
	0x65, 0x72, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x74, 0x64, 0x65, 0x72,
	0x72, 0x12, 0x23, 0x0a, 0x0d, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x65, 0x78, 0x65, 0x72, 0x63, 0x69,
	0x73, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0c, 0x6c, 0x61, 0x73, 0x74, 0x45, 0x78,
	0x65, 0x72, 0x63, 0x69, 0x73, 0x65, 0x12, 0x27, 0x0a, 0x0f, 0x76, 0x65, 0x72, 0x69, 0x66, 0x69,
	0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0e, 0x76, 0x65, 0x72, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x32,
	0xc1, 0x02, 0x0a, 0x06, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x12, 0x2e, 0x0a, 0x04, 0x49, 0x6e,
	0x69, 0x74, 0x12, 0x0c, 0x2e, 0x49, 0x6e, 0x69, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x12, 0x3f, 0x0a, 0x0c, 0x47, 0x65,
	0x74, 0x54, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x73, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70,
	0x74, 0x79, 0x1a, 0x15, 0x2e, 0x47, 0x65, 0x74, 0x54, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67,
	0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x40, 0x0a, 0x0d, 0x53,
	0x74, 0x61, 0x72, 0x74, 0x54, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x12, 0x15, 0x2e, 0x53,
	0x74, 0x61, 0x72, 0x74, 0x54, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x12, 0x3d, 0x0a,
	0x0c, 0x4e, 0x65, 0x78, 0x74, 0x45, 0x78, 0x65, 0x72, 0x63, 0x69, 0x73, 0x65, 0x12, 0x14, 0x2e,
	0x4e, 0x65, 0x78, 0x74, 0x45, 0x78, 0x65, 0x72, 0x63, 0x69, 0x73, 0x65, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x15, 0x2e, 0x4e, 0x65, 0x78, 0x74, 0x45, 0x78, 0x65, 0x72, 0x63, 0x69,
	0x73, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x45, 0x0a, 0x0e,
	0x56, 0x65, 0x72, 0x69, 0x66, 0x79, 0x45, 0x78, 0x65, 0x72, 0x63, 0x69, 0x73, 0x65, 0x12, 0x16,
	0x2e, 0x56, 0x65, 0x72, 0x69, 0x66, 0x79, 0x45, 0x78, 0x65, 0x72, 0x63, 0x69, 0x73, 0x65, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x17, 0x2e, 0x56, 0x65, 0x72, 0x69, 0x66, 0x79, 0x45,
	0x78, 0x65, 0x72, 0x63, 0x69, 0x73, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22,
	0x00, 0x30, 0x01, 0x42, 0x39, 0x5a, 0x37, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x54, 0x68, 0x72, 0x65, 0x65, 0x44, 0x6f, 0x74, 0x73, 0x4c, 0x61, 0x62, 0x73, 0x2f,
	0x63, 0x6c, 0x69, 0x2f, 0x74, 0x64, 0x6c, 0x2d, 0x63, 0x6c, 0x69, 0x2f, 0x74, 0x72, 0x61, 0x69,
	0x6e, 0x69, 0x6e, 0x67, 0x73, 0x2f, 0x67, 0x65, 0x6e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_server_proto_rawDescOnce sync.Once
	file_server_proto_rawDescData = file_server_proto_rawDesc
)

func file_server_proto_rawDescGZIP() []byte {
	file_server_proto_rawDescOnce.Do(func() {
		file_server_proto_rawDescData = protoimpl.X.CompressGZIP(file_server_proto_rawDescData)
	})
	return file_server_proto_rawDescData
}

var file_server_proto_msgTypes = make([]protoimpl.MessageInfo, 11)
var file_server_proto_goTypes = []interface{}{
	(*InitRequest)(nil),            // 0: InitRequest
	(*Training)(nil),               // 1: Training
	(*GetTrainingsResponse)(nil),   // 2: GetTrainingsResponse
	(*StartTrainingRequest)(nil),   // 3: StartTrainingRequest
	(*StartTrainingResponse)(nil),  // 4: StartTrainingResponse
	(*NextExerciseRequest)(nil),    // 5: NextExerciseRequest
	(*NextExerciseResponse)(nil),   // 6: NextExerciseResponse
	(*NextExercise)(nil),           // 7: NextExercise
	(*VerifyExerciseRequest)(nil),  // 8: VerifyExerciseRequest
	(*File)(nil),                   // 9: File
	(*VerifyExerciseResponse)(nil), // 10: VerifyExerciseResponse
	(*emptypb.Empty)(nil),          // 11: google.protobuf.Empty
}
var file_server_proto_depIdxs = []int32{
	1,  // 0: GetTrainingsResponse.trainings:type_name -> Training
	9,  // 1: NextExerciseResponse.files_to_create:type_name -> File
	9,  // 2: NextExercise.files_to_create:type_name -> File
	9,  // 3: VerifyExerciseRequest.files:type_name -> File
	0,  // 4: Server.Init:input_type -> InitRequest
	11, // 5: Server.GetTrainings:input_type -> google.protobuf.Empty
	3,  // 6: Server.StartTraining:input_type -> StartTrainingRequest
	5,  // 7: Server.NextExercise:input_type -> NextExerciseRequest
	8,  // 8: Server.VerifyExercise:input_type -> VerifyExerciseRequest
	11, // 9: Server.Init:output_type -> google.protobuf.Empty
	2,  // 10: Server.GetTrainings:output_type -> GetTrainingsResponse
	11, // 11: Server.StartTraining:output_type -> google.protobuf.Empty
	6,  // 12: Server.NextExercise:output_type -> NextExerciseResponse
	10, // 13: Server.VerifyExercise:output_type -> VerifyExerciseResponse
	9,  // [9:14] is the sub-list for method output_type
	4,  // [4:9] is the sub-list for method input_type
	4,  // [4:4] is the sub-list for extension type_name
	4,  // [4:4] is the sub-list for extension extendee
	0,  // [0:4] is the sub-list for field type_name
}

func init() { file_server_proto_init() }
func file_server_proto_init() {
	if File_server_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_server_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InitRequest); i {
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
		file_server_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Training); i {
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
		file_server_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetTrainingsResponse); i {
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
		file_server_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StartTrainingRequest); i {
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
		file_server_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StartTrainingResponse); i {
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
		file_server_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NextExerciseRequest); i {
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
		file_server_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NextExerciseResponse); i {
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
		file_server_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NextExercise); i {
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
		file_server_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VerifyExerciseRequest); i {
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
		file_server_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*File); i {
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
		file_server_proto_msgTypes[10].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VerifyExerciseResponse); i {
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
			RawDescriptor: file_server_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   11,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_server_proto_goTypes,
		DependencyIndexes: file_server_proto_depIdxs,
		MessageInfos:      file_server_proto_msgTypes,
	}.Build()
	File_server_proto = out.File
	file_server_proto_rawDesc = nil
	file_server_proto_goTypes = nil
	file_server_proto_depIdxs = nil
}
