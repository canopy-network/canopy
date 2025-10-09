# Environment Configuration

This project uses environment variables to configure RPC URLs and other parameters.

## Setup

### 1. Create .env file

Copy the `env.example` file to `.env`:

```bash
cp env.example .env
```

### 2. Configure variables

Edit the `.env` file with your values:

```env
# For local development
VITE_RPC_URL=http://localhost:50002
VITE_ADMIN_RPC_URL=http://localhost:50003
VITE_CHAIN_ID=1
VITE_NODE_ENV=development

# For production (used automatically when VITE_NODE_ENV=production)
VITE_PUBLIC_RPC_URL=https://node1.canopy.us.nodefleet.net/rpc/
VITE_PUBLIC_ADMIN_RPC_URL=https://node1.canopy.us.nodefleet.net/admin/
```

### 3. Available variables

| Variable | Description | Default Value |
|----------|-------------|---------------|
| `VITE_RPC_URL` | RPC URL for development | `http://localhost:50002` |
| `VITE_ADMIN_RPC_URL` | Admin RPC URL for development | `http://localhost:50003` |
| `VITE_CHAIN_ID` | Chain ID | `1` |
| `VITE_NODE_ENV` | Execution mode | `development` |
| `VITE_PUBLIC_RPC_URL` | Public RPC URL for production | `https://node1.canopy.us.nodefleet.net/rpc/` |
| `VITE_PUBLIC_ADMIN_RPC_URL` | Public Admin RPC URL for production | `https://node1.canopy.us.nodefleet.net/admin/` |

## Usage

### Development
```bash
npm run dev
```

### Production
```bash
npm run build
```

## Compatibility

The project maintains compatibility with the previous configuration using `window.__CONFIG__` for cases where configuration is dynamically injected from the server.

## Notes

- Environment variables starting with `VITE_` are exposed to the client
- In production mode (`VITE_NODE_ENV=production`), public URLs are used automatically
- The `.env` file should not be committed to the repository (it's in .gitignore)
