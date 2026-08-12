package contract

// faucet.go implements the devnet-only faucet: a fee-exempt claim_faucet
// tx that credits the signer's AssetBalance ({37}) for all 4 faucet-
// supported assets (BTC, ETH, USDC, CNPY) in one call, gated per-asset by
// FaucetClaimRecord's ({38}) FAUCET_COOLDOWN_BLOCKS cooldown. See
// MessageClaimFaucet's own doc comment in arbor.proto for the full
// design rationale (fee exemption, per-asset-not-all-or-nothing
// semantics, testnet-only status with no build gate yet).
//
// CNPY here is a devnet TEST asset, NOT this chain's native gas/staking
// token -- confirmed against chain.json's "symbol": "ARB". MessageSend's
// own doc comment calling its amount field "uCNPY" is stale wording
// inherited from the upstream Canopy template (same class of leftover
// artifact as the pre-Jul-25 routing-drop bug), not evidence CNPY is
// load-bearing for gas in this fork.

import (
	"google.golang.org/protobuf/types/known/anypb"
)

// FaucetSupportedAssets is the fixed, ordered list of asset IDs the
// faucet dispenses. Order matters only for deterministic event emission
// order within a single claim_faucet tx -- cooldown enforcement itself is
// independent per asset regardless of iteration order.
var FaucetSupportedAssets = []string{"BTC", "ETH", "USDC", "CNPY"}

// FaucetClaimAmounts: flat uint64 units, no per-asset decimals convention
// exists anywhere in this codebase (arbor_state.proto has no decimals
// field), so these are plain round numbers, not scaled to any real-world
// BTC/ETH/USDC decimal precision.
var FaucetClaimAmounts = map[string]uint64{
	"BTC":  5,
	"ETH":  15,
	"USDC": 10000,
	"CNPY": 100,
}

// FaucetCooldownBlocks: 20h equivalent at this chain's 20s block time
// (20 * 3600 / 20 = 3600 blocks). Per-address-per-asset, enforced via
// FaucetClaimRecord ({38}). See that message's doc comment in
// arbor_state.proto for why block-height, not wall-clock, is the basis --
// DeliverTx handlers have no deterministic wall-clock source across
// validators, same reasoning as every other cooldown/staleness check in
// this codebase (e.g. stalenessThresholdTable).
const FaucetCooldownBlocks = 3600

// CheckMessageClaimFaucet statelessly validates a claim_faucet message.
// No asset_id/amount fields to validate (MessageClaimFaucet is
// intentionally empty besides the implicit signer -- see its own doc
// comment in arbor.proto) -- just the 20-byte address shape, mirroring
// every other Check* function's address-length check in this codebase.
func (c *Contract) CheckMessageClaimFaucet(msg *MessageClaimFaucet) *PluginCheckResponse {
	if len(msg.Address) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Address}}
}

// DeliverMessageClaimFaucet applies a validated claim_faucet tx. Per-asset,
// not all-or-nothing (see this file's package doc and MessageClaimFaucet's
// arbor.proto comment for the anti-spam reasoning): each of the 4
// supported assets is independently checked against its own
// FaucetClaimRecord cooldown. An asset still on cooldown is skipped (a
// FaucetClaimEvent with status="skipped_cooldown" is still emitted, so the
// event log alone reconstructs full claim history including skips) rather
// than causing the whole tx to fail -- only a genuine state-read/write
// error fails the tx.
//
// fee is accepted for call-site signature consistency with every other
// Deliver* handler (see DeliverTx's switch), but unused: claim_faucet is
// fee-exempt (see CheckTx's own carve-out), so fee is always 0 here.
func (c *Contract) DeliverMessageClaimFaucet(msg *MessageClaimFaucet, fee uint64) *PluginDeliverResponse {
	if len(msg.Address) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}

	height := c.plugin.CurrentHeight()

	// Build the batched read: for each of the 4 assets, one AssetBalance
	// key and one FaucetClaimRecord key -- 8 keys total, mirroring
	// deposit.go's own multi-key StateRead + entryValue() convention.
	type assetQueryIds struct {
		balanceQ uint64
		claimQ   uint64
	}
	queryIds := make(map[string]assetQueryIds, len(FaucetSupportedAssets))
	keys := make([]*PluginKeyRead, 0, len(FaucetSupportedAssets)*2)
	var nextQ uint64 = 1
	for _, assetID := range FaucetSupportedAssets {
		bq, cq := nextQ, nextQ+1
		nextQ += 2
		queryIds[assetID] = assetQueryIds{balanceQ: bq, claimQ: cq}
		keys = append(keys,
			&PluginKeyRead{QueryId: bq, Key: KeyForAssetBalance(assetID, msg.Address)},
			&PluginKeyRead{QueryId: cq, Key: KeyForFaucetClaimRecord(assetID, msg.Address)},
		)
	}

	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{Keys: keys})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if readResp.Error != nil {
		return &PluginDeliverResponse{Error: readResp.Error}
	}

	sets := make([]*PluginSetOp, 0, len(FaucetSupportedAssets)*2)
	events := make([]*Event, 0, len(FaucetSupportedAssets))

	for _, assetID := range FaucetSupportedAssets {
		qids := queryIds[assetID]

		// Load existing FaucetClaimRecord (zero-value if never claimed).
		claimBytes := entryValue(readResp, qids.claimQ)
		claimRec := &FaucetClaimRecord{}
		if len(claimBytes) > 0 {
			if uErr := Unmarshal(claimBytes, claimRec); uErr != nil {
				return &PluginDeliverResponse{Error: uErr}
			}
		}

		// Cooldown check -- skip this asset, not the whole tx, if still
		// within FaucetCooldownBlocks of the last claim.
		var blocksSinceLast uint64
		onCooldown := false
		if claimRec.LastClaimBlock > 0 {
			if height > claimRec.LastClaimBlock {
				blocksSinceLast = height - claimRec.LastClaimBlock
			}
			if blocksSinceLast < FaucetCooldownBlocks {
				onCooldown = true
			}
		}

		if onCooldown {
			blocksRemaining := FaucetCooldownBlocks - blocksSinceLast
			eventPayload := &FaucetClaimEvent{
				BlockHeight:     height,
				Address:         msg.Address,
				AssetId:         assetID,
				Status:          "skipped_cooldown",
				Amount:          0,
				BlocksRemaining: blocksRemaining,
			}
			anyMsg, aErr := anypb.New(eventPayload)
			if aErr != nil {
				return &PluginDeliverResponse{Error: ErrMarshal(aErr)}
			}
			events = append(events, &Event{
				EventType: "faucet_claim",
				Msg:       &Event_Custom{Custom: &EventCustom{Msg: anyMsg}},
			})
			continue
		}

		// Off cooldown -- load existing AssetBalance (zero-value if never
		// credited before), credit it via the checked-add helper, and
		// stamp FaucetClaimRecord with the current height.
		balBytes := entryValue(readResp, qids.balanceQ)
		bal := &AssetBalance{Address: msg.Address, AssetId: assetID}
		if len(balBytes) > 0 {
			if uErr := Unmarshal(balBytes, bal); uErr != nil {
				return &PluginDeliverResponse{Error: uErr}
			}
			bal.Address = msg.Address
			bal.AssetId = assetID
		}

		amount := FaucetClaimAmounts[assetID]
		if cErr := creditAssetBalanceAmount(assetID, msg.Address, bal, amount); cErr != nil {
			return &PluginDeliverResponse{Error: cErr}
		}

		balBytesOut, mErr := Marshal(bal)
		if mErr != nil {
			return &PluginDeliverResponse{Error: mErr}
		}

		claimRec.Address = msg.Address
		claimRec.AssetId = assetID
		claimRec.LastClaimBlock = height
		claimBytesOut, mErr := Marshal(claimRec)
		if mErr != nil {
			return &PluginDeliverResponse{Error: mErr}
		}

		sets = append(sets,
			&PluginSetOp{Key: KeyForAssetBalance(assetID, msg.Address), Value: balBytesOut},
			&PluginSetOp{Key: KeyForFaucetClaimRecord(assetID, msg.Address), Value: claimBytesOut},
		)

		eventPayload := &FaucetClaimEvent{
			BlockHeight:     height,
			Address:         msg.Address,
			AssetId:         assetID,
			Status:          "credited",
			Amount:          amount,
			BlocksRemaining: 0,
		}
		anyMsg, aErr := anypb.New(eventPayload)
		if aErr != nil {
			return &PluginDeliverResponse{Error: ErrMarshal(aErr)}
		}
		events = append(events, &Event{
			EventType: "faucet_claim",
			Msg:       &Event_Custom{Custom: &EventCustom{Msg: anyMsg}},
		})
	}

	if len(sets) > 0 {
		writeResp, wErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{Sets: sets})
		if wErr != nil {
			return &PluginDeliverResponse{Error: wErr}
		}
		if writeResp.Error != nil {
			return &PluginDeliverResponse{Error: writeResp.Error}
		}
	}

	return &PluginDeliverResponse{Events: events}
}
