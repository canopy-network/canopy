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

### Installation

1. Install dependencies:
```bash
npm install
```

2. Start the development server:
```bash
npm run dev
```

3. Open your browser and navigate to `http://localhost:5173`

### Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint
- `npm run type-check` - Run TypeScript type checking

## Project Structure

```
src/
├── components/     # Reusable components
├── hooks/         # Custom React hooks (including React Query hooks)
├── lib/           # API functions and utilities
├── types/         # TypeScript type definitions
├── utils/         # Utility functions
├── App.tsx        # Main application component
├── main.tsx       # Application entry point
└── index.css      # Global styles with Tailwind
```

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
- Default Admin RPC URL: `http://localhost:50003`
- Default Chain ID: `1`

You can override these settings by setting `window.__CONFIG__` in your HTML.

## License

MIT
