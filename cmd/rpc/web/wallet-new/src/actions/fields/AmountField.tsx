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

    // Get denomination from chain context
    const denom = templateContext?.chain?.denom?.symbol || (field as any).denom || ''
    const showDenom = !!denom

    // Calculate padding based on features and denom
    // Increased padding for better spacing with the MAX button
    const paddingRight = hasFeatures && showDenom ? 'pr-36' : hasFeatures ? 'pr-24' : showDenom ? 'pr-16' : ''

    const common = 'w-full bg-transparent border placeholder-text-muted text-foreground rounded px-3 py-2 focus:outline-none [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none'
    const border = error ? 'border-red-600' : 'border-muted-foreground border-opacity-50'

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
