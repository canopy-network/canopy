import type { NextConfig } from "next";

// Backend destinations are configurable so the SAME build works locally
// (defaults to 127.0.0.1) and on Vercel (set these to your node's public
// URL or a tunnel). Server-side only (NOT NEXT_PUBLIC): the browser always
// calls the same-origin /canopy-rpc, /canopy-admin, /arbor-rpc paths and
// Next proxies them server-side, which also sidesteps CORS entirely.
const canopyRpcDest = process.env.CANOPY_RPC_DEST || "http://127.0.0.1:50002";
const canopyAdminDest = process.env.CANOPY_ADMIN_DEST || "http://127.0.0.1:50003";
const arborRpcDest = process.env.ARBOR_RPC_DEST || "http://127.0.0.1:50010";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      { source: "/canopy-rpc/:path*", destination: `${canopyRpcDest}/:path*` },
      { source: "/canopy-admin/:path*", destination: `${canopyAdminDest}/:path*` },
      { source: "/arbor-rpc/:path*", destination: `${arborRpcDest}/:path*` },
    ];
  },
};

export default nextConfig;
