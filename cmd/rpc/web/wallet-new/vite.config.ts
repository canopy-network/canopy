import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  // Load env file based on `mode` in the current working directory.
  const env = loadEnv(mode, ".", "");

  // Determine base path based on environment
  // Priority: VITE_BASE_PATH env var > production default > development default
  const getBasePath = () => {
    // If explicitly set via environment variable, use it
    if (env.VITE_BASE_PATH) {
      return env.VITE_BASE_PATH;
    }
    // In production, use /wallet/ to match the deployment path
    if (mode === "production") {
      return "/wallet/";
    }
    // In development, use relative paths for local testing
    return "./";
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
    define: {
      // Ensure environment variables are available at build time
      "import.meta.env.VITE_NODE_ENV": JSON.stringify(
        env.VITE_NODE_ENV || "development",
      ),
    },
  };
});
