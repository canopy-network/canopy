/* ===========================
 * Manifest & UI Core Types
 * =========================== */

import React from "react";

export type Manifest = {
  version: string;
  ui?: {
    quickActions?: { max?: number };
    tx: {
      typeMap: Record<string, string>;
      typeIconMap: Record<string, string>;
      fundsWay: Record<string, "in" | "out">;
    };
  };
  actions: Action[];
};

export type PayloadValue =
  | string
  | {
      value: string;
      coerce?: "string" | "number" | "boolean";
    };

export type Action = {
  id: string;
  title?: string; // optional if using label
  icon?: string;
  kind: "tx" | "view" | "utility";
  tags?: string[];
  relatedActions?: string[];
  priority?: number;
  order?: number;
  requiresFeature?: string;
  hidden?: boolean;

  ui?: {
    variant?: "modal" | "page";
    icon?: string;
    slots?: { modal?: { style: React.CSSProperties; className?: string } };
  };

  // Wizard steps support
  steps?: Array<{
    title?: string;
    form?: {
      fields: Field[];
      layout?: {
        grid?: { cols?: number; gap?: number };
        aside?: { show?: boolean; width?: number };
      };
    };
    aside?: {
      widget?: string;
    };
  }>;

  // dynamic form
  form?: {
    fields: Field[];
    layout?: {
      grid?: { cols?: number; gap?: number };
      aside?: { show?: boolean; width?: number };
    };
    info?: {
      title: string;
      items: { label: string; value: string; icons: string }[];
    };
    summary?: {
      title: string;
      items: { label: string; value: string; icons: string }[];
    };
    confirmation: {
      btn: {
        icon: string;
        label: string;
      };
    };
  };
  payload?: Record<string, PayloadValue>;

  // RPC configuration
  rpc?: {
    base: "rpc" | "admin";
    path: string;
    method: string;
    payload?: any;
  };

  // Confirmation step (optional and simple)
  confirm?: {
    title?: string;
    summary?: Array<{ label: string; value: string }>;
    ctaLabel?: string;
    danger?: boolean;
    showPayload?: boolean;
    payloadSource?: "rpc.payload" | "custom";
    payloadTemplate?: any; // if using custom confirmation template
  };

  // Success configuration
  success?: {
    message?: string;
    links?: Array<{
      label: string;
      href: string;
    }>;
  };

  auth?: { type: "sessionPassword" | "none" };

  // Submit (tx or call)
  submit?: Submit;
};

/* ===========================
 * Fields
 * =========================== */

export type FieldBase = {
  id: string;
  name: string;
  label?: string;
  help?: string;
  placeholder?: string;
  readOnly?: boolean;
  required?: boolean;
  disabled?: boolean;
  value?: string;
  // features: copy / paste / set (Max)
  features?: FieldOp[];
  ds?: Record<string, any>;
};

export type AddressField = FieldBase & {
  type: "address";
};

export type AmountField = FieldBase & {
  type: "amount";
  min?: number;
  max?: number;
};

export type NumberField = FieldBase & {
  type: "number";
  min?: number;
  max?: number;
  step?: number | "any";
  integer?: boolean;
};

export type RangeField = FieldBase & {
  type: "range";
  min?: number;
  max?: number;
  step?: number;
  showInput?: boolean;
  suffix?: string;
  marks?: number[];
  presets?: Array<{ label: string; value: number }>;
};

export type TextField = FieldBase & {
  type: "text" | "textarea";
};

export type SwitchField = FieldBase & {
  type: "switch";
};

export type OptionCardField = FieldBase & {
  type: "optionCard";
};

export type DynamicHtml = FieldBase & {
  type: "dynamicHtml";
  html: string;
};

export type OptionField = FieldBase & {
  type: "option";
  inLine?: boolean;
};

export type TableSelectColumn = {
  key: string;
  title: string;
  expr?: string;
  position?: "right" | "left" | "center";
};

export type TableRowAction = {
  title?: string;
  label?: string;
  icon?: string;
  showIf?: string;
  emit?: { op: "set" | "copy"; field?: string; value?: string };
  position?: "right" | "left" | "center";
};

export type TableSelectField = FieldBase & {
  type: "tableSelect";
  id: string;
  name: string;
  label?: string;
  help?: string;
  required?: boolean;
  readOnly?: boolean;
  multiple?: boolean;
  rowKey?: string;
  columns: TableSelectColumn[];
  rows?: any[];
  source?: { uses: string; selector?: string }; // e.g. {uses:'ds', selector:'committees'}
  rowAction?: TableRowAction;
};

export type SelectField = FieldBase & {
  type: "select";
  // Could be a json string or a list of options
  options?: String | Array<{ label: string; value: string }>;
};

export type AdvancedSelectField = FieldBase & {
  type: "advancedSelect";
  allowCreate?: boolean;
  allowFreeInput?: boolean;
  options?: Array<{ label: string; value: string }>;
};

export type Field =
  | AddressField
  | AmountField
  | NumberField
  | RangeField
  | SwitchField
  | OptionCardField
  | OptionField
  | TextField
  | SelectField
  | TableSelectField
  | AdvancedSelectField
  | DynamicHtml;

/* ===========================
 * Field Features (Ops)
 * =========================== */

export type FieldOp =
  | { id: string; op: "copy"; from: string } // copies the resolved value to clipboard
  | { id: string; op: "paste" } // pastes from clipboard to field
  | { id: string; op: "set"; field: string; value: string }; // sets a value (e.g. Max)

/* ===========================
 * UI Ops / Events
 * =========================== */

export type UIOp =
  | { op: "fetch"; source: SourceKey } // triggers a refetch/load of DS on open
  | { op: "notify"; message: string }; // optional: show toast/notification

/* ===========================
 * Submit (HTTP)
 * =========================== */

export type Submit = {
  base: "rpc" | "admin";
  path: string; // e.g. '/v1/admin/tx-send'
  method?: "GET" | "POST";
  headers?: Record<string, string>;
  encoding?: "json" | "text";
  body?: any; // template to resolve or literal value
};

/* ===========================
 * Sources and Selectors
 * =========================== */

export type SourceRef = {
  // where the data to interpolate comes from
  uses: string;
  // path within the source (e.g. 'fee.sendFee', 'amount', 'address')
  selector?: string;
};

// common keys of your current DS; allows free string to grow without touching types
export type SourceKey =
  | "account"
  | "params"
  | "fees"
  | "height"
  | "validators"
  | "activity"
  | "txs.sent"
  | "txs.received"
  | "gov.proposals"
  | string;

/* ===========================
 * Fees (optional, the minimum)
 * =========================== */

export type FeeBuckets = {
  [bucket: string]: { multiplier: number; default?: boolean };
};

export type FeeProviderQuery = {
  type: "query";
  base: "rpc" | "admin";
  path: string;
  method?: "GET" | "POST";
  headers?: Record<string, string>;
  encoding?: "json" | "text";
  selector?: string; // e.g. 'fee' within the response
  cache?: { staleTimeMs?: number; refetchIntervalMs?: number };
};

export type FeeProviderSimulate = {
  type: "simulate";
  base: "rpc" | "admin";
  path: string;
  method?: "GET" | "POST";
  headers?: Record<string, string>;
  encoding?: "json" | "text";
  body?: any;
  gasAdjustment?: number;
  gasPrice?:
    | { type: "static"; value: string }
    | {
        type: "query";
        base: "rpc" | "admin";
        path: string;
        selector?: string;
      };
};

export type FeeProvider = FeeProviderQuery | FeeProviderSimulate;

/* ===========================
 * Templater Context (doc)
 * ===========================
 * Tu resolvedor debe recibir, al menos, este shape:
 * {
 *   chain: { displayName: string; fees?: any; ... },
 *   form: Record<string, any>,
 *   session: { password?: string; ... },
 *   fees: { effective?: string|number; amount?: string|number },
 *   account: { address: string; nickname?: string },
 *   ds: Record<string, any> // e.g. ds.account.amount
 * }
 */
