package internal

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// stubNode serves the two endpoints ConfirmTx depends on. committedHash and
// failedHash are matched against the request body; anything else gets the
// "not found" shapes a real node returns.
func stubNode(t *testing.T, committedHash, failedHash string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := string(raw)
		switch {
		case strings.HasSuffix(r.URL.Path, "/tx-by-hash"):
			if committedHash != "" && strings.Contains(body, committedHash) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{"txHash":%q,"height":4242,"committed":true}`, committedHash)
				return
			}
			// A rejected tx is indistinguishable from one that never existed.
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error":"transaction not found"}`)
		case strings.HasSuffix(r.URL.Path, "/failed-txs"):
			w.WriteHeader(http.StatusOK)
			if failedHash == "" {
				fmt.Fprint(w, `{"results":null,"totalCount":0}`)
				return
			}
			fmt.Fprintf(w, `{"results":[{"txHash":%q,"error":{"code":26,"module":"state_machine","msg":"wrong chain id"}}],"totalCount":1}`, failedHash)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestConfirmTxCommitted(t *testing.T) {
	const hash = "abc123"
	srv := stubNode(t, hash, "")
	defer srv.Close()

	got, err := ConfirmTx(srv.URL, "deadbeef", hash, 3*time.Second)
	if err != nil {
		t.Fatalf("ConfirmTx: %v", err)
	}
	if !got.Committed || got.Failed {
		t.Fatalf("outcome = %+v, want committed", got)
	}
	if got.Height != 4242 {
		t.Errorf("height = %d, want 4242", got.Height)
	}
}

// TestConfirmTxFailedReportsReason is the case this whole change exists for: a
// transaction the RPC accepted and returned a hash for, which the mempool
// re-check then rejected and dropped. It never appears in tx-by-hash, so
// failed-txs is the only endpoint that can say what went wrong.
func TestConfirmTxFailedReportsReason(t *testing.T) {
	const hash = "badbeef"
	srv := stubNode(t, "", hash)
	defer srv.Close()

	got, err := ConfirmTx(srv.URL, "deadbeef", hash, 3*time.Second)
	if err != nil {
		t.Fatalf("ConfirmTx: %v", err)
	}
	if !got.Failed || got.Committed {
		t.Fatalf("outcome = %+v, want failed", got)
	}
	if got.Code != 26 || got.Msg != "wrong chain id" || got.Module != "state_machine" {
		t.Errorf("reason = (%d, %q, %q), want (26, \"wrong chain id\", \"state_machine\")",
			got.Code, got.Msg, got.Module)
	}
	if s := got.Error(); !strings.Contains(s, "wrong chain id") || !strings.Contains(s, "26") {
		t.Errorf("Error() = %q, should carry both the message and the code", s)
	}
}

// TestConfirmTxTimeoutIsNotAFailure covers the third outcome. A tx still
// pending when the deadline passes must not be reported as rejected — the
// error says the outcome is unresolved, not that it failed.
func TestConfirmTxTimeoutIsNotAFailure(t *testing.T) {
	srv := stubNode(t, "", "")
	defer srv.Close()

	got, err := ConfirmTx(srv.URL, "deadbeef", "nevermind", 1200*time.Millisecond)
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if got != nil {
		t.Fatalf("outcome should be nil on timeout, got %+v", got)
	}
	if !strings.Contains(err.Error(), "not confirmed") {
		t.Errorf("error should say the outcome is unresolved, got: %v", err)
	}
}

// TestConfirmTxResolvesFailureImmediately pins the polling order: the failed
// cache is consulted before tx-by-hash, so a rejection reports its reason
// rather than waiting out the full timeout.
func TestConfirmTxResolvesFailureImmediately(t *testing.T) {
	const hash = "cafe01"
	srv := stubNode(t, "", hash)
	defer srv.Close()

	start := time.Now()
	got, err := ConfirmTx(srv.URL, "deadbeef", hash, 10*time.Second)
	if err != nil {
		t.Fatalf("ConfirmTx: %v", err)
	}
	if !got.Failed {
		t.Fatalf("outcome = %+v, want failed", got)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("took %s; a known failure should resolve immediately", elapsed)
	}
}

// TestConfirmTxIgnoresOtherSendersFailures guards the hash match: the
// failed-tx page is scoped to an address, not a hash, so a different failing
// transaction from the same signer must not be mistaken for ours.
func TestConfirmTxIgnoresOtherSendersFailures(t *testing.T) {
	srv := stubNode(t, "", "someoneelse")
	defer srv.Close()

	got, err := ConfirmTx(srv.URL, "deadbeef", "ourhash", 1200*time.Millisecond)
	if err == nil {
		t.Fatalf("a foreign failure was matched to our hash: %+v", got)
	}
}

// TestQueryFailedWalksEveryPage covers a signer whose failure sits beyond the
// first page of the failed-tx cache. queryFailed used to request one page and
// stop, so ConfirmTx reported a timeout ("may still be pending") for a
// transaction the node had definitively rejected.
func TestQueryFailedWalksEveryPage(t *testing.T) {
	const wanted = "deadbeef"
	const totalPages = 3
	var pagesServed []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := string(raw)
		if !strings.HasSuffix(r.URL.Path, "/failed-txs") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		pagesServed = append(pagesServed, body)
		w.WriteHeader(http.StatusOK)
		// The hash lives on the last page only.
		if strings.Contains(body, `"pageNumber":3`) {
			fmt.Fprintf(w, `{"totalPages":%d,"totalCount":201,"results":[{"txHash":%q,"error":{"code":26,"module":"state_machine","msg":"wrong chain id"}}]}`,
				totalPages, wanted)
			return
		}
		fmt.Fprintf(w, `{"totalPages":%d,"totalCount":201,"results":[{"txHash":"0000","error":{"code":1,"module":"x","msg":"other"}}]}`, totalPages)
	}))
	defer srv.Close()

	got := queryFailed(srv.URL, "signeraddr", wanted)
	if got == nil {
		t.Fatal("failure on page 3 was not found; a rejected tx would be reported as a timeout")
	}
	if !got.Failed || got.Msg != "wrong chain id" {
		t.Errorf("wrong outcome: %+v", got)
	}
	if len(pagesServed) != 3 {
		t.Errorf("served %d pages, want 3", len(pagesServed))
	}
}

// TestQueryFailedStopsAtTheLastPage guards the other direction: the walk must
// terminate rather than requesting pages forever when the hash is absent.
func TestQueryFailedStopsAtTheLastPage(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body)
		requests++
		if requests > 10 {
			t.Error("queryFailed did not stop paging")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"totalPages":2,"totalCount":2,"results":[{"txHash":"0000","error":{"code":1,"module":"x","msg":"other"}}]}`)
	}))
	defer srv.Close()

	if got := queryFailed(srv.URL, "signeraddr", "notpresent"); got != nil {
		t.Errorf("found a hash that is not in the cache: %+v", got)
	}
	if requests != 2 {
		t.Errorf("made %d requests, want 2 (stop at totalPages)", requests)
	}
}
