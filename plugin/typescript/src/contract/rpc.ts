/*
This file is a SKELETON HTTP server for the plugin's own custom RPC endpoints.

Canopy core only exposes a single, generic, read-only transport over the unix socket:
`plugin.queryState(height, read)`, which returns raw key/value state at a historical height. The
plugin process owns its HTTP server entirely, so builders may register as many routes as they want
and decode their own keys/protobufs into whatever response shapes they like. Canopy never needs to
know about chain-specific endpoints.

By default this skeleton registers NO routes: it simply starts the server so the wiring exists.
Builders add their own chain-specific endpoints backed by `queryState`. See TUTORIAL.md for a
worked example.

Example of registering a custom route backed by queryState:

    if (url.pathname === '/v1/query/widgets') {
        // perform a detached, read-only state query, e.g.:
        //   const [resp, err] = await plugin.queryState(height, {
        //       keys: [{ queryId: randQueryId(), key: KeyForWidget(address) }],
        //   });
        // then decode resp.results[].entries[].value into your own protobuf type and write JSON.
        return;
    }
*/

import * as http from 'http';

import { Plugin, PLUGIN_BUILD } from './plugin.js';

// StartRPCServer() launches the plugin's own HTTP server that exposes custom, chain-specific RPC
// endpoints. By default no routes are registered; builders add their own routes here, each using
// the detached, read-only queryState() path to fetch state snapshots from Canopy.
export function StartRPCServer(plugin: Plugin): void {
    // resolve the listen address from config
    const addr = plugin.config.rpcAddress;
    // if no address is configured, the RPC server is disabled
    if (!addr) {
        console.log('plugin RPC server disabled (no rpcAddress configured)');
        return;
    }

    const server = http.createServer((_req, res) => {
        // no custom routes are registered by default; builders add their own routes above
        writeJSONError(res, 404, 'not found');
    });

    // split the listen address into host:port (default 0.0.0.0:50010)
    const idx = addr.lastIndexOf(':');
    const host = idx >= 0 ? addr.slice(0, idx) : '0.0.0.0';
    const port = idx >= 0 ? Number(addr.slice(idx + 1)) : Number(addr);

    server.listen(port, host, () => {
        // log the build marker so the running version is obvious in the log
        console.log(`plugin RPC server (${PLUGIN_BUILD}) listening on ${addr}`);
        console.log('plugin RPC server started (no custom routes registered)');
    });

    server.on('error', (err) => {
        console.log(`plugin RPC server error: ${err.message}`);
    });
}

// writeJSONError() writes a JSON error response with the given status code
function writeJSONError(res: http.ServerResponse, status: number, message: string): void {
    res.statusCode = status;
    res.setHeader('Content-Type', 'application/json');
    res.end(JSON.stringify({ error: message }));
}
