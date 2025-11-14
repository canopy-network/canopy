import { templateFns } from "./templaterFunctions";

const banned =
  /(constructor|prototype|__proto__|globalThis|window|document|import|Function|eval)\b/;

function splitArgs(src: string): string[] {
  // divide por comas ignorando comillas y anidación <...>, (...), {{...}}
  const out: string[] = [];
  let cur = "";
  let depthAngle = 0,
    depthParen = 0,
    depthMustache = 0;
  let inS = false,
    inD = false;

  for (let i = 0; i < src.length; i++) {
    const ch = src[i],
      prev = src[i - 1];

    if (!inS && !inD) {
      if (ch === "<") depthAngle++;
      else if (ch === ">") depthAngle = Math.max(0, depthAngle - 1);
      else if (ch === "(") depthParen++;
      else if (ch === ")") depthParen = Math.max(0, depthParen - 1);
      else if (ch === "{" && src[i + 1] === "{") {
        depthMustache++;
        i++;
        cur += "{{";
        continue;
      } else if (ch === "}" && src[i + 1] === "}") {
        depthMustache = Math.max(0, depthMustache - 1);
        i++;
        cur += "}}";
        continue;
      }
    }
    if (ch === "'" && !inD && prev !== "\\") inS = !inS;
    else if (ch === '"' && !inS && prev !== "\\") inD = !inD;

    if (
      ch === "," &&
      !inS &&
      !inD &&
      depthAngle === 0 &&
      depthParen === 0 &&
      depthMustache === 0
    ) {
      out.push(cur.trim());
      cur = "";
      continue;
    }
    cur += ch;
  }
  if (cur.trim() !== "") out.push(cur.trim());
  return out;
}

// evalúa una expresión JS segura usando contexto como argumentos
function evalJsExpression(expr: string, ctx: any): any {
  if (banned.test(expr)) throw new Error("templater: forbidden token");
  const argNames = Object.keys(ctx);
  const argVals = Object.values(ctx);
  // return ( ...expr... );
  // eslint-disable-next-line no-new-func
  const fn = new Function(...argNames, `return (${expr});`);
  return fn(...argVals);
}

function replaceBalanced(
  input: string,
  resolver: (expr: string) => string,
): string {
  let out = "";
  let i = 0;
  while (i < input.length) {
    const start = input.indexOf("{{", i);
    if (start === -1) {
      out += input.slice(i);
      break;
    }
    // texto antes del bloque
    out += input.slice(i, start);

    // buscar cierre balanceado
    let j = start + 2;
    let depth = 1;
    while (j < input.length && depth > 0) {
      if (input.startsWith("{{", j)) {
        depth += 1;
        j += 2;
        continue;
      }
      if (input.startsWith("}}", j)) {
        depth -= 1;
        j += 2;
        if (depth === 0) break;
        continue;
      }
      j += 1;
    }

    // si no se cerró, copia resto y corta
    if (depth !== 0) {
      out += input.slice(start);
      break;
    }

    const exprRaw = input.slice(start + 2, j - 2);
    const replacement = resolver(exprRaw.trim());
    out += replacement;
    i = j;
  }
  return out;
}

/** Evalúa una expresión: función tipo fn<...> o ruta a datos a.b.c */
function evalExpr(expr: string, ctx: any): any {
  if (banned.test(expr)) throw new Error("templater: forbidden token");

  // 1) sintaxis: fn<arg1, arg2, ...>
  const angleCall = expr.match(/^(\w+)<([\s\S]*)>$/);
  if (angleCall) {
    const [, fnName, rawArgs] = angleCall;
    const argStrs = splitArgs(rawArgs);
    const args = argStrs.map((a) => template(a, ctx)); // cada arg puede tener {{...}} anidado
    const fn = (templateFns as Record<string, any>)[fnName];
    if (typeof fn !== "function") return "";
    try {
      return fn(...args);
    } catch {
      return "";
    }
  }

  // 2) sintaxis: fn(arg1, arg2, ...)
  const parenCall = expr.match(/^(\w+)\(([\s\S]*)\)$/);
  if (parenCall) {
    const [, fnName, rawArgs] = parenCall;
    const argStrs = splitArgs(rawArgs);
    const args = argStrs.map((a) => {
      // si el arg es una expresión/plantilla, resuélvela; si es literal, evalúala
      if (/{{.*}}/.test(a)) return template(a, ctx);
      try {
        return evalJsExpression(a, ctx);
      } catch {
        return template(a, ctx);
      }
    });
    const fn = (templateFns as Record<string, any>)[fnName];
    if (typeof fn !== "function") return "";
    try {
      return fn(...args);
    } catch {
      return "";
    }
  }

  // 3) expresión JS libre (p. ej. form.amount * 0.05, Object.keys(ds)...)
  try {
    return evalJsExpression(expr, ctx);
  } catch {
    // 4) ruta normal: a.b.c
    const path = expr
      .split(".")
      .map((s) => s.trim())
      .filter(Boolean);
    let val: any = ctx;
    for (const p of path) val = val?.[p];
    return val == null || typeof val === "object" ? val : String(val);
  }
}

export function resolveTemplatesDeep<T = any>(obj: T, ctx: any): T {
  if (obj == null) return obj as T;
  if (typeof obj === "string") return templateAny(obj, ctx) as any;
  if (Array.isArray(obj))
    return obj.map((x) => resolveTemplatesDeep(x, ctx)) as any;
  if (typeof obj === "object") {
    const out: any = {};
    for (const [k, v] of Object.entries(obj))
      out[k] = resolveTemplatesDeep(v, ctx);
    return out;
  }
  return obj as T;
}

export function extractTemplateDeps(str: string): string[] {
  if (typeof str !== "string" || !str.includes("{{")) return [];

  const blocks: string[] = [];
  const reBlock = /\{\{([\s\S]*?)\}\}/g;
  let m: RegExpExecArray | null;
  while ((m = reBlock.exec(str))) blocks.push(m[1]);

  const ROOTS = ["form", "chain", "params", "fees", "account", "session", "ds"];
  const rootGroup = ROOTS.join("|");
  const rePath = new RegExp(
    `\\b(?:${rootGroup})\\s*(?:\\?\\.)?(?:\\.[A-Za-z0-9_]+|\\[(?:"[^"]+"|'[^']+')\\])+`,
    "g",
  );

  const results: string[] = [];

  for (const code of blocks) {
    const found = code.match(rePath) || [];
    for (let raw of found) {
      raw = raw.replace(/\?\./g, ".");
      raw = raw.replace(
        /\[("([^"]+)"|'([^']+)')\]/g,
        (_s, _g1, g2, g3) => `.${g2 ?? g3}`,
      );
      results.push(raw);
    }
  }

  return Array.from(new Set(results));
}

export function collectDepsFromObject(obj: any): string[] {
  const acc = new Set<string>();
  const walk = (node: any) => {
    if (node == null) return;
    if (typeof node === "string") {
      extractTemplateDeps(node).forEach((d) => acc.add(d));
      return;
    }
    if (Array.isArray(node)) {
      node.forEach(walk);
      return;
    }
    if (typeof node === "object") {
      Object.values(node).forEach(walk);
      return;
    }
  };
  walk(obj);
  return Array.from(acc);
}

export function template(str: unknown, ctx: any): string {
  if (str == null) return "";
  const input = String(str);

  const out = replaceBalanced(input, (expr) => evalExpr(expr, ctx));
  return out;
}

export function templateAny(s: any, ctx: any) {
  if (typeof s !== "string") return s;
  const m = s.match(/^\s*{{\s*([\s\S]+?)\s*}}\s*$/);
  if (m) return evalExpr(m[1], ctx);
  return s.replace(/{{\s*([\s\S]+?)\s*}}/g, (_, e) => {
    const v = evalExpr(e, ctx);
    return v == null ? "" : String(v);
  });
}

export function templateBool(tpl: any, ctx: Record<string, any> = {}): boolean {
  const v = templateAny(tpl, ctx);
  return toBool(v);
}

export function toBool(v: any): boolean {
  if (typeof v === "boolean") return v;
  if (typeof v === "number") return v !== 0 && !Number.isNaN(v);
  if (v == null) return false;
  if (Array.isArray(v)) return v.length > 0;
  if (typeof v === "object") return Object.keys(v).length > 0;
  const s = String(v).trim().toLowerCase();
  if (
    s === "" ||
    s === "0" ||
    s === "false" ||
    s === "no" ||
    s === "off" ||
    s === "null" ||
    s === "undefined"
  )
    return false;
  return true;
}
