// ActionRunner.tsx
import React from 'react'
import {useConfig} from '@/app/providers/ConfigProvider'
import FormRenderer from './FormRenderer'
import {useResolvedFees} from '@/core/fees'
import {useSession, attachIdleRenew} from '@/state/session'
import UnlockModal from '../components/UnlockModal'
import useDebouncedValue from "../core/useDebouncedValue";
import {
    getFieldsFromAction,
    normalizeFormForAction,
    buildPayloadFromAction,
} from '@/core/actionForm'
import {useAccounts} from '@/app/providers/AccountsProvider'
import {template, templateBool} from '@/core/templater'
import { resolveToastFromManifest, resolveRedirectFromManifest } from "@/toast/manifestRuntime";
import { useToast } from "@/toast/ToastContext";
import { genericResultMap } from "@/toast/mappers";
import {LucideIcon} from "@/components/ui/LucideIcon";
import {cx} from "@/ui/cx";
import {motion} from "framer-motion";
import {ToastTemplateOptions} from "@/toast/types";
import {useActionDs} from './useActionDs';



type Stage = 'form' | 'confirm' | 'executing' | 'result'


export default function ActionRunner({actionId, onFinish, className}: { actionId: string, onFinish?: () => void, className?: string}) {
    const toast = useToast();


    const [formHasErrors, setFormHasErrors] = React.useState(false)
    const [stage, setStage] = React.useState<Stage>('form')
    const [form, setForm] = React.useState<Record<string, any>>({})
    const debouncedForm = useDebouncedValue(form, 250)
    const [txRes, setTxRes] = React.useState<any>(null)
    const [localDs, setLocalDs] = React.useState<Record<string, any>>({})
    // Track which fields have been auto-populated at least once
    const [autoPopulatedOnce, setAutoPopulatedOnce] = React.useState<Set<string>>(new Set())

    const {manifest, chain, params, isLoading} = useConfig()
    const {selectedAccount} = useAccounts?.() ?? {selectedAccount: undefined}
    const session = useSession()

    const action = React.useMemo(
        () => manifest?.actions.find((a) => a.id === actionId),
        [manifest, actionId]
    )

    // NEW: Load action-level DS (replaces per-field DS for better performance)
    const actionDsConfig = React.useMemo(() => (action as any)?.ds, [action]);

    // Build context for DS (without ds itself to avoid circular dependency)
    const dsCtx = React.useMemo(() => ({
        form,
        chain,
        account: selectedAccount ? {
            address: selectedAccount.address,
            nickname: selectedAccount.nickname,
            pubKey: selectedAccount.publicKey,
        } : undefined,
        params,
    }), [form, chain, selectedAccount, params]);

    const { ds: actionDs } = useActionDs(
        actionDsConfig,
        dsCtx,
        actionId,
        selectedAccount?.address
    );

    // Merge action-level DS with field-level DS (for backwards compatibility)
    const mergedDs = React.useMemo(() => ({
        ...actionDs,
        ...localDs,
    }), [actionDs, localDs]);
    const feesResolved = useResolvedFees(chain?.fees, {
        actionId: action?.id,
        bucket: 'avg',
        ctx: {chain}
    })


    const ttlSec = chain?.session?.unlockTimeoutSec ?? 900
    React.useEffect(() => {
        attachIdleRenew(ttlSec)
    }, [ttlSec])

    const requiresAuth =
        (action?.auth?.type ??
            (action?.submit?.base === 'admin' ? 'sessionPassword' : 'none')) === 'sessionPassword'
    const [unlockOpen, setUnlockOpen] = React.useState(false)



    const templatingCtx = React.useMemo(() => ({
        form: debouncedForm,
        layout: (action as any)?.form?.layout,
        chain,
        account: selectedAccount ? {
            address: selectedAccount.address,
            nickname: selectedAccount.nickname,
            pubKey: selectedAccount.publicKey,
        } : undefined,
        fees: {
            ...feesResolved
        },
        params: {
            ...params
        },
        ds: mergedDs,  // Use merged DS (action-level + field-level)
        session: {password: session?.password},
        // Unique scope for this action instance to prevent cache collisions
        __scope: `action:${actionId}:${selectedAccount?.address || 'no-account'}`,
    }), [debouncedForm, chain, selectedAccount, feesResolved, session?.password, params, mergedDs, actionId])



    const infoItems = React.useMemo(
        () =>
            (action?.form as any)?.info?.items?.map((it: any) => ({
                label: typeof it.label === 'string' ? template(it.label, templatingCtx) : it.label,
                icon: it.icon,
                value: typeof it.value === 'string' ? template(it.value, templatingCtx) : it.value,
            })) ?? [],
        [action, templatingCtx]
    );

    const rawSummary = React.useMemo(() => {
        const formSum = (action as any)?.form?.confirmation?.summary
        return Array.isArray(formSum) ? formSum : []
    }, [action])

    const summaryTitle = React.useMemo(() => {
        const title = (action as any)?.form?.confirmation?.title
        return typeof title === 'string' ? template(title, templatingCtx) : title
    }, [action, templatingCtx])

    const resolvedSummary = React.useMemo(() => {
        return rawSummary.map((item: any) => ({
            label: typeof item.label === 'string' ? template(item.label, templatingCtx) : item.label,
            icon: item.icon, // opcional
            value: typeof item.value === 'string' ? template(item.value, templatingCtx) : item.value,
        }))
    }, [rawSummary, templatingCtx])

    const hasSummary = resolvedSummary.length > 0

    const confirmBtn = React.useMemo(() => {
        const btn = (action as any)?.form?.confirmation?.btns?.submit
            ?? (action as any)?.form?.confirmation?.btn
            ?? {}
        return {
            label: typeof btn.label === 'string' ? template(btn.label, templatingCtx) : (btn.label ?? 'Confirm'),
            icon: btn.icon ?? undefined,
        }
    }, [action, templatingCtx])

    const isReady = React.useMemo(() => !!action && !!chain, [action, chain])


    const didInitToastRef = React.useRef(false);
    React.useEffect(() => {
        if (!action || !isReady) return;
        if (didInitToastRef.current) return;
        const t = resolveToastFromManifest(action, "onInit", templatingCtx);
        if (t) toast.neutral(t);
        didInitToastRef.current = true;
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [action, isReady]);

    const normForm = React.useMemo(() => normalizeFormForAction(action as any, debouncedForm), [action, debouncedForm])
    const payload = React.useMemo(
        () => buildPayloadFromAction(action as any, {
            form: normForm,
            chain,
            session: {password: session.password},
            account: selectedAccount ? {
                address: selectedAccount.address,
                nickname: selectedAccount.nickname,
                pubKey: selectedAccount.publicKey,
            } : undefined,
            fees: {
                ...feesResolved
            },
            ds: mergedDs,
        }),
        [action, normForm, chain, session.password, feesResolved, selectedAccount, mergedDs]
    )

    const host = React.useMemo(() => {
        if (!action || !chain) return ''
        return action?.submit?.base === 'admin'
            ? chain.rpc.admin ?? chain.rpc.base ?? ''
            : chain.rpc.base ?? ''
    }, [action, chain])


    const doExecute = React.useCallback(async () => {
        if (!isReady) return
        if (requiresAuth && !session.isUnlocked()) {
            setUnlockOpen(true);
            return
        }
        const before = resolveToastFromManifest(action, "onBeforeSubmit", templatingCtx);
        if (before) toast.neutral(before);
        setStage('executing')
        const submitPath = typeof action!.submit?.path === 'string'
            ? template(action!.submit.path, templatingCtx)
            : action!.submit?.path
        const res = await fetch(host + submitPath, {
            method: action!.submit?.method,
            headers: action!.submit?.headers ?? {'Content-Type': 'application/json'},
            body: JSON.stringify(payload),
        }).then((r) => r.json())
        setTxRes(res)

        const key = (res?.ok ?? true) ? "onSuccess" : "onError";
        const t = resolveToastFromManifest(action, key as any, templatingCtx, res);

        if (t) {
            toast.toast(t);
        } else {
            toast.fromResult({
                result: res,
                ctx: templatingCtx,
                map: (r, c) => genericResultMap(r, c),
                fallback: { title: "Processed", variant: "neutral", ctx: templatingCtx } as ToastTemplateOptions
            })
        }
        const fin = resolveToastFromManifest(action, "onFinally", templatingCtx, res);
        if (fin) toast.info(fin);
        setStage('result')
        if (onFinish) onFinish()
    }, [isReady, requiresAuth, session, host, action, payload])

    const onContinue = React.useCallback(() => {
        if (formHasErrors) {
            // opcional: mostrar toast o vibrar el botón
            return
        }
        if (hasSummary) {
            setStage('confirm')
        } else {
            void doExecute()
        }
    }, [formHasErrors, hasSummary, doExecute])

    const onConfirm = React.useCallback(() => {
        if (formHasErrors) {
            // opcional: toast
            return
        }
        void doExecute()
    }, [formHasErrors, doExecute])

    const onBackToForm = React.useCallback(() => {
        setStage('form')
    }, [])


    React.useEffect(() => {
        if (unlockOpen && session.isUnlocked()) {
            setUnlockOpen(false)
            void doExecute()
        }
    }, [unlockOpen, session])

    const onFormChange = React.useCallback((patch: Record<string, any>) => {
        setForm((prev) => ({...prev, ...patch}))
    }, [])

    const [errorsMap, setErrorsMap] = React.useState<Record<string,string>>({})
    const [stepIdx, setStepIdx] = React.useState(0)

    const wizard = React.useMemo(() => (action as any)?.form?.wizard, [action])
    const allFields = React.useMemo(() => getFieldsFromAction(action), [action])


    const steps = React.useMemo(() => {
        if (!wizard) return []
        const declared = Array.isArray(wizard.steps) ? wizard.steps : []
        if (declared.length) return declared
        const uniq = Array.from(new Set(allFields.map((f:any)=>f.step).filter(Boolean)))
        return uniq.map((id:any,i)=>({ id, title: `Step ${i+1}` }))
    }, [wizard, allFields])

    const fieldsForStep = React.useMemo(() => {
        if (!wizard || !steps.length) return allFields
        const cur = steps[stepIdx]?.id ?? (stepIdx+1)
        return allFields.filter((f:any)=> (f.step ?? 1) === cur || String(f.step) === String(cur))
    }, [wizard, steps, stepIdx, allFields])


    const visibleFieldsForStep = React.useMemo(() => {
        const list = fieldsForStep ?? []
        return list.filter((f: any) => {
            if (!f?.showIf) return true
            try {
                return templateBool(f.showIf, { ...templatingCtx, form })
            } catch (e) {
                console.warn('Error evaluating showIf', f.name, e)
                return true
            }
        })
    }, [fieldsForStep, templatingCtx, form])

    // Auto-populate form with default values from field.value when DS data or visible fields change
    const prevStateRef = React.useRef<{ ds: string; fieldNames: string }>({ ds: '', fieldNames: '' })
    React.useEffect(() => {
        const dsSnapshot = JSON.stringify(mergedDs)
        const fieldNamesSnapshot = visibleFieldsForStep.map((f: any) => f.name).join(',')
        const stateSnapshot = { ds: dsSnapshot, fieldNames: fieldNamesSnapshot }

        // Only run when DS or visible fields change
        if (prevStateRef.current.ds === dsSnapshot && prevStateRef.current.fieldNames === fieldNamesSnapshot) {
            return
        }
        prevStateRef.current = stateSnapshot

        setForm(prev => {
            const defaults: Record<string, any> = {}
            let hasDefaults = false

            // Build template context with current form state
            const ctx = {
                form: prev,
                chain,
                account: selectedAccount ? {
                    address: selectedAccount.address,
                    nickname: selectedAccount.nickname,
                    pubKey: selectedAccount.publicKey,
                } : undefined,
                fees: { ...feesResolved },
                params: { ...params },
                ds: mergedDs,
            }

            for (const field of visibleFieldsForStep) {
                const fieldName = (field as any).name
                const fieldValue = (field as any).value
                const autoPopulate = (field as any).autoPopulate ?? 'always' // 'always' | 'once' | false

                // Skip auto-population if field has autoPopulate: false
                if (autoPopulate === false) {
                    continue
                }

                // Skip if autoPopulate: 'once' and field was already populated
                if (autoPopulate === 'once' && autoPopulatedOnce.has(fieldName)) {
                    continue
                }

                // Only set default if form doesn't have a value and field has a default
                if (fieldValue != null && (prev[fieldName] === undefined || prev[fieldName] === '' || prev[fieldName] === null)) {
                    try {
                        const resolved = template(fieldValue, ctx)
                        if (resolved !== undefined && resolved !== '' && resolved !== null) {
                            defaults[fieldName] = resolved
                            hasDefaults = true

                            // Mark as populated if autoPopulate is 'once'
                            if (autoPopulate === 'once') {
                                setAutoPopulatedOnce(prev => new Set([...prev, fieldName]))
                            }
                        }
                    } catch (e) {
                        // Template resolution failed, skip
                    }
                }
            }

            return hasDefaults ? { ...prev, ...defaults } : prev
        })
    }, [mergedDs, visibleFieldsForStep, chain, selectedAccount, feesResolved, params])

    const handleErrorsChange = React.useCallback((errs: Record<string,string>, hasErrors: boolean) => {
        setErrorsMap(errs)
        setFormHasErrors(hasErrors)
    }, [])

    const hasStepErrors = React.useMemo(() => {
        const missingRequired = visibleFieldsForStep.some((f:any) =>
            f.required && (form[f.name] == null || form[f.name] === '')
        );
        const fieldErrors = visibleFieldsForStep.some((f:any) => !!errorsMap[f.name]);
        return missingRequired || fieldErrors;
    }, [visibleFieldsForStep, form, errorsMap]);

    const isLastStep = !wizard || stepIdx >= (steps.length - 1)


    const goNext = React.useCallback(() => {
        if (hasStepErrors) return
        if (!wizard || isLastStep) {
            if (hasSummary) setStage('confirm'); else void doExecute()
        } else {
            setStepIdx(i => i + 1)
        }
    }, [wizard, isLastStep, hasStepErrors, hasSummary, doExecute])

    const goPrev = React.useCallback(() => {
        if (!wizard) return
        setStepIdx(i => Math.max(0, i - 1))
    }, [wizard])


    return (
        <div className="space-y-6">
            {
                stage === 'confirm' && (
                    <button onClick={onBackToForm} className="flex  justify-between items-center gap-2 z-10 p-1 font-bold text-canopy-50 ">
                        <LucideIcon name="arrow-left" />
                        Go back
                    </button>
                )

            }
            <div className={cx("flex flex-col gap-4", className)}>

                {isLoading && <div>Loading…</div>}
                {!isLoading && !isReady && <div>No action "{actionId}" found in manifest</div>}

                {!isLoading && isReady && (
                    <>
                        {
                            stage === 'form' && (
                                <motion.div className="space-y-4">
                                    <FormRenderer
                                        fields={visibleFieldsForStep}
                                        value={form}
                                        onChange={onFormChange}
                                        ctx={templatingCtx}
                                        onErrorsChange={handleErrorsChange}
                                        onDsChange={setLocalDs}
                                    />

                                    {wizard && steps.length > 0 && (
                                        <div className="flex items-center justify-between text-xs text-neutral-400">
                                            <div>{steps[stepIdx]?.title ?? `Step ${stepIdx+1}`}</div>
                                            <div>{stepIdx+1} / {steps.length}</div>
                                        </div>
                                    )}


                                    {infoItems.length > 0 && (
                                        <div className="flex-col h-full p-4 rounded-lg bg-bg-primary">
                                            {action?.form?.info?.title && (
                                                <h4 className="text-canopy-50">{template(action?.form?.info?.title, templatingCtx)}</h4>
                                            )}
                                            <div className="mt-3 space-y-2">
                                                {infoItems.map((d: { icon: string | undefined; label: string | number | boolean | React.ReactElement<any, string | React.JSXElementConstructor<any>> | Iterable<React.ReactNode> | React.ReactPortal | null | undefined; value: any }, i: React.Key | null | undefined) => (
                                                    <div key={i} className="flex items-center justify-between text-md">
                                                        <div
                                                            className="flex items-center gap-2 text-neutral-400 font-light">
                                                            {d.icon ?
                                                                <LucideIcon name={d.icon} className="w-4 h-4"/> : null}
                                                            <span>
                                                            {d.label}
                                                                {d.value && (':')}
                                                        </span>
                                                        </div>
                                                        {d.value && (<span
                                                            className="font-normal text-canopy-50">{String(d.value ?? '—')}</span>)}

                                                    </div>
                                                ))}
                                            </div>
                                        </div>
                                    )}


                                    <div className="flex gap-2">
                                        {wizard && stepIdx > 0 && (
                                            <button onClick={goPrev} className="px-3 py-2 rounded border border-muted text-canopy-50">Back</button>
                                        )}
                                        <button
                                            disabled={hasStepErrors}
                                            onClick={goNext}
                                            className={cx("flex-1 px-3 py-2 bg-primary-500 text-bg-accent-foreground font-bold rounded",
                                                hasStepErrors && "opacity-50 cursor-not-allowed"
                                            )}
                                        >
                                            {(!wizard || isLastStep) ? 'Continue' : 'Next'}
                                        </button>
                                    </div>

                                </motion.div>
                            )}

                        {stage === 'confirm' && (
                            <motion.div className="space-y-4">
                                <div className="flex-col h-full p-4 rounded-lg bg-bg-primary">
                                    {summaryTitle && (
                                        <h4 className="text-canopy-50">{summaryTitle}</h4>
                                    )}

                                    <div className="mt-3 space-y-2">
                                        {resolvedSummary.map((d, i) => (
                                            <div key={i} className="flex items-center justify-between text-md">
                                                <div className="flex items-center gap-2 text-neutral-400 font-light">
                                                    {d.icon ? <LucideIcon name={d.icon} className="w-4 h-4"/> : null}
                                                    <span>{d.label}:</span>
                                                </div>
                                                <span
                                                    className="font-normal text-canopy-50">{String(d.value ?? '—')}</span>
                                            </div>
                                        ))}
                                    </div>
                                </div>

                                <div className="flex flex-col gap-2">
                                    <button
                                        onClick={onConfirm}
                                        className="flex-1 px-3 py-2 bg-primary-500 text-bg-accent-foreground font-bold rounded flex items-center justify-center gap-2"
                                    >
                                        {confirmBtn.icon ?
                                            <LucideIcon name={confirmBtn.icon} className="w-4 h-4"/> : null}
                                        <span>{confirmBtn.label}</span>
                                    </button>
                                </div>
                            </motion.div>
                        )}

                        <UnlockModal address={form.address || ''} ttlSec={ttlSec} open={unlockOpen}
                                     onClose={() => setUnlockOpen(false)}/>

                    </>
                )}
            </div>
        </div>
    )
}
