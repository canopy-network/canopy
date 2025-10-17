/* ===========================
 * Manifest & UI Core Types
 * =========================== */

export type Manifest = {
    version: string;
    ui?: {
        quickActions?: { max?: number }
        tx:{
            typeMap: Record<string, string>;
            typeIconMap: Record<string, string>;
        }
    };
    actions: Action[];
};

export type PayloadValue =
    | string
    | {
    value: string
    coerce?: 'string' | 'number' | 'boolean'
}

export type Action = {
    id: string;
    title?: string;      // opcional si usas label
    label?: string;
    icon?: string;
    kind: 'tx' | 'view' | 'utility';
    tags?: string[];
    relatedActions?: string[];
    priority?: number;
    order?: number;
    requiresFeature?: string;
    hidden?: boolean;

    // Apariencia básica (modal/página)
    ui?: { variant?: 'modal' | 'page'; icon?: string };

    // Slots simples (p.ej. estilos del modal)
    slots?: { modal?: { className?: string } };

    // Form dinámico
    form?: {
        fields: Field[];
        layout?: {
            grid?: { cols?: number; gap?: number };
            aside?: { show?: boolean; width?: number };
        };
    };
    payload?: Record<string, PayloadValue>


    // Paso de confirmación (opcional y simple)
    confirm?: {
        title?: string;
        summary?: Array<{ label: string; value: string }>;
        ctaLabel?: string;
        danger?: boolean;
        showPayload?: boolean;
        payloadSource?: 'rpc.payload' | 'custom';
        payloadTemplate?: any; // si usas plantilla custom de confirmación
    };

    auth?: { type: 'sessionPassword' | 'none' };

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
    value?: string;
    // features: copy / paste / set (Max)
    features?: FieldOp[];
    ds?: Record<string, any>;

};

export type AddressField = FieldBase & {
    type: 'address';
};

export type AmountField = FieldBase & {
    type: 'amount';
    min?: number;
    max?: number;
};

export type TextField = FieldBase & {
    type: 'text' | 'textarea';
};

export type SelectField = FieldBase & {
    type: 'select';
    options?: Array<{ label: string; value: string }>;
    // opciones desde una fuente dinámica (ds/fees/chain…)
    source?: SourceRef;
};

export type Field =
    | AddressField
    | AmountField
    | TextField
    | SelectField;

/* ===========================
 * Field Features (Ops)
 * =========================== */

export type FieldOp =
    | { id: string; op: 'copy'; from: string }                 // copia al clipboard el valor resuelto
    | { id: string; op: 'paste' }                              // pega desde clipboard al field
    | { id: string; op: 'set'; field: string; value: string }; // setea un valor (p.ej. Max)

/* ===========================
 * UI Ops / Events
 * =========================== */

export type UIOp =
    | { op: 'fetch'; source: SourceKey } // dispara un refetch/carga de DS al abrir
    | { op: 'notify'; message: string }; // opcional: mostrar toast/notificación

/* ===========================
 * Submit (HTTP)
 * =========================== */

export type Submit = {
    base: 'rpc' | 'admin';
    path: string;                         // p.ej. '/v1/admin/tx-send'
    method?: 'GET' | 'POST';
    headers?: Record<string, string>;
    encoding?: 'json' | 'text';
    body?: any;                           // plantilla a resolver o valor literal
};

/* ===========================
 * Sources y Selectors
 * =========================== */

export type SourceRef = {
    // de dónde sale el dato que vas a interpolar
    uses: 'chain' | 'ds' | 'fees' | 'form' | 'account' | 'session';
    // ruta dentro de la fuente (p.ej. 'fee.sendFee', 'amount', 'address')
    selector?: string;
};

// claves comunes de tu DS actual; permite string libre para crecer sin tocar tipos
export type SourceKey =
    | 'account'
    | 'params'
    | 'fees'
    | 'height'
    | 'validators'
    | 'activity'
    | 'txs.sent'
    | 'txs.received'
    | 'gov.proposals'
    | string;

/* ===========================
 * Fees (opcional, lo mínimo)
 * =========================== */

export type FeeBuckets = {
    [bucket: string]: { multiplier: number; default?: boolean };
};

export type FeeProviderQuery = {
    type: 'query';
    base: 'rpc' | 'admin';
    path: string;
    method?: 'GET' | 'POST';
    headers?: Record<string, string>;
    encoding?: 'json' | 'text';
    selector?: string; // p.ej. 'fee' dentro del response
    cache?: { staleTimeMs?: number; refetchIntervalMs?: number };
};

export type FeeProviderSimulate = {
    type: 'simulate';
    base: 'rpc' | 'admin';
    path: string;
    method?: 'GET' | 'POST';
    headers?: Record<string, string>;
    encoding?: 'json' | 'text';
    body?: any;
    gasAdjustment?: number;
    gasPrice?:
        | { type: 'static'; value: string }
        | {
        type: 'query';
        base: 'rpc' | 'admin';
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
