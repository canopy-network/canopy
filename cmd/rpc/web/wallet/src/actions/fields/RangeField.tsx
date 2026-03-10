import React from 'react'
import { cx } from '@/ui/cx'
import { FieldWrapper } from './FieldWrapper'
import { BaseFieldProps } from './types'

const toFiniteNumber = (value: any, fallback: number) => {
    const n = Number(value)
    return Number.isFinite(n) ? n : fallback
}

export const RangeField: React.FC<BaseFieldProps> = ({
    field,
    value,
    error,
    templateContext,
    dsValue,
    onChange,
    resolveTemplate,
    setVal,
}) => {
    const rawMin = resolveTemplate((field as any).min)
    const rawMax = resolveTemplate((field as any).max)
    const rawStep = resolveTemplate((field as any).step)

    const min = toFiniteNumber(rawMin, 0)
    const max = toFiniteNumber(rawMax, 100)
    const step = toFiniteNumber(rawStep, 1)

    const fallbackValue = toFiniteNumber(resolveTemplate((field as any).value), min)
    const dsRaw = dsValue?.value ?? dsValue
    const current = toFiniteNumber(value ?? dsRaw, fallbackValue)
    const clamped = Math.min(max, Math.max(min, current))

    const showInput = (field as any).showInput !== false
    const suffix = resolveTemplate((field as any).suffix ?? '')
    const marks = Array.isArray((field as any).marks) ? (field as any).marks : []
    const presets = Array.isArray((field as any).presets) ? (field as any).presets : []
    const hasFeatures = !!(field.features?.length)
    const range = max - min
    const progress = range > 0 ? ((clamped - min) / range) * 100 : 0

    const inputClass = cx(
        'w-full h-11 sm:h-12 bg-background/60 border placeholder:text-muted-foreground/70 text-foreground rounded-xl px-3 focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50 transition-colors [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none',
        error ? 'border-red-600' : 'border-border/70',
        hasFeatures && showInput ? 'pr-24' : '',
    )

    return (
        <FieldWrapper
            field={field}
            error={error}
            templateContext={templateContext}
            resolveTemplate={resolveTemplate}
            hasFeatures={hasFeatures && showInput}
            setVal={setVal}
            currentValue={clamped}
        >
            <div className="space-y-2">
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                    <span>{min}{suffix}</span>
                    <span className="font-semibold text-primary">{clamped}{suffix}</span>
                    <span>{max}{suffix}</span>
                </div>

                <div className="relative py-1">
                    <div className="h-2.5 rounded-full bg-muted/50 overflow-hidden">
                        <div
                            className="h-full bg-primary transition-all duration-200 rounded-full"
                            style={{ width: `${progress}%` }}
                        />
                    </div>
                    <input
                        type="range"
                        min={min}
                        max={max}
                        step={step}
                        className="absolute inset-x-0 -top-1 h-5 w-full opacity-0 cursor-pointer"
                        value={clamped}
                        required={field.required}
                        disabled={field.disabled || field.readOnly}
                        onChange={(e) => onChange(Number(e.currentTarget.value))}
                    />
                </div>

                {marks.length > 0 && (
                    <div
                        className="grid gap-1 text-[10px] text-muted-foreground"
                        style={{ gridTemplateColumns: `repeat(${marks.length}, minmax(0, 1fr))` }}
                    >
                        {marks.map((m: any, i: number) => (
                            <span key={`${m}-${i}`} className="text-center">{m}{suffix}</span>
                        ))}
                    </div>
                )}

                {presets.length > 0 && (
                    <div className="flex flex-wrap gap-1.5 pt-1">
                        {presets.map((preset: any, i: number) => {
                            const presetValue = toFiniteNumber(resolveTemplate(preset?.value), clamped)
                            const active = presetValue === clamped
                            return (
                                <button
                                    key={`${preset?.label ?? 'preset'}-${i}`}
                                    type="button"
                                    className={cx(
                                        'px-2 py-1 text-xs rounded-md border transition-colors',
                                        active
                                            ? 'border-primary bg-primary/15 text-primary'
                                            : 'border-border/70 text-muted-foreground hover:text-foreground hover:border-primary/60',
                                    )}
                                    onClick={() => onChange(presetValue)}
                                >
                                    {resolveTemplate(preset?.label ?? `${presetValue}${suffix}`)}
                                </button>
                            )
                        })}
                    </div>
                )}

                {showInput && (
                    <input
                        type="number"
                        min={min}
                        max={max}
                        step={step}
                        className={inputClass}
                        value={clamped}
                        required={field.required}
                        readOnly={field.readOnly}
                        onChange={(e) => onChange(Number(e.currentTarget.value))}
                    />
                )}
            </div>
        </FieldWrapper>
    )
}
