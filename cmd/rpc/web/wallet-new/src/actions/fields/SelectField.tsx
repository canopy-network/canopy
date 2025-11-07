import React from 'react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/Select'
import { SelectField as SelectFieldType } from '@/manifest/types'
import { template, templateAny } from '@/core/templater'
import { toOptions } from '@/actions/utils/fieldHelpers'
import { FieldWrapper } from './FieldWrapper'
import { BaseFieldProps } from './types'

export const SelectField: React.FC<BaseFieldProps> = ({
    field,
    value,
    error,
    templateContext,
    dsValue,
    onChange,
    resolveTemplate,
}) => {
    const select = field as SelectFieldType
    const staticOptions = Array.isArray(select.options) ? select.options : []
    const rawOptions = dsValue && Object.keys(dsValue).length ? dsValue : staticOptions

    let mappedFromExpr: any[] | null = null
    if (typeof (select as any).map === 'string') {
        try {
            const out = templateAny((select as any).map, templateContext)
            if (Array.isArray(out)) {
                mappedFromExpr = out
            } else if (typeof out === 'string') {
                try {
                    const maybe = JSON.parse(out)
                    if (Array.isArray(maybe)) mappedFromExpr = maybe
                } catch {}
            }
        } catch (err) {
            console.warn('select.map expression error:', err)
        }
    }

    const builtOptions = mappedFromExpr
        ? mappedFromExpr.map((o) => ({
              label: String(o?.label ?? ''),
              value: String(o?.value ?? ''),
          }))
        : toOptions(rawOptions, field, templateContext, template)

    const resolvedDefault = resolveTemplate(field.value)
    const currentValue = value === '' && resolvedDefault != null ? resolvedDefault : value

    return (
        <FieldWrapper field={field} error={error} templateContext={templateContext} resolveTemplate={resolveTemplate}>
            <Select
                value={currentValue ?? ''}
                onValueChange={(val) => onChange(val)}
                disabled={field.readOnly}
                required={field.required}
            >
                <SelectTrigger className="w-full bg-bg-tertiary border-bg-accent text-white h-11 rounded-lg">
                    <SelectValue placeholder={field.placeholder} />
                </SelectTrigger>
                <SelectContent className="bg-bg-tertiary border-bg-accent">
                    {builtOptions.map((o) => (
                        <SelectItem key={o.value} value={o.value} className="text-white">
                            {o.label}
                        </SelectItem>
                    ))}
                </SelectContent>
            </Select>
        </FieldWrapper>
    )
}
