import React from "react";
import { Field } from "@/manifest/types";
import { useDS } from "@/core/useDs";

export function useFieldDs(field: Field, ctx: any) {
    const dsKey = React.useMemo(() => {
        const k = field?.ds ? Object.keys(field.ds)[0] : null;
        return typeof k === "string" ? k : null;
    }, [field]);

    const enabled = !!dsKey;

    const dsParams = React.useMemo(() => {
        if (!enabled) return [];
        // @ts-ignore: dsKey no es null cuando enabled = true
        return field.ds[dsKey] ?? [];
    }, [enabled, field, dsKey]);

    const renderedParams = React.useMemo(() => {
        if (!enabled) return {};
        try {
            const json = JSON.stringify(dsParams).replace(/{{(.*?)}}/g, (_, k) => {
                const path = k.trim().split(".");
                const v = path.reduce((acc: any, cur: string) => acc?.[cur], ctx);
                return v ?? ""; // evita 'undefined'
            });
            return JSON.parse(json);
        } catch {
            return {};
        }
    }, [dsParams, ctx, enabled]);

    const { data, isLoading, error, refetch } = useDS(dsKey ?? "__disabled__", renderedParams, {
        refetchIntervalMs: 3000,
        enabled,
    });

    return {
        data: enabled ? data : null,
        isLoading: enabled ? isLoading : false,
        error: enabled ? error : null,
        refetch,
    };
}
