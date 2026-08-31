import { Field } from '@/manifest/types'
import React from 'react'

export type BaseFieldProps = {
    field: Field
    value: any
    error?: string
    templateContext: Record<string, any>
    dsValue?: any
    onChange: (value: any) => void
    resolveTemplate: (s?: any) => any
    setVal?: (fieldId: string, v: any) => void
}

export type FieldWrapperProps = {
    field: Field
    error?: string
    templateContext: Record<string, any>
    resolveTemplate: (s?: any) => any
    hasFeatures?: boolean
    setVal?: (fieldId: string, v: any) => void
    children: React.ReactNode
    currentValue?: any
}
