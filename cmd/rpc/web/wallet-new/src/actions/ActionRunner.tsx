// ActionRunner.tsx
import React from 'react'
import { useConfig } from '../app/providers/ConfigProvider'
import FormRenderer from './FormRenderer'
import Confirm from './Confirm'
import Result from './Result'
import WizardRunner from './WizardRunner'
import { template } from '../core/templater'
import { useResolvedFee } from '../core/fees'
import { useSession, attachIdleRenew } from '../state/session'
import UnlockModal from '../components/UnlockModal'
import useDebouncedValue from "../core/useDebouncedValue";

type Stage = 'form' | 'confirm' | 'executing' | 'result'


export default function ActionRunner({ actionId }: { actionId: string }) {
    const { manifest, chain, isLoading } = useConfig()
    const action = React.useMemo(
        () => manifest?.actions.find((a) => a.id === actionId),
        [manifest, actionId]
    )

    const [stage, setStage] = React.useState<Stage>('form')
    const [form, setForm] = React.useState<Record<string, any>>({})
    const debouncedForm = useDebouncedValue(form, 250)
    const [txRes, setTxRes] = React.useState<any>(null)

    const session = useSession()
    const ttlSec = chain?.session?.unlockTimeoutSec ?? 900
    React.useEffect(() => { attachIdleRenew(ttlSec) }, [ttlSec])

    const requiresAuth =
        (action?.auth?.type ??
            (action?.rpc.base === 'admin' ? 'sessionPassword' : 'none')) === 'sessionPassword'
    const [unlockOpen, setUnlockOpen] = React.useState(false)

    // ✅ el hook de fee depende del form debounced, no del “en vivo”
    const { data: fee, isFetching } = useResolvedFee(action as any, debouncedForm)

    const isReady = React.useMemo(() => !!action && !!chain, [action, chain])
    const isWizard = React.useMemo(() => action?.flow === 'wizard', [action?.flow])

    const onSubmit = React.useCallback(() => setStage('confirm'), [])

    const payload = React.useMemo(
        () => template(action?.rpc.payload ?? {}, {
            form,
            chain,
            session: { password: session.password },
        }),
        [action?.rpc.payload, form, chain, session.password]
    )

    const confirmSummary = React.useMemo(
        () => (action?.confirm?.summary ?? []).map((s) => ({
            label: s.label,
            value: template(s.value, { form, chain, fees: { effective: fee?.amount } }),
        })),
        [action?.confirm?.summary, form, chain, fee?.amount]
    )

    const host = React.useMemo(() => {
        if (!action || !chain) return ''
        return action.rpc.base === 'admin'
            ? chain.rpc.admin ?? chain.rpc.base ?? ''
            : chain.rpc.base ?? ''
    }, [action, chain])

    const doExecute = React.useCallback(async () => {
        if (!isReady) return
        if (requiresAuth && !session.isUnlocked()) { setUnlockOpen(true); return }
        setStage('executing')
        const res = await fetch(host + action!.rpc.path, {
            method: action!.rpc.method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
        }).then((r) => r.json()).catch(() => ({ hash: '0xDEMO' }))
        setTxRes(res)
        setStage('result')
    }, [isReady, requiresAuth, session, host, action, payload])

    React.useEffect(() => {
        if (unlockOpen && session.isUnlocked()) {
            setUnlockOpen(false)
            void doExecute()
        }
    }, [unlockOpen, session, doExecute])

    const onFormChange = React.useCallback((patch: Record<string, any>) => {
        setForm((prev) => ({ ...prev, ...patch }))
    }, [])

    return (
        <div className="space-y-6">
            <div className="bg-neutral-900 border border-neutral-800 rounded p-4">
                <h2 className="text-lg mb-3">{action?.label ?? 'Action'}</h2>

                {isLoading && <div>Loading…</div>}
                {!isLoading && !isReady && <div>No action "{actionId}" found in manifest</div>}
                {!isLoading && isReady && isWizard && <WizardRunner action={action!} />}

                {!isLoading && isReady && !isWizard && (
                    <>
                        {stage === 'form' && (
                            <div className="space-y-4">
                                {action!.form?.fields ? (
                                    <FormRenderer fields={action!.form?.fields ?? []} value={form} onChange={onFormChange} />
                                ) : (
                                    <div>No form for this action</div>
                                )}

                                {/* Línea de fee sin flicker: mantenemos el último valor mientras “isFetching” */}
                                <div className="flex items-center justify-between">
                                    <div className="text-sm text-neutral-400">
                                        Estimated fee:{' '}
                                        {fee
                                            ? <span className={isFetching ? 'opacity-70 transition-opacity' : ''}>
                          {fee.amount} {chain?.denom.symbol}
                        </span>
                                            : '…'}
                                        {isFetching && <span className="ml-2 animate-pulse">calculating…</span>}
                                    </div>
                                    <button onClick={onSubmit} className="px-3 py-2 bg-emerald-500 text-black rounded">Continue</button>
                                </div>
                            </div>
                        )}

                        {stage === 'confirm' && (
                            <Confirm
                                summary={confirmSummary}
                                ctaLabel={action!.confirm?.ctaLabel ?? (action!.id === 'Send' ? 'Send' : 'Confirm')}
                                danger={!!action!.confirm?.danger}
                                showPayload={!!action!.confirm?.showPayload}
                                payload={action!.confirm?.payloadSource === 'rpc.payload' ? payload : action!.confirm?.payloadTemplate}
                                onBack={() => setStage('form')}
                                onConfirm={doExecute}
                            />
                        )}

                        <UnlockModal address={form.address || ''} ttlSec={ttlSec} open={unlockOpen} onClose={() => setUnlockOpen(false)} />

                        {stage === 'result' && (
                            <Result
                                message={template(action!.success?.message ?? 'Done', { form, chain })}
                                link={
                                    action!.success?.links?.[0]
                                        ? { label: action!.success.links[0].label, href: template(action!.success.links[0].href, { result: txRes }) }
                                        : undefined
                                }
                                onDone={() => setStage('form')}
                            />
                        )}
                    </>
                )}
            </div>
        </div>
    )
}
