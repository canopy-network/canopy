export function template(input: any, ctx: Record<string, any>): any {
  if (input == null) return input
  if (typeof input === 'string') {
    return input.replace(/\{\{\s*([^}]+)\s*}}/g, (_, expr) => {
      try { return expr.split('.').reduce((acc: { [x: string]: any }, k: string | number) => acc?.[k], ctx) ?? '' } catch { return '' }
    })
  }
  if (Array.isArray(input)) return input.map((v) => template(v, ctx))
  if (typeof input === 'object') return Object.fromEntries(Object.entries(input).map(([k, v]) => [k, template(v, ctx)]))
  return input
}
