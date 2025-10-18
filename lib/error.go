package lib

import (
	"fmt"
	"math"
	"runtime"
)

type ErrorI interface {
	Code() ErrorCode     // Returns the error code
	Module() ErrorModule // Returns the error module
	error                // Implements the built-in error interface
}

var _ ErrorI = &Error{} // Ensures *Error implements ErrorI

type ErrorCode uint32 // Defines a type for error codes

type ErrorModule string // Defines a type for error modules

type Error struct {
	ECode   ErrorCode   `json:"code"`   // Error code
	EModule ErrorModule `json:"module"` // Error module
	Msg     string      `json:"msg"`    // Error message
}

func NewError(code ErrorCode, module ErrorModule, msg string) *Error {
	// Constructs a new Error instance
	return &Error{ECode: code, EModule: module, Msg: msg}
}

// Code() returns the associated error code
func (p *Error) Code() ErrorCode { return p.ECode }

// Module() returns module field
func (p *Error) Module() ErrorModule { return p.EModule }

// String() calls Error()
func (p *Error) String() string { return p.Error() }

// Error() returns a formatted string including module, code, message, and stack trace
func (p *Error) Error() string {
	stack, pc := "", make([]uintptr, 1000)
	_ = runtime.Callers(1, pc)
	frames := runtime.CallersFrames(pc)
	if frames == nil {
		return fmt.Sprintf("\nModule:  %s\nCode:    %d\nMessage: %s\n", p.EModule, p.ECode, p.Msg)
	}
	for f, again := frames.Next(); again; f, again = frames.Next() {
		stack += fmt.Sprintf("\n%s L%d", f.File, f.Line)
	}
	//return fmt.Sprintf("\nModule:  %s\nCode:    %d\nMessage: %s\nStack: %s", p.EModule, p.ECode, p.Msg, stack)
	return fmt.Sprintf("\nModule:  %s\nCode:    %d\nMessage: %s", p.EModule, p.ECode, p.Msg)
}

const (
	NoCode ErrorCode = math.MaxUint32

	// Main Module
	MainModule ErrorModule = "main"

	// Main Module Error Codes
	CodeInvalidAddress              ErrorCode = 1
	CodeJSONMarshal                 ErrorCode = 2
	CodeJSONUnmarshal               ErrorCode = 3
	CodeUnmarshal                   ErrorCode = 4
	CodeMarshal                     ErrorCode = 5
	CodeFromAny                     ErrorCode = 6
	CodeToAny                       ErrorCode = 7
	CodeStringToBytes               ErrorCode = 8
	CodeNilBlock                    ErrorCode = 9
	CodeNilBlockHeader              ErrorCode = 10
	CodeInvalidBlockProposerAddress ErrorCode = 11
	CodeInvalidBlockHash            ErrorCode = 12
	CodeWrongLengthBlockHash        ErrorCode = 13
	CodeNilBlockTime                ErrorCode = 14
	CodeWrongLengthLastBlockHash    ErrorCode = 15
	CodeNilNetworkID                ErrorCode = 16
	CodeWrongLengthStateRoot        ErrorCode = 17
	CodeWrongLengthTxRoot           ErrorCode = 18
	CodeWrongLengthValRoot          ErrorCode = 19
	CodeWrongLengthNextValRoot      ErrorCode = 20
	CodeMerkleTree                  ErrorCode = 21
	CodeUnequalBlockHash            ErrorCode = 22
	CodeNewPubKeyFromBytes          ErrorCode = 23
	CodeNewMultiPubKey              ErrorCode = 24
	CodeWriteFile                   ErrorCode = 25
	CodeReadFile                    ErrorCode = 26
	CodeInvalidArgument             ErrorCode = 27
	CodeNilRewardRecipients         ErrorCode = 28
	CodeNoValidators                ErrorCode = 29
	CodeInvalidResultsHash          ErrorCode = 30
	CodeNonNilBlock                 ErrorCode = 31
	CodeProtoParse                  ErrorCode = 32

	// Consensus Module
	ConsensusModule ErrorModule = "consensus"

	// Consensus Module Error Codes
	CodeDuplicateTransaction            ErrorCode = 1
	CodeWrongHighQCHeight               ErrorCode = 2
	CodeMismatchResultsHash             ErrorCode = 3
	CodeDuplicateProposerMessage        ErrorCode = 4
	CodeDuplicateVote                   ErrorCode = 5
	CodeInvalidSignatureLength          ErrorCode = 6
	CodeInvalidPubKey                   ErrorCode = 7
	CodeEmptyView                       ErrorCode = 8
	CodeUnknownConsensusMessage         ErrorCode = 9
	CodeValidatorNotInSet               ErrorCode = 10
	CodeWrongHeight                     ErrorCode = 11
	CodeWrongBlockHeight                ErrorCode = 12
	CodeWrongPhase                      ErrorCode = 13
	CodePartialSignatureEmpty           ErrorCode = 14
	CodeInvalidPartialSignature         ErrorCode = 15
	CodeMismatchConsBlockHash           ErrorCode = 16
	CodeInvalidProposerPubKey           ErrorCode = 17
	CodeNoMaj23                         ErrorCode = 18
	CodeEmptyAggregateSignature         ErrorCode = 19
	CodeInvalidAggregateSignature       ErrorCode = 20
	CodeInvalidAggregateSignatureLen    ErrorCode = 21
	CodeEmptyAggregateSignatureBitmap   ErrorCode = 22
	CodeInvalidAggregateSignatureBitmap ErrorCode = 23
	CodeMismatchPublicKeys              ErrorCode = 24
	CodeEmptyEvidence                   ErrorCode = 25
	CodeAggregateSignature              ErrorCode = 26
	CodeEmptyQuorumCertificate          ErrorCode = 27
	CodeEvidenceTooOld                  ErrorCode = 28
	CodeMismatchProposals               ErrorCode = 29
	CodeFailedSafeNode                  ErrorCode = 30
	CodeInvalidValidatorIndex           ErrorCode = 31
	CodeUnableToAddSigner               ErrorCode = 32
	CodeEmptyMessage                    ErrorCode = 33
	CodeNotSubscribed                   ErrorCode = 34
	CodeInvalidEvidence                 ErrorCode = 35
	CodeMismatchEvidenceAndHeader       ErrorCode = 36
	CodeInvalidTxTime                   ErrorCode = 37
	CodeInvalidRCBuildHeight            ErrorCode = 38
	CodeExpectedBlockSizeLimit          ErrorCode = 39
	CodeNonNilCertResults               ErrorCode = 40
	CodeInvalidMemo                     ErrorCode = 41
	CodeNilCertResult                   ErrorCode = 42
	CodeNilLockOrder                    ErrorCode = 43
	CodeInvalidBuyerReceiveAddress      ErrorCode = 44
	CodeEmptyTransaction                ErrorCode = 45
	CodeHashSize                        ErrorCode = 46
	CodeInvalidLastQC                   ErrorCode = 47
	CodeMaxPort                         ErrorCode = 48
	CodePanic                           ErrorCode = 49
	CodeInvalidVDF                      ErrorCode = 50
	CodeNoSafeNodeJustification         ErrorCode = 51
	CodeNoSavedBlockOrResults           ErrorCode = 52
	CodeInvalidTxHeight                 ErrorCode = 53
	CodeInvalidSigner                   ErrorCode = 54
	CodeMismatchQcBlockHash             ErrorCode = 55
	CodeMismatchHeaderBlockHash         ErrorCode = 56
	CodeEmptyDoubleSigner               ErrorCode = 57
	CodeNonEquivocatingVote             ErrorCode = 58
	CodeInvalidEvidenceHeights          ErrorCode = 59
	CodeInvalidBuyerSendAddress         ErrorCode = 60
	CodeDuplicateCloseOrder             ErrorCode = 61
	CodeDuplicateResetOrder             ErrorCode = 62
	CodeMismatchCertHeight              ErrorCode = 63
	CodeNewHeight                       ErrorCode = 64
	CodeWrongViewHeight                 ErrorCode = 65
	CodeBadPort                         ErrorCode = 66
	CodeBadPortLowLimit                 ErrorCode = 67

	// State Machine Module
	StateMachineModule ErrorModule = "state_machine"

	// State Machine Module Error Codes
	CodeReadGenesisFile  ErrorCode = 1
	CodeFeeBelowState    ErrorCode = 2
	CodeUnauthorizedTx   ErrorCode = 3
	CodeEmptySignature   ErrorCode = 4
	CodeTxSignBytes      ErrorCode = 5
	CodeInvalidTxMessage ErrorCode = 6

	CodeMaxBlockSize              ErrorCode = 8
	CodeMaxTxSize                 ErrorCode = 9
	CodeRejectProposal            ErrorCode = 10
	CodeInvalidNetAddressLen      ErrorCode = 11
	CodeInvalidSignature          ErrorCode = 12
	CodeAddressEmpty              ErrorCode = 13
	CodeAddressSize               ErrorCode = 14
	CodeRecipientAddressEmpty     ErrorCode = 15
	CodeRecipientAddressSize      ErrorCode = 16
	CodeOutputAddressEmpty        ErrorCode = 17
	CodeOutputAddressSize         ErrorCode = 18
	CodeInvalidAmount             ErrorCode = 19
	CodePubKeyEmpty               ErrorCode = 20
	CodePubKeySize                ErrorCode = 21
	CodeParamKeyEmpty             ErrorCode = 22
	CodeParamValEmpty             ErrorCode = 23
	CodeInvalidSubsidy            ErrorCode = 24
	CodeInvalidOpcode             ErrorCode = 25
	CodeWrongChainId              ErrorCode = 26
	CodeUnknownMsg                ErrorCode = 27
	CodeInsufficientFunds         ErrorCode = 28
	CodeValidatorExists           ErrorCode = 29
	CodeValidatorNotExists        ErrorCode = 30
	CodeValidatorUnstaking        ErrorCode = 31
	CodeValidatorPaused           ErrorCode = 32
	CodeValidatorNotPaused        ErrorCode = 33
	CodeEmptyConsParams           ErrorCode = 34
	CodeEmptyValParams            ErrorCode = 35
	CodeEmptyFeeParams            ErrorCode = 36
	CodeEmptyGovParams            ErrorCode = 37
	CodeUnknownParam              ErrorCode = 38
	CodeUnknownParamType          ErrorCode = 39
	CodeUnknownParamSpace         ErrorCode = 40
	CodeInvalidProposalHash       ErrorCode = 41
	CodeInvalidRLPTx              ErrorCode = 42
	CodeInvalidERC20Tx            ErrorCode = 43
	CodeNonSubsidizedCommittee    ErrorCode = 44
	CodeInvalidNumberOfSamples    ErrorCode = 45
	CodeInvalidCertificateResults ErrorCode = 46
	CodePaymentRecipientsCount    ErrorCode = 47
	CodeInvalidPercentAllocation  ErrorCode = 48
	CodeErrNotEmpty               ErrorCode = 49
	CodeInvalidParam              ErrorCode = 50
	CodeErrFailedTransactions     ErrorCode = 51
	CodeInvalidProtocolVersion    ErrorCode = 52
	CodeInvalidDBKey              ErrorCode = 53
	CodeWrongStoreType            ErrorCode = 54
	CodeUnmarshalGenesis          ErrorCode = 55
	CodeInsufficientSupply        ErrorCode = 56
	CodeUnknownMsgName            ErrorCode = 57
	CodeUnknownPageable           ErrorCode = 58

	CodeInvalidBlockRange        ErrorCode = 60
	CodeInvalidPublicKey         ErrorCode = 61
	CodeInvalidDoubleSignHeights ErrorCode = 62
	CodeInvalidDoubleSigner      ErrorCode = 63
	CodeInvalidNumCommittees     ErrorCode = 64

	CodeValidatorIsADelegate ErrorCode = 66

	CodeInvalidChainId           ErrorCode = 68
	CodeWrongNetworkID           ErrorCode = 69
	CodeRootBuildHeight          ErrorCode = 70
	CodeRootHeight               ErrorCode = 71
	CodeInvalidQCCommitteeHeight ErrorCode = 72

	CodeOrderNotFound ErrorCode = 74

	CodeMinimumOrderSize     ErrorCode = 76
	CodeOrderLocked          ErrorCode = 77
	CodeInvalidLockOrder     ErrorCode = 78
	CodeDuplicateLockOrder   ErrorCode = 79
	CodeInvalidBuyerDeadline ErrorCode = 80
	CodeInvalidCloseOrder    ErrorCode = 81

	CodeInvalidCheckpoint        ErrorCode = 83
	CodeInvalidSellOrder         ErrorCode = 84
	CodeStartPollHeight          ErrorCode = 85
	CodeEmptyChainId             ErrorCode = 86
	CodeMismatchCertResults      ErrorCode = 87
	CodeInvalidQCRootChainHeight ErrorCode = 88
	CodeEmptyCertificateResults  ErrorCode = 89
	CodeSlashNonValidator        ErrorCode = 90
	CodeEmptyOrderBook           ErrorCode = 91
	CodeNoSubsidizedCommittees   ErrorCode = 92
	CodeEmptyLotteryWinner       ErrorCode = 93

	// P2P Module
	P2PModule ErrorModule = "p2p"

	// P2P Module Error Codes
	CodeUnknownP2PMessage       ErrorCode = 1
	CodeFailedRead              ErrorCode = 2
	CodeFailedWrite             ErrorCode = 3
	CodeMaxMessageSize          ErrorCode = 4
	CodePongTimeout             ErrorCode = 5
	CodeBlacklisted             ErrorCode = 6
	CodeErrorGroup              ErrorCode = 7
	CodeConnDecrypt             ErrorCode = 8
	CodeChunkLargerThanMax      ErrorCode = 9
	CodeFailedChallenge         ErrorCode = 10
	CodeFailedDiffieHellman     ErrorCode = 11
	CodeFailedHKDF              ErrorCode = 12
	CodePeerAlreadyExists       ErrorCode = 13
	CodePeerNotFound            ErrorCode = 14
	CodeFailedDial              ErrorCode = 15
	CodeMismatchPeerPublicKey   ErrorCode = 16
	CodeFailedListen            ErrorCode = 17
	CodeInvalidPeerPublicKey    ErrorCode = 18
	CodeSignatureSwap           ErrorCode = 19
	CodeMetaSwap                ErrorCode = 20
	CodeBadStream               ErrorCode = 21
	CodeBannedCountry           ErrorCode = 22
	CodeIPLookup                ErrorCode = 23
	CodeBannedIP                ErrorCode = 24
	CodeNonTCPAddr              ErrorCode = 25
	CodeInvalidNetAddressString ErrorCode = 26
	CodeInvalidNetAddressPubKey ErrorCode = 27
	CodeInvalidStateNetAddress  ErrorCode = 28
	CodeMaxOutbound             ErrorCode = 29
	CodeMaxInbound              ErrorCode = 30
	CodeBannedID                ErrorCode = 31
	CodeIncompatiblePeer        ErrorCode = 32
	CodeInvalidNetAddress       ErrorCode = 33

	StorageModule              ErrorModule = "store"
	CodeOpenDB                 ErrorCode   = 1
	CodeCloseDB                ErrorCode   = 2
	CodeStoreSet               ErrorCode   = 3
	CodeStoreGet               ErrorCode   = 4
	CodeStoreDelete            ErrorCode   = 5
	CodeCommitDB               ErrorCode   = 6
	CodeFlushBatch             ErrorCode   = 7
	CodeInvalidKey             ErrorCode   = 8
	CodeReserveKeyWrite        ErrorCode   = 9
	CodeInvalidMerkleTree      ErrorCode   = 10
	CodeInvalidMerkleTreeProof ErrorCode   = 11
	CodeGarbageCollectDB       ErrorCode   = 12
	CodeSetEntry               ErrorCode   = 13
	CodeReadBytes              ErrorCode   = 14
	CodeIndexBlock             ErrorCode   = 15

	RPCModule             ErrorModule = "rpc"
	CodeMempoolStopSignal ErrorCode   = 1
	CodeInvalidParams     ErrorCode   = 2
	CodeNewFSM            ErrorCode   = 3
	CodeTimeMachine       ErrorCode   = 4
	CodePostRequest       ErrorCode   = 5
	CodeGetRequest        ErrorCode   = 6
	CodeHttpStatus        ErrorCode   = 7
	CodeReadBody          ErrorCode   = 8
	CodeStringToCommittee ErrorCode   = 9
)

// error implementations below for the `types` package
func newLogError(err error) ErrorI {
	return NewError(NoCode, MainModule, err.Error())
}

func ErrUnmarshal(err error) ErrorI {
	return NewError(CodeUnmarshal, MainModule, fmt.Sprintf("unmarshal() failed with err: %s", err.Error()))
}

func ErrJSONUnmarshal(err error) ErrorI {
	return NewError(CodeJSONUnmarshal, MainModule, fmt.Sprintf("json.unmarshal() failed with err: %s", err.Error()))
}

func ErrJSONMarshal(err error) ErrorI {
	return NewError(CodeJSONMarshal, MainModule, fmt.Sprintf("json.marshal() failed with err: %s", err.Error()))
}

func ErrFromAny(err error) ErrorI {
	return NewError(CodeFromAny, MainModule, fmt.Sprintf("fromAny() failed with err: %s", err.Error()))
}

func ErrToAny(err error) ErrorI {
	return NewError(CodeToAny, MainModule, fmt.Sprintf("toAny() failed with err: %s", err.Error()))
}

func ErrMarshal(err error) ErrorI {
	return NewError(CodeMarshal, MainModule, fmt.Sprintf("marshal() failed with err: %s", err.Error()))
}

func ErrStringToBytes(err error) ErrorI {
	return NewError(CodeStringToBytes, MainModule, fmt.Sprintf("stringToBytes() failed with err: %s", err.Error()))
}

func ErrNilBlock() ErrorI {
	return NewError(CodeNilBlock, MainModule, "block is nil")
}

func ErrNonNilBlock() ErrorI {
	return NewError(CodeNonNilBlock, MainModule, "block is not nil")
}

func ErrNilRewardRecipients() ErrorI {
	return NewError(CodeNilRewardRecipients, MainModule, "reward recipients is nil")
}

func ErrNilBlockHeader() ErrorI {
	return NewError(CodeNilBlockHeader, MainModule, "block.header is nil")
}

func ErrInvalidBlockProposerAddress() ErrorI {
	return NewError(CodeInvalidBlockProposerAddress, MainModule, "block proposer address is invalid")
}

func ErrInvalidBlockHash() ErrorI {
	return NewError(CodeInvalidBlockHash, MainModule, "invalid block hash")
}

func ErrInvalidResultsHash() ErrorI {
	return NewError(CodeInvalidResultsHash, MainModule, "invalid results hash")
}

func ErrWrongLengthBlockHash() ErrorI {
	return NewError(CodeWrongLengthBlockHash, MainModule, "wrong length block hash")
}

func ErrNilBlockTime() ErrorI {
	return NewError(CodeNilBlockTime, MainModule, "nil block time")
}

func ErrWrongLengthLastBlockHash() ErrorI {
	return NewError(CodeWrongLengthLastBlockHash, MainModule, "wrong length last block hash")
}

func ErrNilNetworkID() ErrorI {
	return NewError(CodeNilNetworkID, MainModule, "nil network id")
}

func ErrWrongLengthTransactionRoot() ErrorI {
	return NewError(CodeWrongLengthTxRoot, MainModule, "wrong length transaction root")
}

func ErrWrongLengthStateRoot() ErrorI {
	return NewError(CodeWrongLengthStateRoot, MainModule, "wrong length state root")
}

func ErrWrongLengthValidatorRoot() ErrorI {
	return NewError(CodeWrongLengthValRoot, MainModule, "wrong length validator root")
}

func ErrWrongLengthNextValidatorRoot() ErrorI {
	return NewError(CodeWrongLengthNextValRoot, MainModule, "wrong length next validator root")
}

func ErrMerkleTree(err error) ErrorI {
	return NewError(CodeMerkleTree, MainModule, fmt.Sprintf("merkle tree failed with err: %s", err.Error()))
}

func ErrUnequalBlockHash() ErrorI {
	return NewError(CodeUnequalBlockHash, MainModule, "unequal block hash")
}

func ErrPubKeyFromBytes(err error) ErrorI {
	return NewError(CodeNewPubKeyFromBytes, MainModule, fmt.Sprintf("publicKeyFromBytes() failed with err: %s", err.Error()))
}

func ErrNewMultiPubKey(err error) ErrorI {
	return NewError(CodeNewMultiPubKey, MainModule, fmt.Sprintf("newMultiPubKey() failed with err: %s", err.Error()))
}

func ErrNoValidators() ErrorI {
	return NewError(CodeNoValidators, MainModule, fmt.Sprintf("there are no validators in the set"))
}

func ErrWrongCertHeight(got, wanted uint64) ErrorI {
	return NewError(CodeWrongHeight, ConsensusModule, fmt.Sprintf("wrong certificate height, got=%d | wanted=%d", got, wanted))
}

func ErrWrongViewHeight(got, wanted uint64) ErrorI {
	return NewError(CodeWrongViewHeight, ConsensusModule, fmt.Sprintf("wrong view height, got=%d | wanted=%d", got, wanted))
}

func ErrMismatchCertBlockHeight(got, wanted uint64) ErrorI {
	return NewError(CodeMismatchCertHeight, ConsensusModule, fmt.Sprintf("mismatch certificate height, got=%d | wanted=%d", got, wanted))
}

func ErrWrongBlockHeight(got, wanted uint64) ErrorI {
	return NewError(CodeWrongBlockHeight, ConsensusModule, fmt.Sprintf("wrong block height, got=%d | wanted=%d", got, wanted))
}

func ErrNewHeight() ErrorI {
	return NewError(CodeNewHeight, ConsensusModule, "new height")
}

func ErrWrongRootHeight() ErrorI {
	return NewError(CodeRootBuildHeight, ConsensusModule, "wrong root height")
}

func ErrWrongRootBuildHeight() ErrorI {
	return NewError(CodeRootHeight, ConsensusModule, "wrong root build height")
}

func ErrInvalidQCCommitteeHeight() ErrorI {
	return NewError(CodeInvalidQCCommitteeHeight, ConsensusModule, "invalid certificate committee height")
}

func ErrInvalidQCRootChainHeight() ErrorI {
	return NewError(CodeInvalidQCRootChainHeight, ConsensusModule, "invalid certificate root-chain height")
}

func ErrInvalidRCBuildHeight(height, stateHeight uint64) ErrorI {
	return NewError(CodeInvalidRCBuildHeight, ConsensusModule, fmt.Sprintf("invalid root chain build height %d with state height %d", height, stateHeight))
}

func ErrEmptyView() ErrorI {
	return NewError(CodeEmptyView, ConsensusModule, "empty view")
}

func ErrWrongPhase() ErrorI {
	return NewError(CodeWrongPhase, ConsensusModule, "wrong phase")
}

func ErrEmptyQuorumCertificate() ErrorI {
	return NewError(CodeEmptyQuorumCertificate, ConsensusModule, "empty quorum certificate")
}

func ErrEmptyAggregateSignature() ErrorI {
	return NewError(CodeEmptyAggregateSignature, ConsensusModule, "empty aggregate signature")
}

func ErrInvalidAggrSignatureLength() ErrorI {
	return NewError(CodeInvalidAggregateSignatureLen, ConsensusModule, "invalid aggregate signature length")
}

func ErrEmptySignerBitmap() ErrorI {
	return NewError(CodeEmptyAggregateSignatureBitmap, ConsensusModule, "empty signer bitmap")
}

func ErrInvalidSignerBitmap(err error) ErrorI {
	return NewError(CodeInvalidAggregateSignatureBitmap, ConsensusModule, fmt.Sprintf("invalid signature bitmap: %s", err.Error()))
}

func ErrInvalidAggrSignature() ErrorI {
	return NewError(CodeInvalidAggregateSignature, ConsensusModule, "invalid aggregate signature")
}

func ErrNoMaj23() ErrorI {
	return NewError(CodeNoMaj23, ConsensusModule, "quorum not reached")
}

func ErrInvalidVDF() ErrorI {
	return NewError(CodeInvalidVDF, ConsensusModule, "invalid verifiable delay proof")
}

func ErrValidatorNotInSet(publicKey []byte) ErrorI {
	return NewError(CodeValidatorNotInSet, ConsensusModule, fmt.Sprintf("validator %s not found in validator set", BytesToString(publicKey)))
}

func ErrInvalidValidatorIndex() ErrorI {
	return NewError(CodeInvalidValidatorIndex, ConsensusModule, "invalid validator index")
}

func ErrNotSubscribed() ErrorI {
	return NewError(CodeNotSubscribed, ConsensusModule, "not subscribed")
}

func ErrInvalidTxHeight() ErrorI {
	return NewError(CodeInvalidTxHeight, ConsensusModule, "invalid tx height")
}

func ErrInvalidTxTime() ErrorI {
	return NewError(CodeInvalidTxTime, ConsensusModule, "invalid tx time")
}

func ErrInvalidMemo() ErrorI {
	return NewError(CodeInvalidMemo, ConsensusModule, "invalid memo")
}

func ErrEmptyEvidence() ErrorI {
	return NewError(CodeEmptyEvidence, ConsensusModule, "evidence is empty")
}

func ErrInvalidEvidence() ErrorI {
	return NewError(CodeInvalidEvidence, ConsensusModule, "evidence is invalid")
}

func ErrInvalidEvidenceHeights() ErrorI {
	return NewError(CodeInvalidEvidenceHeights, ConsensusModule, "evidence heights are invalid")
}

func ErrNonEquivocatingVote() ErrorI {
	return NewError(CodeNonEquivocatingVote, ConsensusModule, "non equivocating vote")
}

func ErrEmptyDoubleSigner() ErrorI {
	return NewError(CodeEmptyDoubleSigner, ConsensusModule, "double signer is empty")
}

func ErrEvidenceTooOld() ErrorI {
	return NewError(CodeEvidenceTooOld, ConsensusModule, "evidence is too old")
}

func ErrInvalidProposerPubKey(expected []byte) ErrorI {
	return NewError(CodeInvalidProposerPubKey, ConsensusModule, fmt.Sprintf("invalid proposer public key, expected %s", BytesToTruncatedString(expected)))
}

func ErrInvalidSigner() ErrorI {
	return NewError(CodeInvalidSigner, ConsensusModule, "invalid cons message signer")
}

func ErrMismatchEvidenceAndHeader() ErrorI {
	return NewError(CodeMismatchEvidenceAndHeader, ConsensusModule, "mismatch evidence and block header")
}

func ErrInvalidLastQuorumCertificate() ErrorI {
	return NewError(CodeInvalidLastQC, ConsensusModule, "last quorum certificate is invalid")
}

func ErrInvalidNetAddrString(s string) ErrorI {
	return NewError(CodeInvalidNetAddressString, P2PModule, fmt.Sprintf("invalid net address string: %s", s))
}

func ErrInvalidNetAddressPubKey(s string) ErrorI {
	return NewError(CodeInvalidNetAddressPubKey, P2PModule, fmt.Sprintf("invalid net address public key: %s", s))
}

func ErrInvalidStateNetAddress(s string) ErrorI {
	return NewError(CodeInvalidStateNetAddress, P2PModule, fmt.Sprintf("invalid net address no ports or subpaths allowed: %s", s))
}

func ErrInvalidNetAddress(s string) ErrorI {
	return NewError(CodeInvalidNetAddress, P2PModule, fmt.Sprintf("invalid net address host and port: %s", s))
}

func ErrWrongHighQCHeight() ErrorI {
	return NewError(CodeWrongHighQCHeight, ConsensusModule, fmt.Sprintf("wrong high qc height"))
}

func ErrWriteFile(err error) ErrorI {
	return NewError(CodeWriteFile, MainModule, fmt.Sprintf("os.WriteFile() failed with err: %s", err.Error()))
}

func ErrReadFile(err error) ErrorI {
	return NewError(CodeReadFile, MainModule, fmt.Sprintf("os.ReadFile() failed with err: %s", err.Error()))
}

func ErrUnknownMessageName(s string) ErrorI {
	return NewError(CodeUnknownMsgName, StateMachineModule, fmt.Sprintf("message name %s is unknown", s))
}

func ErrUnknownPageable(s string) ErrorI {
	return NewError(CodeUnknownPageable, StateMachineModule, fmt.Sprintf("pageable %s is unknown", s))
}

func ErrEmptyTransaction() ErrorI {
	return NewError(CodeEmptyTransaction, StateMachineModule, "transaction is empty")
}

func ErrEmptyMessage() ErrorI {
	return NewError(CodeEmptyMessage, StateMachineModule, "message is empty")
}

func ErrUnknownFields() ErrorI {
	return NewError(CodeUnknownMsg, StateMachineModule, "message contains unknown fields")
}

func ErrNoSavedBlockOrResults() ErrorI {
	return NewError(CodeNoSavedBlockOrResults, StateMachineModule, "no saved block or results to validate the msg")
}

func ErrEmptySignature() ErrorI {
	return NewError(CodeEmptySignature, StateMachineModule, "signature is empty")
}

func ErrInvalidAddress() ErrorI {
	return NewError(CodeInvalidAddress, MainModule, "address is invalid")
}

func ErrInvalidDoubleSignHeights() ErrorI {
	return NewError(CodeInvalidDoubleSignHeights, ConsensusModule, "double sign heights are invalid")
}

func ErrInvalidDoubleSigner() ErrorI {
	return NewError(CodeInvalidDoubleSigner, ConsensusModule, "double signer is invalid")
}

func ErrMismatchResultsHash() ErrorI {
	return NewError(CodeMismatchResultsHash, ConsensusModule, "mismatch results hash")
}

func ErrMismatchConsBlockHash() ErrorI {
	return NewError(CodeMismatchConsBlockHash, ConsensusModule, "mismatch cons block hash")
}

func ErrMismatchQCBlockHash() ErrorI {
	return NewError(CodeMismatchQcBlockHash, ConsensusModule, "mismatch qc block hash")
}

func ErrMismatchHeaderBlockHash() ErrorI {
	return NewError(CodeMismatchHeaderBlockHash, ConsensusModule, "mismatch header block hash")
}

func ErrInvalidPercentAllocation() ErrorI {
	return NewError(CodeInvalidPercentAllocation, StateMachineModule, "invalid percent allocation")
}

func ErrPaymentRecipientsCount() ErrorI {
	return NewError(CodePaymentRecipientsCount, StateMachineModule, "invalid payment recipients count")
}

func ErrWrongNetworkID() ErrorI {
	return NewError(CodeWrongNetworkID, StateMachineModule, "wrong network id")
}

func ErrEmptyChainId() ErrorI {
	return NewError(CodeEmptyChainId, StateMachineModule, "empty chain id")
}

func ErrEmptyOrderBook() ErrorI {
	return NewError(CodeEmptyOrderBook, StateMachineModule, "empty order book")
}

func ErrWrongChainId() ErrorI {
	return NewError(CodeWrongChainId, StateMachineModule, "wrong chain id")
}

func ErrDuplicateTx(hash string) ErrorI {
	return NewError(CodeDuplicateTransaction, ConsensusModule, fmt.Sprintf("tx %s is a duplicate", hash))
}

func ErrMaxTxSize() ErrorI {
	return NewError(CodeMaxTxSize, StateMachineModule, "max tx size")
}

func ErrInvalidArgument() ErrorI {
	return NewError(CodeInvalidArgument, MainModule, "the argument is invalid")
}

func ErrInvalidMessageCast() ErrorI {
	return NewError(CodeInvalidArgument, MainModule, "the message cast failed")
}

func ErrExpectedMaxBlockSize() ErrorI {
	return NewError(CodeExpectedBlockSizeLimit, MainModule, "the block size exceeds the expected limit")
}

func ErrNonNilCertResults() ErrorI {
	return NewError(CodeNonNilCertResults, MainModule, "the certificate results is not empty")
}

func ErrNilLockOrder() ErrorI {
	return NewError(CodeNilLockOrder, MainModule, "lock order is nil")
}

func ErrInvalidBuyerReceiveAddress() ErrorI {
	return NewError(CodeInvalidBuyerReceiveAddress, MainModule, "invalid buyer receive address")
}

func ErrInvalidBuyerSendAddress() ErrorI {
	return NewError(CodeInvalidBuyerSendAddress, MainModule, "invalid buyer send address")
}

func ErrDuplicateResetOrder() ErrorI {
	return NewError(CodeDuplicateResetOrder, MainModule, "duplicate reset order")
}

func ErrDuplicateCloseOrder() ErrorI {
	return NewError(CodeDuplicateCloseOrder, MainModule, "duplicate close order")
}

func ErrNilCertResults() ErrorI {
	return NewError(CodeNilCertResult, MainModule, "the certificate results is empty")
}

func ErrHashSize() ErrorI {
	return NewError(CodeHashSize, MainModule, "wrong hash size")
}

func ErrMaxPort() ErrorI { return NewError(CodeMaxPort, MainModule, "max port exceeded") }

func ErrBadPort() ErrorI { return NewError(CodeBadPort, MainModule, "port not numerical") }

func ErrBadPortLowLimit() ErrorI {
	return NewError(CodeBadPortLowLimit, MainModule, fmt.Sprintf("port must be greater than %d", MinAllowedPort))
}

func ErrProtoParse(err error) ErrorI {
	return NewError(CodeProtoParse, MainModule, fmt.Sprintf("proto parse failed with error: %s", err.Error()))
}

func ErrOrderLocked() ErrorI {
	return NewError(CodeOrderLocked, StateMachineModule, "order locked")
}

func ErrOrderNotFound() ErrorI {
	return NewError(CodeOrderNotFound, StateMachineModule, "order not found")
}

func ErrPanic() ErrorI {
	return NewError(CodePanic, StateMachineModule, "panic")
}

func ErrMempoolStopSignal() ErrorI {
	return NewError(CodeMempoolStopSignal, RPCModule, "mempool stop signal")
}

func ErrInvalidParams(err error) ErrorI {
	bz, _ := MarshalJSON(err)
	return NewError(CodeInvalidParams, RPCModule, fmt.Sprintf("invalid params: %s", string(bz)))
}

func ErrWrongHighQCRootHeight() ErrorI {
	return NewError(CodeNewFSM, RPCModule, fmt.Sprintf("wrong high qc root height"))
}

func ErrNewStore(err error) ErrorI {
	return NewError(CodeNewFSM, RPCModule, fmt.Sprintf("new store failed with err: %s", err.Error()))
}

func ErrTimeMachine(err error) ErrorI {
	return NewError(CodeTimeMachine, RPCModule, fmt.Sprintf("fsm.TimeMachine() failed with err: %s", err.Error()))
}

func ErrPostRequest(err error) ErrorI {
	return NewError(CodePostRequest, RPCModule, fmt.Sprintf("http.Post() failed with err: %s", err.Error()))
}

func ErrGetRequest(err error) ErrorI {
	return NewError(CodeGetRequest, RPCModule, fmt.Sprintf("http.Get() failed with err: %s", err.Error()))
}

func ErrHttpStatus(status string, statusCode int, body []byte) ErrorI {
	return NewError(CodeHttpStatus, RPCModule, fmt.Sprintf("http response bad status %s with code %d and body %s", status, statusCode, body))
}

func ErrReadBody(err error) ErrorI {
	return NewError(CodeReadBody, RPCModule, fmt.Sprintf("io.ReadAll(http.ResponseBody) failed with err: %s", err.Error()))
}

func ErrStringToCommittee(s string) ErrorI {
	return NewError(CodeStringToCommittee, RPCModule, fmt.Sprintf("committee arg %s is invalid, requires a comma separated list of <chainId>=<percent> ex. 0=50,21=25,99=25", s))
}

func ErrNoSubsidizedCommittees(chainId uint64) ErrorI {
	return NewError(CodeNoSubsidizedCommittees, StateMachineModule, fmt.Sprintf("Chain ID %d has no subsidized committees", chainId))
}

func ErrEmptyLotteryWinner() ErrorI {
	return NewError(CodeEmptyLotteryWinner, StateMachineModule, "Lottery winner is empty")
}

func ErrFailedTransactions() ErrorI {
	return NewError(CodeErrFailedTransactions, StateMachineModule, "a block contained failed transactions")
}
