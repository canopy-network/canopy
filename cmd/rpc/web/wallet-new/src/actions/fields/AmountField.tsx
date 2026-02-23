import React from 'react'
import { cx } from '@/ui/cx'
import { FieldWrapper } from './FieldWrapper'
import { BaseFieldProps } from './types'

export const AmountField: React.FC<BaseFieldProps> = ({
    field,
    value,
    error,
    templateContext,
    dsValue,
    onChange,
    resolveTemplate,
    setVal,
}) => {
    const currentValue = value ?? (dsValue?.amount ?? dsValue?.value ?? '')
    const hasFeatures = !!(field.features?.length)

    // Get denomination from chain context.
    // showDenom can be disabled per field from manifest (config-first behavior).
    const explicitShowDenom = (field as any).showDenom
    const denom = (field as any).denom ?? templateContext?.chain?.denom?.symbol ?? ''
    const showDenom = explicitShowDenom === false ? false : !!denom

    // Calculate padding based on features and denom
    // Increased padding for better spacing with the MAX button
    const paddingRight = hasFeatures && showDenom ? 'pr-36' : hasFeatures ? 'pr-24' : showDenom ? 'pr-16' : ''

    const common = 'w-full h-11 sm:h-12 bg-background/60 border placeholder:text-muted-foreground/70 text-foreground rounded-xl px-3 focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50 transition-colors [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none'
    const border = error ? 'border-red-600' : 'border-border/70'

    return (
        <FieldWrapper
            field={field}
            error={error}
            templateContext={templateContext}
            resolveTemplate={resolveTemplate}
            hasFeatures={hasFeatures}
            setVal={setVal}
            currentValue={currentValue}
        >
            <input
                type="number"
                step="any"
                className={cx(common, border, paddingRight)}
                placeholder={resolveTemplate(field.placeholder)}
                value={currentValue ?? ''}
                readOnly={field.readOnly}
                required={field.required}
                onChange={(e) => onChange(e.currentTarget.value)}
                min={(field as any).min}
                max={(field as any).max}
            />
            {showDenom && (
                <div className={cx(
                    "absolute top-1/2 -translate-y-1/2 text-muted-foreground text-sm font-medium pointer-events-none",
                    hasFeatures ? "right-24" : "right-3"
                )}>
                    {denom}
                </div>
            )}
        </FieldWrapper>
    )
}
