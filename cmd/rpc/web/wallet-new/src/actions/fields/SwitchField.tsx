import React from 'react'
import * as Switch from '@radix-ui/react-switch'
import { BaseFieldProps } from './types'

export const SwitchField: React.FC<BaseFieldProps> = ({
    field,
    value,
    onChange,
    resolveTemplate,
}) => {
    const checked = Boolean(value ?? resolveTemplate(field.value) ?? false)

    return (
        <div className="col-span-12 flex flex-col">
            <div className="flex items-center justify-between">
                <div className="text-sm mb-1 text-canopy-50">{resolveTemplate(field.label)}</div>
                <Switch.Root
                    id={field.id}
                    checked={checked}
                    disabled={field.readOnly}
                    onCheckedChange={(next) => onChange(next)}
                    className="relative h-5 w-9 rounded-full bg-muted data-[state=checked]:bg-emerald-500 outline-none shadow-inner transition-colors"
                    aria-label={String(resolveTemplate(field.label) ?? field.name)}
                >
                    <Switch.Thumb className="block h-4 w-4 translate-x-0.5 rounded-full bg-white shadow transition-transform data-[state=checked]:translate-x-[18px]" />
                </Switch.Root>
            </div>
            {field.help && <span className="text-xs text-muted-foreground">{resolveTemplate(field.help)}</span>}
        </div>
    )
}
