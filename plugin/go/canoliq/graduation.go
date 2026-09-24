package canoliq

import "github.com/canopy-network/go-plugin/contract"

// graduation.go implements autonomy-graduation tracking (T5 / WP §10): the
// per-block counters that feed the five graduation metrics, the rolling
// daily-transaction window, and the read-only /v1/graduation surface.
//
// canoLiq begins as a Canopy Nested Chain and may graduate to a sovereign
// chain once it durably clears all five thresholds. T5 only measures and
// surfaces eligibility; the ACTION_AUTONOMY_GRADUATE proposal tier (T1) and
// the actual graduation dispatch (M3) are separate.

// runwayInfinite is the sentinel runway (months) reported when the treasury
// has a positive balance but no measured burn — i.e. effectively unlimited.
const runwayInfinite = 1_000_000

// countGraduationTx advances the in-progress daily-transaction window counter.
// Called from DeliverTx after a successful delivery, so a fresh globals load
// preserves any state the handler already wrote this block.
func (c *Canoliq) countGraduationTx() *contract.PluginError {
	g, err := c.LoadGlobals()
	if err != nil {
		return err
	}
	if !g.GenesisComplete {
		return nil
	}
	g.DailyTxCountWindow++
	return c.SaveGlobals(g)
}

// advanceGraduationWindow rolls the daily-transaction window. On the first
// post-genesis block it anchors the window; once a window spans >= blocksPerDay
// it records the completed count as last_daily_tx_count and resets. Called
// from BeginBlock.
func (c *Canoliq) advanceGraduationWindow(height uint64) *contract.PluginError {
	g, err := c.LoadGlobals()
	if err != nil {
		return err
	}
	if !g.GenesisComplete {
		return nil
	}
	if g.LastWindowCloseHeight == 0 {
		g.LastWindowCloseHeight = height
		return c.SaveGlobals(g)
	}
	if height-g.LastWindowCloseHeight >= blocksPerDay {
		g.LastDailyTxCount = g.DailyTxCountWindow
		g.DailyTxCountWindow = 0
		g.LastWindowCloseHeight = height
		return c.SaveGlobals(g)
	}
	return nil
}

// GraduationMetric is one autonomy threshold and its current measurement.
type GraduationMetric struct {
	Name      string `json:"name"`
	Value     uint64 `json:"value"`
	Threshold uint64 `json:"threshold"`
	Met       bool   `json:"met"`
}

// GraduationView is the /v1/graduation payload: the five metrics plus a
// composite eligibility flag (true only when every metric is met).
type GraduationView struct {
	Metrics  []GraduationMetric `json:"metrics"`
	Eligible bool               `json:"eligible"`
}

// QueryGraduation computes the five WP §10 graduation metrics from the
// snapshot. Each metric uses a strict ">" against its threshold.
func (p *Plugin) QueryGraduation() *GraduationView {
	s := p.Snapshot()
	g, pa := s.Globals, s.Params

	validators := uint64(0)
	if s.ValidatorRegistry != nil {
		validators = uint64(len(s.ValidatorRegistry.Entries))
	}
	turnout := uint64(0)
	if g.TurnoutSampleCount > 0 {
		turnout = g.TurnoutSumBps / g.TurnoutSampleCount
	}
	runway := runwayMonthsFor(s)

	metrics := []GraduationMetric{
		{"tvl_ucnpy", g.TotalPooledCnpy, pa.GraduationMinTvlUcnpy, g.TotalPooledCnpy > pa.GraduationMinTvlUcnpy},
		{"active_validators", validators, pa.GraduationMinValidators, validators > pa.GraduationMinValidators},
		{"turnout_bps", turnout, pa.GraduationMinTurnoutBps, turnout > pa.GraduationMinTurnoutBps},
		{"daily_tx", g.LastDailyTxCount, pa.GraduationMinDailyTx, g.LastDailyTxCount > pa.GraduationMinDailyTx},
		{"runway_months", runway, pa.GraduationMinRunwayMonths, runway > pa.GraduationMinRunwayMonths},
	}
	eligible := true
	for _, m := range metrics {
		if !m.Met {
			eligible = false
			break
		}
	}
	return &GraduationView{Metrics: metrics, Eligible: eligible}
}

// runwayMonthsFor estimates treasury runway: balance / monthly burn, where
// monthly burn is cumulative treasury spend amortized over elapsed months.
// Zero treasury → 0 (no runway); positive treasury with no burn → infinite.
func runwayMonthsFor(s *Snapshot) uint64 {
	if s.TreasuryCNPY == 0 {
		return 0
	}
	elapsedMonths := s.Height / blocksPerMonth
	if elapsedMonths == 0 {
		elapsedMonths = 1
	}
	monthlyBurn := s.Globals.TreasurySpentTotal / elapsedMonths
	if monthlyBurn == 0 {
		return runwayInfinite
	}
	return s.TreasuryCNPY / monthlyBurn
}
