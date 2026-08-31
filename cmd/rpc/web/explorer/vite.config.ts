import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  // Load env file based on `mode` in the current working directory.
  const env = loadEnv(mode, '.', '')

  // Mirror netlify.toml redirects so the dev server speaks the same
  // same-origin proxy paths used in production (/rpc-node1, /admin-node1).
  // VITE_DEV_RPC_PROXY / VITE_DEV_ADMIN_PROXY override the default upstreams.
  const rpcUpstream =
    env.VITE_DEV_RPC_PROXY ||
    env.VITE_PUBLIC_RPC_URL ||
    'https://node1.canopy.us.nodefleet.net/rpc'
  const adminUpstream =
    env.VITE_DEV_ADMIN_PROXY ||
    env.VITE_PUBLIC_ADMIN_RPC_URL ||
    'https://node1.canopy.us.nodefleet.net/admin'

  const stripTrailingSlash = (value: string) => value.replace(/\/+$/, '')

  return {
    plugins: [react()],
    define: {
      // Ensure environment variables are available at build time
      'import.meta.env.VITE_NODE_ENV': JSON.stringify(env.VITE_NODE_ENV || 'development'),
    },
    server: {
      proxy: {
        '/rpc-node1': {
          target: stripTrailingSlash(rpcUpstream),
          changeOrigin: true,
          secure: true,
          rewrite: (path) => path.replace(/^\/rpc-node1/, ''),
        },
        '/admin-node1': {
          target: stripTrailingSlash(adminUpstream),
          changeOrigin: true,
          secure: true,
          rewrite: (path) => path.replace(/^\/admin-node1/, ''),
        },
      },
    },
  }
})
