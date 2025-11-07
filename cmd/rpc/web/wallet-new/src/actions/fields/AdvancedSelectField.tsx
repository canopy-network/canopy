import React from 'react'
import { AdvancedSelectField as AdvancedSelectFieldType } from '@/manifest/types'
import { template, templateAny } from '@/core/templater'
import { toOptions } from '@/actions/utils/fieldHelpers'
import ComboSelectRadix from '@/actions/ComboSelect'
import { FieldWrapper } from './FieldWrapper'
import { BaseFieldProps } from './types'

export const AdvancedSelectField: React.FC<BaseFieldProps> = ({
    field,
    value,
    error,
    templateContext,
    dsValue,
    onChange,
    resolveTemplate,
}) => {
    const select = field as AdvancedSelectFieldType
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
            <ComboSelectRadix
                id={field.id}
                value={currentValue}
                options={builtOptions}
                onChange={(val) => onChange(val)}
                placeholder={field.placeholder}
                allowAssign={(field as any).allowCreate}
                allowFreeInput={(field as any).allowFreeInput}
                disabled={field.disabled}
            />
        </FieldWrapper>
    )
}
