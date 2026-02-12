import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  // Load env file based on `mode` in the current working directory.
  const env = loadEnv(mode, ".", "");

  // Determine base path based on environment
  // Priority: VITE_WALLET_BASE_PATH env var > production default > development default
  const getBasePath = () => {
    // If explicitly set via environment variable, use it
    if (env.VITE_WALLET_BASE_PATH) {
      return env.VITE_WALLET_BASE_PATH;
    }
    // In development, use / for local dev
    if (mode === "development") {
      return "/";
    }
    // In production, use /wallet/ because the app is served behind a reverse proxy
    // at http://node1.localhost/wallet/
    // This ensures:
    // 1. Assets are requested as /wallet/assets/... (Traefik strips /wallet, Go server gets /assets/...)
    // 2. React Router basename is /wallet (matches browser URL)
    return "/wallet/";
  };

  return {
    base: getBasePath(),
    resolve: {
      alias: {
        "@": "/src",
      },
    },
    plugins: [react()],
    build: {
      outDir: "out",
      assetsDir: "assets",
    },

    // Development server configuration
    server: {
      port: 5173,
      proxy: {
        // Proxy /rpc to RPC server
        '/rpc': {
          target: env.VITE_WALLET_RPC_PROXY_TARGET || 'http://localhost:50002',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/rpc/, ''),
        },
        // Proxy /adminrpc to Admin RPC server
        '/adminrpc': {
          target: env.VITE_WALLET_ADMIN_RPC_PROXY_TARGET || 'http://localhost:50003',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/adminrpc/, ''),
        },
      },
    },

    define: {
      // Ensure environment variables are available at build time
      "import.meta.env.VITE_NODE_ENV": JSON.stringify(
        env.VITE_NODE_ENV || "development",
      ),
    },
  };
});
