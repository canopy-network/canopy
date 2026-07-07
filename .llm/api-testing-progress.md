# Launchpad API Testing Progress

## Completed Endpoints

| # | Endpoint | Method | Status | Date |
|---|----------|--------|--------|------|
| 1 | `/api/v1/explorer/pools` | GET | ✅ Complete | 2026-01-18 |
| 2 | `/api/v1/wallet/rewards/history` | GET | ✅ Complete | 2026-01-18 |
| 3 | `/api/v1/validators` | GET | ✅ Complete | 2026-01-18 |
| 4 | `/api/v1/address-book/favorites` | GET | ✅ Complete | 2026-01-18 |
| 5 | `/api/v1/users/chain-favorites/{chain_id}` | DELETE | ✅ Complete | 2026-01-18 |
| 6 | `/api/v1/chains/repository/validate-fork` | POST | ✅ Complete | 2026-01-18 |
| 7 | `/api/v1/explorer/search` | GET | ✅ Complete | 2026-01-18 |
| 8 | `/api/v1/chains/{id}` | DELETE | ✅ Complete | 2026-01-18 |
| 9 | `/api/v1/explorer/trending` | GET | ✅ Complete | 2026-01-18 |

## Notes

### GET /api/v1/explorer/pools
- Returns paginated pool data from multiple chain schemas
- Data verified against indexer DB (chain_1, chain_6, chain_100, chain_101)
- Pagination working correctly (default limit 20, next_cursor provided)

### GET /api/v1/wallet/rewards/history
- Protected endpoint (requires Bearer token)
- Returns reward events grouped by source chain
- Requires `addresses` query param (without 0x prefix)
- Data verified against indexer DB
- Minor: chain_name/chain_symbol null for chain_6

### GET /api/v1/validators
- Returns paginated validator list with multi-chain aggregation
- **Bug fixed:** Removed status override in `validator/service.go` that incorrectly marked validators as "inactive" based on stale height_time
- Data verified against indexer DB (62 validators, status now matches)

### GET /api/v1/address-book/favorites
- Protected endpoint (requires Bearer token)
- Returns only address book entries with `is_favorite: true`
- Pagination working (page, limit, total_count)

### DELETE /api/v1/users/chain-favorites/{chain_id}
- Protected endpoint (requires Bearer token)
- Removes chain preference (like/dislike) for user
- Returns success message, preference becomes null after deletion

### POST /api/v1/chains/repository/validate-fork
- Public endpoint (no auth required)
- Validates GitHub repository is a valid Canopy fork
- Checks: repository_accessible, is_fork, not_archived, not_disabled
- Returns comprehensive validation result with individual check details

### GET /api/v1/explorer/search
- Public endpoint (no auth required)
- Supports search by: block height, address (40-char hex), transaction hash
- Query params: `q` (required), `chain_id` (default: 1), `limit` (default: 10, max: 50)
- Data verified against indexer DB (blocks, txs tables)
- Minor issue: `total_transactions` in address results shows returned count, not actual total (explorer.go:591)

### DELETE /api/v1/chains/{id}
- Protected endpoint (requires Bearer token)
- Deletes a chain owned by the authenticated user
- Chain must be in `draft` or `pending_launch` status (not deletable if active)
- Chain must not have associated transactions
- Full create→delete→verify flow tested successfully
- Not currently referenced in canopy-frontend

### GET /api/v1/explorer/trending
- Public endpoint (no auth required)
- Returns graduated chains ranked by 24h trading volume
- Query param: `limit` (default: 20, max: 100)
- Response includes: rank, chain metrics, 7-day volume history, validators, holders
- Data verified against `crosschain.validators_latest` and `crosschain.accounts_latest`
- Risk field is statically set to "Low" for all chains
