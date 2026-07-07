# Oracle Metrics Reference

This document describes all Prometheus metrics exposed by the Canopy Oracle system for monitoring and observability.

## Overview

The oracle exposes metrics through a Prometheus-compatible endpoint. Metrics are organized into two main categories:

1. **Oracle Metrics** (`canopy_oracle_*`) - High-level oracle operations, order lifecycle, and validation
2. **Eth Block Provider Metrics** (`canopy_eth_*`) - Ethereum connectivity, block processing, and transaction handling

---

## Oracle Metrics

These metrics track the oracle's core functionality including order processing, validation, and submission.

### Block Height Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `canopy_oracle_safe_height` | Gauge | Current safe block height in the oracle |
| `canopy_oracle_source_chain_height` | Gauge | Current source chain height |
| `canopy_oracle_last_processed_height` | Gauge | Last source chain block height processed |
| `canopy_oracle_confirmation_lag` | Gauge | Gap between source chain height and safe height (blocks awaiting confirmation) |
| `canopy_oracle_orders_awaiting_confirmation` | Gauge | Number of orders witnessed but not yet at safe height |

### Order Lifecycle Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `canopy_oracle_orders_witnessed_total` | Counter | - | Total orders witnessed |
| `canopy_oracle_orders_validated_total` | Counter | - | Total orders validated successfully |
| `canopy_oracle_orders_submitted_total` | Counter | - | Total orders submitted for consensus |
| `canopy_oracle_orders_rejected_total` | Counter | - | Total orders rejected during validation |
| `canopy_oracle_orders_not_in_orderbook_total` | Counter | - | Orders witnessed but not found in order book |
| `canopy_oracle_orders_duplicate_total` | Counter | - | Duplicate orders (already in store) |
| `canopy_oracle_orders_archived_total` | Counter | - | Orders successfully archived |
| `canopy_oracle_lock_orders_committed_total` | Counter | - | Lock orders committed via certificate |
| `canopy_oracle_close_orders_committed_total` | Counter | - | Close orders committed via certificate |

### Order Store Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `canopy_oracle_total_orders_stored` | Gauge | Total orders currently stored in order store |
| `canopy_oracle_lock_orders_stored` | Gauge | Total lock orders currently stored |
| `canopy_oracle_close_orders_stored` | Gauge | Total close orders currently stored |

### Validation Failure Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `canopy_oracle_validation_failures_total` | Counter | `reason` | Validation failures by reason |

**Reason Labels:**
- `order_nil` - Order was nil
- `missing_order_type` - Order missing type specification
- `both_order_types` - Order has both lock and close types
- `lock_id_mismatch` - Lock order ID mismatch
- `lock_chain_mismatch` - Lock order chain ID mismatch
- `close_data_mismatch` - Close order data mismatch
- `close_id_mismatch` - Close order ID mismatch
- `close_chain_mismatch` - Close order chain ID mismatch
- `recipient_mismatch` - Recipient address mismatch
- `amount_nil` - Amount is nil
- `amount_mismatch` - Amount mismatch

### Submission Tracking Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `canopy_oracle_orders_held_awaiting_safe_total` | Counter | Orders not submitted due to safe height requirement |
| `canopy_oracle_orders_held_propose_delay_total` | Counter | Orders held by ProposeDelayBlocks |
| `canopy_oracle_orders_held_resubmit_delay_total` | Counter | Orders held by resubmit cooldown |
| `canopy_oracle_lock_order_resubmissions_total` | Counter | Lock orders resubmitted |
| `canopy_oracle_close_order_resubmissions_total` | Counter | Close orders resubmitted |

### Error and Reorg Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `canopy_oracle_chain_reorgs_total` | Counter | Total chain reorganizations detected |
| `canopy_oracle_orders_pruned_total` | Counter | Total orders pruned during cleanup |
| `canopy_oracle_block_processing_errors_total` | Counter | Total block processing errors |
| `canopy_oracle_reorg_rollback_depth` | Histogram | How many blocks reorgs roll back |

### Store Operation Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `canopy_oracle_store_write_errors_total` | Counter | Order store write failures |
| `canopy_oracle_store_read_errors_total` | Counter | Order store read failures |
| `canopy_oracle_store_remove_errors_total` | Counter | Order store remove failures |

### Performance Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `canopy_oracle_order_book_update_time` | Histogram | Time to update order book |
| `canopy_oracle_root_chain_sync_time` | Histogram | Time to sync with root chain |

---

## Eth Block Provider Metrics

These metrics track Ethereum connectivity, block fetching, and transaction processing.

### Connection & Sync Status Metrics (High Priority)

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `canopy_eth_rpc_connection_attempts_total` | Counter | - | Total RPC connection attempts |
| `canopy_eth_rpc_connection_errors_total` | Counter | `error_type` | RPC connection errors by error type |
| `canopy_eth_ws_connection_attempts_total` | Counter | - | Total WebSocket connection attempts |
| `canopy_eth_ws_subscription_errors_total` | Counter | - | WebSocket subscription failures |
| `canopy_eth_connection_state` | Gauge | - | Current connection state |
| `canopy_eth_sync_status` | Gauge | - | Current sync status |

**Connection State Values:**
- `0` = Disconnected
- `1` = Connecting
- `2` = RPC Connected
- `3` = Fully Connected (RPC + WebSocket)

**Sync Status Values:**
- `0` = Unsynced
- `1` = Syncing
- `2` = Synced

### Block Height Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `canopy_eth_chain_head_height` | Gauge | Latest block height from chain head |
| `canopy_eth_last_processed_height` | Gauge | Last block height successfully processed |
| `canopy_eth_safe_height` | Gauge | Current safe (confirmed) block height |
| `canopy_eth_block_height_lag` | Gauge | Number of blocks behind chain head |

### Block Processing Metrics (High Priority)

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `canopy_eth_block_fetch_errors_total` | Counter | `error_type` | Block fetch errors by error type |
| `canopy_eth_block_processing_timeouts_total` | Counter | - | Blocks that timed out during processing |
| `canopy_eth_process_blocks_batch_size` | Histogram | - | Number of blocks processed per batch |
| `canopy_eth_reorg_detected_total` | Counter | - | Chain reorganizations detected |
| `canopy_eth_blocks_processed_total` | Counter | - | Total Ethereum blocks processed |

**Histogram Buckets for Batch Size:** 1, 5, 10, 25, 50, 100, 250, 500, 1000

### Transaction Processing Metrics (Medium Priority)

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `canopy_eth_transactions_total` | Counter | - | Total transactions encountered in blocks |
| `canopy_eth_transactions_processed_total` | Counter | - | Total Ethereum transactions processed |
| `canopy_eth_transaction_parse_errors_total` | Counter | `error_type` | Transaction parsing errors by error type |
| `canopy_eth_transaction_retry_by_attempt_total` | Counter | `attempt` | Transaction retry attempts by attempt number |
| `canopy_eth_transaction_exhausted_retries_total` | Counter | - | Transactions that exhausted all retry attempts |
| `canopy_eth_transaction_success_status_total` | Counter | `status` | Transaction success/failed/unknown breakdown |
| `canopy_eth_receipt_fetch_errors_total` | Counter | - | Receipt fetch failures |
| `canopy_eth_transaction_retries_total` | Counter | - | Total transaction processing retries |

**Status Labels:**
- `success` - Transaction succeeded
- `failed` - Transaction failed
- `unknown` - Unable to determine status

### Order Detection Metrics (Medium Priority)

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `canopy_eth_erc20_transfer_detected_total` | Counter | - | ERC20 transfers detected |
| `canopy_eth_lock_order_detected_total` | Counter | - | Lock orders successfully parsed |
| `canopy_eth_close_order_detected_total` | Counter | - | Close orders successfully parsed |
| `canopy_eth_order_validation_errors_total` | Counter | `order_type`, `error_type` | Order validation errors |

**Order Type Labels:**
- `lock` - Lock order
- `close` - Close order

### Token Cache Error Metrics (Medium Priority)

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `canopy_eth_token_info_fetch_errors_total` | Counter | `field` | Token info fetch errors by field |
| `canopy_eth_token_contract_call_timeouts_total` | Counter | - | Token contract call timeouts |
| `canopy_eth_token_cache_hits_total` | Counter | - | Token cache hits |
| `canopy_eth_token_cache_misses_total` | Counter | - | Token cache misses |

**Field Labels:**
- `name` - Error fetching token name
- `symbol` - Error fetching token symbol
- `decimals` - Error fetching token decimals

### Connection Error Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `canopy_eth_connection_errors_total` | Counter | Total Ethereum connection errors |

---

## Example Prometheus Queries

### Oracle Health

```promql
# Orders pending confirmation
canopy_oracle_orders_awaiting_confirmation

# Validation failure rate by reason
rate(canopy_oracle_validation_failures_total[5m])

# Order processing success rate
rate(canopy_oracle_orders_validated_total[5m]) / rate(canopy_oracle_orders_witnessed_total[5m])
```

### Ethereum Connectivity

```promql
# Connection state (should be 3 for healthy)
canopy_eth_connection_state

# Sync status (should be 2 for synced)
canopy_eth_sync_status

# Block lag (should be low when synced)
canopy_eth_block_height_lag

# RPC error rate
rate(canopy_eth_rpc_connection_errors_total[5m])
```

### Block Processing

```promql
# Blocks processed per second
rate(canopy_eth_blocks_processed_total[5m])

# Block fetch error rate
rate(canopy_eth_block_fetch_errors_total[5m])

# Average batch size
histogram_quantile(0.5, canopy_eth_process_blocks_batch_size)
```

### Transaction Processing

```promql
# Transaction success rate
sum(rate(canopy_eth_transaction_success_status_total{status="success"}[5m])) /
sum(rate(canopy_eth_transaction_success_status_total[5m]))

# Retry distribution
rate(canopy_eth_transaction_retry_by_attempt_total[5m])

# Exhausted retries (potential issues)
rate(canopy_eth_transaction_exhausted_retries_total[5m])
```

### Order Detection

```promql
# Lock orders detected per minute
rate(canopy_eth_lock_order_detected_total[1m]) * 60

# Close orders detected per minute
rate(canopy_eth_close_order_detected_total[1m]) * 60

# Order validation error rate
rate(canopy_eth_order_validation_errors_total[5m])
```

---

## Alerting Recommendations

### Critical Alerts

| Condition | Threshold | Description |
|-----------|-----------|-------------|
| `canopy_eth_connection_state < 3` | > 5 min | Ethereum connection issues |
| `canopy_eth_sync_status < 2` | > 10 min | Oracle not synced |
| `canopy_eth_block_height_lag > 100` | > 5 min | Significant block processing lag |
| `rate(canopy_eth_rpc_connection_errors_total[5m]) > 0.1` | - | High RPC error rate |

### Warning Alerts

| Condition | Threshold | Description |
|-----------|-----------|-------------|
| `canopy_eth_block_height_lag > 10` | > 2 min | Block processing falling behind |
| `rate(canopy_oracle_validation_failures_total[5m]) > 0.01` | - | Validation failures occurring |
| `rate(canopy_eth_transaction_exhausted_retries_total[5m]) > 0` | - | Transactions failing after retries |
| `canopy_oracle_orders_awaiting_confirmation > 50` | - | Many orders pending confirmation |

---

## Grafana Dashboard Tips

1. **Connection Status Panel**: Use stat panel with value mappings for `canopy_eth_connection_state` and `canopy_eth_sync_status`

2. **Block Heights Panel**: Graph showing `canopy_eth_chain_head_height`, `canopy_eth_last_processed_height`, and `canopy_eth_safe_height` together

3. **Order Flow Panel**: Stacked graph of `canopy_eth_lock_order_detected_total` and `canopy_eth_close_order_detected_total` rates

4. **Error Breakdown Panel**: Table showing validation failures grouped by reason label
