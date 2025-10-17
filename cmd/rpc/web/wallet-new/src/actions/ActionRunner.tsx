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
import {template} from '@/core/templater'
import {LucideIcon} from "@/components/ui/LucideIcon";
import {motion} from "framer-motion";
import {cx} from "@/ui/cx";


type Stage = 'form' | 'confirm' | 'executing' | 'result'


export default function ActionRunner({actionId}: { actionId: string }) {
    const [formHasErrors, setFormHasErrors] = React.useState(false)
    const [stage, setStage] = React.useState<Stage>('form')
    const [form, setForm] = React.useState<Record<string, any>>({})
    const debouncedForm = useDebouncedValue(form, 250)
    const [txRes, setTxRes] = React.useState<any>(null)

    const {manifest, chain, isLoading} = useConfig()
    const {selectedAccount} = useAccounts?.() ?? {selectedAccount: undefined}
    const session = useSession()

    const action = React.useMemo(
        () => manifest?.actions.find((a) => a.id === actionId),
        [manifest, actionId]
    )
    const feesResolved = useResolvedFees(chain?.fees, {
        actionId: action?.id,
        bucket: 'avg',
        ctx: {chain}
    })

    const handleErrorsChange = React.useCallback((errs: Record<string,string>, hasErrors: boolean) => {
        setFormHasErrors(hasErrors)
    }, [])

    const fields = React.useMemo(() => getFieldsFromAction(action), [action])


    const ttlSec = chain?.session?.unlockTimeoutSec ?? 900
    React.useEffect(() => {
        attachIdleRenew(ttlSec)
    }, [ttlSec])

    const requiresAuth =
        (action?.auth?.type ??
            (action?.submit?.base === 'admin' ? 'sessionPassword' : 'none')) === 'sessionPassword'
    const [unlockOpen, setUnlockOpen] = React.useState(false)


    const templatingCtx = React.useMemo(() => ({
        form,
        chain,
        account: selectedAccount ? {
            address: selectedAccount.address,
            nickname: selectedAccount.nickname,
        } : undefined,
        fees: {
            ...feesResolved
        },
        session: {password: session?.password},
    }), [form, chain, selectedAccount, feesResolved, session?.password])

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
        return (action as any)?.form?.confirmation?.summary?.title
    }, [action])

    const resolvedSummary = React.useMemo(() => {
        return rawSummary.map((item: any) => ({
            label: item.label,
            icon: item.icon, // opcional
            value: typeof item.value === 'string' ? template(item.value, templatingCtx) : item.value,
        }))
    }, [rawSummary, templatingCtx])

    const hasSummary = resolvedSummary.length > 0

    const confirmBtn = React.useMemo(() => {
        const btn = (action as any)?.form?.confirmation?.btns?.submit
            ?? {}
        return {
            label: btn.label ?? 'Confirm',
            icon: btn.icon ?? undefined,
        }
    }, [action])

    const isReady = React.useMemo(() => !!action && !!chain, [action, chain])


    const normForm = React.useMemo(() => normalizeFormForAction(action as any, debouncedForm), [action, debouncedForm])
    const payload = React.useMemo(
        () => buildPayloadFromAction(action as any, {
            form: normForm,
            chain,
            session: {password: session.password},
            account: selectedAccount ? {
                address: selectedAccount.address,
                nickname: selectedAccount.nickname,
            } : undefined,
            fees: {
                ...feesResolved
            },
        }),
        [action, normForm, chain, session.password, feesResolved]
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
        setStage('executing')
        const res = await fetch(host + action!.submit?.path, {
            method: action!.submit?.method,
            headers: action!.submit?.headers ?? {'Content-Type': 'application/json'},
            body: JSON.stringify(payload),
        }).then((r) => r.json()).catch(() => ({hash: '0xDEMO'}))
        setTxRes(res)
        setStage('result')
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
            <div>

                {isLoading && <div>Loading…</div>}
                {!isLoading && !isReady && <div>No action "{actionId}" found in manifest</div>}

                {!isLoading && isReady && (
                    <>
                        {
                            stage === 'form' && (
                                <motion.div className="space-y-4">
                                    <FormRenderer fields={fields} value={form} onChange={onFormChange} ctx={templatingCtx} onErrorsChange={handleErrorsChange}/>


                                    {infoItems.length > 0 && (
                                        <div className="flex-col h-full p-4 rounded-lg bg-bg-primary">
                                            {action?.form?.info?.title && (
                                                <h4 className="text-canopy-50">{action?.form?.info?.title}</h4>
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
                                    {action?.submit && (
                                        <button
                                            disabled={formHasErrors}
                                            onClick={onContinue}
                                            className={cx("w-full px-3 py-2 bg-primary-500 text-bg-accent-foreground font-bold rounded",
                                                formHasErrors && "opacity-50 cursor-not-allowed cursor-not-allowed"
                                            )}>
                                            Continue
                                        </button>
                                    )}

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
