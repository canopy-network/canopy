package canoliq

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// rpc_test.go exercises the read-only HTTP query layer end-to-end against
// the in-process fakeStore. It covers (a) per-helper query correctness on
// seeded state, (b) HTTP route → JSON shape, and (c) the conservation
// equation visible through /v1/pools after a reward sweep.

// newTestRPC mounts the canoliq mux backed by the same fakeStore-driven
// *Plugin used elsewhere in the test suite. Returns the test server, the
// plugin (so callers can adjust height), and the underlying store.
func newTestRPC(t *testing.T) (*httptest.Server, *Plugin, *fakeStore) {
	t.Helper()
	c, store := newTestCanoliq()
	rpc := &RPCServer{plugin: c.plugin}
	mux := http.NewServeMux()
	rpc.registerRoutes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, c.plugin, store
}

// getJSON issues a GET against the test server and decodes into out.
// Asserts the expected status code so failure modes surface a clear message.
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

func TestRPCHealthAndGlobals(t *testing.T) {
	srv, p, s := newTestRPC(t)
	g := &contract.CanoliqGlobals{
		TotalCcnpySupply: 1_500_000,
		TotalPooledCnpy:  2_000_000,
		GenesisComplete:  true,
		CliqTotalSupply:  CLIQTotalSupply,
	}
	s.set(KeyForGlobals(), mustMarshal(g))
	p.setHeight(42)

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
	srv, _, s := newTestRPC(t)
	p := DefaultParams()
	p.FeeBps = 2000 // override to a non-default so we can spot it
	s.set(KeyForParams(), mustMarshal(p))

	var got contract.CanoliqParams
	getJSON(t, srv, "/v1/params", http.StatusOK, &got)
	if got.FeeBps != 2000 {
		t.Fatalf("FeeBps: got %d want 2000", got.FeeBps)
	}
	if got.UserRebateBps+got.TreasuryBps+got.ValidatorBps+got.BuybackBps != 10_000 {
		t.Fatalf("bps must sum to 10000: %+v", &got)
	}
}

func TestRPCAccountComposite(t *testing.T) {
	srv, _, s := newTestRPC(t)
	user := addr20(0x42)
	seedAccount(s, user, 7_000_000)
	s.set(KeyForCCNPYBalance(user), EncodeUint64(500_000))
	s.set(KeyForCLIQBalance(user), EncodeUint64(123_456))
	s.set(KeyForCLIQStake(user), mustMarshal(&contract.CLIQStake{
		Address: user, Amount: 999_999, StakedAtHeight: 5,
	}))
	s.set(KeyForValidatorIncentives(user), EncodeUint64(7_777))

	hex := "0x" + strings.Repeat("42", 20)
	var view AccountView
	getJSON(t, srv, "/v1/account/"+hex, http.StatusOK, &view)
	if view.Address != hex || view.CNPY != 7_000_000 || view.CCNPY != 500_000 ||
		view.CLIQLiquid != 123_456 || view.ValidatorIncentive != 7_777 {
		t.Fatalf("account view: %+v", view)
	}
	if view.CLIQStake == nil || view.CLIQStake.Amount != 999_999 {
		t.Fatalf("stake missing or wrong: %+v", view.CLIQStake)
	}
}

func TestRPCAccountRejectsBadAddress(t *testing.T) {
	srv, _, _ := newTestRPC(t)
	resp, err := http.Get(srv.URL + "/v1/account/notahexaddr")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] == "" {
		t.Fatalf("missing error body: %+v", body)
	}
}

func TestRPCPoolsConservationAfterReward(t *testing.T) {
	srv, p, s := newTestRPC(t)
	// Same setup pattern as TestWhitepaperSection7Reconciliation: fund the
	// committee pool with the post-DAO 0.95X amount, run EndBlock, and
	// verify the HTTP /v1/pools view satisfies the same conservation
	// equation enforced by TestInsuranceConservationFullSplit.
	const X uint64 = 1000
	const postDao = X - X/20 // 950
	g := &contract.CanoliqGlobals{GenesisComplete: true}
	s.set(KeyForGlobals(), mustMarshal(g))
	s.set(KeyForParams(), mustMarshal(DefaultParams()))
	s.set(contract.KeyForFeePool(2), mustMarshal(&contract.Pool{Id: 2, Amount: postDao}))

	c := &Canoliq{Config: Config{ChainId: 2}, plugin: p, fsmId: 1}
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("ProcessRewards: %v", err)
	}

	var pools PoolsView
	getJSON(t, srv, "/v1/pools", http.StatusOK, &pools)
	gAfter := loadGlobals(t, s)
	// User accrual lives on globals.TotalPooledCnpy; the rest splits across
	// the plugin scalars surfaced through /v1/pools.
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
	srv, _, s := newTestRPC(t)
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

	// Missing id returns 404.
	getJSON(t, srv, "/v1/proposal/9999", http.StatusNotFound, nil)
}

func TestRPCSpendAndApprovals(t *testing.T) {
	srv, _, s := newTestRPC(t)
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

func TestRPCBuybackOrder(t *testing.T) {
	srv, _, s := newTestRPC(t)
	order := &contract.BuybackOrder{
		ProposalId:   7,
		CnpyDrawn:    100_000,
		CliqAcquired: 500_000,
		Mode:         contract.BuybackMode_BUYBACK_BURN,
		Executed:     true,
	}
	s.set(KeyForBuybackOrder(7), mustMarshal(order))

	var got contract.BuybackOrder
	getJSON(t, srv, "/v1/buyback/7", http.StatusOK, &got)
	if got.ProposalId != 7 || got.CnpyDrawn != 100_000 || !got.Executed {
		t.Fatalf("buyback round-trip: %+v", &got)
	}
	getJSON(t, srv, "/v1/buyback/8888", http.StatusNotFound, nil)
}

func TestRPCRedemptionAndVesting(t *testing.T) {
	srv, p, s := newTestRPC(t)
	user := addr20(0xDD)

	// Redemption
	red := &contract.Redemption{
		Id: 4, Address: user, CnpyAmount: 250_000, UnbondCompleteHeight: 1000,
	}
	s.set(KeyForRedemption(user, 4), mustMarshal(red))
	hex := "0x" + strings.Repeat("dd", 20)
	var got contract.Redemption
	getJSON(t, srv, "/v1/redemption/"+hex+"/4", http.StatusOK, &got)
	if got.Id != 4 || got.CnpyAmount != 250_000 {
		t.Fatalf("redemption: %+v", &got)
	}

	// Vesting: cliff at 100, end at 200, current height 150 ⇒ 50% unlocked.
	sched := &contract.VestingSchedule{
		Address:     user,
		TotalAmount: 1_000_000,
		CliffHeight: 100,
		StartHeight: 100,
		EndHeight:   200,
	}
	s.set(KeyForVesting(user, 0), mustMarshal(sched))
	s.set(KeyForVestingIndex(user), mustMarshal(&contract.VestingIndex{ScheduleIds: []uint64{0}}))
	p.setHeight(150)

	var vbody struct {
		Address   string         `json:"address"`
		Schedules []*VestingView `json:"schedules"`
	}
	getJSON(t, srv, "/v1/vesting/"+hex, http.StatusOK, &vbody)
	if vbody.Address != hex || len(vbody.Schedules) != 1 {
		t.Fatalf("vesting body: %+v", vbody)
	}
	got0 := vbody.Schedules[0]
	if got0.UnlockedToDate != 500_000 || got0.ClaimableNow != 500_000 || got0.CurrentHeight != 150 {
		t.Fatalf("vesting math: %+v", got0)
	}
}

func TestRPCMethodNotAllowed(t *testing.T) {
	srv, _, _ := newTestRPC(t)
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
