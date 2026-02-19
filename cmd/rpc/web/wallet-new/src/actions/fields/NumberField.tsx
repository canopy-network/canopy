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
    const common = 'w-full bg-transparent border placeholder-text-muted text-foreground rounded px-3 py-2 focus:outline-none [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none'
    const border = error ? 'border-red-600' : 'border-muted-foreground border-opacity-50'
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
