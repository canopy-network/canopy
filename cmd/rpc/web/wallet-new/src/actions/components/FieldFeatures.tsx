import React from 'react'
import { FieldOp } from '@/manifest/types'
import { template } from '@/core/templater'
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard'

type FieldFeaturesProps = {
    fieldId: string
    features?: FieldOp[]
    ctx: Record<string, any>
    setVal: (fieldId: string, v: any) => void
    currentValue?: any
}

export const FieldFeatures: React.FC<FieldFeaturesProps> = ({ features, ctx, setVal, fieldId, currentValue }) => {
    const { copyToClipboard } = useCopyToClipboard()

    if (!features?.length) return null

    const resolve = (s?: any) => (typeof s === 'string' ? template(s, ctx) : s)

    // Check if this field was programmatically prefilled (from prefilledData or modules)
    const isProgrammaticallyPrefilled = ctx?.__programmaticallyPrefilled?.has(fieldId) ?? false

    // Only hide paste button if field is programmatically prefilled AND has a value
    const shouldHidePaste = isProgrammaticallyPrefilled && currentValue !== undefined && currentValue !== null && currentValue !== ''

    const labelFor = (op: FieldOp) => {
        const opAny = op as any
        if (opAny.op === 'copy') return 'Copy'
        if (opAny.op === 'paste') return 'Paste'
        if (opAny.op === 'set' || opAny.op === 'max') {
            // Custom label or default to "Max" for set/max operations
            return opAny.label ?? 'Max'
        }
        return opAny.op
    }

    const handle = async (op: FieldOp) => {
        const opAny = op as any
        switch (opAny.op) {
            case 'copy': {
                const txt = String(resolve(opAny.from) ?? '')
                await copyToClipboard(txt, opAny.label || 'Field value')
                return
            }
            case 'paste': {
                const txt = await navigator.clipboard.readText()
                setVal(fieldId, txt)
                return
            }
            case 'set':
            case 'max': {
                // Resolve the value from manifest (can be a template expression)
                const v = resolve(opAny.value)
                setVal(opAny.field ?? fieldId, v)
                return
            }
        }
    }

    // Filter features: hide paste button ONLY when field is programmatically prefilled
    const visibleFeatures = features.filter((op) => {
        const opAny = op as any
        // Hide paste button only if field was programmatically prefilled (not from autopopulate/DS)
        if (opAny.op === 'paste' && shouldHidePaste) {
            return false
        }
        return true
    })

    // Don't render if no visible features
    if (!visibleFeatures.length) return null

    return (
        <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1.5 z-10">
            {visibleFeatures.map((op) => (
                <button
                    key={op.id}
                    type="button"
                    onClick={() => handle(op)}
                    className="text-xs px-3 py-1.5 rounded-md font-semibold border border-primary/60 bg-primary/10 text-primary hover:bg-primary hover:text-bg-primary transition-all duration-200 shadow-sm hover:shadow-md active:scale-95"
                >
                    {labelFor(op)}
                </button>
            ))}
        </div>
    )
}
