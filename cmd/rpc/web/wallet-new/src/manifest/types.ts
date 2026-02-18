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
  title?: string; // opcional si usas label
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

  // Paso de confirmación (opcional y simple)
  confirm?: {
    title?: string;
    summary?: Array<{ label: string; value: string }>;
    ctaLabel?: string;
    danger?: boolean;
    showPayload?: boolean;
    payloadSource?: "rpc.payload" | "custom";
    payloadTemplate?: any; // si usas plantilla custom de confirmación
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

  // Envío (tx o llamada)
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
  source?: { uses: string; selector?: string }; // p.ej. {uses:'ds', selector:'committees'}
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
  | { id: string; op: "copy"; from: string } // copia al clipboard el valor resuelto
  | { id: string; op: "paste" } // pega desde clipboard al field
  | { id: string; op: "set"; field: string; value: string }; // setea un valor (p.ej. Max)

/* ===========================
 * UI Ops / Events
 * =========================== */

export type UIOp =
  | { op: "fetch"; source: SourceKey } // dispara un refetch/carga de DS al abrir
  | { op: "notify"; message: string }; // opcional: mostrar toast/notificación

/* ===========================
 * Submit (HTTP)
 * =========================== */

export type Submit = {
  base: "rpc" | "admin";
  path: string; // p.ej. '/v1/admin/tx-send'
  method?: "GET" | "POST";
  headers?: Record<string, string>;
  encoding?: "json" | "text";
  body?: any; // plantilla a resolver o valor literal
};

/* ===========================
 * Sources y Selectors
 * =========================== */

export type SourceRef = {
  // de dónde sale el dato que vas a interpolar
  uses: string;
  // ruta dentro de la fuente (p.ej. 'fee.sendFee', 'amount', 'address')
  selector?: string;
};

// claves comunes de tu DS actual; permite string libre para crecer sin tocar tipos
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
 * Fees (opcional, lo mínimo)
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
  selector?: string; // p.ej. 'fee' dentro del response
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
 *   ds: Record<string, any> // p.ej. ds.account.amount
 * }
 */
