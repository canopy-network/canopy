import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        source: "/canopy-rpc/:path*",
        destination: "http://127.0.0.1:50002/:path*",
      },
      {
        source: "/canopy-admin/:path*",
        destination: "http://127.0.0.1:50003/:path*",
      },
      {
        source: "/arbor-rpc/:path*",
        destination: "http://127.0.0.1:50010/:path*",
      },
    ];
  },
};

export default nextConfig;
