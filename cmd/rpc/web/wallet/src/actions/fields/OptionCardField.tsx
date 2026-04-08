import React from 'react'
import { OptionCard, OptionCardOpt } from '@/actions/OptionCard'
import { FieldWrapper } from './FieldWrapper'
import { BaseFieldProps } from './types'

export const OptionCardField: React.FC<BaseFieldProps> = ({
    field,
    value,
    error,
    templateContext,
    onChange,
    resolveTemplate,
}) => {
    const opts: OptionCardOpt[] = Array.isArray((field as any).options) ? (field as any).options : []
    const resolvedDefault = resolveTemplate(field.value)
    const currentValue = (value === '' || value == null) && resolvedDefault != null ? resolvedDefault : value

    return (
        <FieldWrapper field={field} error={error} templateContext={templateContext} resolveTemplate={resolveTemplate}>
            <div role="radiogroup" aria-label={String(resolveTemplate(field.label) ?? field.name)} className="grid grid-cols-1 sm:grid-cols-2 gap-3 w-full">
                {opts.map((o, i) => {
                    const label = resolveTemplate(o.label)
                    const help = resolveTemplate(o.help)
                    const rawValue = resolveTemplate(o.value) ?? i

                    // Normalize values for comparison (handle booleans, strings, numbers)
                    const normalizeValue = (v: any) => {
                        if (v === true || v === 'true') return true
                        if (v === false || v === 'false') return false
                        return v
                    }

                    const normalizedOptionValue = normalizeValue(rawValue)
                    const normalizedCurrentValue = normalizeValue(currentValue)
                    const selected = normalizedCurrentValue === normalizedOptionValue

                    return (
                        <div key={String(rawValue)}>
                            <OptionCard
                                selected={selected}
                                disabled={field.readOnly}
                                onSelect={() => onChange(normalizedOptionValue)}
                                label={label}
                                help={help}
                            />
                        </div>
                    )
                })}
            </div>
        </FieldWrapper>
    )
}
