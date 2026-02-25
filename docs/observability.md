# Observability Notes (P2P Non-Signers / Heartbeat)

This repo exposes Prometheus metrics from the node at `/metrics`.

## Heartbeat / Pong Debugging

New low-cardinality metrics (no per-peer labels):
- `canopy_p2p_heartbeat_ping_sent_total`
- `canopy_p2p_heartbeat_ping_recv_total`
- `canopy_p2p_heartbeat_pong_sent_total`
- `canopy_p2p_heartbeat_pong_recv_total`
- `canopy_p2p_heartbeat_rtt_seconds` (histogram)
- `canopy_p2p_heartbeat_timeout_total`

Useful PromQL:
```promql
rate(canopy_p2p_heartbeat_timeout_total[5m])
```

```promql
rate(canopy_p2p_heartbeat_ping_sent_total[5m])
rate(canopy_p2p_heartbeat_pong_recv_total[5m])
```

```promql
histogram_quantile(0.99, rate(canopy_p2p_heartbeat_rtt_seconds_bucket[5m]))
```

On heartbeat timeout, the node logs a single line with:
`lastHeardAge`, `lastPongAge`, `lastPingSentAge`, `lastPingRecvAge`, `lastPongSentAge`.

## Dial / Peer-Book Churn (Ephemeral Port Pollution)

New metrics (low-cardinality):
- `canopy_p2p_dial_attempt_total{expected_port="true|false|unknown"}`
- `canopy_p2p_dial_success_total{expected_port="true|false|unknown"}`
- `canopy_p2p_dial_timeout_total{expected_port="true|false|unknown"}`
- `canopy_p2p_peer_book_add_total{expected_port="true|false|unknown"}`

`expected_port=true` means the address uses the chain's expected P2P port (e.g. 9001 for chain 1).

Useful PromQL:
```promql
rate(canopy_p2p_dial_timeout_total{expected_port="false"}[5m])
```

```promql
rate(canopy_p2p_peer_book_add_total{expected_port="false"}[5m])
```

If `expected_port="false"` dominates during incident windows, the peer-book is likely being populated with undialable endpoints (e.g. inbound ephemeral source ports), amplifying churn and increasing the probability of missed heartbeats/votes.

## Host TCP Loss Signals (Node Exporter)

To prove whether incidents are driven by network loss/jitter, scrape *host* TCP stats (not container namespace).

Recommended PromQL:
```promql
rate(node_netstat_Tcp_RetransSegs[5m])
rate(node_netstat_TcpExt_TCPSynRetrans[5m])
rate(node_netstat_TcpExt_TCPTimeouts[5m])
```

If you run `node-exporter` in Docker, use host namespaces (example):
- `network_mode: host`
- `pid: host`
- mount host `/proc` and `/sys`
- run with `--path.procfs=/host/proc --path.sysfs=/host/sys`

