import React from 'react'
import { cx } from '@/ui/cx'
import { FieldWrapper } from './FieldWrapper'
import { BaseFieldProps } from './types'

export const AddressField: React.FC<BaseFieldProps> = ({
    field,
    value,
    error,
    templateContext,
    onChange,
    resolveTemplate,
    setVal,
}) => {
    const resolved = resolveTemplate(field.value)
    const currentValue = value === '' && resolved != null ? resolved : value

    const hasFeatures = !!(field.features?.length)
    const common = 'w-full bg-transparent border placeholder-text-muted text-foreground rounded px-3 py-2 focus:outline-none'
    const paddingRight = hasFeatures ? 'pr-20' : ''
    const border = error ? 'border-red-600' : 'border-muted-foreground border-opacity-50'

    return (
        <FieldWrapper
            field={field}
            error={error}
            templateContext={templateContext}
            resolveTemplate={resolveTemplate}
            hasFeatures={hasFeatures}
            setVal={setVal}
        >
            <input
                className={cx(common, border, paddingRight)}
                placeholder={resolveTemplate(field.placeholder) ?? 'address'}
                value={currentValue ?? ''}
                readOnly={field.readOnly}
                required={field.required}
                onChange={(e) => onChange(e.target.value)}
            />
        </FieldWrapper>
    )
}
