import React from 'react'
import { cx } from '@/ui/cx'
import { FieldWrapper } from './FieldWrapper'
import { BaseFieldProps } from './types'

export const NumberField: React.FC<BaseFieldProps> = ({
    field,
    value,
    error,
    templateContext,
    dsValue,
    onChange,
    resolveTemplate,
    setVal,
}) => {
    const currentValue = value ?? (dsValue?.value ?? '')
    const hasFeatures = !!(field.features?.length)

    const step = (field as any).integer ? 1 : (field as any).step ?? 'any'
    const common = 'w-full h-11 sm:h-12 bg-background/60 border placeholder:text-muted-foreground/70 text-foreground rounded-xl px-3 focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50 transition-colors [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none'
    const border = error ? 'border-red-600' : 'border-border/70'
    const paddingRight = hasFeatures ? 'pr-24' : ''

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
                step={step}
                className={cx(common, border, paddingRight)}
                placeholder={resolveTemplate(field.placeholder)}
                value={currentValue ?? ''}
                readOnly={field.readOnly}
                required={field.required}
                onChange={(e) => onChange(e.currentTarget.value)}
                min={(field as any).min}
                max={(field as any).max}
            />
        </FieldWrapper>
    )
}
