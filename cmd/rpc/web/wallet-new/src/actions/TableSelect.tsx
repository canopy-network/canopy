import * as React from 'react'
import { templateBool } from '@/core/templater' // ajusta la ruta si aplica

/** Tipos básicos del manifest */
type ColAlign = 'left' | 'center' | 'right'
type ColumnType = 'text' | 'image' | 'html'

export type TableSelectColumn = {
    key?: string
    title?: string
    align?: ColAlign
    type?: ColumnType

    /** TEXT */
    expr?: string

    /** IMAGE */
    src?: string                   // expr o key -> URL de imagen (si no hay, cae a avatar)
    alt?: string                   // expr opcional para alt
    initialsFrom?: string          // expr/llave para derivar iniciales y color si no hay 'src'
    size?: number                  // tamaño del avatar/imagen en px (default 28)

    /** HTML */
    html?: string                  // HTML templated (se renderiza con dangerouslySetInnerHTML)
}

export type TableRowAction = {
    title?: string                 // título de cabecera para la columna de acción
    label?: string                 // template del label del botón
    icon?: string                  // (reservado) por si luego usas un icon set central
    showIf?: string                // template condicional
    emit?: {
        op: 'set' | 'copy' | 'select'        // select: marcar selección; set: setear otro field; copy: al portapapeles
        field?: string                        // requerido para 'set'
        value?: string                        // template
    }
}

/** Config del field en manifest */
export type TableSelectField = {
    id: string
    name: string
    type: 'tableSelect'
    label?: string
    help?: string
    required?: boolean
    readOnly?: boolean
    multiple?: boolean
    rowKey?: string
    columns: TableSelectColumn[]
    rows?: any[]                             // data estática
    source?: { uses: string; selector?: string } // data dinámica: p.ej. {uses:'ds', selector:'committees'}
    rowAction?: TableRowAction
    /** cómo se selecciona */
    selectMode?: 'row' | 'action' | 'none'   // 'row' (default): click en fila; 'action': sólo botón; 'none': deshabilitado
}

/** Props del componente */
export type TableSelectProps = {
    field: TableSelectField
    currentValue: any
    onChange: (next: any) => void
    errors?: Record<string, string>
    resolveTemplate: (v: any) => any
    template: (tpl: string, ctx?: any) => any
    templateContext?: any
}

/** Utils locales */
const cx = (...a: Array<string | false | null | undefined>) => a.filter(Boolean).join(' ')
const asArray = (x: any) => Array.isArray(x) ? x : (x == null ? [] : [x])
const pick = (obj: any, path?: string) => !path ? obj : path.split('.').reduce((acc, k) => acc?.[k], obj)
const safe = (v: any) => v == null ? '' : String(v)

/** Mobile-first: span según cantidad de columnas totales (12 = full) */
function spanResponsiveByCount(colCount: number): string {
    if (colCount <= 1) return 'col-span-12'
    if (colCount === 2) return 'col-span-12 sm:col-span-6 md:col-span-6'
    if (colCount === 3) return 'col-span-12 sm:col-span-6 md:col-span-4 lg:col-span-4'
    if (colCount === 4) return 'col-span-12 sm:col-span-6 md:col-span-3 lg:col-span-3'
    if (colCount === 5) return 'col-span-12 sm:col-span-6 md:col-span-2 lg:col-span-2'
    if (colCount === 6) return 'col-span-12 sm:col-span-6 md:col-span-2 lg:col-span-2'
    return 'col-span-12 sm:col-span-6 md:col-span-2 lg:col-span-1' // 7+
}

/** Avatar helpers (para fallback cuando no hay imagen) */
function hashColor(input: string): string {
    let h = 0
    for (let i = 0; i < input.length; i++) h = (h << 5) - h + input.charCodeAt(i)
    const hue = Math.abs(h) % 360
    return `hsl(${hue} 65% 45%)`
}
function getInitials(text?: string) {
    const p = (text ?? '').trim().split(/\s+/)
    const first = p[0]?.[0] ?? ''
    const last = p.length > 1 ? p[p.length - 1]?.[0] ?? '' : ''
    return (first + last).toUpperCase() || (text?.[0]?.toUpperCase() ?? '•')
}

const TableSelect: React.FC<TableSelectProps> = ({
                                                     field: tf,
                                                     currentValue,
                                                     onChange,
                                                     errors = {},
                                                     resolveTemplate,
                                                     template,
                                                     templateContext
                                                 }) => {
    const columns = React.useMemo(
        () => (tf.columns ?? []).map(c => ({ ...c, title: c.title ? resolveTemplate(c.title) : undefined })),
        [tf.columns, resolveTemplate]
    )
    const keyField = tf.rowKey ?? 'id'
    const label = resolveTemplate(tf.label)
    const selectMode = tf.selectMode ?? 'row'

    const base = tf.source ? templateContext?.[tf.source.uses] : undefined
    const dsRows = tf.source ? asArray(pick(base, tf.source.selector)) : []
    const staticRows = asArray(tf.rows)
    const rows = React.useMemo(
        () => (dsRows.length ? dsRows : staticRows).map((r: any, idx: number) => ({ __idx: idx, ...r })),
        [dsRows, staticRows]
    )

    const selectedKeys: string[] = React.useMemo(() => {
        return tf.multiple
            ? asArray(currentValue).map(String)
            : (currentValue != null && currentValue !== '' ? [String(currentValue)] : [])
    }, [currentValue, tf.multiple])

    const setSelectedKey = (k: string) => {
        if (tf.readOnly) return
        if (tf.multiple) {
            const next = selectedKeys.includes(k) ? selectedKeys.filter(x => x !== k) : [...selectedKeys, k]
            onChange(next)
        } else {
            onChange(selectedKeys[0] === k ? '' : k)
        }
    }

    const toggleRow = (row: any) => {
        if (selectMode !== 'row' || tf.readOnly) return
        const k = String(row[keyField] ?? row.__idx)
        setSelectedKey(k)
    }

    const renderAction = (row: any) => {
        const ra = tf.rowAction
        if (!ra) return null
        const localCtx = { ...templateContext, row }
        const visible = ra.showIf == null ? true : templateBool(ra.showIf, localCtx)
        if (!visible) return null

        const btnLabel = ra.label ? template(ra.label, localCtx) : 'Action'
        const onClick = async (e: React.MouseEvent) => {
            e.stopPropagation()
            if (!ra.emit) return
            if (ra.emit.op === 'set') {
                const val = ra.emit.value ? template(ra.emit.value, localCtx) : undefined
                onChange(val)
            } else if (ra.emit.op === 'copy') {
                const val = ra.emit.value ? template(ra.emit.value, localCtx) : JSON.stringify(row)
                await navigator.clipboard.writeText(String(val ?? ''))
            } else if (ra.emit.op === 'select') {
                if (tf.readOnly) return
                const k = String(row[keyField] ?? row.__idx)
                setSelectedKey(k)
            }
        }
        return (
            <button
                type="button"
                onClick={onClick}
                className="px-2 py-1 rounded border border-primary text-primary hover:bg-primary hover:text-secondary text-xs font-bold"
            >
                {safe(btnLabel)}
            </button>
        )
    }

    /** 4) Pintado */
    const colCount = columns.length + (tf.rowAction ? 1 : 0)
    const colSpanCls = spanResponsiveByCount(colCount)
    const cellAlign = (a?: ColAlign) =>
        a === 'right' ? 'text-right' : a === 'center' ? 'text-center' : 'text-left'

    const renderImageCell = (col: TableSelectColumn, row: any) => {
        const local = { ...templateContext, row }
        const size = (col.size ?? 28)
        const src = col.src ? safe(template(col.src, local)) : ''
        const alt = col.alt ? safe(template(col.alt, local)) : safe((col.key ? row[col.key] : row.name) ?? '')
        const basis = col.initialsFrom ? safe(template(col.initialsFrom, local)) : safe((col.key ? row[col.key] : row.name) ?? '')
        const initials = getInitials(basis)
        const color = hashColor(basis)

        if (src) {
            return (
                <img
                    src={src}
                    alt={alt}
                    width={size}
                    height={size}
                    className="rounded-full object-cover inline-block"
                    style={{ width: size, height: size }}
                />
            )
        }
        return (
            <span
                className="inline-flex items-center justify-center rounded-full text-xs font-semibold"
                style={{ width: size, height: size, background: color, color: 'white' }}
                aria-label={alt}
            >
        {initials}
      </span>
        )
    }

    const renderCell = (c: TableSelectColumn, row: any) => {
        const local = { ...templateContext, row }
        if (c.type === 'image') return renderImageCell(c, row)

        if (c.type === 'html' && c.html) {
            const htmlString = template(c.html, local)
            return <div className="truncate" dangerouslySetInnerHTML={{ __html: htmlString }} />
        }

        const cellVal = c.expr
            ? template(c.expr, local)
            : (c.key ? row[c.key] : '')
        return <span className="truncate">{safe(cellVal ?? '—')}</span>
    }

    return (
        <div className="col-span-12 w-full">
            {!!label && <div className="text-sm mb-2 text-text-muted">{label}</div>}

            <div className="rounded-lg border border-neutral-700 overflow-hidden">
                {/* Header */}
                <div className="grid grid-cols-12 gap-2 px-3 py-2 border-b border-muted text-[11px] uppercase tracking-wide text-neutral-400">
                    {columns.map((c, i) => (
                        <div key={c.key ?? i} className={cx(colSpanCls, cellAlign(c.align), 'truncate')}>
                            {safe(c.title)}
                        </div>
                    ))}
                    {tf.rowAction?.title && (
                        <div className={cx(colSpanCls, cellAlign('right'), 'truncate')}>
                            {resolveTemplate(tf.rowAction.title)}
                        </div>
                    )}
                </div>

                {/* Rows */}
                <div className="divide-y divide-neutral-800">
                    {rows.map((row: any) => {
                        const k = String(row[keyField] ?? row.__idx)
                        const selected = selectedKeys.includes(k)
                        return (
                            <button
                                type="button"
                                key={k}
                                onClick={() => toggleRow(row)}
                                className={cx(
                                    'w-full grid grid-cols-12 gap-2 items-center px-3 py-2 text-sm hover:bg-bg-primary/40 transition-colors text-canopy-50',
                                    selected && 'bg-primary/10',
                                    selectMode !== 'row' && 'cursor-default' // si no se permite click en fila
                                )}
                                aria-pressed={selected}
                            >
                                {columns.map((c, i) => (
                                    <div key={c.key ?? i} className={cx(colSpanCls, cellAlign(c.align))}>
                                        {renderCell(c, row)}
                                    </div>
                                ))}
                                {tf.rowAction && (
                                    <div className={cx(colSpanCls, 'flex justify-end')}>
                                        {renderAction(row)}
                                    </div>
                                )}
                            </button>
                        )
                    })}
                    {rows.length === 0 && (
                        <div className="px-3 py-6 text-center text-xs text-neutral-500">No data</div>
                    )}
                </div>
            </div>

            {(errors[tf.name]) && (
                <div className={cx('text-xs mt-1', errors[tf.name] ? 'text-red-400' : 'text-text-muted')}>
                    {errors[tf.name]}
                </div>
            )}
        </div>
    )
}

export default TableSelect