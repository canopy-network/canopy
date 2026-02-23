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

    // Track previous resolved value to sync when template changes
    const prevResolvedRef = React.useRef<string | null>(null)

    // Sync field value when the resolved template changes (e.g., table selection)
    // This allows computed fields to stay in sync while still being editable
    React.useEffect(() => {
        if (field.value && resolvedValue != null) {
            const resolvedStr = String(resolvedValue)
            if (prevResolvedRef.current !== null && prevResolvedRef.current !== resolvedStr) {
                // Template value changed, sync the input
                onChange(resolvedStr)
            }
            prevResolvedRef.current = resolvedStr
        }
    }, [resolvedValue, field.value, onChange])

    // For readOnly fields with a value template, always use the resolved template
    // For editable fields, use form value but initialize from template if empty
    const currentValue =
        field.readOnly && field.value && resolvedValue != null
            ? resolvedValue
            : value === '' && resolvedValue != null
                ? resolvedValue
                : value || (dsValue?.amount ?? dsValue?.value ?? '')

    const hasFeatures = !!(field.features?.length)
    const commonBase = 'w-full bg-background/60 border placeholder:text-muted-foreground/70 text-foreground rounded-xl px-3 focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50 transition-colors'
    const common = isTextarea
        ? `${commonBase} py-2.5 min-h-[112px] resize-y`
        : `${commonBase} h-11 sm:h-12`
    const paddingRight = hasFeatures ? 'pr-24' : '' // Increased padding for better button spacing
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
            <Component
                className={cx(common, border, paddingRight)}
                type={!isTextarea ? (field as any).inputType ?? 'text' : undefined}
                placeholder={resolveTemplate(field.placeholder)}
                value={currentValue ?? ''}
                readOnly={field.readOnly}
                required={field.required}
                onChange={(e: any) => onChange(e.currentTarget.value)}
            />
        </FieldWrapper>
    )
}
