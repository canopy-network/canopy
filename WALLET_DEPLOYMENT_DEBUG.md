# Wallet Deployment Debugging Guide

## Problem
Assets not loading when accessing wallet at `https://node1.canoliq.org/wallet/`

## Root Cause Analysis

### Current Setup
- **URL**: `https://node1.canoliq.org/wallet/`
- **Go Server**: Listens on port 50000, serves at root `/`
- **Traefik**: Reverse proxy handling `/wallet/` path

### The Issue
When the HTML is at `/wallet/` but uses relative paths `./assets/...`, the browser looks for:
- `https://node1.canoliq.org/wallet/./assets/file.js` ✅ (correct)

BUT, if Traefik is NOT stripping the `/wallet/` prefix, the Go server receives:
- Request: `/wallet/assets/file.js`
- But Go server serves files from: `/assets/file.js` (root)
- Result: 404

## Solutions

### Solution 1: Configure Traefik to Strip Path Prefix (RECOMMENDED)

Update your Traefik configuration to strip `/wallet/` before proxying to the Go server.

**Example Traefik Configuration (docker-compose labels):**

```yaml
services:
  canopy:
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.wallet.rule=Host(`node1.canoliq.org`) && PathPrefix(`/wallet`)"
      - "traefik.http.routers.wallet.entrypoints=websecure"
      - "traefik.http.services.wallet.loadbalancer.server.port=50000"

      # ADD THIS: Strip the /wallet prefix before sending to backend
      - "traefik.http.middlewares.wallet-stripprefix.stripprefix.prefixes=/wallet"
      - "traefik.http.routers.wallet.middlewares=wallet-stripprefix"
```

**OR in Traefik static config (traefik.yml):**

```yaml
http:
  middlewares:
    wallet-stripprefix:
      stripPrefix:
        prefixes:
          - "/wallet"
  routers:
    wallet:
      rule: "Host(`node1.canoliq.org`) && PathPrefix(`/wallet`)"
      service: wallet
      middlewares:
        - wallet-stripprefix
  services:
    wallet:
      loadBalancer:
        servers:
          - url: "http://localhost:50000"
```

### Solution 2: Serve Wallet at Root Domain (Alternative)

If you want wallet at the root:

```yaml
services:
  canopy:
    labels:
      - "traefik.http.routers.wallet.rule=Host(`wallet.canoliq.org`)"
      - "traefik.http.routers.wallet.entrypoints=websecure"
      - "traefik.http.services.wallet.loadbalancer.server.port=50000"
```

Then access at: `https://wallet.canoliq.org/`

### Solution 3: Configure Vite with Absolute Base Path (NOT RECOMMENDED)

This would require knowing the exact deployment path:

```typescript
// vite.config.ts
export default defineConfig({
  base: "/wallet/",  // Hard-coded path
  // ...
});
```

**Downsides:**
- Won't work locally (expects /wallet/ path)
- Less flexible
- Doesn't work if you change the path

## Testing Steps

### 1. Verify Current Traefik Configuration

```bash
# On your server
docker exec <traefik-container> cat /etc/traefik/traefik.yml
# or
docker compose config
```

Look for PathPrefix stripping configuration.

### 2. Test Asset Loading

Open browser DevTools (F12) → Network tab, then access:
`https://node1.canoliq.org/wallet/`

Check the failing requests:
- If they request: `/wallet/assets/file.js` → Traefik IS passing the prefix
- If they request: `/assets/file.js` → Path stripping is working

### 3. Verify Go Server Response

```bash
# Direct test to Go server (if accessible)
curl -I http://localhost:50000/assets/index-CdcGvnGe.js

# Through Traefik
curl -I https://node1.canoliq.org/wallet/assets/index-CdcGvnGe.js
```

## Quick Fix Checklist

- [ ] Updated `vite.config.ts` with `base: "./"` ✅ (already done)
- [ ] Rebuilt wallet with `npm run build` ✅ (already done)
- [ ] Added `build/new-wallet` target to Makefile ✅ (already done)
- [ ] Committed and pushed changes to git ✅ (already done)
- [ ] **Configure Traefik StripPrefix middleware** ← DO THIS NOW
- [ ] Rebuild Docker image: `docker compose build --no-cache`
- [ ] Restart containers: `docker compose down && docker compose up -d`
- [ ] Clear browser cache or hard refresh (Ctrl+Shift+R)

## Expected Behavior After Fix

1. User visits: `https://node1.canoliq.org/wallet/`
2. Traefik strips `/wallet/` and forwards to Go server as: `/`
3. Go server returns `index.html`
4. Browser requests: `https://node1.canoliq.org/wallet/assets/file.js`
5. Traefik strips `/wallet/` and forwards as: `/assets/file.js`
6. Go server serves embedded file ✅

## How to Find Your Traefik Config

Common locations:
- Docker Compose labels: `docker-compose.yaml` or `docker-compose.yml`
- Traefik static config: `/etc/traefik/traefik.yml`
- Traefik dynamic config: `/etc/traefik/dynamic/` or labels in compose file
- Monitoring stack: `monitoring-stack/loadbalancer/traefik.yml`
- Service definitions: `monitoring-stack/loadbalancer/services/prod.yaml`
