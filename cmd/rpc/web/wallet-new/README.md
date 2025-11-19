# Canopy Wallet

Modern React wallet for the Canopy Network, built with Vite + TypeScript.

## Development

### Prerequisites

- Node.js 18+
- npm

### Setup

1. Install dependencies:
   ```bash
   npm install
   ```

2. Copy environment file:
   ```bash
   cp .env.example .env
   ```

3. Start development server:
   ```bash
   npm run dev
   ```

The wallet will be available at `http://localhost:5173`

## Building for Production

### Environment Configuration

The build process uses the `VITE_BASE_PATH` environment variable to configure the deployment path.

**Default production path**: `/wallet/`

To customize the base path, create a `.env` file:

```bash
# For deployment at https://example.com/wallet/
VITE_BASE_PATH=/wallet/

# For deployment at root domain https://wallet.example.com/
VITE_BASE_PATH=/

# For custom subdirectory
VITE_BASE_PATH=/my-custom-path/
```

### Build Commands

```bash
# Production build (uses /wallet/ by default)
npm run build

# Build with custom base path
VITE_BASE_PATH=/custom/ npm run build

# Preview production build
npm run preview
```

The build output will be in the `out/` directory.

## Deployment

### Docker Build

The wallet is automatically built during the Docker image build process via the Makefile:

```bash
# From project root
make build/wallet
```

This is automatically called by the Dockerfile.

### Manual Deployment

1. Build the wallet:
   ```bash
   npm run build
   ```

2. The compiled assets will be embedded in the Go binary during the build process via `//go:embed` directives.

### Reverse Proxy Configuration

When deploying behind a reverse proxy (like Traefik), ensure the proxy is configured to strip the path prefix:

**Example Traefik Configuration:**

```yaml
http:
  middlewares:
    strip-wallet-prefix:
      stripPrefix:
        prefixes:
          - "/wallet"
        forceSlash: false

  routers:
    wallet:
      rule: "Host(`example.com`) && PathPrefix(`/wallet`)"
      service: wallet
      middlewares:
        - strip-wallet-prefix
```

This ensures that requests to `/wallet/assets/file.js` are forwarded to the Go server as `/assets/file.js`.

## Troubleshooting

### Assets not loading in production

**Problem**: CSS and JS files return 404 or wrong MIME type.

**Solution**:
1. Verify `VITE_BASE_PATH` matches your deployment path
2. Ensure reverse proxy is configured to strip the path prefix
3. Rebuild the Docker image after changing the base path
