package contract

import (
	"encoding/json"
	"math/big"
	"net/http"

	"google.golang.org/protobuf/proto"
)

// handleQueryVaultDebt serves GET /v1/query/vaultdebt?vaultId=<id>
// Returns the NASM vault's LIVE, stability-fee-scaled current debt --
// i.e. exactly the ScaledNusdDebt(vault, sfIndexNow) figure
// DeliverMessageBurnNusd itself compares msg.NusdAmount against at Step 6
// -- rather than the raw, unscaled NasmVault.nusd_principal field.
//
// [FIX] This route exists because the frontend's Burn NUSD form was
// reading vault.nusdPrincipal directly (the principal snapshotted at
// last accrual) and presenting it as "current debt." Between that
// snapshot and the stability fee's per-block BeginBlock accrual, actual
// debt grows past nusd_principal. A burn submitted for exactly the
// stale figure lands short of currentDebt on-chain, so
// DeliverMessageBurnNusd's fullClosure check (msg.NusdAmount >=
// currentDebt) is false -- the vault is NOT deleted, its escrow Pool is
// NOT deleted, and the vault is left open with a small residual
// nusd_principal (dust), even though the user believed they'd repaid in
// full. This route gives the frontend a live figure to both display and
// submit against, closing that gap.
//
// Follows handleQueryMaxMintableNusd's own [FIX]-documented pattern
// exactly: NEVER fabricate a bare &Contract{plugin: p} and call
// ScaledNusdDebt via a Contract-shaped helper that internally expects a
// real, FSM-issued c.fsmId -- QueryState is the only safe entry point
// for a custom RPC handler with no live tx/block context (see
// QueryState's own doc comment). This handler reads the NasmVault and
// StabilityFeeIndex records directly via p.QueryState and calls
// ScaledNusdDebt (nasm_scaled_debt.go), which takes the decoded structs
// directly and does no state access of its own -- safe to call from
// here unmodified, unlike ResolvePrice/GetAssetTier-style helpers that
// call c.plugin.StateRead internally.
func (p *Plugin) handleQueryVaultDebt(w http.ResponseWriter, r *http.Request) {
	vaultId := r.URL.Query().Get("vaultId")
	if vaultId == "" {
		http.Error(w, `{"error":"missing vaultId query param"}`, http.StatusBadRequest)
		return
	}
	if err := ValidateVaultID(vaultId); err != nil {
		http.Error(w, `{"error":"invalid vaultId: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	const (
		qVault = iota
		qSfIndex
	)
	readResp, rErr := p.QueryState(0, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: qVault, Key: KeyForNasmVault(vaultId)},
			{QueryId: qSfIndex, Key: KeyForStabilityFeeIndex()},
		},
	})
	if rErr != nil {
		http.Error(w, `{"error":"query state failed: `+rErr.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if readResp.Error != nil {
		http.Error(w, `{"error":"state read error: `+readResp.Error.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	vaultBytes := entryValue(readResp, qVault)
	if len(vaultBytes) == 0 {
		http.Error(w, `{"error":"vault not found"}`, http.StatusNotFound)
		return
	}
	vault := &NasmVault{}
	if uErr := proto.Unmarshal(vaultBytes, vault); uErr != nil {
		http.Error(w, `{"error":"failed to decode NasmVault record: `+uErr.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// found=false at genesis defaults to RAY, matching every other read
	// site of this same {35}/StabilityFeeIndex record (burn_nusd.go,
	// liquidate_nasm_vault.go, GetStabilityFeeIndex's own doc comment).
	var sfIndexNow *big.Int
	sfIndexBytes := entryValue(readResp, qSfIndex)
	if len(sfIndexBytes) == 0 {
		sfIndexNow = RAY
	} else {
		sf := &StabilityFeeIndex{}
		if uErr := proto.Unmarshal(sfIndexBytes, sf); uErr != nil {
			http.Error(w, `{"error":"failed to decode StabilityFeeIndex record: `+uErr.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		sfIndexNow = DecodeUint128(sf.SfIndex)
	}

	currentDebt, sdErr := ScaledNusdDebt(vault, sfIndexNow)
	if sdErr != nil {
		http.Error(w, `{"error":"`+sdErr.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"vaultId":       vaultId,
		"nusdPrincipal": vault.NusdPrincipal,
		"currentDebt":   currentDebt,
		"note":          "currentDebt is live, stability-fee-scaled -- use this for burn_nusd, not nusdPrincipal",
	})
}
