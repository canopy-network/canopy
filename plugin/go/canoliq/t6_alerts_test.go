package canoliq

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/canopy-network/go-plugin/contract"
)

// t6_alerts_test.go covers the push-alert subsystem (T6): condition firing,
// debounce, resolution, threshold edges (via the synchronous test hook), and
// webhook delivery + format adapters + 500-resilience (via httptest.Server).

// newAlertTest wires a synchronous capture hook and a default alert config.
func newAlertTest(t *testing.T) (*Canoliq, *fakeStore, *[]AlertEnvelope) {
	t.Helper()
	c, s := newTestCanoliq()
	var got []AlertEnvelope
	c.plugin.alertHook = func(e AlertEnvelope) { got = append(got, e) }
	c.plugin.config.Alerts = &AlertConfig{} // empty URL ok: the hook intercepts
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})
	return c, s, &got
}

// evalAt seeds the buyback pool / pooled TVL, refreshes the snapshot, and runs
// the alert evaluation at the given height.
func evalAt(t *testing.T, c *Canoliq, s *fakeStore, height, buyback, pooled uint64) {
	t.Helper()
	s.set(KeyForBuybackPool(), EncodeUint64(buyback))
	g := loadGlobals(t, s)
	g.TotalPooledCnpy = pooled
	seedGlobals(s, g)
	if err := c.refreshSnapshot(height); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if err := c.evaluateAlerts(height); err != nil {
		t.Fatalf("evaluateAlerts: %v", err)
	}
}

func countKind(got []AlertEnvelope, kind string) int {
	n := 0
	for _, e := range got {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

// TestT6BuybackDrainFires anchors a window then drains the pool 60% and expects
// one buyback-drain alert with the schema-versioned details.
func TestT6BuybackDrainFires(t *testing.T) {
	c, s, got := newAlertTest(t)
	evalAt(t, c, s, 1, 1000, 0) // anchor baseline 1000
	if countKind(*got, AlertBuybackDrain) != 0 {
		t.Fatal("window-anchor block should not fire")
	}
	evalAt(t, c, s, 2, 400, 0) // 60% drain > 50%
	if countKind(*got, AlertBuybackDrain) != 1 {
		t.Fatalf("expected one buyback-drain alert, got %d", countKind(*got, AlertBuybackDrain))
	}
	e := (*got)[len(*got)-1]
	if e.Severity != severityWarn || e.Height != 2 {
		t.Errorf("envelope: severity=%s height=%d", e.Severity, e.Height)
	}
	if e.Details["schemaVersion"] != alertSchemaVersion {
		t.Errorf("missing schemaVersion: %v", e.Details["schemaVersion"])
	}
}

// TestT6ConcentrationFires: one validator over 66% of committee stake fires.
func TestT6ConcentrationFires(t *testing.T) {
	c, s, got := newAlertTest(t)
	s.set(KeyForValidatorRegistry(), mustMarshal(&contract.ValidatorRegistry{Entries: []*contract.ValidatorRegistryEntry{
		{Address: addr20(0x01), Stake: 70}, {Address: addr20(0x02), Stake: 30}, // 70%
	}}))
	evalAt(t, c, s, 1, 0, 0)
	if countKind(*got, AlertValidatorConcentration) != 1 {
		t.Fatalf("expected concentration alert, got %d", countKind(*got, AlertValidatorConcentration))
	}
	// Balanced set (50/50) does not fire.
	c2, s2, got2 := newAlertTest(t)
	s2.set(KeyForValidatorRegistry(), mustMarshal(&contract.ValidatorRegistry{Entries: []*contract.ValidatorRegistryEntry{
		{Address: addr20(0x01), Stake: 50}, {Address: addr20(0x02), Stake: 50},
	}}))
	evalAt(t, c2, s2, 1, 0, 0)
	if countKind(*got2, AlertValidatorConcentration) != 0 {
		t.Error("balanced validators should not fire concentration")
	}
}

// TestT6TVLDropFires: TVL falling 30% over the window fires a crit alert.
func TestT6TVLDropFires(t *testing.T) {
	c, s, got := newAlertTest(t)
	evalAt(t, c, s, 1, 0, 1000) // anchor pooled baseline 1000
	evalAt(t, c, s, 2, 0, 700)  // 30% drop > 20%
	if countKind(*got, AlertTVLDrop) != 1 {
		t.Fatalf("expected tvl-drop alert, got %d", countKind(*got, AlertTVLDrop))
	}
	if (*got)[len(*got)-1].Severity != severityCrit {
		t.Error("tvl-drop should be crit severity")
	}
}

// TestT6DebounceAndResolution: a persistent condition fires once within the
// debounce window; once resolved the watermark clears and it can fire again.
func TestT6DebounceAndResolution(t *testing.T) {
	c, s, got := newAlertTest(t)
	evalAt(t, c, s, 1, 1000, 0) // anchor
	evalAt(t, c, s, 2, 300, 0)  // drain → fire
	evalAt(t, c, s, 3, 200, 0)  // still draining, within debounce → no re-fire
	if n := countKind(*got, AlertBuybackDrain); n != 1 {
		t.Fatalf("debounce: expected 1 fire, got %d", n)
	}
	// Resolve: pool back at/above baseline → clears watermark.
	evalAt(t, c, s, 4, 1000, 0)
	// Drain again → fires (watermark was cleared).
	evalAt(t, c, s, 5, 300, 0)
	if n := countKind(*got, AlertBuybackDrain); n != 2 {
		t.Fatalf("post-resolution: expected 2 total fires, got %d", n)
	}
}

// TestT6ThresholdEdge: exactly at threshold does not fire; one bps above does.
func TestT6ThresholdEdge(t *testing.T) {
	// drain default 5000 bps. baseline 1000.
	c, s, got := newAlertTest(t)
	evalAt(t, c, s, 1, 1000, 0)
	evalAt(t, c, s, 2, 500, 0) // exactly 5000 bps → not > 5000
	if countKind(*got, AlertBuybackDrain) != 0 {
		t.Errorf("at-threshold drain should not fire")
	}
	c2, s2, got2 := newAlertTest(t)
	evalAt(t, c2, s2, 1, 1000, 0)
	evalAt(t, c2, s2, 2, 499, 0) // 5010 bps → fires
	if countKind(*got2, AlertBuybackDrain) != 1 {
		t.Errorf("just-above-threshold drain should fire")
	}
}

// TestT6PostAlertDelivery exercises the real HTTP path: json/slack/discord
// formats reach a mock receiver, and a 500 response is handled gracefully.
func TestT6PostAlertDelivery(t *testing.T) {
	received := make(chan []byte, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		received <- b
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	env := AlertEnvelope{Kind: "buyback_drain", Height: 7, Severity: "warn", Message: "draining", Details: map[string]any{"dropBps": 6000}}

	postAlert(&AlertConfig{WebhookURL: srv.URL, Format: "json"}, env)
	if b := waitBody(t, received); !strings.Contains(string(b), `"kind":"buyback_drain"`) {
		t.Errorf("json payload missing kind: %s", b)
	}
	postAlert(&AlertConfig{WebhookURL: srv.URL, Format: "slack"}, env)
	if b := waitBody(t, received); !strings.Contains(string(b), `"text"`) {
		t.Errorf("slack payload missing text: %s", b)
	}
	postAlert(&AlertConfig{WebhookURL: srv.URL, Format: "discord"}, env)
	if b := waitBody(t, received); !strings.Contains(string(b), `"content"`) {
		t.Errorf("discord payload missing content: %s", b)
	}

	// 500 from the receiver must not panic or block.
	srv500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv500.Close()
	postAlert(&AlertConfig{WebhookURL: srv500.URL, Format: "json"}, env) // returns cleanly
}

// TestT6AuthHeaderSent confirms the configured Authorization header is set.
func TestT6AuthHeaderSent(t *testing.T) {
	gotAuth := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth <- r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	postAlert(&AlertConfig{WebhookURL: srv.URL, AuthHeader: "Bearer secret"}, AlertEnvelope{Kind: "x"})
	select {
	case a := <-gotAuth:
		if a != "Bearer secret" {
			t.Errorf("auth header: got %q", a)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no request received")
	}
}

// TestT6DisabledIsNoop: with neither hook nor webhook, evaluation is a no-op.
func TestT6DisabledIsNoop(t *testing.T) {
	c, s := newTestCanoliq()
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})
	if err := c.refreshSnapshot(1); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if err := c.evaluateAlerts(1); err != nil {
		t.Fatalf("disabled evaluateAlerts should be a no-op: %v", err)
	}
}

func waitBody(t *testing.T, ch chan []byte) []byte {
	t.Helper()
	select {
	case b := <-ch:
		return b
	case <-time.After(2 * time.Second):
		t.Fatal("no webhook request received")
		return nil
	}
}
