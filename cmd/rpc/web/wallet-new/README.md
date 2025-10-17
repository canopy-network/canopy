# 🧩 CNPY Wallet — Config-First Manifest System

This document explains how to create and maintain **action manifests** for the Config-First wallet.  
The system allows new blockchain transactions and UI flows to be defined through **JSON files**, without modifying application code.

---

## 📁 Overview

Each chain defines:
- `chain.json` → RPC configuration, fee buckets, and session parameters.  
- `manifest.json` → List of **actions** (transaction templates) to render dynamically in the wallet.

At runtime:
- The wallet loads the manifest and generates dynamic forms.
- Context objects (`ctx`) provide access to chain, account, DS (data sources), session, and fee data.
- Payloads are resolved from templates and sent to the defined RPC endpoints.

---

## ⚙️ Manifest Structure

Each action entry follows this schema:

```jsonc
{
  "id": "send",
  "title": "Send",
  "icon": "Send",
  "ui": { "variant": "modal" },
  "tags": ["quick"],
  "form": { ... },
  "events": { ... },
  "payload": { ... },
  "submit": { "base": "admin", "path": "/v1/admin/tx-send", "method": "POST" }
}
```

### Top-Level Fields

| Key | Type | Description |
|-----|------|-------------|
| `id` | string | Unique identifier of the action. |
| `title` | string | Display name in UI. |
| `icon` | string | Lucide icon name. |
| `tags` | string[] | Used for grouping (“quick”, “dashboard”, etc.). |
| `ui` | object | UI behavior (e.g., modal or drawer). |
| `slots.modal.className` | string | Tailwind class to style the modal container. |

---

## 🧠 Dynamic Form Definition

Each field inside `form.fields` is declarative and can include bindings, data source fetches, and UI helpers.

Example:

```json
{
  "id": "amount",
  "name": "amount",
  "type": "amount",
  "label": "Amount",
  "min": 0,
  "features": [
    {
      "id": "maxBtn",
      "op": "set",
      "field": "amount",
      "value": "{{ds.account.amount}} - {{fees.effective}}"
    }
  ]
}
```

### Supported Field Types
- `text`, `number`, `amount`, `address`, `select`, `textarea`.

### Features
Declarative interactions:
- `"op": "copy"` → copies value to clipboard.  
- `"op": "paste"` → pastes clipboard value.  
- `"op": "set"` → programmatically sets another field’s value.  

---

## 🔄 Data Source (`ds`) Integration

Each field can declare a `ds` block to automatically populate its value:

```json
"ds": {
  "account": {
    "account": { "address": "{{account.address}}" }
  }
}
```

When declared, the field’s value will update once the data source returns results.

---

## 🧩 Payload Construction

Payloads define how data is sent to the backend RPC endpoint.  
They support templating (`{{...}}`) and coercion (`string`, `number`, `boolean`).

```json
"payload": {
  "address": { "value": "{{account.address}}", "coerce": "string" },
  "output": { "value": "{{form.output}}", "coerce": "string" },
  "amount": { "value": "{{toUcnpy<{{form.amount}}>}}", "coerce": "number" },
  "fee": { "value": "{{fees.raw.sendFee}}", "coerce": "number" },
  "password": { "value": "{{session.password}}", "coerce": "string" }
}
```

### Supported Coercions
- `"string"` → converts to string.  
- `"number"` → parses and converts to number.  
- `"boolean"` → interprets `"true"`, `"1"`, etc. as `true`.

---

## 🧮 Templating Engine

Templates use double braces and can call functions:

```txt
{{chain.displayName}} (Balance: formatToCoin<{{ds.account.amount}}>)
```

Functions like `formatToCoin` or `toUcnpy` are defined in `templaterFunctions.ts`.  
Nested evaluation is supported.

---

## ⚡ Custom Template Functions

Example definitions (`templaterFunctions.ts`):

```ts
export const templateFns = {
  formatToCoin: (v: any) => (Number(v) / 1_000_000).toFixed(2),
  toUcnpy: (v: any) => Math.round(Number(v) * 1_000_000),
}
```

They can be used anywhere in the manifest, in field values or payloads.

---

## 🧩 Context Available in Templates

When rendering or submitting, the wallet provides:

| Key | Description |
|-----|--------------|
| `chain` | Chain configuration from `chain.json`. |
| `account` | Selected account (`address`, `nickname`, `publicKey`). |
| `form` | Current form state. |
| `session` | Current session data (e.g., password). |
| `fees` | Fetched fee parameters (`raw`, `amount`, `denom`). |
| `ds` | Results from registered data sources. |

---

## 🧾 Example Action Manifest

```json
{
  "id": "send",
  "title": "Send",
  "icon": "Send",
  "ui": { "variant": "modal" },
  "tags": ["quick"],
  "form": {
    "fields": [
      {
        "id": "address",
        "name": "address",
        "type": "text",
        "label": "From Address",
        "value": "{{account.address}}",
        "readOnly": true
      },
      {
        "id": "output",
        "name": "output",
        "type": "text",
        "label": "To Address",
        "required": true,
        "features": [{ "id": "pasteBtn", "op": "paste" }]
      },
      {
        "id": "asset",
        "name": "asset",
        "type": "text",
        "label": "Asset",
        "value": "{{chain.displayName}} (Balance: formatToCoin<{{ds.account.amount}}>)",
        "readOnly": true,
        "ds": {
          "account": {
            "account": { "address": "{{account.address}}" }
          }
        }
      }
    ]
  },
  "payload": {
    "address": { "value": "{{account.address}}", "coerce": "string" },
    "output": { "value": "{{form.output}}", "coerce": "string" },
    "amount": { "value": "{{toUcnpy<{{form.amount}}>}}", "coerce": "number" },
    "fee": { "value": "{{fees.raw.sendFee}}", "coerce": "number" },
    "submit": { "value": true, "coerce": "boolean" },
    "password": { "value": "{{session.password}}", "coerce": "string" }
  },
  "submit": {
    "base": "admin",
    "path": "/v1/admin/tx-send",
    "method": "POST"
  }
}
```

---

## 🧭 Guidelines

✅ **DO**
- Keep `manifest.json` declarative — no inline JS logic.  
- Use `{{ }}` placeholders with clear paths.  
- Prefer template functions (`formatToCoin`, `toUcnpy`, etc.) for conversions.  
- Reuse fee selectors and buckets from `chain.json`.

🚫 **DON’T**
- Hardcode user or chain-specific values.  
- Access unregistered DS keys — always declare them.  
- Mix UI logic (like validation messages) into payloads.

---

## 🧪 Debugging Tips

- Enable `console.log(resolved)` in `buildPayloadFromAction()` to inspect final payload values.  
- Check the rendered form fields to confirm DS bindings populate correctly.  
- When debugging template parsing, log `template(str, ctx)` output before submission.
