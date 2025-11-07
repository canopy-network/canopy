import React from 'react'
import { cx } from '@/ui/cx'
import { FieldWrapper } from './FieldWrapper'
import { BaseFieldProps } from './types'

export const TextField: React.FC<BaseFieldProps> = ({
    field,
    value,
    error,
    templateContext,
    dsValue,
    onChange,
    resolveTemplate,
    setVal,
}) => {
    const isTextarea = field.type === 'textarea'
    const Component: any = isTextarea ? 'textarea' : 'input'

    const resolvedValue = resolveTemplate(field.value)
    const currentValue =
        value === '' && resolvedValue != null
            ? resolvedValue
            : value || (dsValue?.amount ?? dsValue?.value ?? '')

    const hasFeatures = !!(field.features?.length)
    const common = 'w-full bg-transparent border placeholder-text-muted text-white rounded px-3 py-2 focus:outline-none'
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
            <Component
                className={cx(common, border, paddingRight)}
                placeholder={resolveTemplate(field.placeholder)}
                value={currentValue ?? ''}
                readOnly={field.readOnly}
                required={field.required}
                onChange={(e: any) => onChange(e.currentTarget.value)}
            />
        </FieldWrapper>
    )
}
