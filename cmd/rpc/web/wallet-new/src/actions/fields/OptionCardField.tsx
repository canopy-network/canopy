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
            <div role="radiogroup" aria-label={String(resolveTemplate(field.label) ?? field.name)} className="grid grid-cols-12 gap-3 w-full">
                {opts.map((o, i) => {
                    const label = resolveTemplate(o.label)
                    const help = resolveTemplate(o.help)
                    const val = String(resolveTemplate(o.value) ?? i)
                    const selected = String(currentValue ?? '') === val

                    return (
                        <div key={val} className="col-span-12">
                            <OptionCard
                                selected={selected}
                                disabled={field.readOnly}
                                onSelect={() => onChange(val)}
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
