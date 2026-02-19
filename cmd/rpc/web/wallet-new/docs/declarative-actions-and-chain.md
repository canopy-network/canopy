# Declarative Actions and Chain Configuration

This wallet follows a **config-first architecture**: behavior is defined in configuration files, and the runtime executes that configuration.

The two core files are:

- `public/plugin/canopy/chain.json`
- `public/plugin/canopy/manifest.json`

## Why This Exists

The goal is to avoid hardcoded blockchain behavior in UI components.

With this model, you can:

- Change RPC endpoints without rewriting app code.
- Add or modify wallet actions (send, stake, governance, etc.) without creating new React forms manually.
- Keep business rules in a single, auditable configuration layer.
- Reuse the same runtime engine across different chains or environments.

## `chain.json`: Network and Data Contract

`chain.json` defines chain-level and API-level behavior:

- RPC base URLs (`rpc`, `admin`)
- denomination metadata (`base`, `symbol`, `decimals`)
- explorers and links
- data sources (`ds`) used by hooks and dynamic forms
- fee providers and session settings

Think of `chain.json` as the **integration contract** with the blockchain.

## `manifest.json`: Declarative UI + Action Contract

`manifest.json` defines what the app can do and how forms behave:

- `actions[]`: each user operation (send, stake, edit stake, governance vote, etc.)
- dynamic fields, validation, and conditional visibility
- wizard steps and confirmation summaries
- payload mapping and coercion
- submit endpoint (`base`, `path`, `method`)
- success/error notifications

Think of `manifest.json` as the **interaction contract** between user intent and transaction payloads.

## End-to-End Runtime Flow

1. User opens an action.
2. Runtime loads action definition from `manifest.json`.
3. Runtime resolves DS dependencies from `chain.json` (`ds.*` calls).
4. Form is rendered dynamically from field config.
5. Values are validated/coerced according to manifest rules.
6. Payload is built declaratively.
7. Request is submitted to the configured endpoint.
8. Notifications and UI state are resolved from action config.

## Runtime Layers in Practice

In this wallet, declarative actions are executed by a runtime pipeline:

- `ActionModalProvider` / `ActionsModal`: opens and hosts action UI.
- `ActionRunner`: orchestrates DS fetch, form state, payload build, submit, and notifications.
- `FormRenderer` + field registry: resolves and renders each field type.
- `usePopulateController`: controls initialization/autopopulate phases and DS readiness.

This separation is intentional:

- UI shell remains generic.
- business behavior lives in config.
- dynamic data and validation are resolved centrally.

## Example: Stake / Edit Stake

The `stake` action can route to different endpoints based on current state:

- New validator -> `/v1/admin/tx-stake`
- Existing validator -> `/v1/admin/tx-edit-stake`

This is defined in manifest submit config, not in component code.

The same action can also adapt:

- field requirements
- labels/help text
- payload normalization

based on whether `ds.validator` exists.

## Field Types (Manifest-Driven)

Field rendering is mapped through a field registry. Current types include:

- `text`, `textarea`
- `amount`, `number`, `address`
- `select`, `advancedSelect`
- `switch`, `option`, `optionCard`
- `tableSelect`
- `dynamicHtml`
- layout/structure: `section`, `divider`, `spacer`, `heading`, `collapsibleGroup`

Why this matters:

- new flows can be created by reusing existing field types in manifest.
- runtime does not need action-specific React forms.
- unsupported types fail safely and are visible during integration.

## Modal Data Population and Prefill Model

When opening a dynamic action modal, data can come from multiple sources:

1. action defaults (`field.value`)
2. DS results (`ds.*`)
3. runtime/session context (`account`, `chain`, `fees`, `session`)
4. explicit `prefilledData` passed when opening the action (for edit flows)

### Population phases

Population is not a single step. It follows phases:

- `waiting`: critical DS is not ready, form can show loading/skeleton behavior.
- `initializing`: first safe population pass from templates/defaults.
- `ready`: form becomes interactive; only explicit `autoPopulate: "always"` keeps updating from DS.

### Precedence and safety

- `prefilledData` is treated as authoritative for those fields during init.
- default templates do not overwrite prefilled fields.
- after initialization, DS updates do not blindly overwrite user input unless field explicitly opts in with `autoPopulate: "always"`.

This avoids race conditions and accidental input loss when DS refreshes.

## Responsive Behavior (Declarative + Runtime)

Responsive form layout is controlled through field span config and runtime helpers.

### Grid model

- Forms render in a 12-column grid.
- Each field can define `span` (or `ui.grid.colSpan`) with breakpoints:
  - `base`, `sm`, `md`, `lg`, `xl`

Example:

```json
{
  "id": "fee",
  "name": "fee",
  "type": "amount",
  "span": { "base": 12, "md": 6 }
}
```

Runtime behavior:

- mobile-first: defaults to full width on small screens.
- breakpoints progressively apply wider multi-column layouts.
- tabs and modal content regions are scroll-safe to prevent overflow.

### Modal responsiveness

Dynamic action modals are designed to:

- keep viewport constraints (`dvh`-based max height).
- maintain internal scrolling in content region.
- preserve close controls and headers.
- avoid sidebar/topbar overlap through controlled overlay stacking.

## Data Source (DS) Interaction Pattern

DS definitions live in `chain.json`. Action DS blocks reference them declaratively.

Typical usage:

- action declares DS dependencies (`account`, `validator`, `keystore`, etc.).
- DS options control staleness/refetch behavior (`staleTimeMs`, `refetchIntervalMs`, `watch`, `critical`).
- templates consume DS via `{{ ds.someKey.someValue }}`.

This provides consistent data-fetch semantics for all actions.

## Template and Coercion Model

Templates are resolved at runtime using context objects:

- `form`
- `ds`
- `account`
- `chain`
- `fees`
- `params`
- `session`

Payload fields can define coercion:

- `string`
- `number`
- `boolean`
- raw objects where needed

For blockchain payloads, this is critical for safe numeric conversions
(e.g., display denom -> micro denom) and type correctness.

## Design Rules for New Actions

When adding a new action, keep these rules:

- Put business behavior in manifest, not React components.
- Keep payload transformations explicit (`coerce`, conversion helpers, guards).
- Fetch external state via DS, never inline endpoint strings in UI components.
- Prefer conditional config (`showIf`, dynamic `required`, templated values) over branching UI code.
- Keep confirmation summaries and notifications in config so users can review intent before submit.

Additional rules for dynamic modals/forms:

- Use `prefilledData` for edit workflows instead of hardcoding edit components.
- Use `critical` DS keys when form correctness depends on external data.
- Use `autoPopulate: "once"` for default suggestions and `"always"` only when continuous sync is required.
- Keep long help text and labels concise to preserve mobile readability.

## Benefits and Tradeoffs

Benefits:

- Faster iteration for product/business rules.
- Lower risk of UI/backend drift.
- Easier portability to new chains.

Tradeoffs:

- More responsibility on config quality.
- Requires strict validation and review of manifest changes.
- Debugging often means inspecting resolved templates and DS outputs.

## Recommended Review Checklist

Before shipping manifest or chain changes:

- Validate JSON syntax.
- Verify DS endpoints, methods, and selectors.
- Confirm numeric conversions (micro/base denom) are correct.
- Confirm `submit.path` and `payload` align with RPC contract.
- Test both success and failure toasts/messages.
- Test responsive behavior for long labels/help/validation messages.
