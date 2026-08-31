# Repository Guidelines

## Project Structure & Module Organization
- `src/` contains application code.
- `src/app/` holds routing, page shells, and providers.
- `src/components/` contains feature UI and reusable primitives (`src/components/ui`).
- `src/actions/` implements the dynamic action/form runtime.
- `src/core/` and `src/hooks/` contain DS fetch logic, RPC helpers, and data hooks.
- `src/manifest/` contains manifest loaders and types.
- `public/plugin/canopy/` is the config surface (`chain.json`, `manifest.json`, `chain.json.template`).
- `scripts/generate-chain-config.js` is executed before production builds.
- `docs/declarative-actions-and-chain.md` documents the config-first architecture.
- Build artifacts are emitted to `out/`.

## Build, Test, and Development Commands
- `npm install` installs dependencies (a `pnpm-lock.yaml` is also present if you use pnpm).
- `npm run dev` starts the Vite dev server on `http://localhost:5173`.
- `npm run build` runs prebuild config generation, TypeScript project checks, and Vite production build.
- `npm run preview` serves the production bundle locally.
- No `npm test` script exists in this package yet; use `npm run build` as the minimum validation step.

## Coding Style & Naming Conventions
- TypeScript is strict (`"strict": true`); keep all new code fully typed.
- Prefer React function components and hooks.
- Naming conventions:
  - Components: `PascalCase.tsx` (example: `ValidatorList.tsx`)
  - Hooks: `useX.ts`/`useX.tsx` (example: `useValidators.ts`)
  - Utilities: `camelCase.ts`
- Use the `@/` alias for imports rooted at `src`.
- Preserve the config-first model: new chain actions/endpoints should be defined in `public/plugin/canopy/*.json`, not hardcoded in UI flows.
- Follow local formatting in the file you touch (quote style is mixed across the repo).

## Testing Guidelines
- Automated tests are currently not configured in this workspace.
- Before opening a PR, run `npm run build` and manually verify affected flows in `npm run dev`.
- If you add tests, colocate them as `*.test.ts`/`*.test.tsx` near the implementation and prioritize action runtime, DS fetching, and critical page paths.

## Commit & Pull Request Guidelines
- Follow Conventional Commit style used in history: `feat:`, `fix:`, `chore:`.
- Keep commits small and focused; avoid mixing unrelated refactors with feature work.
- PRs should include:
  - a concise summary and rationale,
  - linked issue/task,
  - screenshots or short recordings for UI changes,
  - notes on `chain.json`/`manifest.json` compatibility impacts.
