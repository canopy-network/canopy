package atrium

import (
    "encoding/json"
    "fmt"
    "github.com/canopy-network/canopy/plugin"
)

// ── STATE MODELS ──────────────────────────────────────────

type Listing struct {
    ID            string  `json:"id"`
    Creator       string  `json:"creator"`
    Name          string  `json:"name"`
    Description   string  `json:"description"`
    Category      string  `json:"category"`
    PricePerMonth float64 `json:"pricePerMonth"`
    IsVerified    bool    `json:"isVerified"`
    IsActive      bool    `json:"isActive"`
    CreatedAt     uint64  `json:"createdAt"`
}

type AccessRecord struct {
    ID         string  `json:"id"`
    ListingID  string  `json:"listingId"`
    Buyer      string  `json:"buyer"`
    ExpiresAt  uint64  `json:"expiresAt"`
    PaidAmount float64 `json:"paidAmount"`
}

type MarketplaceState struct {
    Listings     map[string]Listing      `json:"listings"`
    AccessRecords map[string]AccessRecord `json:"accessRecords"`
    TotalVolume  float64                 `json:"totalVolume"`
    Treasury     string                  `json:"treasury"`
}

// ── TRANSACTION TYPES ─────────────────────────────────────

const (
    TxListAgent        = "ListAgent"
    TxBuyAccess        = "BuyAccess"
    TxVerifyListing    = "VerifyListing"
    TxDeactivateListing = "DeactivateListing"
)

// ── PLUGIN INTERFACE IMPLEMENTATION ───────────────────────

type AtriumPlugin struct {
    state MarketplaceState
}

func New() *AtriumPlugin {
    return &AtriumPlugin{
        state: MarketplaceState{
            Listings:      make(map[string]Listing),
            AccessRecords: make(map[string]AccessRecord),
            Treasury:      "0xAtriumTreasury",
        },
    }
}

// CheckTx — validate before mempool admission
func (a *AtriumPlugin) CheckTx(tx plugin.Transaction) error {
    switch tx.Type {
    case TxListAgent:
        if tx.Params["name"] == "" || tx.Params["category"] == "" {
            return fmt.Errorf("name and category required")
        }
    case TxBuyAccess:
        listingID := tx.Params["listingId"]
        listing, ok := a.state.Listings[listingID]
        if !ok || !listing.IsActive {
            return fmt.Errorf("listing not found or inactive")
        }
    }
    return nil
}

// ApplyTransaction — mutate state (called per block)
func (a *AtriumPlugin) ApplyTransaction(tx plugin.Transaction, blockHeight uint64) error {
    switch tx.Type {

    case TxListAgent:
        id := fmt.Sprintf("%x", tx.Hash())
        a.state.Listings[id] = Listing{
            ID:            id,
            Creator:       tx.Caller,
            Name:          tx.Params["name"],
            Description:   tx.Params["description"],
            Category:      tx.Params["category"],
            PricePerMonth: parseFloat(tx.Params["pricePerMonth"]),
            IsActive:      true,
            CreatedAt:     blockHeight,
        }

    case TxBuyAccess:
        listing := a.state.Listings[tx.Params["listingId"]]
        months := parseInt(tx.Params["durationMonths"])
        cost := listing.PricePerMonth * float64(months)
        id := fmt.Sprintf("%x", tx.Hash())
        a.state.AccessRecords[id] = AccessRecord{
            ID:         id,
            ListingID:  tx.Params["listingId"],
            Buyer:      tx.Caller,
            ExpiresAt:  blockHeight + uint64(months*43200),
            PaidAmount: cost,
        }
        a.state.TotalVolume += cost

    case TxVerifyListing:
        if l, ok := a.state.Listings[tx.Params["listingId"]]; ok {
            l.IsVerified = true
            a.state.Listings[tx.Params["listingId"]] = l
        }

    case TxDeactivateListing:
        if l, ok := a.state.Listings[tx.Params["listingId"]]; ok {
            l.IsActive = false
            a.state.Listings[tx.Params["listingId"]] = l
        }
    }
    return nil
}

// Query — serves your frontend's /state/... RPC calls
func (a *AtriumPlugin) Query(path string, params map[string]string) ([]byte, error) {
    switch path {
    case "state/listings":
        listings := []Listing{}
        for _, l := range a.state.Listings {
            if cat, ok := params["category"]; ok && cat != "" {
                if l.Category != cat { continue }
            }
            listings = append(listings, l)
        }
        return json.Marshal(listings)

    case "state/access":
        records := []AccessRecord{}
        for _, r := range a.state.AccessRecords {
            if buyer, ok := params["buyer"]; ok && r.Buyer == buyer {
                records = append(records, r)
            }
        }
        return json.Marshal(records)

    case "state/marketplace":
        return json.Marshal(map[string]interface{}{
            "totalVolume": a.state.TotalVolume,
            "treasury":    a.state.Treasury,
        })
    }
    return nil, fmt.Errorf("unknown path: %s", path)
}
