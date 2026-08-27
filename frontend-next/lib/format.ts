// Formatting helpers — ported from Frontend/proto.js (PRX = 1e6 uPRX).
export function fmtPRX(n: bigint | number | string | null | undefined): string {
  if (n === null || n === undefined || n === "") return "—";
  const x = Number(n) / 1_000_000;
  if (x >= 1e9) return (x / 1e9).toFixed(2) + "B";
  if (x >= 1e6) return (x / 1e6).toFixed(2) + "M";
  if (x >= 1000) return (x / 1000).toFixed(2) + "k";
  if (x >= 1) return x.toFixed(2);
  return x.toFixed(6);
}

export function fmtA(n: number | null | undefined): string {
  if (n === null || n === undefined) return "—";
  if (n >= 1e9) return (n / 1e9).toFixed(2) + "B";
  if (n >= 1e6) return (n / 1e6).toFixed(2) + "M";
  if (n >= 1e3) return (n / 1e3).toFixed(1) + "k";
  return String(n);
}

export function h2b(hex: string): Uint8Array {
  const s = hex.trim().toLowerCase();
  if (s.length % 2) throw new Error("Odd hex");
  const o = new Uint8Array(s.length / 2);
  for (let i = 0; i < o.length; i++) o[i] = parseInt(s.slice(i * 2, i * 2 + 2), 16);
  return o;
}

export function b2h(b: Uint8Array): string {
  return Array.from(b)
    .map((x) => x.toString(16).padStart(2, "0"))
    .join("");
}
