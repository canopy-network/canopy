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
}) => {
    const help = error || resolveTemplate(field.help)

    return (
        <div className={spanClasses(field, templateContext?.layout)}>
            <label className="block">
                {resolveTemplate(field.label) && (
                    <div className="text-sm mb-1 text-text-muted">{resolveTemplate(field.label)}</div>
                )}
                <div className="relative">
                    {children}
                    {hasFeatures && field.features && setVal && (
                        <FieldFeatures
                            fieldId={field.name}
                            features={field.features}
                            ctx={templateContext}
                            setVal={setVal}
                        />
                    )}
                </div>
                {help && (
                    <div
                        className={cx(
                            'text-xs mt-1 break-words',
                            error ? 'text-red-400' : 'text-text-muted'
                        )}
                    >
                        {help}
                    </div>
                )}
            </label>
        </div>
    )
}
