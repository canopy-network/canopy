import React from 'react'
import { cx } from '@/ui/cx'
import { OptionField as OptionFieldType } from '@/manifest/types'
import { Option, OptionItem } from '@/actions/Option'
import { FieldWrapper } from './FieldWrapper'
import { BaseFieldProps } from './types'

export const OptionField: React.FC<BaseFieldProps> = ({
    field,
    value,
    error,
    templateContext,
    onChange,
    resolveTemplate,
}) => {
    const optionField = field as OptionFieldType
    const isInLine = optionField.inLine
    const opts: OptionItem[] = Array.isArray((field as any).options) ? (field as any).options : []
    const resolvedDefault = resolveTemplate(field.value)
    const currentValue = (value === '' || value == null) && resolvedDefault != null ? resolvedDefault : value

    return (
        <FieldWrapper field={field} error={error} templateContext={templateContext} resolveTemplate={resolveTemplate}>
            <div
                role="radiogroup"
                aria-label={String(resolveTemplate(field.label) ?? field.name)}
                className={cx('w-full gap-3', isInLine ? 'flex flex-wrap justify-between items-center' : 'grid grid-cols-12')}
            >
                {opts.map((o, i) => {
                    const label = resolveTemplate(o.label)
                    const help = resolveTemplate(o.help)
                    const val = String(resolveTemplate(o.value) ?? i)
                    const selected = String(currentValue ?? '') === val

                    return (
                        <div key={val} className={cx(isInLine ? 'flex-1 min-w-[120px] max-w-full' : 'col-span-12')}>
                            <Option
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
