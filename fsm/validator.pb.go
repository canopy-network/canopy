// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.29.3
// source: validator.proto

package fsm

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// *****************************************************************************************************
// This file is auto-generated from source files in `/lib/.proto/*` using Protocol Buffers (protobuf)
//
// Protobuf is a language-neutral, platform-neutral serialization format. It allows users
// to define objects in a way that’s both efficient to store and fast to transmit over the network.
// These definitions are compiled into code that *enables different systems and programming languages
// to communicate in a byte-perfect manner*
//
// To update these structures, make changes to the source .proto files, then recompile
// to regenerate this file.
// These auto-generated files are easily recognized by checking for a `.pb.go` ending
// *****************************************************************************************************
// _
// _
// _
//
// A Validator is an actor in the blockchain network responsible for verifying transactions, creating new blocks,
// and maintaining the blockchain's integrity and security. In Canopy, Validators provide this service for multiple
// chains by restaking their tokens across multiple 'committees' and run a blockchain client for each
// Both the Operator and the Output private key may sign transactions on behalf of the Validator, but only the
// Output private key may change the Output address.
type Validator struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// address: the short version of the public key of the operator of the service, corresponding to the 2
	Address []byte `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	// public_key: the public cryptographic identity of the operator of the service. This key must be an aggregable BLS
	// key for efficiency in the BFT process.
	PublicKey []byte `protobuf:"bytes,2,opt,name=public_key,json=publicKey,proto3" json:"publicKey"` // @gotags: json:"publicKey"
	// net_address: the tcp peer-to-peer address of the node to enable easy discovery of the peer for multi-consensus
	NetAddress string `protobuf:"bytes,3,opt,name=net_address,json=netAddress,proto3" json:"netAddress"` // @gotags: json:"netAddress"
	// staked_amount: the amount of tokens locked as a surety bond against malicious behavior. These tokens may be
	// increased by auto-compounding rewards or by an edit-stake command. This bond is returned to the output address
	// after executing an unstake command and waiting the unstaking period.
	StakedAmount uint64 `protobuf:"varint,4,opt,name=staked_amount,json=stakedAmount,proto3" json:"stakedAmount"` // @gotags: json:"stakedAmount"
	// committees: a list of ids of the committees the validator is offering Validation or Delegation services for
	Committees []uint64 `protobuf:"varint,5,rep,packed,name=committees,proto3" json:"committees,omitempty"`
	// max_paused_height: if the Validator is paused, this value tracks the maximum height it may be paused before it
	// automatically begins unstaking
	MaxPausedHeight uint64 `protobuf:"varint,6,opt,name=max_paused_height,json=maxPausedHeight,proto3" json:"maxPausedHeight"` // @gotags: json:"maxPausedHeight"
	// unstaking_height: if the Validator is unstaking, this value tracks the future block height a Validator's surety
	// bond will be returned
	UnstakingHeight uint64 `protobuf:"varint,7,opt,name=unstaking_height,json=unstakingHeight,proto3" json:"unstakingHeight"` // @gotags: json:"unstakingHeight"
	// output: the address where early-withdrawal rewards and the unstaking surety bond are transferred to
	Output []byte `protobuf:"bytes,8,opt,name=output,proto3" json:"output,omitempty"`
	// delegate: signals whether the Validator is a Delegate or not. If true, the Validator only passively participates
	// in Validation-as-a-Service by dedicating their funds to help the chain qualify for protocol subsidisation
	Delegate bool `protobuf:"varint,9,opt,name=delegate,proto3" json:"delegate,omitempty"`
	// compound: signals whether the Validator is auto-compounding (increasing their stake with new rewards) or
	// withdrawing rewards early to their output address as they come in
	Compound      bool `protobuf:"varint,10,opt,name=compound,proto3" json:"compound,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Validator) Reset() {
	*x = Validator{}
	mi := &file_validator_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Validator) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Validator) ProtoMessage() {}

func (x *Validator) ProtoReflect() protoreflect.Message {
	mi := &file_validator_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Validator.ProtoReflect.Descriptor instead.
func (*Validator) Descriptor() ([]byte, []int) {
	return file_validator_proto_rawDescGZIP(), []int{0}
}

func (x *Validator) GetAddress() []byte {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *Validator) GetPublicKey() []byte {
	if x != nil {
		return x.PublicKey
	}
	return nil
}

func (x *Validator) GetNetAddress() string {
	if x != nil {
		return x.NetAddress
	}
	return ""
}

func (x *Validator) GetStakedAmount() uint64 {
	if x != nil {
		return x.StakedAmount
	}
	return 0
}

func (x *Validator) GetCommittees() []uint64 {
	if x != nil {
		return x.Committees
	}
	return nil
}

func (x *Validator) GetMaxPausedHeight() uint64 {
	if x != nil {
		return x.MaxPausedHeight
	}
	return 0
}

func (x *Validator) GetUnstakingHeight() uint64 {
	if x != nil {
		return x.UnstakingHeight
	}
	return 0
}

func (x *Validator) GetOutput() []byte {
	if x != nil {
		return x.Output
	}
	return nil
}

func (x *Validator) GetDelegate() bool {
	if x != nil {
		return x.Delegate
	}
	return false
}

func (x *Validator) GetCompound() bool {
	if x != nil {
		return x.Compound
	}
	return false
}

// ValidatorsList is stored as a single blob (instead of using prefixed keys) for the following reasons:
//
//   - Filesystem database iterators (e.g., in LevelDB/BadgerDB) introduce significant overhead and are often slow,
//     especially when reading or updating large sets of keys every block.
//
//   - The total number of validators is expected to remain below 100,000, and each entry is relatively small (~200 bytes),
//     making a full in-memory deserialization manageable and performant.
//
//   - Validators are expected to participate in most committees, so the majority of the list is relevant to each context,
//     and filtering from a single list is faster than multiple disk reads.
//
//   - Storing separate, per-committee lists (especially if sorted and updated every block) would create substantial
//     additional write and storage overhead compared to maintaining a single, compact list.
type ValidatorsList struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	List          []*Validator           `protobuf:"bytes,1,rep,name=List,proto3" json:"List,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ValidatorsList) Reset() {
	*x = ValidatorsList{}
	mi := &file_validator_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ValidatorsList) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ValidatorsList) ProtoMessage() {}

func (x *ValidatorsList) ProtoReflect() protoreflect.Message {
	mi := &file_validator_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ValidatorsList.ProtoReflect.Descriptor instead.
func (*ValidatorsList) Descriptor() ([]byte, []int) {
	return file_validator_proto_rawDescGZIP(), []int{1}
}

func (x *ValidatorsList) GetList() []*Validator {
	if x != nil {
		return x.List
	}
	return nil
}

// NonSignerInfo is information that tracks the number of blocks not signed by the Validator within the Non-Sign-Window
type NonSigner struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// address: shorter version of the operator public key of the non signer
	Address []byte `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	// counter: increments when a Validator doesn't sign a block and resets every non-sign-window
	Counter       uint64 `protobuf:"varint,2,opt,name=counter,proto3" json:"counter,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *NonSigner) Reset() {
	*x = NonSigner{}
	mi := &file_validator_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NonSigner) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NonSigner) ProtoMessage() {}

func (x *NonSigner) ProtoReflect() protoreflect.Message {
	mi := &file_validator_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NonSigner.ProtoReflect.Descriptor instead.
func (*NonSigner) Descriptor() ([]byte, []int) {
	return file_validator_proto_rawDescGZIP(), []int{2}
}

func (x *NonSigner) GetAddress() []byte {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *NonSigner) GetCounter() uint64 {
	if x != nil {
		return x.Counter
	}
	return 0
}

// NonSignerList is a list of information that tracks the number of blocks not signed by the Validator within the Non-Sign-Window
type NonSignerList struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	List          []*NonSigner           `protobuf:"bytes,1,rep,name=List,proto3" json:"List,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *NonSignerList) Reset() {
	*x = NonSignerList{}
	mi := &file_validator_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NonSignerList) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NonSignerList) ProtoMessage() {}

func (x *NonSignerList) ProtoReflect() protoreflect.Message {
	mi := &file_validator_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NonSignerList.ProtoReflect.Descriptor instead.
func (*NonSignerList) Descriptor() ([]byte, []int) {
	return file_validator_proto_rawDescGZIP(), []int{3}
}

func (x *NonSignerList) GetList() []*NonSigner {
	if x != nil {
		return x.List
	}
	return nil
}

var File_validator_proto protoreflect.FileDescriptor

const file_validator_proto_rawDesc = "" +
	"\n" +
	"\x0fvalidator.proto\x12\x05types\"\xd1\x02\n" +
	"\tValidator\x12\x18\n" +
	"\aaddress\x18\x01 \x01(\fR\aaddress\x12\x1d\n" +
	"\n" +
	"public_key\x18\x02 \x01(\fR\tpublicKey\x12\x1f\n" +
	"\vnet_address\x18\x03 \x01(\tR\n" +
	"netAddress\x12#\n" +
	"\rstaked_amount\x18\x04 \x01(\x04R\fstakedAmount\x12\x1e\n" +
	"\n" +
	"committees\x18\x05 \x03(\x04R\n" +
	"committees\x12*\n" +
	"\x11max_paused_height\x18\x06 \x01(\x04R\x0fmaxPausedHeight\x12)\n" +
	"\x10unstaking_height\x18\a \x01(\x04R\x0funstakingHeight\x12\x16\n" +
	"\x06output\x18\b \x01(\fR\x06output\x12\x1a\n" +
	"\bdelegate\x18\t \x01(\bR\bdelegate\x12\x1a\n" +
	"\bcompound\x18\n" +
	" \x01(\bR\bcompound\"6\n" +
	"\x0eValidatorsList\x12$\n" +
	"\x04List\x18\x01 \x03(\v2\x10.types.ValidatorR\x04List\"?\n" +
	"\tNonSigner\x12\x18\n" +
	"\aaddress\x18\x01 \x01(\fR\aaddress\x12\x18\n" +
	"\acounter\x18\x02 \x01(\x04R\acounter\"5\n" +
	"\rNonSignerList\x12$\n" +
	"\x04List\x18\x01 \x03(\v2\x10.types.NonSignerR\x04ListB&Z$github.com/canopy-network/canopy/fsmb\x06proto3"

var (
	file_validator_proto_rawDescOnce sync.Once
	file_validator_proto_rawDescData []byte
)

func file_validator_proto_rawDescGZIP() []byte {
	file_validator_proto_rawDescOnce.Do(func() {
		file_validator_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_validator_proto_rawDesc), len(file_validator_proto_rawDesc)))
	})
	return file_validator_proto_rawDescData
}

var file_validator_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_validator_proto_goTypes = []any{
	(*Validator)(nil),      // 0: types.Validator
	(*ValidatorsList)(nil), // 1: types.ValidatorsList
	(*NonSigner)(nil),      // 2: types.NonSigner
	(*NonSignerList)(nil),  // 3: types.NonSignerList
}
var file_validator_proto_depIdxs = []int32{
	0, // 0: types.ValidatorsList.List:type_name -> types.Validator
	2, // 1: types.NonSignerList.List:type_name -> types.NonSigner
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_validator_proto_init() }
func file_validator_proto_init() {
	if File_validator_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_validator_proto_rawDesc), len(file_validator_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_validator_proto_goTypes,
		DependencyIndexes: file_validator_proto_depIdxs,
		MessageInfos:      file_validator_proto_msgTypes,
	}.Build()
	File_validator_proto = out.File
	file_validator_proto_goTypes = nil
	file_validator_proto_depIdxs = nil
}
