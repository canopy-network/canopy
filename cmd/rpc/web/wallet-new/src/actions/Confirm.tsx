import React from 'react'
import { cx } from '../ui/cx'

function ConfirmInner({
  summary, payload, showPayload = false, ctaLabel = 'Confirm', danger = false, onBack, onConfirm
}: {
  summary: { label: string; value: string }[]
  payload?: any
  showPayload?: boolean
  ctaLabel?: string
  danger?: boolean
  onBack: () => void
  onConfirm: () => void
}) {
  const [open, setOpen] = React.useState(showPayload)

  return (
    <div className="space-y-4">
      <div className="bg-neutral-900 border border-neutral-800 rounded p-4">
        <ul className="space-y-2">
          {summary.map((s, i) => (
            <li key={i} className="grid grid-cols-3 gap-2">
              <span className="text-neutral-400 col-span-1">{s.label}</span>
              <span className="col-span-2">{s.value}</span>
            </li>
          ))}
        </ul>
      </div>

      {payload != null && (
        <div className="bg-neutral-900 border border-neutral-800 rounded">
          <div className="flex items-center justify-between px-4 py-2">
            <div className="text-sm text-neutral-300">Raw Payload</div>
            <button className="text-sm text-emerald-400" onClick={()=>setOpen(!open)}>{open ? 'Hide' : 'Show'}</button>
          </div>
          {open && (
            <pre className="px-4 py-3 text-xs overflow-auto border-t border-neutral-800">
{JSON.stringify(payload, null, 2)}
            </pre>
          )}
        </div>
      )}

      <div className="flex gap-2">
        <button onClick={onBack} className="px-3 py-2 bg-neutral-800 rounded">Back</button>
        <button
          onClick={onConfirm}
          className={cx(
            'px-3 py-2 text-black rounded',
            danger ? 'bg-red-500 hover:bg-red-400' : 'bg-emerald-500 hover:bg-emerald-400'
          )}
        >
          {ctaLabel}
        </button>
      </div>
    </div>
  )
}

export default React.memo(ConfirmInner);



