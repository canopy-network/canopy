import React from 'react'
import { FieldOp } from '@/manifest/types'
import { template } from '@/core/templater'

type FieldFeaturesProps = {
    fieldId: string
    features?: FieldOp[]
    ctx: Record<string, any>
    setVal: (fieldId: string, v: any) => void
}

export const FieldFeatures: React.FC<FieldFeaturesProps> = ({ features, ctx, setVal, fieldId }) => {
    if (!features?.length) return null

    const resolve = (s?: any) => (typeof s === 'string' ? template(s, ctx) : s)

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
                await navigator.clipboard.writeText(txt)
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

    return (
        <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1">
            {features.map((op) => (
                <button
                    key={op.id}
                    type="button"
                    onClick={() => handle(op)}
                    className="text-xs px-2 py-1 rounded font-semibold border border-primary text-primary hover:bg-primary hover:text-secondary transition-colors"
                >
                    {labelFor(op)}
                </button>
            ))}
        </div>
    )
}
