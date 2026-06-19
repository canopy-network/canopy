package canoliq

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/canopy-network/go-plugin/contract"
)

// alerts.go is the T6 push-alert subsystem (WP §11): a non-blocking webhook
// dispatcher plus the on-chain condition checks that feed it.
//
// Design rationale:
//   - Pull vs push: the RPC surface (Phase 3 §1) already exposes everything
//     for polling; alerts add an *unattended* push path for the few events an
//     operator must react to (pool draining, stake concentration, TVL crash).
//   - Dedup in state: a watermark per alert kind (KeyForAlertState) survives
//     restarts, so the same condition does not re-page every block. The
//     watermark clears on resolution so the next occurrence fires immediately.
//   - Goroutine dispatcher: webhook POSTs run off-thread with a short timeout
//     so a slow or dead receiver can never stall EndBlock / consensus.

// Alert kinds. Used as the AlertEnvelope.Kind and the KeyForAlertState suffix.
const (
	AlertBuybackDrain           = "buyback_drain"
	AlertValidatorConcentration = "validator_concentration"
	AlertTVLDrop                = "tvl_drop"
	AlertStuckRedemption        = "stuck_redemption"
)

// Alert severities.
const (
	severityWarn = "warn"
	severityCrit = "crit"
)

// alertSchemaVersion versions the details payload for downstream consumers.
const alertSchemaVersion = 1

// Defaults for the AlertConfig knobs (overridable via config JSON).
const (
	alertDefaultMinInterval         = 100   // blocks between re-fires of the same kind
	alertDefaultWindowBlocks        = 100   // tumbling window for drain/drop checks
	alertDefaultDrainBps            = 5_000 // 50%
	alertDefaultConcentBps          = 6_600 // 66%
	alertDefaultTVLDropBps          = 2_000 // 20%
	alertDefaultStuckRedemptionCnt  = 10    // mature unclaimed redemptions
)

// AlertEnvelope is the canonical alert payload. Slack / Discord adapters
// project this into their own shapes; the json format sends it verbatim.
type AlertEnvelope struct {
	Kind     string         `json:"kind"`
	Height   uint64         `json:"height"`
	Severity string         `json:"severity"`
	Message  string         `json:"message"`
	Details  map[string]any `json:"details"`
}

// alertsEnabled reports whether a usable alert webhook is configured.
func (p *Plugin) alertsEnabled() bool {
	return p.alertHook != nil || (p.config.Alerts != nil && p.config.Alerts.WebhookURL != "")
}

// fireAlert delivers an alert. With the test hook set it is synchronous;
// otherwise it spawns a goroutine so EndBlock never blocks on network IO.
func (p *Plugin) fireAlert(env AlertEnvelope) {
	if env.Details == nil {
		env.Details = map[string]any{}
	}
	env.Details["schemaVersion"] = alertSchemaVersion
	if p.alertHook != nil {
		p.alertHook(env)
		return
	}
	cfg := p.config.Alerts
	if cfg == nil || cfg.WebhookURL == "" {
		return
	}
	go postAlert(cfg, env)
}

// postAlert POSTs env to the configured webhook with a 5s timeout. Failures
// are logged at WARN and otherwise swallowed — alert delivery is best-effort.
func postAlert(cfg *AlertConfig, env AlertEnvelope) {
	body, err := encodeAlert(cfg.Format, env)
	if err != nil {
		log.Printf("canoliq: WARN alert encode failed: %v", err)
		return
	}
	req, err := http.NewRequest(http.MethodPost, cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("canoliq: WARN alert request build failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.AuthHeader != "" {
		req.Header.Set("Authorization", cfg.AuthHeader)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("canoliq: WARN alert POST %s failed: %v", env.Kind, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("canoliq: WARN alert POST %s returned %d", env.Kind, resp.StatusCode)
	}
}

// encodeAlert renders the envelope in the configured format. Unknown formats
// fall back to json.
func encodeAlert(format string, env AlertEnvelope) ([]byte, error) {
	switch format {
	case "slack":
		return json.Marshal(map[string]any{"text": alertText(env)})
	case "discord":
		return json.Marshal(map[string]any{"content": alertText(env)})
	default: // "json" or unset
		return json.Marshal(env)
	}
}

// alertText is the human-readable one-liner used by the Slack / Discord
// adapters.
func alertText(env AlertEnvelope) string {
	return "[" + env.Severity + "] canoLiq " + env.Kind + " @ h" +
		uintToString(env.Height) + ": " + env.Message
}

// uintToString avoids pulling strconv into a hot path for a single use.
func uintToString(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// --- effective-config getters (nil-safe, default-applying) ---

func (cfg *AlertConfig) windowBlocks() uint64 {
	if cfg != nil && cfg.WindowBlocks > 0 {
		return cfg.WindowBlocks
	}
	return alertDefaultWindowBlocks
}
func (cfg *AlertConfig) drainBps() uint64 {
	if cfg != nil && cfg.DrainAlertBps > 0 {
		return cfg.DrainAlertBps
	}
	return alertDefaultDrainBps
}
func (cfg *AlertConfig) concentBps() uint64 {
	if cfg != nil && cfg.ConcentrationAlertBps > 0 {
		return cfg.ConcentrationAlertBps
	}
	return alertDefaultConcentBps
}
func (cfg *AlertConfig) tvlDropBps() uint64 {
	if cfg != nil && cfg.TvlDropBps > 0 {
		return cfg.TvlDropBps
	}
	return alertDefaultTVLDropBps
}
func (cfg *AlertConfig) stuckRedemptionCount() uint64 {
	if cfg != nil && cfg.StuckRedemptionCount > 0 {
		return cfg.StuckRedemptionCount
	}
	return alertDefaultStuckRedemptionCnt
}
func (cfg *AlertConfig) minInterval(kind string) uint64 {
	if cfg != nil {
		if v, ok := cfg.MinIntervalBlocks[kind]; ok {
			return v
		}
		if cfg.DefaultMinIntervalBlocks > 0 {
			return cfg.DefaultMinIntervalBlocks
		}
	}
	return alertDefaultMinInterval
}

// --- condition evaluation (called from EndBlock after refreshSnapshot) ---

// evaluateAlerts runs every alert condition against the freshly-published
// snapshot and dispatches any that fire. A no-op when alerts are disabled.
func (c *Canoliq) evaluateAlerts(height uint64) *contract.PluginError {
	if !c.plugin.alertsEnabled() {
		return nil
	}
	cfg := c.plugin.config.Alerts // may be nil under the test hook; getters default
	s := c.plugin.Snapshot()
	if err := c.checkBuybackDrain(s, height, cfg); err != nil {
		return err
	}
	if err := c.checkValidatorConcentration(s, height, cfg); err != nil {
		return err
	}
	if err := c.checkTVLDrop(s, height, cfg); err != nil {
		return err
	}
	if err := c.checkStuckRedemption(height, cfg); err != nil {
		return err
	}
	return nil
}

// checkStuckRedemption fires when the count of mature unclaimed redemptions
// exceeds stuckRedemptionCount. Uses the global mature-redemption index
// maintained by deliver.go (write on queue, delete on claim). Instantaneous
// — no tumbling window — and `crit` severity.
//
// The threshold detection only needs `count > N`; the range scan is capped
// at `threshold + 1` so we never haul a large index back over the wire.
func (c *Canoliq) checkStuckRedemption(height uint64, cfg *AlertConfig) *contract.PluginError {
	st, err := c.loadAlertState(AlertStuckRedemption)
	if err != nil {
		return err
	}
	threshold := cfg.stuckRedemptionCount()
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Ranges: []*contract.PluginRangeRead{{
			QueryId: q,
			Prefix:  MatureRedemptionPrefix(),
			// +1 so we observe the boundary cross. The index sorts
			// ascending by mature height, so the first `threshold+1`
			// entries are the oldest — exactly what we want to count.
			Limit: threshold + 1,
		}},
	})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	count := uint64(0)
	for _, r := range resp.Results {
		for _, e := range r.Entries {
			mh, ok := ParseMatureRedemptionHeight(e.Key)
			if !ok {
				continue
			}
			if mh <= height {
				count++
			}
		}
	}
	fired := count > threshold
	return c.applyAlert(AlertStuckRedemption, fired, height, severityCrit,
		"mature unclaimed redemptions above threshold", map[string]any{
			"count": count, "thresholdCount": threshold,
		}, st)
}

// checkBuybackDrain fires when the buyback pool drains more than drainBps over
// a tumbling window.
func (c *Canoliq) checkBuybackDrain(s *Snapshot, height uint64, cfg *AlertConfig) *contract.PluginError {
	st, err := c.loadAlertState(AlertBuybackDrain)
	if err != nil {
		return err
	}
	current := s.BuybackPool
	rolled := c.rollWindow(st, height, current, cfg.windowBlocks())
	fired, dropBps := false, uint64(0)
	if !rolled && st.WindowBaseline > 0 && current < st.WindowBaseline {
		dropBps = mulDiv(st.WindowBaseline-current, 10_000, st.WindowBaseline)
		fired = dropBps > cfg.drainBps()
	}
	return c.applyAlert(AlertBuybackDrain, fired, height, severityWarn,
		"buyback pool draining faster than threshold", map[string]any{
			"baseline": st.WindowBaseline, "current": current, "dropBps": dropBps, "thresholdBps": cfg.drainBps(),
		}, st)
}

// checkTVLDrop fires when total pooled CNPY drops more than tvlDropBps over a
// tumbling window.
func (c *Canoliq) checkTVLDrop(s *Snapshot, height uint64, cfg *AlertConfig) *contract.PluginError {
	st, err := c.loadAlertState(AlertTVLDrop)
	if err != nil {
		return err
	}
	current := s.Globals.TotalPooledCnpy
	rolled := c.rollWindow(st, height, current, cfg.windowBlocks())
	fired, dropBps := false, uint64(0)
	if !rolled && st.WindowBaseline > 0 && current < st.WindowBaseline {
		dropBps = mulDiv(st.WindowBaseline-current, 10_000, st.WindowBaseline)
		fired = dropBps > cfg.tvlDropBps()
	}
	return c.applyAlert(AlertTVLDrop, fired, height, severityCrit,
		"TVL dropped faster than threshold", map[string]any{
			"baseline": st.WindowBaseline, "current": current, "dropBps": dropBps, "thresholdBps": cfg.tvlDropBps(),
		}, st)
}

// checkValidatorConcentration fires when one committee validator holds more
// than concentBps of total committee stake. Instantaneous (no window).
func (c *Canoliq) checkValidatorConcentration(s *Snapshot, height uint64, cfg *AlertConfig) *contract.PluginError {
	st, err := c.loadAlertState(AlertValidatorConcentration)
	if err != nil {
		return err
	}
	var total, max uint64
	if s.ValidatorRegistry != nil {
		for _, e := range s.ValidatorRegistry.Entries {
			total += e.Stake
			if e.Stake > max {
				max = e.Stake
			}
		}
	}
	fired, concentBps := false, uint64(0)
	if total > 0 {
		concentBps = mulDiv(max, 10_000, total)
		fired = concentBps > cfg.concentBps()
	}
	return c.applyAlert(AlertValidatorConcentration, fired, height, severityWarn,
		"validator stake concentration above threshold", map[string]any{
			"maxStake": max, "totalStake": total, "concentrationBps": concentBps, "thresholdBps": cfg.concentBps(),
		}, st)
}

// rollWindow resets the tumbling-window baseline when the window opens or has
// elapsed. Returns true when it just (re)anchored, so the caller skips a
// same-block comparison against a stale baseline.
func (c *Canoliq) rollWindow(st *contract.AlertState, height, current, windowBlocks uint64) bool {
	if st.WindowStartHeight == 0 || height-st.WindowStartHeight >= windowBlocks {
		st.WindowStartHeight = height
		st.WindowBaseline = current
		return true
	}
	return false
}

// applyAlert handles debounce + dispatch + watermark maintenance, then
// persists the alert state. On resolution the watermark clears so the next
// occurrence fires immediately.
func (c *Canoliq) applyAlert(kind string, fired bool, height uint64, severity, message string, details map[string]any, st *contract.AlertState) *contract.PluginError {
	cfg := c.plugin.config.Alerts
	if fired {
		if st.LastFiredHeight == 0 || height-st.LastFiredHeight >= cfg.minInterval(kind) {
			c.plugin.fireAlert(AlertEnvelope{Kind: kind, Height: height, Severity: severity, Message: message, Details: details})
			st.LastFiredHeight = height
		}
	} else {
		st.LastFiredHeight = 0
	}
	return c.saveAlertState(kind, st)
}

// loadAlertState reads a per-kind AlertState, returning a zero value when absent.
func (c *Canoliq) loadAlertState(kind string) (*contract.AlertState, *contract.PluginError) {
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForAlertState(kind)}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	st := new(contract.AlertState)
	if len(resp.Results) > 0 && len(resp.Results[0].Entries) > 0 {
		if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, st); e != nil {
			return nil, e
		}
	}
	return st, nil
}

func (c *Canoliq) saveAlertState(kind string, st *contract.AlertState) *contract.PluginError {
	bz, e := contract.Marshal(st)
	if e != nil {
		return e
	}
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
		Sets: []*contract.PluginSetOp{{Key: KeyForAlertState(kind), Value: bz}},
	}); err != nil {
		return err
	}
	return nil
}
