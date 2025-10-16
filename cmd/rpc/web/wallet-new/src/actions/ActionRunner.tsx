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
import {microToDisplay} from "@/core/format";
import { useAccounts } from '@/app/providers/AccountsProvider'



type Stage = 'form' | 'confirm' | 'executing' | 'result'


export default function ActionRunner({actionId}: { actionId: string }) {
    const {manifest, chain, isLoading} = useConfig()
    const { selectedAccount } = useAccounts?.() ?? { selectedAccount: undefined }

    const action = React.useMemo(
        () => manifest?.actions.find((a) => a.id === actionId),
        [manifest, actionId]
    )

    const fields = React.useMemo(() => getFieldsFromAction(action), [action])

    const [stage, setStage] = React.useState<Stage>('form')
    const [form, setForm] = React.useState<Record<string, any>>({})
    const debouncedForm = useDebouncedValue(form, 250)
    const [txRes, setTxRes] = React.useState<any>(null)

    const session = useSession()
    const ttlSec = chain?.session?.unlockTimeoutSec ?? 900
    React.useEffect(() => {
        attachIdleRenew(ttlSec)
    }, [ttlSec])

    const requiresAuth =
        (action?.auth?.type ??
            (action?.submit?.base === 'admin' ? 'sessionPassword' : 'none')) === 'sessionPassword'
    const [unlockOpen, setUnlockOpen] = React.useState(false)

    const feesResolved = useResolvedFees(chain?.fees, {
        actionId: action?.id,
        bucket: 'avg',
        ctx: { chain }
    })

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
        session: { password: session?.password },
    }), [form, chain, selectedAccount, feesResolved, session?.password])

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
            headers:  action!.submit?.headers ?? {'Content-Type': 'application/json'},
            body: JSON.stringify(payload),
        }).then((r) => r.json()).catch(() => ({hash: '0xDEMO'}))
        setTxRes(res)
        setStage('result')
    }, [isReady, requiresAuth, session, host, action, payload])


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
            <div className="">

                {isLoading && <div>Loading…</div>}
                {!isLoading && !isReady && <div>No action "{actionId}" found in manifest</div>}

                {!isLoading && isReady  && (
                    <>
                        {stage === 'form' && (
                            <div className="space-y-4">
                                <FormRenderer fields={fields} value={form} onChange={onFormChange} ctx={templatingCtx}/>

                                <div className="flex-col h-full p-4 rounded-lg bg-bg-primary items-center">
                                    <h4 className={'text-canopy-50'}>Network Fee</h4>
                                    <div className="flex text-md  mt-3 justify-between">
                                        <span className={'text-neutral-400 font-light'}>
                                            Estimated fee:
                                        </span>
                                        {feesResolved
                                            ?
                                         <span className={'font-normal text-canopy-50'}>
                                                {microToDisplay(Number(feesResolved.amount), chain?.denom?.decimals ?? 6)} {chain?.denom.symbol}
                                        </span>
                                            : '…'}
                                    </div>
                                </div>
                                <button onClick={doExecute}
                                        className="w-full px-3 py-2 bg-primary-500 text-bg-accent-foreground font-bold rounded">Continue
                                </button>
                            </div>
                        )}


                        <UnlockModal address={form.address || ''} ttlSec={ttlSec} open={unlockOpen}
                                     onClose={() => setUnlockOpen(false)}/>

                    </>
                )}
            </div>
        </div>
    )
}
