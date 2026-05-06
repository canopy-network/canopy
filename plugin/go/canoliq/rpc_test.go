package canoliq

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// rpc_test.go exercises the read-only HTTP query layer end-to-end against
// the in-process fakeStore. Each test seeds state via fakeStore, ticks
// EndBlock to publish a snapshot, and then asserts the HTTP route output.
//
// Tests that depend on per-address routes (account composite, vesting,
// redemption, vote, buyback) are intentionally absent — those routes are
// deferred to Phase 3 §1.1; per-address records are not enumerable from
// a snapshot built off canoliq's existing indexes.

// newTestRPC mounts the canoliq mux backed by the same fakeStore-driven
// *Plugin used elsewhere in the test suite. Tests must call refresh() to
// populate the snapshot before issuing HTTP requests.
func newTestRPC(t *testing.T) (*httptest.Server, *Canoliq, *fakeStore, func(uint64)) {
	t.Helper()
	c, store := newTestCanoliq()
	rpc := &RPCServer{plugin: c.plugin}
	mux := http.NewServeMux()
	rpc.registerRoutes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	refresh := func(h uint64) {
		c.plugin.setHeight(h)
		if err := c.refreshSnapshot(h); err != nil {
			t.Fatalf("refreshSnapshot: %v", err)
		}
	}
	return srv, c, store, refresh
}

// getJSON issues a GET against the test server and decodes into out.
// Asserts the expected status code so failure modes surface clearly.
func getJSON(t *testing.T, srv *httptest.Server, path string, wantStatus int, out any) {
	t.Helper()
	resp, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("GET %s status: got %d want %d", path, resp.StatusCode, wantStatus)
	}
	if out == nil {
		return
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("GET %s decode: %v", path, err)
	}
}

func TestRPCHealthBeforeSnapshot(t *testing.T) {
	srv, _, _, _ := newTestRPC(t)
	// No snapshot yet: should still answer with sane defaults.
	var health HealthView
	getJSON(t, srv, "/v1/health", http.StatusOK, &health)
	if health.Height != 0 || health.GenesisComplete || health.ChainID != 2 {
		t.Fatalf("cold-start health: %+v", health)
	}
}

func TestRPCHealthAndGlobalsAfterRefresh(t *testing.T) {
	srv, _, s, refresh := newTestRPC(t)
	g := &contract.CanoliqGlobals{
		TotalCcnpySupply: 1_500_000,
		TotalPooledCnpy:  2_000_000,
		GenesisComplete:  true,
		CliqTotalSupply:  CLIQTotalSupply,
	}
	s.set(KeyForGlobals(), mustMarshal(g))
	refresh(42)

	var health HealthView
	getJSON(t, srv, "/v1/health", http.StatusOK, &health)
	if health.Height != 42 || !health.GenesisComplete || health.ChainID != 2 {
		t.Fatalf("health: %+v", health)
	}
	var got contract.CanoliqGlobals
	getJSON(t, srv, "/v1/globals", http.StatusOK, &got)
	if got.TotalCcnpySupply != g.TotalCcnpySupply || got.TotalPooledCnpy != g.TotalPooledCnpy {
		t.Fatalf("globals round-trip: got %+v", &got)
	}
}

func TestRPCParamsRoundTrip(t *testing.T) {
	srv, _, s, refresh := newTestRPC(t)
	p := DefaultParams()
	p.FeeBps = 2000
	s.set(KeyForParams(), mustMarshal(p))
	refresh(1)

	var got contract.CanoliqParams
	getJSON(t, srv, "/v1/params", http.StatusOK, &got)
	if got.FeeBps != 2000 {
		t.Fatalf("FeeBps: got %d want 2000", got.FeeBps)
	}
	if got.UserRebateBps+got.TreasuryBps+got.ValidatorBps+got.BuybackBps != 10_000 {
		t.Fatalf("bps must sum to 10000: %+v", &got)
	}
}

func TestRPCPoolsConservationAfterReward(t *testing.T) {
	srv, c, s, refresh := newTestRPC(t)
	// Same setup pattern as TestWhitepaperSection7Reconciliation: fund the
	// committee pool with the post-DAO 0.95X amount, run EndBlock (which
	// applies the fee split AND refreshes the snapshot), then verify the
	// HTTP /v1/pools view satisfies conservation.
	const X uint64 = 1000
	const postDao = X - X/20 // 950
	g := &contract.CanoliqGlobals{GenesisComplete: true}
	s.set(KeyForGlobals(), mustMarshal(g))
	s.set(KeyForParams(), mustMarshal(DefaultParams()))
	s.set(contract.KeyForFeePool(2), mustMarshal(&contract.Pool{Id: 2, Amount: postDao}))

	resp := c.EndBlock(&contract.PluginEndRequest{Height: 1})
	if resp.Error != nil {
		t.Fatalf("EndBlock: %v", resp.Error)
	}
	_ = refresh // EndBlock already published; refresh helper unused here.

	var pools PoolsView
	getJSON(t, srv, "/v1/pools", http.StatusOK, &pools)
	gAfter := loadGlobals(t, s)
	valSum := uint64(0)
	for _, v := range pools.ValidatorIncentives {
		valSum += v.Amount
	}
	total := gAfter.TotalPooledCnpy + pools.TreasuryCNPY + pools.InsurancePool +
		valSum + pools.BuybackPool
	if total != postDao {
		t.Fatalf("conservation: %d != %d (pooled=%d treasury=%d insurance=%d val=%d buyback=%d)",
			total, postDao, gAfter.TotalPooledCnpy, pools.TreasuryCNPY, pools.InsurancePool, valSum, pools.BuybackPool)
	}
}

func TestRPCProposalLifecycle(t *testing.T) {
	srv, _, s, refresh := newTestRPC(t)
	prop := &contract.Proposal{
		Id:                  3,
		Proposer:            addr20(0xAA),
		CreationHeight:      10,
		ExpiryHeight:        100,
		SnapshotTotalStaked: 10_000,
		Status:              contract.ProposalStatus_PROPOSAL_ACTIVE,
	}
	s.set(KeyForProposal(3), mustMarshal(prop))
	s.set(KeyForProposalIndex(), mustMarshal(&contract.ProposalIndex{Ids: []uint64{3}}))
	refresh(50)

	var idx struct {
		Ids []uint64 `json:"ids"`
	}
	getJSON(t, srv, "/v1/proposals", http.StatusOK, &idx)
	if len(idx.Ids) != 1 || idx.Ids[0] != 3 {
		t.Fatalf("proposal index: %+v", idx)
	}

	var got contract.Proposal
	getJSON(t, srv, "/v1/proposal/3", http.StatusOK, &got)
	if got.Id != 3 || got.SnapshotTotalStaked != 10_000 {
		t.Fatalf("proposal round-trip: %+v", &got)
	}
	getJSON(t, srv, "/v1/proposal/9999", http.StatusNotFound, nil)
}

func TestRPCSpendAndApprovals(t *testing.T) {
	srv, _, s, refresh := newTestRPC(t)
	signer := addr20(0xBB)
	params := DefaultParams()
	params.MultisigSigners = [][]byte{signer}
	params.MultisigThreshold = 1
	s.set(KeyForParams(), mustMarshal(params))

	spend := &contract.TreasurySpend{
		Id:               5,
		ProposalId:       11,
		ExecutableHeight: 200,
		RequiresMultisig: true,
		Payload: &contract.ProposalTreasurySpend{
			Recipient:    addr20(0xCC),
			Amount:       1_000_000_000,
			Denomination: contract.SpendDenomination_SPEND_CNPY,
		},
	}
	s.set(KeyForTreasurySpend(5), mustMarshal(spend))
	s.set(KeyForSpendIndex(), mustMarshal(&contract.ProposalIndex{Ids: []uint64{5}}))
	s.set(KeyForMultisigApproval(5, signer), mustMarshal(&contract.MultisigApproval{
		SpendId: 5, Signer: signer, Height: 50,
	}))
	refresh(100)

	var idx struct {
		Ids []uint64 `json:"ids"`
	}
	getJSON(t, srv, "/v1/spends", http.StatusOK, &idx)
	if len(idx.Ids) != 1 || idx.Ids[0] != 5 {
		t.Fatalf("spend index: %+v", idx)
	}

	var got contract.TreasurySpend
	getJSON(t, srv, "/v1/spend/5", http.StatusOK, &got)
	if got.Id != 5 || got.Payload.GetAmount() != 1_000_000_000 {
		t.Fatalf("spend round-trip: %+v", &got)
	}

	var apv MultisigApprovalsView
	getJSON(t, srv, "/v1/spend/5/approvals", http.StatusOK, &apv)
	if apv.Threshold != 1 || len(apv.Approvals) != 1 || apv.Approvals[0].SpendId != 5 {
		t.Fatalf("approvals: %+v", apv)
	}
}

func TestRPCValidatorsAndStakers(t *testing.T) {
	srv, _, s, refresh := newTestRPC(t)
	val := addr20(0xEE)
	staker := addr20(0xFF)
	registry := &contract.ValidatorRegistry{
		Entries: []*contract.ValidatorRegistryEntry{
			{Address: val, Stake: 1_000_000},
		},
	}
	s.set(KeyForValidatorRegistry(), mustMarshal(registry))
	s.set(KeyForValidatorIncentives(val), EncodeUint64(42))
	s.set(KeyForCLIQStake(staker), mustMarshal(&contract.CLIQStake{
		Address: staker, Amount: 500_000, StakedAtHeight: 7,
	}))
	s.set(KeyForCLIQStakeIndex(), mustMarshal(&contract.CLIQStakeIndex{
		Addresses: [][]byte{staker},
	}))
	refresh(100)

	var reg contract.ValidatorRegistry
	getJSON(t, srv, "/v1/validators", http.StatusOK, &reg)
	if len(reg.Entries) != 1 || reg.Entries[0].Stake != 1_000_000 {
		t.Fatalf("validators: %+v", reg.Entries)
	}

	var stakers struct {
		Stakers []*StakerView `json:"stakers"`
	}
	getJSON(t, srv, "/v1/stakers", http.StatusOK, &stakers)
	if len(stakers.Stakers) != 1 || stakers.Stakers[0].Amount != 500_000 ||
		stakers.Stakers[0].StakedAtHeight != 7 {
		t.Fatalf("stakers: %+v", stakers.Stakers)
	}
}

func TestRPCMethodNotAllowed(t *testing.T) {
	srv, _, _, _ := newTestRPC(t)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/health", nil)
	if err != nil {
		t.Fatalf("new req: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status: got %d want 405", resp.StatusCode)
	}
}

func TestRPCStartRPCServerEmptyAddrDisabled(t *testing.T) {
	srv, err := StartRPCServer(&Plugin{}, "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if srv != nil {
		t.Fatalf("expected nil server when addr empty")
	}
}
