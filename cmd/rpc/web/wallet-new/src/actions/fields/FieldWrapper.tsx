import React from 'react'
import { cx } from '@/ui/cx'
import { spanClasses } from '@/actions/utils/fieldHelpers'
import { FieldFeatures } from '@/actions/components/FieldFeatures'
import { FieldWrapperProps } from './types'

export const FieldWrapper: React.FC<FieldWrapperProps> = ({
    field,
    error,
    templateContext,
    resolveTemplate,
    hasFeatures,
    setVal,
    children,
    currentValue,
}) => {
    const help = error || resolveTemplate(field.help)

    return (
        <div className={spanClasses(field, templateContext?.layout)}>
            <label className="block">
                {resolveTemplate(field.label) && (
                    <div className="text-[11px] sm:text-xs uppercase tracking-[0.08em] mb-1.5 text-muted-foreground/90">
                        {resolveTemplate(field.label)}
                    </div>
                )}
                <div className="relative">
                    {children}
                    {hasFeatures && field.features && setVal && (
                        <FieldFeatures
                            fieldId={field.name}
                            features={field.features}
                            ctx={templateContext}
                            setVal={setVal}
                            currentValue={currentValue}
                        />
                    )}
                </div>
                {help && (
                    <div
                        className={cx(
                            'text-[11px] sm:text-xs mt-1.5 break-words overflow-wrap-anywhere',
                            error ? 'text-red-400' : 'text-muted-foreground'
                        )}
                    >
                        {help}
                    </div>
                )}
            </label>
        </div>
    )
}
