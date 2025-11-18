import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  // Load env file based on `mode` in the current working directory.
  const env = loadEnv(mode, ".", "");

  return {
    base: "./",
    resolve: {
      alias: {
        "@": "/src",
      },
    },
    plugins: [react()],
    build: {
      outDir: "out",
    },
    define: {
      // Ensure environment variables are available at build time
      "import.meta.env.VITE_NODE_ENV": JSON.stringify(
        env.VITE_NODE_ENV || "development",
      ),
    },
  };
});
