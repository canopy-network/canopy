/*
This file contains an EXAMPLE HTTP server that demonstrates how a plugin builder exposes their own
custom RPC endpoints for their chain.

Canopy core only exposes a single, generic, read-only transport over the unix socket:
`Plugin.queryState(height, read)`, which returns raw key/value state at a historical height. The
plugin process owns its HTTP server entirely, so builders may register as many routes as they want
and decode their own keys/protobufs into whatever response shapes they like. Canopy never needs to
know about chain-specific endpoints.

The endpoints below are intentionally plugin-specific (faucet and reward records) so they showcase
data that does NOT exist in the Canopy node's own RPC. Account/pool queries already exist in core,
so they make poor examples of a *custom* endpoint.
*/

import * as http from 'http';
import Long from 'long';

import { types } from '../proto/types.js';
import { IPluginError } from './error.js';
import { Plugin, PLUGIN_BUILD, Unmarshal } from './plugin.js';
import { KeyForFaucet, FaucetPrefix, KeyForReward, RewardPrefix } from './contract.js';

// StartRPCServer() launches the plugin's own HTTP server exposing custom, chain-specific RPC endpoints.
// Builders are free to register any number of routes; each handler uses the detached, read-only
// queryState() path to fetch state snapshots from Canopy.
export function StartRPCServer(plugin: Plugin): void {
    // resolve the listen address from config
    const addr = plugin.config.rpcAddress;
    // if no address is configured, the RPC server is disabled
    if (!addr) {
        console.log('plugin RPC server disabled (no rpcAddress configured)');
        return;
    }

    const server = http.createServer((req, res) => {
        const url = new URL(req.url || '', 'http://localhost');
        // GET /v1/query/faucets[?address=<hex>][&height=<uint64>]
        if (url.pathname === '/v1/query/faucets') {
            handleQueryFaucets(plugin, url, res).catch((e) => {
                writeJSONError(res, 500, (e as Error).message);
            });
            return;
        }
        // GET /v1/query/rewards[?address=<hex>][&height=<uint64>]
        if (url.pathname === '/v1/query/rewards') {
            handleQueryRewards(plugin, url, res).catch((e) => {
                writeJSONError(res, 500, (e as Error).message);
            });
            return;
        }
        writeJSONError(res, 404, 'not found');
    });

    // split the listen address into host:port (default 0.0.0.0:50010)
    const idx = addr.lastIndexOf(':');
    const host = idx >= 0 ? addr.slice(0, idx) : '0.0.0.0';
    const port = idx >= 0 ? Number(addr.slice(idx + 1)) : Number(addr);

    server.listen(port, host, () => {
        // log the build marker and the registered routes so the running version is obvious in the log
        console.log(`plugin RPC server (${PLUGIN_BUILD}) listening on ${addr}`);
        console.log(
            'plugin RPC routes registered: GET /v1/query/faucets, GET /v1/query/rewards'
        );
    });

    server.on('error', (err) => {
        console.log(`plugin RPC server error: ${err.message}`);
    });
}

// handleQueryFaucets() returns faucet records. With ?address=<hex> it returns a single recipient's
// record; otherwise it returns every faucet record via a range read over the faucet prefix.
async function handleQueryFaucets(
    plugin: Plugin,
    url: URL,
    res: http.ServerResponse
): Promise<void> {
    // optional single-record lookup by recipient address
    const addrHex = url.searchParams.get('address');
    if (addrHex) {
        if (!/^[0-9a-fA-F]{40}$/.test(addrHex)) {
            writeJSONError(res, 400, 'address must be a 20-byte hex string');
            return;
        }
        const address = Buffer.from(addrHex, 'hex');
        const height = parseHeight(url);
        const [value, err] = await queryValue(plugin, height, KeyForFaucet(address));
        if (err) {
            writeJSONError(res, 500, err.msg);
            return;
        }
        const [faucet, uErr] = Unmarshal(value || new Uint8Array(), types.Faucet);
        if (uErr) {
            writeJSONError(res, 500, uErr.msg);
            return;
        }
        writeJSON(res, { faucet: faucetToJSON(faucet), height });
        return;
    }
    // otherwise return all faucet records via a range read
    const height = parseHeight(url);
    const [entries, err] = await queryRange(plugin, height, FaucetPrefix());
    if (err) {
        writeJSONError(res, 500, err.msg);
        return;
    }
    // decode each entry into the plugin's own Faucet type
    const faucets: Record<string, unknown>[] = [];
    for (const entry of entries || []) {
        const [faucet, uErr] = Unmarshal(entry.value, types.Faucet);
        if (uErr) {
            console.log(
                `faucet decode error: key=${bytesToHex(entry.key)} value=${bytesToHex(entry.value)} err=${uErr.msg}`
            );
            writeJSONError(res, 500, uErr.msg);
            return;
        }
        faucets.push(faucetToJSON(faucet));
    }
    writeJSON(res, { faucets, count: faucets.length, height });
}

// handleQueryRewards() returns reward records. With ?address=<hex> it returns a single recipient's
// record; otherwise it returns every reward record via a range read over the reward prefix.
async function handleQueryRewards(
    plugin: Plugin,
    url: URL,
    res: http.ServerResponse
): Promise<void> {
    // optional single-record lookup by recipient address
    const addrHex = url.searchParams.get('address');
    if (addrHex) {
        if (!/^[0-9a-fA-F]{40}$/.test(addrHex)) {
            writeJSONError(res, 400, 'address must be a 20-byte hex string');
            return;
        }
        const address = Buffer.from(addrHex, 'hex');
        const height = parseHeight(url);
        const [value, err] = await queryValue(plugin, height, KeyForReward(address));
        if (err) {
            writeJSONError(res, 500, err.msg);
            return;
        }
        const [reward, uErr] = Unmarshal(value || new Uint8Array(), types.Reward);
        if (uErr) {
            writeJSONError(res, 500, uErr.msg);
            return;
        }
        writeJSON(res, { reward: rewardToJSON(reward), height });
        return;
    }
    // otherwise return all reward records via a range read
    const height = parseHeight(url);
    const [entries, err] = await queryRange(plugin, height, RewardPrefix());
    if (err) {
        writeJSONError(res, 500, err.msg);
        return;
    }
    // decode each entry into the plugin's own Reward type
    const rewards: Record<string, unknown>[] = [];
    for (const entry of entries || []) {
        const [reward, uErr] = Unmarshal(entry.value, types.Reward);
        if (uErr) {
            console.log(
                `reward decode error: key=${bytesToHex(entry.key)} value=${bytesToHex(entry.value)} err=${uErr.msg}`
            );
            writeJSONError(res, 500, uErr.msg);
            return;
        }
        rewards.push(rewardToJSON(reward));
    }
    writeJSON(res, { rewards, count: rewards.length, height });
}

// faucetToJSON() shapes a Faucet record into a JSON-friendly map (hex-encoding addresses)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function faucetToJSON(faucet: any): Record<string, unknown> {
    return {
        recipientAddress: bytesToHex(faucet?.recipientAddress),
        totalAmount: uint64ToJSON(faucet?.totalAmount),
        count: uint64ToJSON(faucet?.count)
    };
}

// rewardToJSON() shapes a Reward record into a JSON-friendly map (hex-encoding addresses)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function rewardToJSON(reward: any): Record<string, unknown> {
    return {
        recipientAddress: bytesToHex(reward?.recipientAddress),
        lastAdminAddress: bytesToHex(reward?.lastAdminAddress),
        totalAmount: uint64ToJSON(reward?.totalAmount),
        count: uint64ToJSON(reward?.count)
    };
}

// queryValue() performs a single-key detached read and returns the raw value bytes (null = not found)
async function queryValue(
    plugin: Plugin,
    height: number,
    key: Uint8Array
): Promise<[Uint8Array | null, IPluginError | null]> {
    // execute a detached, read-only state query for the single key
    const [resp, err] = await plugin.queryState(height, {
        keys: [{ queryId: randQueryId(), key }]
    });
    if (err) {
        return [null, err];
    }
    // extract the first entry value if present (null means 'not found')
    const entries = resp?.results?.[0]?.entries;
    if (!entries || entries.length === 0) {
        return [null, null];
    }
    return [entries[0].value, null];
}

// queryRange() performs a detached range read over a key prefix
async function queryRange(
    plugin: Plugin,
    height: number,
    prefix: Uint8Array
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
): Promise<[any[] | null, IPluginError | null]> {
    // execute a detached, read-only range query over the prefix
    const [resp, err] = await plugin.queryState(height, {
        ranges: [{ queryId: randQueryId(), prefix }]
    });
    if (err) {
        return [null, err];
    }
    // return the entries of the first (only) range result, if present
    if (!resp?.results || resp.results.length === 0) {
        return [[], null];
    }
    return [resp.results[0].entries || [], null];
}

// randQueryId() generates a random query id to correlate batch state read requests
function randQueryId(): Long {
    return Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));
}

// parseHeight() reads the optional 'height' query parameter, defaulting to 0 (latest committed)
function parseHeight(url: URL): number {
    const h = url.searchParams.get('height');
    if (!h) {
        return 0;
    }
    const n = Number(h);
    return Number.isFinite(n) && n >= 0 ? Math.floor(n) : 0;
}

// bytesToHex() hex-encodes a bytes field (may be Uint8Array/Buffer or undefined)
function bytesToHex(bytes: Uint8Array | undefined | null): string {
    if (!bytes) {
        return '';
    }
    return Buffer.from(bytes).toString('hex');
}

// uint64ToJSON() converts a protobuf uint64 (Long or number) into JSON output, using a number when
// it fits safely and falling back to a string to avoid precision loss for very large values
function uint64ToJSON(v: Long | number | undefined | null): number | string {
    if (v === undefined || v === null) {
        return 0;
    }
    const l = Long.isLong(v) ? v : Long.fromNumber(v as number);
    return l.lessThanOrEqual(Long.fromNumber(Number.MAX_SAFE_INTEGER)) ? l.toNumber() : l.toString();
}

// writeJSON() writes a JSON success response
function writeJSON(res: http.ServerResponse, body: unknown): void {
    res.setHeader('Content-Type', 'application/json');
    res.end(JSON.stringify(body));
}

// writeJSONError() writes a JSON error response with the given status code
function writeJSONError(res: http.ServerResponse, status: number, message: string): void {
    res.statusCode = status;
    res.setHeader('Content-Type', 'application/json');
    res.end(JSON.stringify({ error: message }));
}
