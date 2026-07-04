package sol

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/gagliardetto/solana-go"
)

const solanaBlockchain = "solana"

var (
	memoProgramID          = solana.MustPublicKeyFromBase58("MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr")
	computeBudgetProgramID = solana.MustPublicKeyFromBase58("ComputeBudget111111111111111111111111111111")
)

// OrderValidator validates canopy order JSON found in a Memo instruction.
type OrderValidator interface {
	ValidateOrderJsonBytes(jsonBytes []byte, orderType types.OrderType) error
}

var _ types.TransactionI = &Transaction{}

// instruction is a resolved (program-id + account pubkeys + data) view of one top-level
// instruction, decoupled from solana-go's compiled/indexed form so parsing is pure and testable.
type instruction struct {
	programID solana.PublicKey
	accounts  []solana.PublicKey
	data      []byte
}

// Transaction wraps a Solana transaction and implements types.TransactionI.
type Transaction struct {
	signature string        // base58 transaction signature (used as Hash/TransactionID)
	feePayer  string        // first signer account, used as From()
	instrs    []instruction // resolved top-level instructions

	order *types.WitnessedOrder // parsed order, if any

	// transfer fields, populated for close orders
	isTransfer      bool
	destination     string   // token account (SPL) or wallet (native) receiving funds
	mint            string   // SPL mint (base58), empty for native SOL or when needsMintLookup is true
	amount          *big.Int // base-unit amount
	decimals        uint8    // resolved by the provider via mint cache; 0 for native SOL by convention
	needsMintLookup bool     // true for a bare SPL Transfer (tag 3), whose instruction omits the mint;
	// the provider must resolve it from the destination token account before decimals can be looked up
}

// From returns the fee payer (first signer).
func (t *Transaction) From() string { return t.feePayer }

// To returns the transfer destination token/wallet account (empty for lock orders).
func (t *Transaction) To() string { return t.destination }

// Hash returns the transaction signature.
func (t *Transaction) Hash() string { return t.signature }

// Blockchain returns the chain identifier.
func (t *Transaction) Blockchain() string { return solanaBlockchain }

// Order returns the parsed witnessed order, if any.
func (t *Transaction) Order() *types.WitnessedOrder { return t.order }

// clearOrder resets parsed state (mirrors eth's clearOrder for retry paths).
func (t *Transaction) clearOrder() {
	t.order = nil
	t.isTransfer = false
	t.needsMintLookup = false
}

// TokenTransfer returns the generic token transfer view.
func (t *Transaction) TokenTransfer() types.TokenTransfer {
	return types.TokenTransfer{
		Blockchain:       solanaBlockchain,
		TokenInfo:        types.TokenInfo{Decimals: t.decimals},
		TransactionID:    t.signature,
		SenderAddress:    t.feePayer,
		RecipientAddress: t.destination,
		TokenBaseAmount:  t.amount,
		ContractAddress:  t.mint,
	}
}

// findInstruction returns the first top-level instruction whose program id matches, or nil.
func (t *Transaction) findInstruction(programID solana.PublicKey) *instruction {
	for i := range t.instrs {
		if t.instrs[i].programID.Equals(programID) {
			return &t.instrs[i]
		}
	}
	return nil
}

// parseInstructions scans top-level instructions for a Memo (order JSON) and, for close orders,
// a transfer instruction. No order found is not an error (returns nil, order stays nil) - a memo
// that matches neither schema is only logged (Debugf), not returned as an error, since on a real
// chain most memos are unrelated third-party traffic and would otherwise spam logs/callers.
func (t *Transaction) parseInstructions(v OrderValidator, logger lib.LoggerI) error {
	memo := t.findInstruction(memoProgramID)
	if memo == nil {
		return nil // not an order transaction
	}
	// try lock order first (no transfer required)
	lockErr := v.ValidateOrderJsonBytes(memo.data, types.LockOrderType)
	if lockErr == nil {
		lo := &lib.LockOrder{}
		if err := lo.UnmarshalJSON(memo.data); err != nil {
			return fmt.Errorf("failed to unmarshal lock order json: %w", err)
		}
		t.order = &types.WitnessedOrder{OrderId: lo.OrderId, LockOrder: lo}
		return nil
	}
	// try close order (requires a transfer instruction)
	closeErr := v.ValidateOrderJsonBytes(memo.data, types.CloseOrderType)
	if closeErr == nil {
		co := &lib.CloseOrder{}
		if err := co.UnmarshalJSON(memo.data); err != nil {
			return fmt.Errorf("failed to unmarshal close order json: %w", err)
		}
		t.order = &types.WitnessedOrder{OrderId: co.OrderId, CloseOrder: co}
		if err := t.parseTransfer(); err != nil {
			return err
		}
		return nil
	}
	// memo present but matched neither schema - this used to be entirely silent, which hid a
	// real bug where a genuine close order memo failed schema validation for a subtle reason
	// and was indistinguishable from ordinary non-order memo traffic. Debug (not Warn/Error)
	// because on a real chain this fires for every unrelated third-party memo transaction.
	if logger != nil {
		logger.Debugf("[SOL-TX] memo present but matched neither order schema (lock: %v; close: %v); treating as non-order memo", lockErr, closeErr)
	}
	return nil
}

// parseTransfer locates and decodes the funds-transfer instruction (SPL token or native SOL).
func (t *Transaction) parseTransfer() error {
	// SPL token transfer takes precedence when present
	if spl := t.findInstruction(solana.TokenProgramID); spl != nil {
		return t.parseSPLTransfer(spl)
	}
	// otherwise native SOL transfer
	if sys := t.findInstruction(solana.SystemProgramID); sys != nil {
		return t.parseNativeTransfer(sys)
	}
	return fmt.Errorf("close order transaction has no transfer instruction")
}

// parseSPLTransfer decodes SPL Token `Transfer` (tag 3) or `TransferChecked` (tag 12).
func (t *Transaction) parseSPLTransfer(in *instruction) error {
	if len(in.data) < 9 {
		return fmt.Errorf("spl transfer data too short")
	}
	tag := in.data[0]
	amount := binary.LittleEndian.Uint64(in.data[1:9])
	switch tag {
	case 3: // Transfer: accounts [source, destination, owner]
		if len(in.accounts) < 2 {
			return fmt.Errorf("spl transfer missing accounts")
		}
		t.destination = in.accounts[1].String()
		// mint not present in bare Transfer; the provider resolves it from the destination
		// token account (see needsMintLookup) so decimals can still be looked up
		t.needsMintLookup = true
	case 12: // TransferChecked: accounts [source, mint, destination, owner]
		if len(in.accounts) < 3 {
			return fmt.Errorf("spl transferChecked missing accounts")
		}
		t.mint = in.accounts[1].String()
		t.destination = in.accounts[2].String()
	default:
		return fmt.Errorf("unsupported spl token instruction tag %d", tag)
	}
	t.amount = new(big.Int).SetUint64(amount)
	t.isTransfer = true
	return nil
}

// parseNativeTransfer decodes a System Program `Transfer` (discriminator 2, u64 lamports).
func (t *Transaction) parseNativeTransfer(in *instruction) error {
	if len(in.data) < 12 {
		return fmt.Errorf("system transfer data too short")
	}
	if binary.LittleEndian.Uint32(in.data[0:4]) != 2 {
		return fmt.Errorf("not a system transfer instruction")
	}
	if len(in.accounts) < 2 {
		return fmt.Errorf("system transfer missing accounts")
	}
	lamports := binary.LittleEndian.Uint64(in.data[4:12])
	t.destination = in.accounts[1].String() // recipient wallet
	t.amount = new(big.Int).SetUint64(lamports)
	t.mint = "" // native SOL, no mint
	t.isTransfer = true
	return nil
}

// MatchesOrderDestination verifies the transfer went to the seller's expected destination.
// For SPL transfers, the expected destination is the Associated Token Account derived from
// (recipient wallet, mint) — recipientBytes is the seller's WALLET pubkey and contractBytes is
// the mint. For native SOL transfers, the destination is compared directly to recipientBytes.
func (t *Transaction) MatchesOrderDestination(contractBytes, recipientBytes []byte) (bool, error) {
	if !t.isTransfer {
		return false, nil
	}
	recipient := solana.PublicKeyFromBytes(recipientBytes)
	// native SOL: no mint/contract, compare wallet directly
	if len(contractBytes) == 0 {
		return recipient.String() == t.destination, nil
	}
	mint := solana.PublicKeyFromBytes(contractBytes)
	ata, _, err := solana.FindAssociatedTokenAddress(recipient, mint)
	if err != nil {
		return false, err
	}
	return ata.String() == t.destination, nil
}

// newTransaction builds a Transaction from resolved instructions and identity fields.
func newTransaction(signature, feePayer string, instrs []instruction) *Transaction {
	return &Transaction{signature: signature, feePayer: feePayer, instrs: instrs}
}
