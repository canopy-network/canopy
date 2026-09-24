package canoliq

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/canopy-network/go-plugin/contract"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/types/known/anypb"
)

// mainnet_replay_parity_test.go pins consensus parity with the plugin that ran
// mainnet committee 29 from its activation (plugin-go-v2026.266.23).
//
// Every node that syncs committee 29 from genesis re-executes the blocks that
// v2026.266.23 produced, but with whatever plugin binary it runs today. Any
// consensus-visible change must therefore be gated on a height (see
// rewardfix.go) and leave every earlier block byte-identical. This test drives
// a mainnet-shaped scenario — pinned-height genesis from genesis.mainnet.json,
// an uncapped deposit while Canopy stake is still zero, a foreign validator
// and a foreign delegate compounding, a second deposit, a redemption — and
// compares the full plugin-visible state after every block against hashes
// captured from v2026.266.23 itself (testdata/mainnet_prefix_parity.golden).
//
// Regenerate only from the v2026.266.23 tree:
//   PARITY_DUMP=1 go test -run TestMainnetPreFixReplayParity ./canoliq/

const parityGoldenPath = "testdata/mainnet_prefix_parity.golden"

func parityHex(h string) []byte {
	b, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	return b
}

func parityStateHash(s *fakeStore) string {
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte{byte(len(k) >> 8), byte(len(k))})
		h.Write([]byte(k))
		v := s.data[k]
		h.Write([]byte{byte(len(v) >> 24), byte(len(v) >> 16), byte(len(v) >> 8), byte(len(v))})
		h.Write(v)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// paritySetValidator writes a Canopy core Validator record exactly as core
// serializes it (lib/.proto/validator.proto). It is hand-encoded because the
// plugin's contract.Validator mirror gained fields over time, and the
// fixture must be the same bytes whichever plugin tree runs it.
func paritySetValidator(t *testing.T, s *fakeStore, addr []byte, stake uint64, delegate bool) {
	t.Helper()
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendBytes(b, addr)
	b = protowire.AppendTag(b, 4, protowire.VarintType)
	b = protowire.AppendVarint(b, stake)
	b = protowire.AppendTag(b, 5, protowire.BytesType)
	b = protowire.AppendBytes(b, protowire.AppendVarint(nil, 29))
	b = protowire.AppendTag(b, 8, protowire.BytesType)
	b = protowire.AppendBytes(b, addr)
	if delegate {
		b = protowire.AppendTag(b, 9, protowire.VarintType)
		b = protowire.AppendVarint(b, 1)
	}
	b = protowire.AppendTag(b, 10, protowire.VarintType)
	b = protowire.AppendVarint(b, 1)
	s.set(contract.KeyForValidator(addr), b)
}

func paritySetSupply(t *testing.T, s *fakeStore, staked uint64) {
	t.Helper()
	bz, err := contract.Marshal(&contract.Supply{Staked: staked})
	if err != nil {
		t.Fatal(err)
	}
	s.set(contract.KeyForSupply(), bz)
}

// runMainnetParityScenario returns the state hash after every block.
func runMainnetParityScenario(t *testing.T) []string {
	t.Helper()
	c, s := newTestCanoliq()
	cfg := Config{
		ChainId:          29,
		DataDirPath:      "/tmp/canoliq-test",
		Profile:          ProfileMainnet,
		GenesisPath:      "genesis.mainnet.json",
		ActivationHeight: 10,
	}
	c.Config = cfg
	c.plugin.config = cfg

	valA := parityHex("737ef80e8476450b6970dd6c1e44c93699c8e0c0")
	del := parityHex("725f4f9983aaa3c60cc318bff1a196f86c65eb97")
	userA := parityHex("111e0583347c901fa3ccf3bc1de4840349565e61")
	userB := parityHex("9ddb1fe7e9539e08e035118f15f50b2c089485e3")
	seedAccount(s, userA, 100_000_000)
	seedAccount(s, userB, 600_000_000)

	valStake := uint64(10_000_000_000)
	delStake := uint64(0)
	paritySetSupply(t, s, 0)
	paritySetValidator(t, s, valA, valStake, false)

	var hashes []string
	for h := uint64(1); h <= 80; h++ {
		c.plugin.setHeight(h)
		if r := c.BeginBlock(&contract.PluginBeginRequest{Height: h}); r.Error != nil {
			t.Fatalf("begin block %d: %v", h, r.Error)
		}
		var tx *anypb.Any
		var err error
		switch h {
		case 15: // uncapped: Canopy Supply.Staked is still 0
			tx, err = anypb.New(&contract.MessageCanoliqDeposit{FromAddress: userA, Amount: 10_000_000})
		case 35:
			tx, err = anypb.New(&contract.MessageCanoliqDeposit{FromAddress: userB, Amount: 508_500_661})
		case 45:
			tx, err = anypb.New(&contract.MessageCanoliqRedeem{FromAddress: userA, CcnpyAmount: 2_000_000})
		}
		if err != nil {
			t.Fatal(err)
		}
		if tx != nil {
			r := c.DeliverTx(&contract.PluginDeliverRequest{Tx: &contract.Transaction{Msg: tx, Fee: 10_000}})
			if r.Error != nil {
				t.Fatalf("deliver at %d: %v", h, r.Error)
			}
		}
		// Core compounds committee reward into the bonds before the plugin's
		// EndBlock runs (fsm/automatic.go: DistributeCommitteeRewards).
		if h > 10 {
			valStake += 11_400_000
			paritySetValidator(t, s, valA, valStake, false)
		}
		if h == 30 {
			delStake = 30_000_000_000
		} else if h > 30 {
			delStake += 1_425_000
		}
		if delStake > 0 {
			paritySetValidator(t, s, del, delStake, true)
		}
		if h >= 25 {
			paritySetSupply(t, s, valStake+delStake)
		}
		if r := c.EndBlock(&contract.PluginEndRequest{Height: h}); r.Error != nil {
			t.Fatalf("end block %d: %v", h, r.Error)
		}
		hashes = append(hashes, parityStateHash(s))
	}
	return hashes
}

func TestMainnetPreFixReplayParity(t *testing.T) {
	got := runMainnetParityScenario(t)
	if os.Getenv("PARITY_DUMP") != "" {
		bz, _ := json.MarshalIndent(got, "", " ")
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(parityGoldenPath, bz, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %d block hashes to %s", len(got), parityGoldenPath)
		return
	}
	bz, err := os.ReadFile(parityGoldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var want []string
	if err := json.Unmarshal(bz, &want); err != nil {
		t.Fatal(err)
	}
	if len(want) != len(got) {
		t.Fatalf("block count: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("state diverges from v2026.266.23 after block %d", i+1)
		}
	}
}
