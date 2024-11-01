// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v3.19.1
// source: consensus.proto

package lib

import (
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

type Phase int32

const (
	Phase_UNKNOWN         Phase = 0
	Phase_ELECTION        Phase = 1
	Phase_ELECTION_VOTE   Phase = 2
	Phase_PROPOSE         Phase = 3
	Phase_PROPOSE_VOTE    Phase = 4
	Phase_PRECOMMIT       Phase = 5
	Phase_PRECOMMIT_VOTE  Phase = 6
	Phase_COMMIT          Phase = 7
	Phase_COMMIT_PROCESS  Phase = 8
	Phase_ROUND_INTERRUPT Phase = 9
	Phase_PACEMAKER       Phase = 10
)

// Enum value maps for Phase.
var (
	Phase_name = map[int32]string{
		0:  "UNKNOWN",
		1:  "ELECTION",
		2:  "ELECTION_VOTE",
		3:  "PROPOSE",
		4:  "PROPOSE_VOTE",
		5:  "PRECOMMIT",
		6:  "PRECOMMIT_VOTE",
		7:  "COMMIT",
		8:  "COMMIT_PROCESS",
		9:  "ROUND_INTERRUPT",
		10: "PACEMAKER",
	}
	Phase_value = map[string]int32{
		"UNKNOWN":         0,
		"ELECTION":        1,
		"ELECTION_VOTE":   2,
		"PROPOSE":         3,
		"PROPOSE_VOTE":    4,
		"PRECOMMIT":       5,
		"PRECOMMIT_VOTE":  6,
		"COMMIT":          7,
		"COMMIT_PROCESS":  8,
		"ROUND_INTERRUPT": 9,
		"PACEMAKER":       10,
	}
)

func (x Phase) Enum() *Phase {
	p := new(Phase)
	*p = x
	return p
}

func (x Phase) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Phase) Descriptor() protoreflect.EnumDescriptor {
	return file_consensus_proto_enumTypes[0].Descriptor()
}

func (Phase) Type() protoreflect.EnumType {
	return &file_consensus_proto_enumTypes[0]
}

func (x Phase) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Phase.Descriptor instead.
func (Phase) EnumDescriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{0}
}

type Proposers struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Addresses [][]byte `protobuf:"bytes,1,rep,name=addresses,proto3" json:"addresses,omitempty"`
}

func (x *Proposers) Reset() {
	*x = Proposers{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Proposers) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Proposers) ProtoMessage() {}

func (x *Proposers) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use Proposers.ProtoReflect.Descriptor instead.
func (*Proposers) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{0}
}

func (x *Proposers) GetAddresses() [][]byte {
	if x != nil {
		return x.Addresses
	}
	return nil
}

type QuorumCertificate struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Header      *View               `protobuf:"bytes,1,opt,name=header,proto3" json:"header,omitempty"`                              // replica vote view
	Results     *CertificateResult  `protobuf:"bytes,2,opt,name=results,proto3" json:"results,omitempty"`                            // used for PROPOSE
	ResultsHash []byte              `protobuf:"bytes,3,opt,name=results_hash,json=resultsHash,proto3" json:"results_hash,omitempty"` // used after PROPOSE
	Block       []byte              `protobuf:"bytes,4,opt,name=block,proto3" json:"block,omitempty"`                                // used for PROPOSE
	BlockHash   []byte              `protobuf:"bytes,5,opt,name=block_hash,json=blockHash,proto3" json:"block_hash,omitempty"`       // used after PROPOSE
	ProposerKey []byte              `protobuf:"bytes,6,opt,name=proposer_key,json=proposerKey,proto3" json:"proposer_key,omitempty"` // only EV and PROPOSE
	Signature   *AggregateSignature `protobuf:"bytes,7,opt,name=signature,proto3" json:"signature,omitempty"`                        // aggregate signature from the current proposer message
}

func (x *QuorumCertificate) Reset() {
	*x = QuorumCertificate{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QuorumCertificate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QuorumCertificate) ProtoMessage() {}

func (x *QuorumCertificate) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use QuorumCertificate.ProtoReflect.Descriptor instead.
func (*QuorumCertificate) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{1}
}

func (x *QuorumCertificate) GetHeader() *View {
	if x != nil {
		return x.Header
	}
	return nil
}

func (x *QuorumCertificate) GetResults() *CertificateResult {
	if x != nil {
		return x.Results
	}
	return nil
}

func (x *QuorumCertificate) GetResultsHash() []byte {
	if x != nil {
		return x.ResultsHash
	}
	return nil
}

func (x *QuorumCertificate) GetBlock() []byte {
	if x != nil {
		return x.Block
	}
	return nil
}

func (x *QuorumCertificate) GetBlockHash() []byte {
	if x != nil {
		return x.BlockHash
	}
	return nil
}

func (x *QuorumCertificate) GetProposerKey() []byte {
	if x != nil {
		return x.ProposerKey
	}
	return nil
}

func (x *QuorumCertificate) GetSignature() *AggregateSignature {
	if x != nil {
		return x.Signature
	}
	return nil
}

type View struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// the height of the integrated chain
	// number of committed blocks in the integrated blockchain
	Height uint64 `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	// step within the consensus protocol where validators attempt to agree on the next block
	// each round a new Proposer is selected to lead the validators to agree on the next block
	// if they fail, the round is incremented, more time is granted for consensus timeouts, and the
	// cycle starts over
	Round uint64 `protobuf:"varint,2,opt,name=round,proto3" json:"round,omitempty"`
	// represents the smallest unit in the consensus process. Each round consists of multiple phases, and these phases are
	// executed sequentially to achieve consensus on the next block.
	// ELECTION->ELECTION-VOTE->PROPOSE->PROPOSE-VOTE->PRECOMMIT->PRECOMMIT-VOTE->COMMIT->COMMIT-PROCESS
	Phase Phase `protobuf:"varint,3,opt,name=phase,proto3,enum=types.Phase" json:"phase,omitempty"`
	// the Canopy chain height also the height that the committee validator set may be verified
	CommitteeHeight uint64 `protobuf:"varint,4,opt,name=committee_height,json=committeeHeight,proto3" json:"committee_height,omitempty"`
	// the identifier of the network preventing cross-play between different networks (testnet / mainnet / forks)
	NetworkId uint64 `protobuf:"varint,5,opt,name=network_id,json=networkId,proto3" json:"network_id,omitempty"`
	// maps to a specific committee on Canopy
	CommitteeId uint64 `protobuf:"varint,6,opt,name=committee_id,json=committeeId,proto3" json:"committee_id,omitempty"`
}

func (x *View) Reset() {
	*x = View{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *View) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*View) ProtoMessage() {}

func (x *View) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use View.ProtoReflect.Descriptor instead.
func (*View) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{2}
}

func (x *View) GetHeight() uint64 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *View) GetRound() uint64 {
	if x != nil {
		return x.Round
	}
	return 0
}

func (x *View) GetPhase() Phase {
	if x != nil {
		return x.Phase
	}
	return Phase_UNKNOWN
}

func (x *View) GetCommitteeHeight() uint64 {
	if x != nil {
		return x.CommitteeHeight
	}
	return 0
}

func (x *View) GetNetworkId() uint64 {
	if x != nil {
		return x.NetworkId
	}
	return 0
}

func (x *View) GetCommitteeId() uint64 {
	if x != nil {
		return x.CommitteeId
	}
	return 0
}

// Verifiable Delay Function
type VDF struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Proof      []byte `protobuf:"bytes,1,opt,name=proof,proto3" json:"proof,omitempty"`            // proof of function completion given a specific seed
	Iterations uint64 `protobuf:"varint,2,opt,name=iterations,proto3" json:"iterations,omitempty"` // number of iterations (proxy for time)
}

func (x *VDF) Reset() {
	*x = VDF{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VDF) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VDF) ProtoMessage() {}

func (x *VDF) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use VDF.ProtoReflect.Descriptor instead.
func (*VDF) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{3}
}

func (x *VDF) GetProof() []byte {
	if x != nil {
		return x.Proof
	}
	return nil
}

func (x *VDF) GetIterations() uint64 {
	if x != nil {
		return x.Iterations
	}
	return 0
}

type AggregateSignature struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Signature []byte `protobuf:"bytes,1,opt,name=signature,proto3" json:"signature,omitempty"`
	Bitmap    []byte `protobuf:"bytes,2,opt,name=bitmap,proto3" json:"bitmap,omitempty"`
}

func (x *AggregateSignature) Reset() {
	*x = AggregateSignature{}
	if protoimpl.UnsafeEnabled {
		mi := &file_consensus_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AggregateSignature) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AggregateSignature) ProtoMessage() {}

func (x *AggregateSignature) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use AggregateSignature.ProtoReflect.Descriptor instead.
func (*AggregateSignature) Descriptor() ([]byte, []int) {
	return file_consensus_proto_rawDescGZIP(), []int{4}
}

func (x *AggregateSignature) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

func (x *AggregateSignature) GetBitmap() []byte {
	if x != nil {
		return x.Bitmap
	}
	return nil
}

var File_consensus_proto protoreflect.FileDescriptor

var file_consensus_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x63, 0x6f, 0x6e, 0x73, 0x65, 0x6e, 0x73, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x05, 0x74, 0x79, 0x70, 0x65, 0x73, 0x1a, 0x0e, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73,
	0x61, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x29, 0x0a, 0x09, 0x50, 0x72, 0x6f, 0x70,
	0x6f, 0x73, 0x65, 0x72, 0x73, 0x12, 0x1c, 0x0a, 0x09, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73,
	0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x09, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73,
	0x73, 0x65, 0x73, 0x22, 0xa0, 0x02, 0x0a, 0x11, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43, 0x65,
	0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65, 0x12, 0x23, 0x0a, 0x06, 0x68, 0x65, 0x61,
	0x64, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0b, 0x2e, 0x74, 0x79, 0x70, 0x65,
	0x73, 0x2e, 0x56, 0x69, 0x65, 0x77, 0x52, 0x06, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x12, 0x32,
	0x0a, 0x07, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x18, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x43, 0x65, 0x72, 0x74, 0x69, 0x66, 0x69, 0x63,
	0x61, 0x74, 0x65, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x52, 0x07, 0x72, 0x65, 0x73, 0x75, 0x6c,
	0x74, 0x73, 0x12, 0x21, 0x0a, 0x0c, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73, 0x5f, 0x68, 0x61,
	0x73, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0b, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74,
	0x73, 0x48, 0x61, 0x73, 0x68, 0x12, 0x14, 0x0a, 0x05, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x1d, 0x0a, 0x0a, 0x62,
	0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x09, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x12, 0x21, 0x0a, 0x0c, 0x70, 0x72,
	0x6f, 0x70, 0x6f, 0x73, 0x65, 0x72, 0x5f, 0x6b, 0x65, 0x79, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x0b, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x65, 0x72, 0x4b, 0x65, 0x79, 0x12, 0x37, 0x0a,
	0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x19, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x41, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61,
	0x74, 0x65, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x52, 0x09, 0x73, 0x69, 0x67,
	0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x22, 0xc5, 0x01, 0x0a, 0x04, 0x56, 0x69, 0x65, 0x77, 0x12,
	0x16, 0x0a, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x72, 0x6f, 0x75, 0x6e, 0x64,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x72, 0x6f, 0x75, 0x6e, 0x64, 0x12, 0x22, 0x0a,
	0x05, 0x70, 0x68, 0x61, 0x73, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0c, 0x2e, 0x74,
	0x79, 0x70, 0x65, 0x73, 0x2e, 0x50, 0x68, 0x61, 0x73, 0x65, 0x52, 0x05, 0x70, 0x68, 0x61, 0x73,
	0x65, 0x12, 0x29, 0x0a, 0x10, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x74, 0x65, 0x65, 0x5f, 0x68,
	0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0f, 0x63, 0x6f, 0x6d,
	0x6d, 0x69, 0x74, 0x74, 0x65, 0x65, 0x48, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x1d, 0x0a, 0x0a,
	0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x5f, 0x69, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x09, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x49, 0x64, 0x12, 0x21, 0x0a, 0x0c, 0x63,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x74, 0x65, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28,
	0x04, 0x52, 0x0b, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x74, 0x65, 0x65, 0x49, 0x64, 0x22, 0x3b,
	0x0a, 0x03, 0x56, 0x44, 0x46, 0x12, 0x14, 0x0a, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x12, 0x1e, 0x0a, 0x0a, 0x69,
	0x74, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x0a, 0x69, 0x74, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x22, 0x4a, 0x0a, 0x12, 0x41,
	0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72,
	0x65, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x12,
	0x16, 0x0a, 0x06, 0x62, 0x69, 0x74, 0x6d, 0x61, 0x70, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x06, 0x62, 0x69, 0x74, 0x6d, 0x61, 0x70, 0x2a, 0xbb, 0x01, 0x0a, 0x05, 0x50, 0x68, 0x61, 0x73,
	0x65, 0x12, 0x0b, 0x0a, 0x07, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x0c,
	0x0a, 0x08, 0x45, 0x4c, 0x45, 0x43, 0x54, 0x49, 0x4f, 0x4e, 0x10, 0x01, 0x12, 0x11, 0x0a, 0x0d,
	0x45, 0x4c, 0x45, 0x43, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x56, 0x4f, 0x54, 0x45, 0x10, 0x02, 0x12,
	0x0b, 0x0a, 0x07, 0x50, 0x52, 0x4f, 0x50, 0x4f, 0x53, 0x45, 0x10, 0x03, 0x12, 0x10, 0x0a, 0x0c,
	0x50, 0x52, 0x4f, 0x50, 0x4f, 0x53, 0x45, 0x5f, 0x56, 0x4f, 0x54, 0x45, 0x10, 0x04, 0x12, 0x0d,
	0x0a, 0x09, 0x50, 0x52, 0x45, 0x43, 0x4f, 0x4d, 0x4d, 0x49, 0x54, 0x10, 0x05, 0x12, 0x12, 0x0a,
	0x0e, 0x50, 0x52, 0x45, 0x43, 0x4f, 0x4d, 0x4d, 0x49, 0x54, 0x5f, 0x56, 0x4f, 0x54, 0x45, 0x10,
	0x06, 0x12, 0x0a, 0x0a, 0x06, 0x43, 0x4f, 0x4d, 0x4d, 0x49, 0x54, 0x10, 0x07, 0x12, 0x12, 0x0a,
	0x0e, 0x43, 0x4f, 0x4d, 0x4d, 0x49, 0x54, 0x5f, 0x50, 0x52, 0x4f, 0x43, 0x45, 0x53, 0x53, 0x10,
	0x08, 0x12, 0x13, 0x0a, 0x0f, 0x52, 0x4f, 0x55, 0x4e, 0x44, 0x5f, 0x49, 0x4e, 0x54, 0x45, 0x52,
	0x52, 0x55, 0x50, 0x54, 0x10, 0x09, 0x12, 0x0d, 0x0a, 0x09, 0x50, 0x41, 0x43, 0x45, 0x4d, 0x41,
	0x4b, 0x45, 0x52, 0x10, 0x0a, 0x42, 0x20, 0x5a, 0x1e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x69, 0x6e, 0x63, 0x68, 0x75, 0x63, 0x6f, 0x2f, 0x67, 0x69, 0x6e,
	0x63, 0x68, 0x75, 0x2f, 0x6c, 0x69, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
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
var file_consensus_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_consensus_proto_goTypes = []interface{}{
	(Phase)(0),                 // 0: types.Phase
	(*Proposers)(nil),          // 1: types.Proposers
	(*QuorumCertificate)(nil),  // 2: types.QuorumCertificate
	(*View)(nil),               // 3: types.View
	(*VDF)(nil),                // 4: types.VDF
	(*AggregateSignature)(nil), // 5: types.AggregateSignature
	(*CertificateResult)(nil),  // 6: types.CertificateResult
}
var file_consensus_proto_depIdxs = []int32{
	3, // 0: types.QuorumCertificate.header:type_name -> types.View
	6, // 1: types.QuorumCertificate.results:type_name -> types.CertificateResult
	5, // 2: types.QuorumCertificate.signature:type_name -> types.AggregateSignature
	0, // 3: types.View.phase:type_name -> types.Phase
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_consensus_proto_init() }
func file_consensus_proto_init() {
	if File_consensus_proto != nil {
		return
	}
	file_proposal_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_consensus_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Proposers); i {
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
			switch v := v.(*QuorumCertificate); i {
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
			switch v := v.(*View); i {
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
			switch v := v.(*VDF); i {
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
			switch v := v.(*AggregateSignature); i {
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
			NumMessages:   5,
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
