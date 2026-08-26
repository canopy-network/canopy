package contract

/*
faucet_pool.go implements a POOL-BASED native-token (ARB) faucet — distinct
from faucet.go's MessageClaimFaucet, which MINTS 4 devnet test assets
(BTC/ETH/USDC/CNPY) out of thin air with no funding step. This faucet does
the opposite: it drains a real, funded account (the "pool") via ordinary
signed MessageSend transactions, so the pool can run dry and claims stop
naturally once it does — there is no minting here, only redistribution of
whatever the pool holds.

WHY THIS CAN'T BE A DeliverTx / STATE-MACHINE MESSAGE LIKE MessageClaimFaucet:
the whole point (per the person operating this) is that CLAIMANTS PAY NO
GAS. A DeliverTx message is signed and fee-paid by whoever submits it — so
a gasless claim can only work if something OTHER than the claimant signs
and pays for the transfer. That means an off-chain, always-on process has
to hold the pool address's private key and sign ordinary MessageSend txs on
the claimant's behalf. This is fundamentally different from every other
plugin message in this codebase: it is the one place in the plugin that
autonomously moves real funds without a human co-signing each transfer.
Treat any change here with the same care as the signing code in
scripts/submit_tx.go and scripts/local_keystore.go, which this reuses.

DESIGN:
  - Pool key stays in the operator's local Canopy keystore (~/.canopy/keystore.json),
    decrypted ONCE at process startup via the address+password given in
    FAUCET_POOL_ADDRESS / FAUCET_POOL_PASSWORD, then held decrypted in memory
    for the life of the process. It is never re-read from disk per-claim,
    never logged, and never echoed back in any HTTP response.
  - Claim amount is a single fixed value (FAUCET_POOL_CLAIM_AMOUNT_UARB,
    default below), not a randomized range — a random per-claim amount adds
    claim-log ambiguity and invites "did I get shorted" support questions
    for no real benefit over a flat, predictable number.
  - Cooldown is per-address only (24h equivalent in blocks), enforced by
    this process's own state — not the state machine, since the transfer
    itself isn't a state-machine message. This is the SAME persistence
    pattern as quest_xp.go: an in-memory map, snapshotted to a JSON file on
    every claim, reloaded on startup. See questXPSaveState/LoadState for
    the identical approach and its rationale (single-process-restart
    protection only, not consensus-replicated — there is exactly one pool
    signer process, by design, so that scope match is intentional here,
    not a limitation to fix later).
  - No pool-balance safety floor: claims are checked against the pool's
    CURRENT on-chain balance immediately before sending (not a cached
    number), and a claim that would exceed the pool's balance is refused
    with a clear "pool is empty" error rather than attempted and failed
    on-chain. The pool is simply allowed to run to zero — that's the
    explicit, deliberate behavior asked for, not an oversight.
  - No per-IP tracking. This is a DELIBERATE scope decision, not a gap:
    per-IP limiting was considered and explicitly deferred to a future
    frontend/proxy-layer check if ever needed — it cannot live here
    correctly anyway, since this process only sees whatever the reverse
    proxy forwards it, and trusting a spoofable header for a security
    control would be worse than no IP check at all.

CONFIGURATION (env vars):
  FAUCET_POOL_ADDRESS         — hex address of the funded pool account (required to
                                 enable this faucet; if unset, claims are refused
                                 with a clear "faucet not configured" error rather
                                 than the process failing to start — same
                                 fail-closed pattern as QUEST_ADMIN_TOKEN).
  FAUCET_POOL_PASSWORD        — password to decrypt that address's key from the
                                 local keystore at startup. Required alongside
                                 FAUCET_POOL_ADDRESS.
  FAUCET_POOL_CLAIM_AMOUNT_UARB — amount sent per successful claim, in base units
                                 (micro-ARB, 6 decimals per chain.json — so
                                 2_000_000 = 2 ARB). Default 2_000_000 (2 ARB).
  FAUCET_POOL_COOLDOWN_BLOCKS — blocks between claims per address. Default 4320
                                 (24h at this chain's 20s block time — matches
                                 quest_xp.go's own dayBlocks constant exactly,
                                 reused here rather than redefined).
  FAUCET_POOL_STATE_FILE      — path to the cooldown-state JSON snapshot. Default
                                 "./faucet_pool_state.json". Set an absolute path
                                 in production, same reasoning as
                                 QUEST_XP_STATE_FILE.
  FAUCET_POOL_FEE_UARB        — fee this process pays (as sender) per claim send,
                                 in base units. Default 10000, matching every
                                 other tx fee constant in submit_tx.go.
*/

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	bls12381 "github.com/drand/kyber-bls12381"
	"github.com/drand/kyber/sign/bdn"
)

// ---- config ----

const (
	defaultFaucetPoolClaimAmountUarb = 2_000_000 // 2 ARB, per your call — flat amount, not a range
	defaultFaucetPoolCooldownBlocks  = dayBlocks // reuse quest_xp.go's existing 24h-equivalent constant
	defaultFaucetPoolFeeUarb         = 10000     // matches submit_tx.go's fee convention
)

func faucetPoolAddress() string {
	return strings.ToLower(strings.TrimPrefix(os.Getenv("FAUCET_POOL_ADDRESS"), "0x"))
}

func faucetPoolPassword() string {
	return os.Getenv("FAUCET_POOL_PASSWORD")
}

func faucetPoolClaimAmountUarb() uint64 {
	if v := os.Getenv("FAUCET_POOL_CLAIM_AMOUNT_UARB"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return defaultFaucetPoolClaimAmountUarb
}

func faucetPoolCooldownBlocks() int64 {
	if v := os.Getenv("FAUCET_POOL_COOLDOWN_BLOCKS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return int64(defaultFaucetPoolCooldownBlocks)
}

func faucetPoolFeeUarb() uint64 {
	if v := os.Getenv("FAUCET_POOL_FEE_UARB"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			return n
		}
	}
	return defaultFaucetPoolFeeUarb
}

func faucetPoolNetworkID() uint64 {
	// Reuses ARBOR_NETWORK_ID, the same env var scripts/submit_tx.go already
	// reads for this — one override name across the codebase rather than a
	// second, faucet-specific one. Per the grad node reference guide, this
	// can be left unset there too: networkID is 1 on both local devnet and
	// the grad node observed so far. Only chainID has been seen to differ
	// between deployments.
	if v := os.Getenv("ARBOR_NETWORK_ID"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			return n
		}
	}
	return 1
}

func faucetPoolChainID() uint64 {
	// Reuses ARBOR_CHAIN_ID, same reasoning as faucetPoolNetworkID above.
	// DO NOT hardcode this to 1 — confirmed directly against a live grad
	// node block header that chainID there is 407, not 1. A hardcoded 1
	// here would build transactions signed for the wrong chain and get
	// them rejected (or worse, silently misinterpreted) the moment this
	// runs against anything other than local devnet.
	if v := os.Getenv("ARBOR_CHAIN_ID"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			return n
		}
	}
	return 1
}

func faucetPoolStateFilePath() string {
	if v := os.Getenv("FAUCET_POOL_STATE_FILE"); v != "" {
		return v
	}
	return "./faucet_pool_state.json"
}

// ---- pool signing key: decrypted once at startup, held in memory ----

type faucetPoolKey struct {
	Address    string // hex, lowercase, no 0x prefix
	PublicKey  string // hex
	PrivateKey string // hex — sensitive; never logged, never serialized to the state file
}

// ---- local keystore decryption ----
//
// Deliberately reimplements local_keystore.go's approach (read+decrypt
// ~/.canopy/keystore.json directly on this machine) rather than calling a
// node's /v1/admin/keystore-get RPC. That older remote-decrypt approach was
// already tried and abandoned for scripts/submit_tx.go specifically because
// it requires the node itself to hold and serve key material — the wrong
// assumption for a shared grad node this process doesn't control. The same
// reasoning applies here, doubly so: this key controls real pool funds.

const (
	faucetKdfDefaultTime     = 3
	faucetKdfDefaultMemoryKB = 32 * 1024
	faucetKdfDefaultThreads  = 4
	faucetKdfKeyLen          = 32
)

type faucetEncryptedPrivateKey struct {
	PublicKey   string `json:"publicKey"`
	Salt        string `json:"salt"`
	Encrypted   string `json:"encrypted"`
	KeyAddress  string `json:"keyAddress"`
	KeyNickname string `json:"keyNickname,omitempty"`
	KdfTime     uint32 `json:"kdfTime,omitempty"`
	KdfMemoryKB uint32 `json:"kdfMemoryKb,omitempty"`
	KdfThreads  uint8  `json:"kdfThreads,omitempty"`
}

type faucetLocalKeystoreFile struct {
	AddressMap map[string]*faucetEncryptedPrivateKey `json:"addressMap"`
}

func faucetPoolLoadKeyFromLocalKeystore(address, password string) (*faucetPoolKey, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir: %w", err)
	}
	path := filepath.Join(home, ".canopy", "keystore.json")
	bz, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read keystore at %s: %w", path, err)
	}
	var ks faucetLocalKeystoreFile
	if err := json.Unmarshal(bz, &ks); err != nil {
		return nil, fmt.Errorf("parse keystore: %w", err)
	}
	epk, ok := ks.AddressMap[address]
	if !ok {
		return nil, fmt.Errorf("address %s not found in local keystore %s", address, path)
	}
	privKeyBytes, err := faucetDecryptPrivateKeyLocal(epk, []byte(password))
	if err != nil {
		return nil, fmt.Errorf("decrypt key for %s: %w", address, err)
	}
	return &faucetPoolKey{
		Address:    address,
		PublicKey:  epk.PublicKey,
		PrivateKey: hex.EncodeToString(privKeyBytes),
	}, nil
}

func faucetDecryptPrivateKeyLocal(epk *faucetEncryptedPrivateKey, password []byte) ([]byte, error) {
	salt, err := hex.DecodeString(epk.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}
	encrypted, err := hex.DecodeString(epk.Encrypted)
	if err != nil {
		return nil, fmt.Errorf("decode encrypted blob: %w", err)
	}
	kt, km, kthr := epk.KdfTime, epk.KdfMemoryKB, uint32(epk.KdfThreads)
	if kt == 0 && km == 0 && kthr == 0 {
		kt, km, kthr = faucetKdfDefaultTime, faucetKdfDefaultMemoryKB, faucetKdfDefaultThreads
	}
	key := argon2.Key(password, salt, kt, km, uint8(kthr), faucetKdfKeyLen)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := key[:12]
	plainText, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("gcm open (likely wrong password): %w", err)
	}
	return plainText, nil
}

var (
	faucetPoolKeyMu     sync.RWMutex
	faucetPoolLoadedKey *faucetPoolKey // nil until faucetPoolInit succeeds
	faucetPoolInitErr   string         // human-readable reason claims are refused, if init failed/was skipped
)

// faucetPoolInit decrypts the pool key once, at process startup, exactly
// like keystoreGetKeyLocal in scripts/local_keystore.go (that file lives
// under package main in scripts/, so its logic is intentionally
// reimplemented here rather than imported — this package cannot import
// package main). If FAUCET_POOL_ADDRESS/PASSWORD are unset, this is NOT an
// error: the faucet is simply disabled, and every claim request will
// return a clear "faucet not configured" response rather than the plugin
// process failing to start. That mirrors QUEST_ADMIN_TOKEN's fail-closed
// pattern elsewhere in this codebase.
func faucetPoolInit() {
	addr := faucetPoolAddress()
	pass := faucetPoolPassword()
	if addr == "" || pass == "" {
		faucetPoolInitErr = "faucet not configured (FAUCET_POOL_ADDRESS/FAUCET_POOL_PASSWORD unset)"
		fmt.Printf("[faucet_pool] disabled: %s\n", faucetPoolInitErr)
		return
	}

	key, err := faucetPoolLoadKeyFromLocalKeystore(addr, pass)
	if err != nil {
		faucetPoolInitErr = fmt.Sprintf("could not load pool key: %v", err)
		fmt.Printf("[faucet_pool] disabled: %s\n", faucetPoolInitErr)
		return
	}

	faucetPoolKeyMu.Lock()
	faucetPoolLoadedKey = key
	faucetPoolInitErr = ""
	faucetPoolKeyMu.Unlock()

	fmt.Printf("[faucet_pool] enabled — pool address %s, claim amount %d uARB, cooldown %d blocks\n",
		addr, faucetPoolClaimAmountUarb(), faucetPoolCooldownBlocks())

	faucetPoolLoadState()
}

func faucetPoolGetKey() (*faucetPoolKey, string) {
	faucetPoolKeyMu.RLock()
	defer faucetPoolKeyMu.RUnlock()
	if faucetPoolLoadedKey == nil {
		return nil, faucetPoolInitErr
	}
	return faucetPoolLoadedKey, ""
}

// ---- cooldown state: in-memory + file-backed, same pattern as quest_xp.go ----

var (
	faucetClaimMu       sync.RWMutex
	faucetLastClaimAddr = map[string]int64{} // address -> block height of last successful claim
)

type faucetPoolSnapshot struct {
	LastClaimAddr map[string]int64 `json:"lastClaimAddr"`
	SavedAt       int64            `json:"savedAt"`
}

func faucetPoolSaveState() {
	faucetClaimMu.RLock()
	snap := faucetPoolSnapshot{
		LastClaimAddr: make(map[string]int64, len(faucetLastClaimAddr)),
		SavedAt:       time.Now().UnixMilli(),
	}
	for k, v := range faucetLastClaimAddr {
		snap.LastClaimAddr[k] = v
	}
	faucetClaimMu.RUnlock()

	path := faucetPoolStateFilePath()
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		fmt.Printf("[faucet_pool] state save failed (marshal): %v\n", err)
		return
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".faucet_pool_state.*.tmp")
	if err != nil {
		fmt.Printf("[faucet_pool] state save failed (tempfile in %s): %v\n", dir, err)
		return
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		fmt.Printf("[faucet_pool] state save failed (write): %v\n", err)
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		fmt.Printf("[faucet_pool] state save failed (close): %v\n", err)
		return
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		fmt.Printf("[faucet_pool] state save failed (rename into %s): %v\n", path, err)
		return
	}
}

func faucetPoolLoadState() {
	path := faucetPoolStateFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("[faucet_pool] no state file at %s, starting fresh\n", path)
			return
		}
		fmt.Printf("[faucet_pool] state load failed (read %s): %v — starting with empty state\n", path, err)
		return
	}
	var snap faucetPoolSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		fmt.Printf("[faucet_pool] state load failed (parse %s): %v — file left untouched, starting with empty state\n", path, err)
		return
	}
	faucetClaimMu.Lock()
	if snap.LastClaimAddr != nil {
		faucetLastClaimAddr = snap.LastClaimAddr
	}
	faucetClaimMu.Unlock()
	fmt.Printf("[faucet_pool] state loaded from %s (saved at %d): %d addresses with claim history\n",
		path, snap.SavedAt, len(snap.LastClaimAddr))
}

// ---- BDN-over-BLS12-381 signing, reimplemented locally ----
//
// plugin/go/crypto's StringToBLS12381PrivateKey / Sign are exactly what's
// needed here, but that package's signing.go imports THIS package
// (contract) — so importing crypto from faucet_pool.go creates
// contract -> crypto -> contract, an import cycle Go refuses to build.
// Rather than restructure the crypto package or move faucet claiming into
// a separate standalone process (a materially bigger change, discussed
// and declined — see the design note at the top of this file), this
// reimplements just the two operations actually needed (decode a private
// key, sign a message) directly against the same underlying
// drand/kyber-bls12381 + drand/kyber/sign/bdn libraries crypto/bls.go
// itself calls. Same scheme (BDN, not the plain-BLS variant quest_xp.go
// uses for its own unrelated signature format), same suite construction,
// same signing call — verified line-for-line against plugin/go/crypto/bls.go
// at the time this was written. If that file's signing logic ever changes,
// this needs updating to match.
func faucetPoolBDNSign(privKeyHex string, msg []byte) ([]byte, error) {
	bz, err := hex.DecodeString(privKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode private key hex: %w", err)
	}
	suite := bls12381.NewBLS12381Suite()
	scalar := suite.G2().Scalar()
	if err := scalar.UnmarshalBinary(bz); err != nil {
		return nil, fmt.Errorf("unmarshal private key scalar: %w", err)
	}
	scheme := bdn.NewSchemeOnG2(suite)
	sig, err := scheme.Sign(scalar, msg)
	if err != nil {
		return nil, fmt.Errorf("bdn sign: %w", err)
	}
	return sig, nil
}

// ---- balance query + signed send ----

// faucetPoolQueryBalance calls the core RPC's /v1/query/account route and
// returns the pool address's current spendable amount, in base units.
func faucetPoolQueryBalance(addressHex string, height int64) (uint64, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"address": addressHex,
		"height":  height,
	})
	resp, err := http.Post(coreRPCURL()+"/v1/query/account", "application/json", strings.NewReader(string(reqBody)))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Amount uint64 `json:"amount"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, fmt.Errorf("parse account response: %w, body=%s", err, string(body))
	}
	return parsed.Amount, nil
}

// faucetPoolSendNative builds, signs, and broadcasts a MessageSend from the
// pool address to the claimant, reusing the exact tx-construction and
// signing shape proven in scripts/submit_tx.go's sendNativeSend — NOT
// buildSignAndSendTx's generic msgTypeUrl/msgBytes hex-proto envelope,
// which is for plugin-registered message types. Native send is core-level
// and independent of plugin routing (confirmed in submit_tx.go's own
// comment on sendNativeSend), and /v1/tx expects the message fields
// inlined under "msg" as a plain JSON object for this path — sending the
// generic envelope shape here was tried first and failed with a
// protobuf-decode-style error from /v1/tx (HTTP 400, "Offset": 0),
// confirming the two message-submission shapes are not interchangeable.
// Signing itself (the deterministic proto marshal of the Any-wrapped
// message for sign-bytes) is unaffected by this — only the OUTGOING HTTP
// JSON body shape differs between the two paths.
//
// KNOWN RACE, ACCEPTED: the balance check in faucetPoolHandleClaim and the
// broadcast here are two separate network round-trips, not one atomic
// operation. Two claims arriving in the same narrow window could both pass
// the balance check before either lands on-chain, and the second send
// could then fail on-chain (or succeed and overdraw the pool below zero,
// depending on how the state machine handles an insufficient-balance send
// — expected to simply reject the tx, same as any ordinary send). This
// process handles exactly one HTTP request at a time for claims in
// practice (low volume, single faucet instance), so the actual exposure is
// small; a proper fix would serialize claims through a single in-process
// queue rather than relying on request-arrival timing, which is worth
// doing if claim volume ever grows enough to make the race observable.
func faucetPoolSendNative(key *faucetPoolKey, toAddrHex string, amount, fee uint64, height int64) (string, error) {
	fromAddr, err := hex.DecodeString(key.Address)
	if err != nil {
		return "", fmt.Errorf("decode pool address: %w", err)
	}
	toAddr, err := hex.DecodeString(toAddrHex)
	if err != nil {
		return "", fmt.Errorf("decode recipient address: %w", err)
	}

	msg := &MessageSend{
		FromAddress: fromAddr,
		ToAddress:   toAddr,
		Amount:      amount,
	}
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal message: %w", err)
	}
	msgAny := &anypb.Any{TypeUrl: "type.googleapis.com/types.MessageSend", Value: msgBytes}
	txTime := uint64(time.Now().UnixMicro())

	tx := &Transaction{
		MessageType:   "send",
		Msg:           msgAny,
		Signature:     nil,
		CreatedHeight: uint64(height),
		Time:          txTime,
		Fee:           fee,
		Memo:          "arbor faucet",
		NetworkId:     faucetPoolNetworkID(),
		ChainId:       faucetPoolChainID(),
	}
	signBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(tx)
	if err != nil {
		return "", fmt.Errorf("marshal sign bytes: %w", err)
	}

	signature, err := faucetPoolBDNSign(key.PrivateKey, signBytes)
	if err != nil {
		return "", fmt.Errorf("sign transaction: %w", err)
	}

	pubKeyBytes, err := hex.DecodeString(key.PublicKey)
	if err != nil {
		return "", fmt.Errorf("decode public key: %w", err)
	}

	txJSON := map[string]interface{}{
		"type": "send",
		"msg": map[string]interface{}{
			"fromAddress": hex.EncodeToString(fromAddr),
			"toAddress":   hex.EncodeToString(toAddr),
			"amount":      amount,
		},
		"signature": map[string]string{
			"publicKey": hex.EncodeToString(pubKeyBytes),
			"signature": hex.EncodeToString(signature),
		},
		"time":          txTime,
		"createdHeight": height,
		"fee":           fee,
		"memo":          "arbor faucet",
		"networkID":     faucetPoolNetworkID(),
		"chainID":       faucetPoolChainID(),
	}
	body, err := json.Marshal(txJSON)
	if err != nil {
		return "", fmt.Errorf("marshal tx json: %w", err)
	}

	resp, err := http.Post(coreRPCURL()+"/v1/tx", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("post tx: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	var txHash string
	if err := json.Unmarshal(respBody, &txHash); err != nil {
		return "", fmt.Errorf("parse tx response: %w, body=%s", err, string(respBody))
	}
	return txHash, nil
}

// ---- HTTP handler ----

type faucetPoolClaimRequest struct {
	Address string `json:"address"`
}

type faucetPoolClaimResponse struct {
	Ok              bool   `json:"ok"`
	Error           string `json:"error,omitempty"`
	TxHash          string `json:"txHash,omitempty"`
	AmountUarb      uint64 `json:"amountUarb,omitempty"`
	BlocksRemaining uint64 `json:"blocksRemaining,omitempty"` // only set on cooldown refusal
}

func faucetPoolValidAddress(addr string) bool {
	if len(addr) != 40 {
		return false
	}
	_, err := hex.DecodeString(addr)
	return err == nil
}

// faucetPoolHandleClaim is the only mutating faucet_pool endpoint. It is
// intentionally synchronous end-to-end (query height -> query pool balance
// -> check cooldown -> sign -> broadcast -> record) so the HTTP response
// itself is the definitive answer to "did this claim succeed" — no
// polling, no webhook, no eventual consistency for the caller to handle.
func faucetPoolHandleClaim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		faucetPoolWriteJSON(w, http.StatusMethodNotAllowed, faucetPoolClaimResponse{Ok: false, Error: "POST only"})
		return
	}

	key, keyErr := faucetPoolGetKey()
	if key == nil {
		faucetPoolWriteJSON(w, http.StatusServiceUnavailable, faucetPoolClaimResponse{Ok: false, Error: keyErr})
		return
	}

	var req faucetPoolClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		faucetPoolWriteJSON(w, http.StatusBadRequest, faucetPoolClaimResponse{Ok: false, Error: "invalid JSON body"})
		return
	}
	claimAddr := strings.ToLower(strings.TrimPrefix(req.Address, "0x"))
	if !faucetPoolValidAddress(claimAddr) {
		faucetPoolWriteJSON(w, http.StatusBadRequest, faucetPoolClaimResponse{Ok: false, Error: "address is not a well-formed 20-byte hex address"})
		return
	}

	height, err := fetchCurrentHeight()
	if err != nil {
		faucetPoolWriteJSON(w, http.StatusBadGateway, faucetPoolClaimResponse{Ok: false, Error: "could not reach core RPC for height"})
		return
	}

	// Cooldown check — read-then-decide, lock held only for the map read/write,
	// never across the network calls below (same lock-scoping discipline as
	// quest_xp.go's mutation handlers).
	cooldownBlocks := faucetPoolCooldownBlocks()
	faucetClaimMu.RLock()
	lastClaim, hasClaimed := faucetLastClaimAddr[claimAddr]
	faucetClaimMu.RUnlock()
	if hasClaimed {
		var blocksSince int64
		if height > lastClaim {
			blocksSince = height - lastClaim
		}
		if blocksSince < cooldownBlocks {
			faucetPoolWriteJSON(w, http.StatusTooManyRequests, faucetPoolClaimResponse{
				Ok:              false,
				Error:           "cooldown active — one claim per address per 24h",
				BlocksRemaining: uint64(cooldownBlocks - blocksSince),
			})
			return
		}
	}

	claimAmount := faucetPoolClaimAmountUarb()
	fee := faucetPoolFeeUarb()

	// Balance check against the pool's CURRENT on-chain balance, queried
	// fresh for this claim — not a cached figure — so two concurrent
	// claims near pool exhaustion can't both pass a stale check. (This
	// does not eliminate the race entirely — see the doc comment above
	// faucetPoolSendNative — but it does mean the check reflects reality
	// at the moment it's made, rather than a number that could be
	// arbitrarily stale.)
	poolBalance, err := faucetPoolQueryBalance(key.Address, height)
	if err != nil {
		faucetPoolWriteJSON(w, http.StatusBadGateway, faucetPoolClaimResponse{Ok: false, Error: fmt.Sprintf("could not check pool balance: %v", err)})
		return
	}
	if poolBalance < claimAmount+fee {
		faucetPoolWriteJSON(w, http.StatusServiceUnavailable, faucetPoolClaimResponse{Ok: false, Error: "faucet pool is empty — ask the operator to refill it"})
		return
	}

	txHash, err := faucetPoolSendNative(key, claimAddr, claimAmount, fee, height)
	if err != nil {
		faucetPoolWriteJSON(w, http.StatusBadGateway, faucetPoolClaimResponse{Ok: false, Error: fmt.Sprintf("send failed: %v", err)})
		return
	}

	faucetClaimMu.Lock()
	faucetLastClaimAddr[claimAddr] = height
	faucetClaimMu.Unlock()
	faucetPoolSaveState()

	faucetPoolWriteJSON(w, http.StatusOK, faucetPoolClaimResponse{Ok: true, TxHash: txHash, AmountUarb: claimAmount})
}

func faucetPoolWriteJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func faucetPoolCORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // tighten to specific origins before relying on this beyond testnet
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func init() {
	faucetPoolInit()
}
