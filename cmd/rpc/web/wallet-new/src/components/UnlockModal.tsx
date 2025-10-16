import React, {useState} from 'react'
import {useSession} from '../state/session'
import {LockOpenIcon, XIcon} from "lucide-react";

export default function UnlockModal({address, ttlSec, open, onClose}:
                                    { address: string; ttlSec: number; open: boolean; onClose: () => void }) {
    const [pwd, setPwd] = useState('')
    const [err, setErr] = useState<string>('')
    const unlock = useSession(s => s.unlock)
    if (!open) return null

    const submit = async () => {
        if (!pwd) {
            setErr('Password required');
            return
        }
        unlock(address, pwd, ttlSec)
        onClose()
    }

    return (
        <div className="fixed inset-0 z-50 grid place-items-center bg-black/60">
            <div className="w-full max-w-sm bg-bg-secondary border border-neutral-800 rounded-xl p-4">
                <h3 className="text-lg text-canopy-50 font-semibold mb-2">Unlock wallet</h3>
                <p className="text-sm text-neutral-400 mb-3">Authorize transactions for the
                    next {Math.round(ttlSec / 60)} minutes.</p>
                <input
                    type="password"
                    value={pwd}
                    onChange={e => setPwd(e.target.value)}
                    placeholder="Password"
                    className="w-full bg-transparent text-canopy-50 border border-muted rounded-md px-3 py-2"
                />
                {err && <div className="text-status-error text-sm mt-2">{err}</div>}
                <div className="flex justify-end gap-2 mt-4">
                    <button onClick={onClose}
                            className=" flex items-center gap-2 px-3 py-2 bg-muted text-canopy-50 rounded-lg hover:bg-muted/50 ">
                        <XIcon className={'w-4'}/>
                        Cancel
                    </button>

                    <button onClick={submit}
                            className="flex justify-center items-center gap-2 px-3 py-2 bg-primary text-bg-primary rounded-lg font-bold hover:bg-primary/50">
                        <LockOpenIcon className={'w-4'}/>
                        Unlock
                    </button>

                </div>
            </div>
        </div>
    )
}
