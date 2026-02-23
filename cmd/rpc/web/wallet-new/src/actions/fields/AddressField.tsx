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
    const common = 'w-full h-11 sm:h-12 bg-background/60 border placeholder:text-muted-foreground/70 text-foreground rounded-xl px-3 focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50 transition-colors'
    const paddingRight = hasFeatures ? 'pr-20' : ''
    const border = error ? 'border-red-600' : 'border-border/70'

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
