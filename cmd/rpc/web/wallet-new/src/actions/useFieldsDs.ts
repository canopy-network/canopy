import React from "react";
import {Field} from "@/manifest/types";
import {useDS} from "@/core/useDs";

export function useFieldDs(field: Field, ctx: any) {
    const dsKey = field?.ds ? Object.keys(field.ds)[0] : ''

    if(!dsKey) return {data: null, isLoading: false, error: null, refetch: () => {}}


    const dsParams = field?.ds?.[dsKey] ?? []
    const enabled = Boolean(dsKey)

    const renderedParams = React.useMemo(() => {
        if (!enabled) return null
        return JSON.parse(
            JSON.stringify(dsParams).replace(/{{(.*?)}}/g, (_, k) => {
                const path = k.trim().split('.')
                return path.reduce((acc: { [x: string]: any; }, cur: string | number) => acc?.[cur], ctx)
            })
        )
    }, [dsParams, ctx, enabled])


    const { data, isLoading, error, refetch } = useDS(dsKey, renderedParams, {refetchIntervalMs: 3000, enabled })

    return { data, isLoading, error, refetch }
}
