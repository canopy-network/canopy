package canoliq

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/canopy-network/go-plugin/contract"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

// CanoliqConfig is the plugin identity advertised to the FSM during handshake.
// Registers all canoLiq-specific transaction types alongside the standard
// 'send' message reused from the existing tx.proto.
var CanoliqConfig = &contract.PluginConfig{
	Name:    "canoliq_plugin",
	Id:      2,
	Version: 1,
	SupportedTransactions: []string{
		"send",
		"canoliq_deposit",
		"canoliq_redeem",
		"canoliq_claim_redemption",
		"cliq_transfer",
		"cliq_claim_vested",
	},
	TransactionTypeUrls: []string{
		"type.googleapis.com/types.MessageSend",
		"type.googleapis.com/types.MessageCanoliqDeposit",
		"type.googleapis.com/types.MessageCanoliqRedeem",
		"type.googleapis.com/types.MessageCanoliqClaimRedemption",
		"type.googleapis.com/types.MessageCLIQTransfer",
		"type.googleapis.com/types.MessageCLIQClaimVested",
	},
	EventTypeUrls: nil,
}

// init seeds CanoliqConfig.FileDescriptorProtos with the proto files the FSM
// needs to decode canoLiq transactions. Mirrors the registration pattern in
// contract/contract.go::init.
func init() {
	var fds [][]byte
	for _, file := range []protoreflect.FileDescriptor{
		anypb.File_google_protobuf_any_proto,
		contract.File_account_proto,
		contract.File_event_proto,
		contract.File_plugin_proto,
		contract.File_tx_proto,
		contract.File_canoliq_proto,
	} {
		fd, _ := proto.Marshal(protodesc.ToFileDescriptorProto(file))
		fds = append(fds, fd)
	}
	CanoliqConfig.FileDescriptorProtos = fds
}

// Config is the canoLiq plugin runtime configuration. It mirrors
// contract.Config but adds the chainId of the canoLiq committee (used to
// derive the committee reward pool key) and the path to genesis.json.
type Config struct {
	ChainId     uint64 `json:"chainId"`
	DataDirPath string `json:"dataDirPath"`
	GenesisPath string `json:"genesisPath"`
}

// DefaultConfig returns reasonable defaults: the next free committee chainId
// and the standard plugin socket directory.
func DefaultConfig() Config {
	return Config{
		ChainId:     2,
		DataDirPath: filepath.Join("/tmp/plugin/"),
		GenesisPath: "",
	}
}

// NewConfigFromFile populates a Config from a JSON file, falling back to
// DefaultConfig values for any missing fields.
func NewConfigFromFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	c := DefaultConfig()
	if err = json.Unmarshal(data, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// DefaultParams returns the canonical canoLiq fee/split parameters from the
// whitepaper: 12% protocol fee with a 40/30/15/15 split.
func DefaultParams() *contract.CanoliqParams {
	return &contract.CanoliqParams{
		FeeBps:          1200,
		UserRebateBps:   4000,
		TreasuryBps:     3000,
		ValidatorBps:    1500,
		BuybackBps:      1500,
		DepositFee:      10_000,
		RedeemFee:       10_000,
		ClaimFee:        10_000,
		CliqTransferFee: 10_000,
	}
}

// ValidateParams enforces invariants on a CanoliqParams record before it is
// stored or used. The four split bps fields must total 10000.
func ValidateParams(p *contract.CanoliqParams) *contract.PluginError {
	if p == nil {
		return ErrInvalidParams()
	}
	if p.FeeBps > 10_000 {
		return ErrInvalidParams()
	}
	if p.UserRebateBps+p.TreasuryBps+p.ValidatorBps+p.BuybackBps != 10_000 {
		return ErrInvalidParams()
	}
	return nil
}
