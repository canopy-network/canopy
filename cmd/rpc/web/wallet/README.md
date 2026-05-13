# Canopy Wallet

A modern, **config-first blockchain wallet** built with React, TypeScript, and Tailwind CSS. The wallet features a dynamic, configuration-driven architecture where blockchain interactions, UI forms, and data sources are defined through JSON configuration files rather than hardcoded application logic.

## 🌟 Features

### Core Functionality
- **Multi-Account Management**: Create, import, and manage multiple blockchain accounts
- **Transaction Management**: Send, receive, and track transactions with real-time status updates
- **Staking Operations**: Stake, unstake, pause/unpause validators with comprehensive management tools
- **Governance Participation**: Vote on proposals and create new governance proposals
- **Real-time Monitoring**: Monitor node performance, network peers, system resources, and logs

### Architecture Highlights
- **Config-First Approach**: All blockchain interactions defined in `chain.json` and `manifest.json`
- **Data Source (DS) Pattern**: Centralized API configuration and caching
- **Dynamic Form Generation**: Transaction forms generated from JSON configuration
- **Real-time Updates**: Live data updates using React Query with configurable intervals
- **Responsive Design**: Modern UI with dark theme and responsive layouts

## 🏗️ Architecture Overview

### Config-First Design
The wallet operates on a **config-first** principle where blockchain-specific configurations are externalized into JSON files:

```
public/plugin/canopy/
├── chain.json          # RPC endpoints, data sources, parameters
└── manifest.json       # Transaction forms, UI definitions, actions
```

### Data Source (DS) Pattern
All API calls use a centralized DS system defined in `chain.json`:

```json
{
  "ds": {
    "height": {
      "source": { "base": "rpc", "path": "/v1/query/height", "method": "POST" },
      "selector": "",
      "coerce": { "response": { "": "int" } }
    },
    "admin": {
      "consensusInfo": {
        "source": { "base": "admin", "path": "/v1/admin/consensus-info", "method": "GET" }
      }
    }
  }
}
```

### Component Structure
```
src/
├── app/
│   ├── pages/           # Main application pages
│   └── providers/       # React context providers
├── components/
│   ├── dashboard/       # Dashboard widgets
│   ├── monitoring/      # Node monitoring components
│   ├── staking/         # Staking management UI
│   └── ui/              # Reusable UI components
├── hooks/               # Custom React hooks
├── core/                # Core utilities and DS system
└── manifest/            # Manifest parsing and types
```

## 🚀 Getting Started

### Prerequisites
- Node.js 18+ and npm/pnpm
- A running Canopy node with RPC and Admin endpoints

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd canopy-wallet
   ```

2. **Install dependencies**
   ```bash
   npm install
   # or
   pnpm install
   ```

3. **Configure your node connection**

   Edit `public/plugin/canopy/chain.json`:
   ```json
   {
     "rpc": {
       "base": "http://your-node-ip:50002",
       "admin": "http://your-node-ip:50003"
     }
   }
   ```


4. Copy environment file:
   ```bash
   cp .env.example .env
   ```


5. **Start the development server**
   ```bash
   npm run dev
   ```

6. **Open your browser**
   ```
   http://localhost:5173
   ```

## 📁 Configuration Files

### chain.json
Defines blockchain-specific configuration:

- **RPC Endpoints**: Base and admin API URLs
- **Data Sources**: API endpoint definitions with caching strategies
- **Fee Configuration**: Transaction fee parameters and providers
- **Network Parameters**: Chain ID, denomination, explorer URLs
- **Session Settings**: Unlock timeouts and security preferences

### manifest.json
Defines dynamic UI and transaction forms:

- **Actions**: Transaction templates (send, stake, governance)
- **Form Fields**: Dynamic form generation with validation
- **UI Mapping**: Icons, labels, and transaction categorization
- **Payload Construction**: Data transformation for API calls

### Declarative Architecture Docs
- **Declarative Actions + Chain Guide**: `docs/declarative-actions-and-chain.md`
- Covers the config-first runtime flow, purpose of `chain.json` and `manifest.json`, and implementation guidelines.

## 🖥️ Main Features

### Dashboard
- Account balance overview with 24h change tracking
- Recent transaction history with status indicators
- Quick action buttons for common operations
- Network status and validator information

### Account Management
- Create new accounts with secure key generation
- Import existing accounts from private keys
- Export account information and QR codes
- Multi-account switching and management

### Staking
- Comprehensive validator management
- Real-time staking statistics and rewards tracking
- Bulk operations for multiple validators
- Performance metrics and chain participation

### Governance
- View active proposals with voting status
- Cast votes with detailed proposal information
- Create new governance proposals
- Track voting history and participation

### Monitoring
- **Real-time Node Status**: Sync status, block height, consensus information
- **Network Peers**: Connected peers, network topology
- **System Resources**: CPU, memory, disk usage monitoring
- **Live Logs**: Real-time log streaming with export functionality
- **Performance Metrics**: Block production, network I/O, system health

## 🔧 Development

### Project Structure
- **React 18**: Modern React with hooks and concurrent features
- **TypeScript**: Full type safety throughout the application
- **Tailwind CSS**: Utility-first styling with custom design system
- **React Router**: Client-side routing with protected routes
- **React Query**: Server state management with caching
- **Framer Motion**: Smooth animations and transitions
- **Zustand**: Lightweight state management

### Key Development Patterns

#### Data Fetching
All data fetching uses the DS pattern through custom hooks:
```typescript
const dsFetch = useDSFetcher();
const data = await dsFetch('admin.consensusInfo');
```

#### Form Handling
Forms are generated dynamically from manifest configuration:
```typescript
const { openAction } = useActionModal();
openAction('send'); // Opens send transaction form
```

#### Error Handling
Consistent error handling with user-friendly messages:
```typescript
const { data, error, isLoading } = useQuery({
  queryKey: ['nodeData'],
  queryFn: () => dsFetch('admin.consensusInfo'),
  retry: 2,
  retryDelay: 1000,
});
```

### Adding New Features

1. **Define Data Sources**: Add new DS endpoints in `chain.json`
2. **Create Hooks**: Build custom hooks for data fetching
3. **Build Components**: Create UI components using design system
4. **Add Actions**: Define new transaction types in `manifest.json`

### Environment Variables
```bash
VITE_DEFAULT_CHAIN=canopy
VITE_CONFIG_MODE=embedded
VITE_NODE_ENV=development
```

## 🛠️ Deployment

### Production Build
```bash
npm run build
```

### Configuration for Production
1. Update `chain.json` with production RPC endpoints
2. Configure proper CORS settings on your node
3. Set appropriate session timeouts and security parameters
4. Ensure SSL/TLS is configured for secure connections

### Docker Deployment
The wallet can be deployed alongside Canopy nodes:
```bash
# Build the application
npm run build

# Serve with any static file server
npx serve -s dist -p 3000
```

## 🔐 Security

### Key Management
- Private keys are encrypted with user passwords
- Keys stored locally in browser secure storage
- Session-based key unlocking with configurable timeouts

### Network Security
- All API calls over HTTPS in production
- CORS configuration required on node endpoints
- Session timeout and re-authentication for sensitive operations

### Best Practices
- Regular password changes recommended
- Backup recovery phrases securely
- Use hardware wallets for large amounts
- Verify transaction details before signing

## 📚 API Reference

### Core Hooks
- `useAccountData()`: Account balances and information
- `useNodeData()`: Node status and monitoring data
- `useValidators()`: Validator information and staking data
- `useTransactions()`: Transaction history and status

### DS Endpoints
All API endpoints are defined in `chain.json` under the `ds` section:
- **Query endpoints**: Height, accounts, validators, transactions
- **Admin endpoints**: Consensus info, peer info, logs, resources
- **Transaction endpoints**: Send, stake, governance operations

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes following the existing patterns
4. Add appropriate tests and documentation
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Code Style
- Use TypeScript for all new code
- Follow existing naming conventions
- Add JSDoc comments for complex functions
- Use the established DS pattern for API calls
- Maintain responsive design principles

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

For support and questions:
- Check the documentation in `/docs`
- Review existing issues on GitHub
- Join our community discussions
- Contact the development team

## 🗂️ Configuration Examples

### Basic Node Configuration
```json
{
  "chainId": "1",
  "displayName": "Canopy",
  "rpc": {
    "base": "http://localhost:50002",
    "admin": "http://localhost:50003"
  },
  "denom": {
    "base": "ucnpy",
    "symbol": "CNPY",
    "decimals": 6
  }
}
```

### Simple Transaction Action
```json
{
  "id": "send",
  "title": "Send",
  "icon": "Send",
  "form": {
    "fields": [
      {
        "id": "output",
        "name": "output",
        "type": "text",
        "label": "Recipient Address",
        "required": true
      }
    ]
  },
  "submit": {
    "base": "admin",
    "path": "/v1/admin/tx-send",
    "method": "POST"
  }
}
```


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


**Built with ❤️ for the Canopy ecosystem**

