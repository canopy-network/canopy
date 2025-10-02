import { normalizeEvmAddress } from '../core/address'

export type FieldError = { name: string; message: string }

export async function validateField(f: any, value: any, ctx: any): Promise<FieldError | null> {
  const rules = f.rules ?? {}
  if (f.required && (value === '' || value == null)) return { name: f.name, message: 'Required' }
  if (f.type === 'number' && value !== '' && value != null) {
    const n = Number(value)
    if (Number.isNaN(n)) return { name: f.name, message: 'Invalid number' }
    if (rules.min != null && n < rules.min) return { name: f.name, message: `Min ${rules.min}` }
    if (rules.max != null && n > rules.max) return { name: f.name, message: `Max ${rules.max}` }
    if (rules.gt != null && !(n > rules.gt)) return { name: f.name, message: `Must be > ${rules.gt}` }
    if (rules.lt != null && !(n < rules.lt)) return { name: f.name, message: `Must be < ${rules.lt}` }
  }
  if (f.type === 'address' || rules.address) {
    const { ok } = normalizeEvmAddress(String(value || ''))
    if (!ok) return { name: f.name, message: 'Invalid address' }
  }
  if (rules.regex) {
    try { if (!(new RegExp(rules.regex).test(String(value ?? '')))) return { name: f.name, message: rules.message ?? 'Invalid format' } }
    catch {}
  }
  if (rules.remote && value) {
    const host = rules.remote.base === 'admin' ? ctx.chain.rpc.admin : ctx.chain.rpc.base
    const res = await fetch(host + rules.remote.path, {
      method: rules.remote.method ?? 'GET',
      headers: { 'Content-Type': 'application/json' },
      body: rules.remote.method === 'POST' ? JSON.stringify(rules.remote.body ?? {}) : undefined
    }).then(r => r.json()).catch(() => ({}))
    const ok = rules.remote.selector ? !!rules.remote.selector.split('.').reduce((a: any,k:string)=>a?.[k],res) : !!res
    if (!ok) return { name: f.name, message: rules.message ?? 'Remote validation failed' }
  }
  return null
}
