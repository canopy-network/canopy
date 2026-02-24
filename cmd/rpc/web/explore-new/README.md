# Explore New

A modern React application built with Vite, TypeScript, Tailwind CSS, React Hook Form, Framer Motion, and React Query for efficient data fetching and state management.

## Features

- ⚡ **Vite** - Fast build tool and dev server
- ⚛️ **React 18** - Latest React features
- 🔷 **TypeScript** - Type safety and better developer experience
- 🎨 **Tailwind CSS** - Utility-first CSS framework
- 📝 **React Hook Form** - Performant forms with easy validation
- ✨ **Framer Motion** - Production-ready motion library for React
- 🔄 **React Query** - Powerful data fetching and caching library

## Getting Started

### Prerequisites

- Node.js (version 18 or higher)
- npm or yarn
- Canopy blockchain node running on port 50001

### Installation

1. Clone the repository and navigate to the project directory:
```bash
cd cmd/rpc/web/explore-new
```

2. Install dependencies:
```bash
npm install
```

3. Ensure your Canopy blockchain node is running on port 50001:
```bash
# Your Canopy node should be accessible at:
# http://localhost:50001
```

4. Start the development server:
```bash
npm run dev
```

5. Open your browser and navigate to `http://localhost:5173`

### Quick Setup

The application will automatically connect to your Canopy node at `http://localhost:50001`. If your node is running on a different port, you can configure it by setting `window.__CONFIG__` in your HTML or modifying the API configuration.

### Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint
- `npm run type-check` - Run TypeScript type checking

## Project Structure

```
src/
├── components/           # Reusable components
│   ├── analytics/       # Analytics dashboard components
│   │   ├── AnalyticsFilters.tsx
│   │   ├── BlockProductionRate.tsx
│   │   ├── FeeTrends.tsx
│   │   ├── KeyMetrics.tsx
│   │   ├── NetworkActivity.tsx
│   │   ├── NetworkAnalyticsPage.tsx
│   │   ├── StakingTrends.tsx
│   │   ├── TransactionTypes.tsx
│   │   └── ValidatorWeights.tsx
│   ├── block/          # Block-related components
│   │   ├── BlockTransactions.tsx
│   │   ├── BlocksFilters.tsx
│   │   ├── BlocksPage.tsx
│   │   └── BlocksTable.tsx
│   ├── Home/           # Home page components
│   │   ├── ExtraTables.tsx
│   │   ├── HomePage.tsx
│   │   └── TableCard.tsx
│   ├── transaction/    # Transaction components
│   │   ├── TransactionsPage.tsx
│   │   └── TransactionsTable.tsx
│   ├── validator/      # Validator components
│   │   ├── ValidatorsFilters.tsx
│   │   ├── ValidatorsPage.tsx
│   │   └── ValidatorsTable.tsx
│   ├── token-swaps/    # Token swap components
│   │   ├── RecentSwapsTable.tsx
│   │   ├── SwapFilters.tsx
│   │   └── TokenSwapsPage.tsx
│   ├── common/         # Shared UI components
│   │   ├── Footer.tsx
│   │   ├── Logo.tsx
│   │   └── Navbar.tsx
│   └── ui/            # Basic UI components
│       ├── AnimatedNumber.tsx
│       ├── LoadingSpinner.tsx
│       └── SearchInput.tsx
├── hooks/             # Custom React hooks
│   ├── useApi.ts      # React Query hooks for API calls
│   └── useSearch.ts   # Search functionality hook
├── lib/               # API functions and utilities
│   └── api.ts         # All API endpoint functions
├── types/             # TypeScript type definitions
│   ├── api.ts         # API response types
│   └── common.ts      # Common type definitions
├── data/              # Static data and configurations
│   ├── blocks.json    # Block-related text content
│   ├── navbar.json    # Navigation menu configuration
│   └── transactions.json # Transaction-related text content
├── App.tsx            # Main application component
├── main.tsx           # Application entry point
└── index.css          # Global styles with Tailwind
```

### Component Mapping

| Component | Purpose | Location |
|-----------|---------|----------|
| **Analytics** | Dashboard with network metrics and charts | `/analytics` |
| **Blocks** | Block explorer with filtering and pagination | `/blocks` |
| **Transactions** | Transaction history and details | `/transactions` |
| **Validators** | Validator information and ranking | `/validators` |
| **Token Swaps** | Token swap orders and trading | `/token-swaps` |
| **Home** | Main dashboard with overview tables | `/` |

## API Integration

This project includes a complete API integration system with React Query:

### API Functions (`src/lib/api.ts`)
- All backend API calls from the original explorer project
- TypeScript support for better type safety
- Error handling and response processing

### React Query Hooks (`src/hooks/useApi.ts`)
- Custom hooks for each API endpoint
- Automatic caching and background updates
- Loading and error states
- Optimistic updates support

### Available Hooks
- `useBlocks(page)` - Fetch blocks data
- `useTransactions(page, height)` - Fetch transactions
- `useAccounts(page)` - Fetch accounts
- `useValidators(page)` - Fetch validators
- `useCommittee(page, chainId)` - Fetch committee data
- `useDAO(height)` - Fetch DAO data
- `useAccount(height, address)` - Fetch account details
- `useParams(height)` - Fetch parameters
- `useSupply(height)` - Fetch supply data
- `useCardData()` - Fetch dashboard card data
- `useTableData(page, category, committee)` - Fetch table data
- And many more...

### Usage Example
```typescript
import { useBlocks, useValidators } from './hooks/useApi'

function MyComponent() {
  const { data: blocks, isLoading, error } = useBlocks(1)
  const { data: validators } = useValidators(1)

  if (isLoading) return <div>Loading...</div>
  if (error) return <div>Error: {error.message}</div>

  return (
    <div>
      <h2>Blocks: {blocks?.totalCount}</h2>
      <h2>Validators: {validators?.totalCount}</h2>
    </div>
  )
}
```

## Technologies Used

- **Vite** - Build tool and dev server
- **React** - UI library
- **TypeScript** - Type safety
- **Tailwind CSS** - Styling
- **React Hook Form** - Form handling
- **Framer Motion** - Animations
- **React Query** - Data fetching and caching

## Development

This project uses:
- ESLint for code linting
- Prettier for code formatting
- TypeScript for type checking
- React Query DevTools for debugging queries

## API Configuration

The application automatically configures API endpoints based on the environment:
- Default RPC URL: `http://localhost:50002`
- Default Admin RPC URL: `http://localhost:50002`
- Default Chain ID: `1`

You can override these settings by setting `window.__CONFIG__` in your HTML.

## License

MIT
