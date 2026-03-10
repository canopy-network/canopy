import React from 'react'
import { template } from '@/core/templater'
import TableSelect from '@/actions/TableSelect'
import { BaseFieldProps } from './types'

type TableSelectFieldProps = BaseFieldProps & {
    errors: Record<string, string>
}

export const TableSelectField: React.FC<TableSelectFieldProps> = ({
    field,
    value,
    errors,
    templateContext,
    dsValue,
    onChange,
    resolveTemplate,
}) => {
    // Track if we've initialized from DS
    const hasInitializedRef = React.useRef(false)

    // Auto-populate from DS when it loads (for pre-selecting committees)
    React.useEffect(() => {
        // Only auto-populate if:
        // 1. Field has a value template (e.g., "{{ ds.validator?.committees ?? [] }}")
        // 2. Current value is empty
        // 3. We haven't initialized yet
        if ((field as any).value && !hasInitializedRef.current) {
            const resolved = resolveTemplate((field as any).value)

            // Check if resolved value is non-empty
            const hasResolvedValue = resolved && (Array.isArray(resolved) ? resolved.length > 0 : resolved !== '')
            const hasCurrentValue = value && (Array.isArray(value) ? value.length > 0 : value !== '')

            if (hasResolvedValue && !hasCurrentValue) {
                onChange(resolved)
                hasInitializedRef.current = true
            }
        }
    }, [templateContext, field, value, onChange, resolveTemplate])

    return (
        <TableSelect
            field={field as any}
            currentValue={value}
            onChange={(next) => onChange(next)}
            errors={errors}
            resolveTemplate={resolveTemplate}
            template={template}
            templateContext={templateContext}
        />
    )
}
