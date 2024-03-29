// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.19.1
// source: consensus.proto

package consensus

import (
	types "github.com/ginchuco/ginchu/types"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type MsgType int32

const (
	MsgType_UNKNOWN        MsgType = 0
	MsgType_PREPARE        MsgType = 1
	MsgType_PREPARE_VOTE   MsgType = 2
	MsgType_PRECOMMIT      MsgType = 3
	MsgType_PRECOMMIT_VOTE MsgType = 4
	MsgType_COMMIT         MsgType = 5
	MsgType_COMMIT_VOTE    MsgType = 6
	MsgType_DECIDE         MsgType = 7
	MsgType_NEW_VIEW       MsgType = 8
)

// Enum value maps for MsgType.
var (
	MsgType_name = map[int32]string{
		0: "UNKNOWN",
		1: "PREPARE",
		2: "PREPARE_VOTE",
		3: "PRECOMMIT",
		4: "PRECOMMIT_VOTE",
		5: "COMMIT",
		6: "COMMIT_VOTE",
		7: "DECIDE",
		8: "NEW_VIEW",
	}
	MsgType_value = map[string]int32{
		"UNKNOWN":        0,
		"PREPARE":        1,
		"PREPARE_VOTE":   2,
		"PRECOMMIT":      3,
		"PRECOMMIT_VOTE": 4,
		"COMMIT":         5,
		"COMMIT_VOTE":    6,
		"DECIDE":         7,
		"NEW_VIEW":       8,
	}
)

func (x MsgType) Enum() *MsgType {
	p := new(MsgType)
	*p = x
	return p
}

func (x MsgType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (MsgType) Descriptor() protoreflect.EnumDescriptor {
	return file_consensus_proto_enumTypes[0].Descriptor()
}

func (MsgType) Type() protoreflect.EnumType {
	return &file_consensus_proto_enumTypes[0]
}

func (x MsgType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use MsgType.Descriptor instead.
func (MsgType) EnumDescriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{0}
}

type QuorumCert struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Height    uint64  `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	ViewNum   uint64  `protobuf:"varint,2,opt,name=viewNum,proto3" json:"viewNum,omitempty"`
	BlockHash []byte  `protobuf:"bytes,3,opt,name=BlockHash,proto3" json:"BlockHash,omitempty"`
	Type      MsgType `protobuf:"varint,4,opt,name=type,proto3,enum=types.MsgType" json:"type,omitempty"`
	Signature []byte  `protobuf:"bytes,5,opt,name=signature,proto3" json:"signature,omitempty"`
	VrfOuts   []byte  `protobuf:"bytes,6,opt,name=vrf_outs,json=vrfOuts,proto3" json:"vrf_outs,omitempty"`
}

func (x *QuorumCert) Reset() {
	*x = QuorumCert{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QuorumCert) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QuorumCert) ProtoMessage() {}

func (x *QuorumCert) ProtoReflect() protoreflect.Message {
	mi := &file_consensus_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QuorumCert.ProtoReflect.Descriptor instead.
func (*QuorumCert) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{0}
}

func (x *QuorumCert) GetHeight() uint64 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *QuorumCert) GetViewNum() uint64 {
	if x != nil {
		return x.ViewNum
	}
	return 0
}

func (x *QuorumCert) GetBlockHash() []byte {
	if x != nil {
		return x.BlockHash
	}
	return nil
}

func (x *QuorumCert) GetType() MsgType {
	if x != nil {
		return x.Type
	}
	return MsgType_UNKNOWN
}

func (x *QuorumCert) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

func (x *QuorumCert) GetVrfOuts() []byte {
	if x != nil {
		return x.VrfOuts
	}
	return nil
}

type VRFs struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Vrfs []*VRF `protobuf:"bytes,1,rep,name=vrfs,proto3" json:"vrfs,omitempty"`
}

func (x *VRFs) Reset() {
	*x = VRFs{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VRFs) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VRFs) ProtoMessage() {}

func (x *VRFs) ProtoReflect() protoreflect.Message {
	mi := &file_consensus_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VRFs.ProtoReflect.Descriptor instead.
func (*VRFs) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{1}
}

func (x *VRFs) GetVrfs() []*VRF {
	if x != nil {
		return x.Vrfs
	}
	return nil
}

type VRF struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	PublicKey []byte `protobuf:"bytes,1,opt,name=PublicKey,proto3" json:"PublicKey,omitempty"`
	Out       []byte `protobuf:"bytes,2,opt,name=Out,proto3" json:"Out,omitempty"`
}

func (x *VRF) Reset() {
	*x = VRF{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VRF) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VRF) ProtoMessage() {}

func (x *VRF) ProtoReflect() protoreflect.Message {
	mi := &file_consensus_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VRF.ProtoReflect.Descriptor instead.
func (*VRF) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{2}
}

func (x *VRF) GetPublicKey() []byte {
	if x != nil {
		return x.PublicKey
	}
	return nil
}

func (x *VRF) GetOut() []byte {
	if x != nil {
		return x.Out
	}
	return nil
}

type Prepare struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Height      uint64       `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	ViewNum     uint64       `protobuf:"varint,2,opt,name=viewNum,proto3" json:"viewNum,omitempty"`
	CurProposal *types.Block `protobuf:"bytes,3,opt,name=curProposal,proto3" json:"curProposal,omitempty"`
	HighQC      *QuorumCert  `protobuf:"bytes,4,opt,name=highQC,proto3" json:"highQC,omitempty"`
}

func (x *Prepare) Reset() {
	*x = Prepare{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Prepare) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Prepare) ProtoMessage() {}

func (x *Prepare) ProtoReflect() protoreflect.Message {
	mi := &file_consensus_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Prepare.ProtoReflect.Descriptor instead.
func (*Prepare) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{3}
}

func (x *Prepare) GetHeight() uint64 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *Prepare) GetViewNum() uint64 {
	if x != nil {
		return x.ViewNum
	}
	return 0
}

func (x *Prepare) GetCurProposal() *types.Block {
	if x != nil {
		return x.CurProposal
	}
	return nil
}

func (x *Prepare) GetHighQC() *QuorumCert {
	if x != nil {
		return x.HighQC
	}
	return nil
}

type PrepareVote struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Height     uint64           `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	ViewNum    uint64           `protobuf:"varint,2,opt,name=viewNum,proto3" json:"viewNum,omitempty"`
	BlockHash  []byte           `protobuf:"bytes,3,opt,name=blockHash,proto3" json:"blockHash,omitempty"`
	Qc         *QuorumCert      `protobuf:"bytes,4,opt,name=qc,proto3" json:"qc,omitempty"`
	PartialSig *types.Signature `protobuf:"bytes,5,opt,name=partialSig,proto3" json:"partialSig,omitempty"`
}

func (x *PrepareVote) Reset() {
	*x = PrepareVote{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PrepareVote) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PrepareVote) ProtoMessage() {}

func (x *PrepareVote) ProtoReflect() protoreflect.Message {
	mi := &file_consensus_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PrepareVote.ProtoReflect.Descriptor instead.
func (*PrepareVote) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{4}
}

func (x *PrepareVote) GetHeight() uint64 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *PrepareVote) GetViewNum() uint64 {
	if x != nil {
		return x.ViewNum
	}
	return 0
}

func (x *PrepareVote) GetBlockHash() []byte {
	if x != nil {
		return x.BlockHash
	}
	return nil
}

func (x *PrepareVote) GetQc() *QuorumCert {
	if x != nil {
		return x.Qc
	}
	return nil
}

func (x *PrepareVote) GetPartialSig() *types.Signature {
	if x != nil {
		return x.PartialSig
	}
	return nil
}

type PreCommit struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Height    uint64      `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	ViewNum   uint64      `protobuf:"varint,2,opt,name=viewNum,proto3" json:"viewNum,omitempty"`
	PrepareQC *QuorumCert `protobuf:"bytes,3,opt,name=prepareQC,proto3" json:"prepareQC,omitempty"`
}

func (x *PreCommit) Reset() {
	*x = PreCommit{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PreCommit) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PreCommit) ProtoMessage() {}

func (x *PreCommit) ProtoReflect() protoreflect.Message {
	mi := &file_consensus_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PreCommit.ProtoReflect.Descriptor instead.
func (*PreCommit) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{5}
}

func (x *PreCommit) GetHeight() uint64 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *PreCommit) GetViewNum() uint64 {
	if x != nil {
		return x.ViewNum
	}
	return 0
}

func (x *PreCommit) GetPrepareQC() *QuorumCert {
	if x != nil {
		return x.PrepareQC
	}
	return nil
}

type PreCommitVote struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Height     uint64           `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	ViewNum    uint64           `protobuf:"varint,2,opt,name=viewNum,proto3" json:"viewNum,omitempty"`
	BlockHash  []byte           `protobuf:"bytes,3,opt,name=blockHash,proto3" json:"blockHash,omitempty"`
	Qc         *QuorumCert      `protobuf:"bytes,4,opt,name=qc,proto3" json:"qc,omitempty"`
	PartialSig *types.Signature `protobuf:"bytes,5,opt,name=partialSig,proto3" json:"partialSig,omitempty"`
}

func (x *PreCommitVote) Reset() {
	*x = PreCommitVote{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PreCommitVote) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PreCommitVote) ProtoMessage() {}

func (x *PreCommitVote) ProtoReflect() protoreflect.Message {
	mi := &file_consensus_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PreCommitVote.ProtoReflect.Descriptor instead.
func (*PreCommitVote) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{6}
}

func (x *PreCommitVote) GetHeight() uint64 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *PreCommitVote) GetViewNum() uint64 {
	if x != nil {
		return x.ViewNum
	}
	return 0
}

func (x *PreCommitVote) GetBlockHash() []byte {
	if x != nil {
		return x.BlockHash
	}
	return nil
}

func (x *PreCommitVote) GetQc() *QuorumCert {
	if x != nil {
		return x.Qc
	}
	return nil
}

func (x *PreCommitVote) GetPartialSig() *types.Signature {
	if x != nil {
		return x.PartialSig
	}
	return nil
}

type Commit struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Height      uint64      `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	ViewNum     uint64      `protobuf:"varint,2,opt,name=viewNum,proto3" json:"viewNum,omitempty"`
	PreCommitQC *QuorumCert `protobuf:"bytes,3,opt,name=preCommitQC,proto3" json:"preCommitQC,omitempty"`
}

func (x *Commit) Reset() {
	*x = Commit{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Commit) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Commit) ProtoMessage() {}

func (x *Commit) ProtoReflect() protoreflect.Message {
	mi := &file_consensus_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Commit.ProtoReflect.Descriptor instead.
func (*Commit) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{7}
}

func (x *Commit) GetHeight() uint64 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *Commit) GetViewNum() uint64 {
	if x != nil {
		return x.ViewNum
	}
	return 0
}

func (x *Commit) GetPreCommitQC() *QuorumCert {
	if x != nil {
		return x.PreCommitQC
	}
	return nil
}

type CommitVote struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Height     uint64           `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	ViewNum    uint64           `protobuf:"varint,2,opt,name=viewNum,proto3" json:"viewNum,omitempty"`
	BlockHash  []byte           `protobuf:"bytes,3,opt,name=blockHash,proto3" json:"blockHash,omitempty"`
	Qc         *QuorumCert      `protobuf:"bytes,4,opt,name=qc,proto3" json:"qc,omitempty"`
	PartialSig *types.Signature `protobuf:"bytes,5,opt,name=partialSig,proto3" json:"partialSig,omitempty"`
}

func (x *CommitVote) Reset() {
	*x = CommitVote{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CommitVote) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CommitVote) ProtoMessage() {}

func (x *CommitVote) ProtoReflect() protoreflect.Message {
	mi := &file_consensus_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CommitVote.ProtoReflect.Descriptor instead.
func (*CommitVote) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{8}
}

func (x *CommitVote) GetHeight() uint64 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *CommitVote) GetViewNum() uint64 {
	if x != nil {
		return x.ViewNum
	}
	return 0
}

func (x *CommitVote) GetBlockHash() []byte {
	if x != nil {
		return x.BlockHash
	}
	return nil
}

func (x *CommitVote) GetQc() *QuorumCert {
	if x != nil {
		return x.Qc
	}
	return nil
}

func (x *CommitVote) GetPartialSig() *types.Signature {
	if x != nil {
		return x.PartialSig
	}
	return nil
}

type Decide struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Height   uint64      `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	ViewNum  uint64      `protobuf:"varint,2,opt,name=viewNum,proto3" json:"viewNum,omitempty"`
	CommitQC *QuorumCert `protobuf:"bytes,3,opt,name=commitQC,proto3" json:"commitQC,omitempty"`
}

func (x *Decide) Reset() {
	*x = Decide{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Decide) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Decide) ProtoMessage() {}

func (x *Decide) ProtoReflect() protoreflect.Message {
	mi := &file_consensus_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Decide.ProtoReflect.Descriptor instead.
func (*Decide) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{9}
}

func (x *Decide) GetHeight() uint64 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *Decide) GetViewNum() uint64 {
	if x != nil {
		return x.ViewNum
	}
	return 0
}

func (x *Decide) GetCommitQC() *QuorumCert {
	if x != nil {
		return x.CommitQC
	}
	return nil
}

type NewView struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Height     uint64           `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	ViewNum    uint64           `protobuf:"varint,2,opt,name=viewNum,proto3" json:"viewNum,omitempty"`
	PrepareQC  *QuorumCert      `protobuf:"bytes,3,opt,name=prepareQC,proto3" json:"prepareQC,omitempty"`
	PartialSig *types.Signature `protobuf:"bytes,4,opt,name=partialSig,proto3" json:"partialSig,omitempty"`
}

func (x *NewView) Reset() {
	*x = NewView{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[10]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NewView) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NewView) ProtoMessage() {}

func (x *NewView) ProtoReflect() protoreflect.Message {
	mi := &file_consensus_proto_msgTypes[10]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NewView.ProtoReflect.Descriptor instead.
func (*NewView) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{10}
}

func (x *NewView) GetHeight() uint64 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *NewView) GetViewNum() uint64 {
	if x != nil {
		return x.ViewNum
	}
	return 0
}

func (x *NewView) GetPrepareQC() *QuorumCert {
	if x != nil {
		return x.PrepareQC
	}
	return nil
}

func (x *NewView) GetPartialSig() *types.Signature {
	if x != nil {
		return x.PartialSig
	}
	return nil
}

var File_consensus_proto protoreflect.FileDescriptor

var file_consensus_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x63, 0x6f, 0x6e, 0x73, 0x65, 0x6e, 0x73, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x05, 0x74, 0x79, 0x70, 0x65, 0x73, 0x1a, 0x0b, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x08, 0x74, 0x78, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0xb9, 0x01, 0x0a, 0x0a, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x12, 0x16,
	0x0a, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06,
	0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75,
	0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d,
	0x12, 0x1c, 0x0a, 0x09, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x09, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x12, 0x22,
	0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0e, 0x2e, 0x74,
	0x79, 0x70, 0x65, 0x73, 0x2e, 0x4d, 0x73, 0x67, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79,
	0x70, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65,
	0x12, 0x19, 0x0a, 0x08, 0x76, 0x72, 0x66, 0x5f, 0x6f, 0x75, 0x74, 0x73, 0x18, 0x06, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x07, 0x76, 0x72, 0x66, 0x4f, 0x75, 0x74, 0x73, 0x22, 0x26, 0x0a, 0x04, 0x56,
	0x52, 0x46, 0x73, 0x12, 0x1e, 0x0a, 0x04, 0x76, 0x72, 0x66, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x0a, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x56, 0x52, 0x46, 0x52, 0x04, 0x76,
	0x72, 0x66, 0x73, 0x22, 0x35, 0x0a, 0x03, 0x56, 0x52, 0x46, 0x12, 0x1c, 0x0a, 0x09, 0x50, 0x75,
	0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x50,
	0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x4f, 0x75, 0x74, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x03, 0x4f, 0x75, 0x74, 0x22, 0x96, 0x01, 0x0a, 0x07, 0x50,
	0x72, 0x65, 0x70, 0x61, 0x72, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x18,
	0x0a, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d, 0x12, 0x2e, 0x0a, 0x0b, 0x63, 0x75, 0x72, 0x50,
	0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0c, 0x2e,
	0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x0b, 0x63, 0x75, 0x72,
	0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x12, 0x29, 0x0a, 0x06, 0x68, 0x69, 0x67, 0x68,
	0x51, 0x43, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73,
	0x2e, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x52, 0x06, 0x68, 0x69, 0x67,
	0x68, 0x51, 0x43, 0x22, 0xb2, 0x01, 0x0a, 0x0b, 0x50, 0x72, 0x65, 0x70, 0x61, 0x72, 0x65, 0x56,
	0x6f, 0x74, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x04, 0x52, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x76,
	0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x76, 0x69,
	0x65, 0x77, 0x4e, 0x75, 0x6d, 0x12, 0x1c, 0x0a, 0x09, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x61,
	0x73, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x48,
	0x61, 0x73, 0x68, 0x12, 0x21, 0x0a, 0x02, 0x71, 0x63, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x11, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43, 0x65,
	0x72, 0x74, 0x52, 0x02, 0x71, 0x63, 0x12, 0x30, 0x0a, 0x0a, 0x70, 0x61, 0x72, 0x74, 0x69, 0x61,
	0x6c, 0x53, 0x69, 0x67, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x74, 0x79, 0x70,
	0x65, 0x73, 0x2e, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x52, 0x0a, 0x70, 0x61,
	0x72, 0x74, 0x69, 0x61, 0x6c, 0x53, 0x69, 0x67, 0x22, 0x6e, 0x0a, 0x09, 0x50, 0x72, 0x65, 0x43,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x18, 0x0a,
	0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x07,
	0x76, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d, 0x12, 0x2f, 0x0a, 0x09, 0x70, 0x72, 0x65, 0x70, 0x61,
	0x72, 0x65, 0x51, 0x43, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x74, 0x79, 0x70,
	0x65, 0x73, 0x2e, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x52, 0x09, 0x70,
	0x72, 0x65, 0x70, 0x61, 0x72, 0x65, 0x51, 0x43, 0x22, 0xb4, 0x01, 0x0a, 0x0d, 0x50, 0x72, 0x65,
	0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x56, 0x6f, 0x74, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x68, 0x65,
	0x69, 0x67, 0x68, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x68, 0x65, 0x69, 0x67,
	0x68, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x04, 0x52, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d, 0x12, 0x1c, 0x0a, 0x09,
	0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x09, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x12, 0x21, 0x0a, 0x02, 0x71, 0x63,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x51,
	0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x52, 0x02, 0x71, 0x63, 0x12, 0x30, 0x0a,
	0x0a, 0x70, 0x61, 0x72, 0x74, 0x69, 0x61, 0x6c, 0x53, 0x69, 0x67, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x10, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74,
	0x75, 0x72, 0x65, 0x52, 0x0a, 0x70, 0x61, 0x72, 0x74, 0x69, 0x61, 0x6c, 0x53, 0x69, 0x67, 0x22,
	0x6f, 0x0a, 0x06, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x68, 0x65, 0x69,
	0x67, 0x68, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68,
	0x74, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x04, 0x52, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d, 0x12, 0x33, 0x0a, 0x0b, 0x70,
	0x72, 0x65, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x51, 0x43, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x11, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43,
	0x65, 0x72, 0x74, 0x52, 0x0b, 0x70, 0x72, 0x65, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x51, 0x43,
	0x22, 0xb1, 0x01, 0x0a, 0x0a, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x56, 0x6f, 0x74, 0x65, 0x12,
	0x16, 0x0a, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e,
	0x75, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75,
	0x6d, 0x12, 0x1c, 0x0a, 0x09, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x12,
	0x21, 0x0a, 0x02, 0x71, 0x63, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x74, 0x79,
	0x70, 0x65, 0x73, 0x2e, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x52, 0x02,
	0x71, 0x63, 0x12, 0x30, 0x0a, 0x0a, 0x70, 0x61, 0x72, 0x74, 0x69, 0x61, 0x6c, 0x53, 0x69, 0x67,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x53,
	0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x52, 0x0a, 0x70, 0x61, 0x72, 0x74, 0x69, 0x61,
	0x6c, 0x53, 0x69, 0x67, 0x22, 0x69, 0x0a, 0x06, 0x44, 0x65, 0x63, 0x69, 0x64, 0x65, 0x12, 0x16,
	0x0a, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06,
	0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75,
	0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d,
	0x12, 0x2d, 0x0a, 0x08, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x51, 0x43, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x11, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x51, 0x75, 0x6f, 0x72, 0x75,
	0x6d, 0x43, 0x65, 0x72, 0x74, 0x52, 0x08, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x51, 0x43, 0x22,
	0x9e, 0x01, 0x0a, 0x07, 0x4e, 0x65, 0x77, 0x56, 0x69, 0x65, 0x77, 0x12, 0x16, 0x0a, 0x06, 0x68,
	0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x68, 0x65, 0x69,
	0x67, 0x68, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d, 0x12, 0x2f, 0x0a,
	0x09, 0x70, 0x72, 0x65, 0x70, 0x61, 0x72, 0x65, 0x51, 0x43, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x11, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43,
	0x65, 0x72, 0x74, 0x52, 0x09, 0x70, 0x72, 0x65, 0x70, 0x61, 0x72, 0x65, 0x51, 0x43, 0x12, 0x30,
	0x0a, 0x0a, 0x70, 0x61, 0x72, 0x74, 0x69, 0x61, 0x6c, 0x53, 0x69, 0x67, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x10, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x53, 0x69, 0x67, 0x6e, 0x61,
	0x74, 0x75, 0x72, 0x65, 0x52, 0x0a, 0x70, 0x61, 0x72, 0x74, 0x69, 0x61, 0x6c, 0x53, 0x69, 0x67,
	0x2a, 0x8f, 0x01, 0x0a, 0x07, 0x4d, 0x73, 0x67, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0b, 0x0a, 0x07,
	0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x50, 0x52, 0x45,
	0x50, 0x41, 0x52, 0x45, 0x10, 0x01, 0x12, 0x10, 0x0a, 0x0c, 0x50, 0x52, 0x45, 0x50, 0x41, 0x52,
	0x45, 0x5f, 0x56, 0x4f, 0x54, 0x45, 0x10, 0x02, 0x12, 0x0d, 0x0a, 0x09, 0x50, 0x52, 0x45, 0x43,
	0x4f, 0x4d, 0x4d, 0x49, 0x54, 0x10, 0x03, 0x12, 0x12, 0x0a, 0x0e, 0x50, 0x52, 0x45, 0x43, 0x4f,
	0x4d, 0x4d, 0x49, 0x54, 0x5f, 0x56, 0x4f, 0x54, 0x45, 0x10, 0x04, 0x12, 0x0a, 0x0a, 0x06, 0x43,
	0x4f, 0x4d, 0x4d, 0x49, 0x54, 0x10, 0x05, 0x12, 0x0f, 0x0a, 0x0b, 0x43, 0x4f, 0x4d, 0x4d, 0x49,
	0x54, 0x5f, 0x56, 0x4f, 0x54, 0x45, 0x10, 0x06, 0x12, 0x0a, 0x0a, 0x06, 0x44, 0x45, 0x43, 0x49,
	0x44, 0x45, 0x10, 0x07, 0x12, 0x0c, 0x0a, 0x08, 0x4e, 0x45, 0x57, 0x5f, 0x56, 0x49, 0x45, 0x57,
	0x10, 0x08, 0x42, 0x26, 0x5a, 0x24, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x67, 0x69, 0x6e, 0x63, 0x68, 0x75, 0x63, 0x6f, 0x2f, 0x67, 0x69, 0x6e, 0x63, 0x68, 0x75,
	0x2f, 0x63, 0x6f, 0x6e, 0x73, 0x65, 0x6e, 0x73, 0x75, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_consensus_proto_rawDescOnce sync.Once
	file_consensus_proto_rawDescData = file_consensus_proto_rawDesc
)

func file_consensus_proto_rawDescGZIP() []byte {
	file_consensus_proto_rawDescOnce.Do(func() {
		file_consensus_proto_rawDescData = protoimpl.X.CompressGZIP(file_consensus_proto_rawDescData)
	})
	return file_consensus_proto_rawDescData
}

var file_consensus_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_consensus_proto_msgTypes = make([]protoimpl.MessageInfo, 11)
var file_consensus_proto_goTypes = []interface{}{
	(MsgType)(0),            // 0: types.MsgType
	(*QuorumCert)(nil),      // 1: types.QuorumCert
	(*VRFs)(nil),            // 2: types.VRFs
	(*VRF)(nil),             // 3: types.VRF
	(*Prepare)(nil),         // 4: types.Prepare
	(*PrepareVote)(nil),     // 5: types.PrepareVote
	(*PreCommit)(nil),       // 6: types.PreCommit
	(*PreCommitVote)(nil),   // 7: types.PreCommitVote
	(*Commit)(nil),          // 8: types.Commit
	(*CommitVote)(nil),      // 9: types.CommitVote
	(*Decide)(nil),          // 10: types.Decide
	(*NewView)(nil),         // 11: types.NewView
	(*types.Block)(nil),     // 12: types.Block
	(*types.Signature)(nil), // 13: types.Signature
}
var file_consensus_proto_depIdxs = []int32{
	0,  // 0: types.QuorumCert.type:type_name -> types.MsgType
	3,  // 1: types.VRFs.vrfs:type_name -> types.VRF
	12, // 2: types.Prepare.curProposal:type_name -> types.Block
	1,  // 3: types.Prepare.highQC:type_name -> types.QuorumCert
	1,  // 4: types.PrepareVote.qc:type_name -> types.QuorumCert
	13, // 5: types.PrepareVote.partialSig:type_name -> types.Signature
	1,  // 6: types.PreCommit.prepareQC:type_name -> types.QuorumCert
	1,  // 7: types.PreCommitVote.qc:type_name -> types.QuorumCert
	13, // 8: types.PreCommitVote.partialSig:type_name -> types.Signature
	1,  // 9: types.Commit.preCommitQC:type_name -> types.QuorumCert
	1,  // 10: types.CommitVote.qc:type_name -> types.QuorumCert
	13, // 11: types.CommitVote.partialSig:type_name -> types.Signature
	1,  // 12: types.Decide.commitQC:type_name -> types.QuorumCert
	1,  // 13: types.NewView.prepareQC:type_name -> types.QuorumCert
	13, // 14: types.NewView.partialSig:type_name -> types.Signature
	15, // [15:15] is the sub-list for method output_type
	15, // [15:15] is the sub-list for method input_type
	15, // [15:15] is the sub-list for extension type_name
	15, // [15:15] is the sub-list for extension extendee
	0,  // [0:15] is the sub-list for field type_name
}

func init() { file_consensus_proto_init() }
func file_consensus_proto_init() {
	if File_consensus_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_consensus_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QuorumCert); i {
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
		file_consensus_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VRFs); i {
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
		file_consensus_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VRF); i {
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
		file_consensus_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Prepare); i {
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
		file_consensus_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PrepareVote); i {
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
		file_consensus_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PreCommit); i {
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
		file_consensus_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PreCommitVote); i {
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
		file_consensus_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Commit); i {
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
		file_consensus_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CommitVote); i {
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
		file_consensus_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Decide); i {
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
		file_consensus_proto_msgTypes[10].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NewView); i {
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
			RawDescriptor: file_consensus_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   11,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_consensus_proto_goTypes,
		DependencyIndexes: file_consensus_proto_depIdxs,
		EnumInfos:         file_consensus_proto_enumTypes,
		MessageInfos:      file_consensus_proto_msgTypes,
	}.Build()
	File_consensus_proto = out.File
	file_consensus_proto_rawDesc = nil
	file_consensus_proto_goTypes = nil
	file_consensus_proto_depIdxs = nil
}
