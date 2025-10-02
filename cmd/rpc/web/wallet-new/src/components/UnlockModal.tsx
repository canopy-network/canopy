import React, { useState } from 'react'
import { useSession } from '../state/session'

export default function UnlockModal({ address, ttlSec, open, onClose }:
  { address: string; ttlSec: number; open: boolean; onClose: () => void }) {
  const [pwd, setPwd] = useState('')
  const [err, setErr] = useState<string>('')
  const unlock = useSession(s => s.unlock)
  if (!open) return null

  const submit = async () => {
    if (!pwd) { setErr('Password required'); return }
    unlock(address, pwd, ttlSec)
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 grid place-items-center bg-black/60">
      <div className="w-full max-w-sm bg-neutral-900 border border-neutral-800 rounded-xl p-4">
        <h3 className="text-lg font-semibold mb-2">Unlock wallet</h3>
        <p className="text-sm text-neutral-400 mb-3">Authorize transactions for the next {Math.round(ttlSec/60)} minutes.</p>
        <input
          type="password"
          value={pwd}
          onChange={e=>setPwd(e.target.value)}
          placeholder="Password"
          className="w-full bg-neutral-950 border border-neutral-800 rounded px-3 py-2"
        />
        {err && <div className="text-red-500 text-sm mt-2">{err}</div>}
        <div className="flex justify-end gap-2 mt-4">
          <button onClick={onClose} className="px-3 py-2 bg-neutral-800 rounded">Cancel</button>
          <button onClick={submit} className="px-3 py-2 bg-emerald-500 text-black rounded">Unlock</button>
        </div>
      </div>
    </div>
  )
}
