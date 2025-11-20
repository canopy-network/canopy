import React from 'react'
import { FieldWrapper } from './FieldWrapper'
import { BaseFieldProps } from './types'

export const DynamicHtmlField: React.FC<BaseFieldProps> = ({
    field,
    error,
    templateContext,
    resolveTemplate,
}) => {
    const resolvedHtml = resolveTemplate((field as any).html)

    return (
        <FieldWrapper field={field} error={error} templateContext={templateContext} resolveTemplate={resolveTemplate}>
            <div
                className="text-sm text-text-muted w-full"
                dangerouslySetInnerHTML={{ __html: resolvedHtml ?? '' }}
            />
        </FieldWrapper>
    )
}
