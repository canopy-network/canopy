# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Canopy Wallet is a **config-first** React wallet application for the Canopy blockchain. Behavior is defined in configuration files (`chain.json`, `manifest.json`) and a runtime engine executes that configuration, avoiding hardcoded blockchain logic in UI components.

## Build & Development Commands

```bash
# Install dependencies (pnpm or npm)
npm install

# Start dev server (http://localhost:5173)
npm run dev

# Production build (runs prebuild config generation + tsc + vite)
npm run build

# Preview production build locally
npm run preview
```

**No test script exists.** Use `npm run build` as minimum validation. TypeScript is strict mode.

## Project Structure

```
src/
├── app/              # Routing, page shells, providers
│   ├── pages/        # Route components (Dashboard, Staking, Governance, etc.)
│   ├── providers/    # Context providers (Config, Accounts, ActionModal, Toast)
│   └── routes.tsx    # React Router configuration
├── actions/          # Dynamic action/form runtime
│   ├── ActionRunner.tsx    # Orchestrates DS fetch, form state, payload build, submit
│   ├── FormRenderer.tsx    # Renders fields from manifest config
│   ├── WizardRunner.tsx    # Multi-step wizard flows
│   └── fields/       # Field type implementations (text, amount, select, etc.)
├── components/       # Feature UI and reusable primitives
│   ├── ui/           # Primitives (buttons, inputs, badges)
│   └── layouts/      # Navbar, Sidebar, Footer
├── core/             # DS fetch logic, RPC helpers, templating
│   ├── dsCore.ts     # Data source resolution, request building, response parsing
│   ├── templater.ts  # Template resolution ({{ ds.key.value }} syntax)
│   └── useDs.ts      # React Query wrapper for DS fetching
├── hooks/            # Data hooks (useAccounts, useValidators, useTransactions, etc.)
├── manifest/         # Manifest loaders and types
├── state/            # Zustand session state
└── toast/            # Toast notification system

public/plugin/canopy/
├── chain.json        # Network config: RPC URLs, denom, DS definitions
├── chain.json.template  # Template for env-based URL injection
└── manifest.json     # Action definitions: forms, fields, payloads, submit endpoints

scripts/
└── generate-chain-config.js  # Pre-build script: injects RPC URLs from env vars
```

## Architecture: Config-First Model

### Core Concept
- **`chain.json`**: Integration contract with blockchain (RPC endpoints, denomination, data sources)
- **`manifest.json`**: Interaction contract (actions, form fields, payload mapping, submit config)

### Data Sources (DS)
DS definitions in `chain.json` are referenced declaratively in actions:
- Template syntax: `{{ ds.account.amount }}`, `{{ form.recipient }}`, `{{ account.address }}`
- Coercion: Automatic type conversion (string → number, micro/base denom)
- Pagination: Built-in page/cursor strategies

### Action Runtime Flow
1. User triggers action → `ActionModalProvider` opens modal
2. `ActionRunner` loads action definition from `manifest.json`
3. DS dependencies fetched via `dsCore.ts`
4. `FormRenderer` renders fields dynamically from config
5. Values validated/coerced per manifest rules
6. Payload built declaratively, submitted to configured endpoint
7. Success/error handled from action config

### Key Runtime Files
- `src/app/providers/ActionModalProvider.tsx` - Modal host, action switching
- `src/actions/ActionRunner.tsx` - Form state, DS orchestration, submit
- `src/core/dsCore.ts` - DS resolution, request/response handling
- `src/core/templater.ts` - `{{ }}` template resolution

## Development Guidelines

### Adding New Actions
1. Define action in `public/plugin/canopy/manifest.json`
2. Add required DS endpoints to `public/plugin/canopy/chain.json`
3. Use existing field types: `text`, `amount`, `select`, `advancedSelect`, `tableSelect`, `switch`, `optionCard`, `dynamicHtml`, `section`, `divider`, `heading`
4. Keep payload transformations explicit with `coerce` config
5. Never hardcode RPC endpoints in UI components

### Field Types
Manifest-driven field rendering supports: `text`, `textarea`, `amount`, `number`, `address`, `select`, `advancedSelect`, `switch`, `option`, `optionCard`, `tableSelect`, `dynamicHtml`, `section`, `divider`, `spacer`, `heading`, `collapsibleGroup`

### Import Alias
Use `@/` for imports rooted at `src/`:
```typescript
import { useConfig } from '@/app/providers/ConfigProvider'
```

### Styling
- Tailwind CSS with custom Canopy palette (canopy green: `#35CD48`)
- Design tokens in `tailwind.config.js`
- Dark theme only (`darkMode: ['class']`)

### Environment Variables
```bash
VITE_WALLET_RPC_PROXY_TARGET     # RPC base URL (default: http://localhost:50002)
VITE_WALLET_ADMIN_RPC_PROXY_TARGET  # Admin RPC URL (default: http://localhost:50003)
VITE_ROOT_WALLET_RPC_PROXY_TARGET   # Root chain RPC (default: same as RPC)
VITE_WALLET_BASE_PATH            # Base path for routing (default: / in dev, /wallet/ in prod)
```

## Coding Conventions

- TypeScript strict mode required
- Components: `PascalCase.tsx`
- Hooks: `useX.ts`
- Utilities: `camelCase.ts`
- Prefer function components and hooks
- Use `@/` alias for src imports
- Follow existing formatting in the file you touch

## Commit Style

Conventional Commits: `feat:`, `fix:`, `chore:`

## Key Dependencies

- React 18 + React Router 7
- Vite 5 + TypeScript 5
- TanStack React Query
- Radix UI primitives
- Tailwind CSS + class-variance-authority
- Framer Motion
- Zustand (state management)
- viem (blockchain utilities)
